package main

import (
	"archive/zip"
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/pkg/sftp"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	maxStagedFiles  = 20000
	previewItemCap  = 400
	uploadItemLimit = 20
)

type UploadResult struct {
	OK      bool     `json:"ok"`
	Done    int      `json:"done"`
	Failed  int      `json:"failed"`
	Errors  []string `json:"errors"`
	Skipped bool     `json:"skipped"`
	Dest    string   `json:"dest"`
	Label   string   `json:"label"`
	Kind    string   `json:"kind"`
	Bytes   int64    `json:"bytes"`
	Time    string   `json:"time"`
}

type UploadItem struct {
	Rel  string `json:"rel"`
	Size int64  `json:"size"`
}

type UploadPreview struct {
	Kind    string       `json:"kind"`
	Dest    string       `json:"dest"`
	Files   int          `json:"files"`
	Folders int          `json:"folders"`
	Bytes   int64        `json:"bytes"`
	Items   []UploadItem `json:"items"`
	Skipped bool         `json:"skipped"`
	Error   string       `json:"error"`
}

type stagedFile struct {
	Rel   string
	Size  int64
	Local string
}

type stagedUpload struct {
	ProjectID int64
	Dest      string
	Kind      string
	Local     string
	Files     []stagedFile
	Folders   int
	Bytes     int64
}

type uploadProgress struct {
	Done  int    `json:"done"`
	Total int    `json:"total"`
	Path  string `json:"path"`
	Error string `json:"error"`
}

func skipUploadName(name string) bool {
	if name == "" || strings.HasPrefix(name, "._") {
		return true
	}
	switch name {
	case ".DS_Store", "Thumbs.db", "desktop.ini":
		return true
	}
	return false
}

func skipUploadRel(rel string) bool {
	rel = strings.TrimPrefix(filepath.ToSlash(rel), "/")
	if rel == "" || strings.Contains(rel, "..") {
		return true
	}
	for _, p := range strings.Split(rel, "/") {
		if p == "__MACOSX" || skipUploadName(p) {
			return true
		}
	}
	return false
}

func uniqueParentDirs(files []stagedFile) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, f := range files {
		d := path.Dir(strings.TrimPrefix(filepath.ToSlash(f.Rel), "/"))
		for d != "." && d != "" && d != "/" {
			if _, ok := seen[d]; ok {
				break
			}
			seen[d] = struct{}{}
			out = append(out, d)
			d = path.Dir(d)
		}
	}
	return out
}

func (a *App) setStaged(st *stagedUpload) {
	a.stageMu.Lock()
	a.staged = st
	a.stageMu.Unlock()
}

func (a *App) getStaged() *stagedUpload {
	a.stageMu.Lock()
	defer a.stageMu.Unlock()
	return a.staged
}

func (a *App) CancelUpload() {
	a.setStaged(nil)
}

func (a *App) previewFromStaged(st *stagedUpload) UploadPreview {
	if st == nil {
		return UploadPreview{Skipped: true}
	}
	n := len(st.Files)
	if n > previewItemCap {
		n = previewItemCap
	}
	out := UploadPreview{
		Kind:    st.Kind,
		Dest:    st.Dest,
		Files:   len(st.Files),
		Folders: st.Folders,
		Bytes:   st.Bytes,
		Items:   make([]UploadItem, 0, n),
	}
	for i := 0; i < n; i++ {
		f := st.Files[i]
		out.Items = append(out.Items, UploadItem{Rel: f.Rel, Size: f.Size})
	}
	return out
}

func (a *App) stageForProject(projectID int64, dir string) (Project, UploadPreview, error) {
	p, err := a.store.GetProject(projectID)
	if err != nil {
		return p, UploadPreview{Error: err.Error()}, err
	}
	dest, err := a.resolveActive(p, dir)
	if err != nil {
		return p, UploadPreview{Error: err.Error()}, err
	}
	return p, UploadPreview{Dest: dest}, nil
}

func (a *App) StageUploadFile(projectID int64, dir string) UploadPreview {
	p, prev, err := a.stageForProject(projectID, dir)
	if err != nil {
		return prev
	}
	local, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Upload file",
	})
	if err != nil {
		return UploadPreview{Error: err.Error()}
	}
	if local == "" {
		return UploadPreview{Skipped: true}
	}
	info, err := os.Stat(local)
	if err != nil || info.IsDir() {
		return UploadPreview{Error: "could not read file"}
	}
	if skipUploadName(info.Name()) {
		return UploadPreview{Error: "that file is skipped"}
	}
	st := &stagedUpload{
		ProjectID: p.ID,
		Dest:      prev.Dest,
		Kind:      "file",
		Local:     local,
		Files:     []stagedFile{{Rel: path.Base(filepath.ToSlash(local)), Size: info.Size(), Local: local}},
		Bytes:     info.Size(),
	}
	a.setStaged(st)
	return a.previewFromStaged(st)
}

func (a *App) StageUploadFolder(projectID int64, dir string) UploadPreview {
	p, prev, err := a.stageForProject(projectID, dir)
	if err != nil {
		return prev
	}
	local, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Upload folder",
	})
	if err != nil {
		return UploadPreview{Error: err.Error()}
	}
	if local == "" {
		return UploadPreview{Skipped: true}
	}
	st, err := scanLocalFolder(p.ID, prev.Dest, local)
	if err != nil {
		return UploadPreview{Error: err.Error()}
	}
	a.setStaged(st)
	return a.previewFromStaged(st)
}

func (a *App) StageUploadZip(projectID int64, dir string) UploadPreview {
	p, prev, err := a.stageForProject(projectID, dir)
	if err != nil {
		return prev
	}
	local, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   "Upload ZIP",
		Filters: []runtime.FileFilter{{DisplayName: "ZIP", Pattern: "*.zip"}},
	})
	if err != nil {
		return UploadPreview{Error: err.Error()}
	}
	if local == "" {
		return UploadPreview{Skipped: true}
	}
	st, err := scanZip(p.ID, prev.Dest, local)
	if err != nil {
		return UploadPreview{Error: err.Error()}
	}
	a.setStaged(st)
	return a.previewFromStaged(st)
}

func scanLocalFolder(projectID int64, dest, local string) (*stagedUpload, error) {
	st := &stagedUpload{ProjectID: projectID, Dest: dest, Kind: "folder", Local: local}
	dirs := map[string]struct{}{}
	err := filepath.Walk(local, func(fp string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		rel, e := filepath.Rel(local, fp)
		if e != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		if info.IsDir() {
			if skipUploadName(info.Name()) || skipUploadRel(rel) {
				return filepath.SkipDir
			}
			dirs[rel] = struct{}{}
			return nil
		}
		if skipUploadRel(rel) {
			return nil
		}
		if len(st.Files) >= maxStagedFiles {
			return errors.New("too many files")
		}
		st.Files = append(st.Files, stagedFile{Rel: rel, Size: info.Size(), Local: fp})
		st.Bytes += info.Size()
		if d := path.Dir(rel); d != "." && d != "" {
			dirs[d] = struct{}{}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	st.Folders = len(dirs)
	return st, nil
}

func scanZip(projectID int64, dest, local string) (*stagedUpload, error) {
	zr, err := zip.OpenReader(local)
	if err != nil {
		return nil, errors.New("could not open zip")
	}
	defer zr.Close()
	st := &stagedUpload{ProjectID: projectID, Dest: dest, Kind: "zip", Local: local}
	dirs := map[string]struct{}{}
	for _, f := range zr.File {
		rel := strings.TrimPrefix(filepath.ToSlash(f.Name), "/")
		if strings.HasSuffix(rel, "/") || f.FileInfo().IsDir() {
			rel = strings.TrimSuffix(rel, "/")
			if rel != "" && !skipUploadRel(rel) {
				dirs[rel] = struct{}{}
			}
			continue
		}
		if skipUploadRel(rel) {
			continue
		}
		if len(st.Files) >= maxStagedFiles {
			return nil, errors.New("too many files")
		}
		st.Files = append(st.Files, stagedFile{Rel: rel, Size: int64(f.UncompressedSize64)})
		st.Bytes += int64(f.UncompressedSize64)
		if d := path.Dir(rel); d != "." && d != "" {
			dirs[d] = struct{}{}
		}
	}
	st.Folders = len(dirs)
	return st, nil
}

func openZipEntry(zr *zip.ReadCloser, rel string) (io.ReadCloser, error) {
	if zr == nil {
		return nil, errors.New("zip not open")
	}
	want := filepath.ToSlash(rel)
	for _, f := range zr.File {
		if filepath.ToSlash(f.Name) == want {
			return f.Open()
		}
	}
	return nil, errors.New("missing zip entry")
}

func (a *App) ConfirmUpload() UploadResult {
	st := a.getStaged()
	if st == nil || len(st.Files) == 0 {
		return UploadResult{Skipped: true, Errors: []string{"nothing to upload"}}
	}
	p, err := a.store.GetProject(st.ProjectID)
	if err != nil {
		return UploadResult{Errors: []string{err.Error()}}
	}
	dest := st.Dest
	if dest == "" {
		dest = p.RemotePath
	}
	if err := a.mgr.EnsureDirRaw(p.ServerID, dest); err != nil {
		return UploadResult{Errors: []string{err.Error()}}
	}
	for _, d := range uniqueParentDirs(st.Files) {
		if err := a.mgr.MkdirP(p.ServerID, dest, d); err != nil {
			return UploadResult{Errors: []string{"mkdir " + d + ": " + err.Error()}}
		}
	}

	total := len(st.Files)
	workers := uploadWorkers
	if total < workers {
		workers = total
	}
	if workers < 1 {
		workers = 1
	}
	type worker struct {
		cli    *sftp.Client
		closer func()
		zr     *zip.ReadCloser
	}
	var pool []worker
	for i := 0; i < workers; i++ {
		cli, closer, err := a.mgr.NewSFTP(p.ServerID)
		if err != nil {
			continue
		}
		w := worker{cli: cli, closer: closer}
		if st.Kind == "zip" {
			zr, zerr := zip.OpenReader(st.Local)
			if zerr != nil {
				closer()
				continue
			}
			w.zr = zr
		}
		pool = append(pool, w)
	}
	if len(pool) == 0 {
		return UploadResult{Errors: []string{"could not open upload connection"}}
	}
	defer func() {
		for _, w := range pool {
			if w.zr != nil {
				_ = w.zr.Close()
			}
			w.closer()
		}
	}()

	jobs := make(chan stagedFile)
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		out  UploadResult
		done int
	)
	runtime.EventsEmit(a.ctx, "upload:progress", uploadProgress{Done: 0, Total: total})
	for _, w := range pool {
		w := w
		wg.Add(1)
		go func() {
			defer wg.Done()
			for f := range jobs {
				err := putStagedFile(w.cli, w.zr, dest, st, f)
				mu.Lock()
				done++
				prog := uploadProgress{Done: done, Total: total, Path: f.Rel}
				if err != nil {
					out.Failed++
					prog.Error = err.Error()
					if len(out.Errors) < uploadItemLimit {
						out.Errors = append(out.Errors, f.Rel)
					}
				} else {
					out.Done++
				}
				runtime.EventsEmit(a.ctx, "upload:progress", prog)
				mu.Unlock()
			}
		}()
	}
	for _, f := range st.Files {
		jobs <- f
	}
	close(jobs)
	wg.Wait()
	out.OK = out.Failed == 0
	out.Dest = dest
	out.Kind = st.Kind
	out.Bytes = st.Bytes
	out.Time = time.Now().UTC().Format(time.RFC3339)
	if len(st.Files) == 1 {
		out.Label = st.Files[0].Rel
	} else if st.Kind == "zip" {
		out.Label = path.Base(filepath.ToSlash(st.Local))
	} else {
		out.Label = path.Base(filepath.ToSlash(st.Local))
		if out.Label == "" || out.Label == "." {
			out.Label = st.Kind
		}
	}
	status := "ok"
	if !out.OK {
		status = "failed"
	}
	a.store.RecordUpload(p.ID, out.Label, out.Bytes, status)
	a.setStaged(nil)
	a.store.AddAudit(&p.ServerID, "file.upload", dest)
	return out
}

func putStagedFile(cli *sftp.Client, zr *zip.ReadCloser, dest string, st *stagedUpload, f stagedFile) error {
	var (
		src io.ReadCloser
		err error
	)
	if st.Kind == "zip" {
		src, err = openZipEntry(zr, f.Rel)
	} else {
		src, err = os.Open(f.Local)
	}
	if err != nil {
		return err
	}
	defer src.Close()
	return uploadOn(cli, dest, f.Rel, src)
}
