package main

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx   context.Context
	store *Store
	mgr   *Manager
	term  *termHub
}

func NewApp(store *Store, mgr *Manager) *App {
	return &App{store: store, mgr: mgr, term: newTermHub()}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) shutdown(_ context.Context) {
	a.term.CloseAll()
	a.mgr.Close()
}

type Meta struct {
	DataDir string `json:"dataDir"`
}

func (a *App) Meta() Meta {
	return Meta{DataDir: a.store.Dir}
}

func (a *App) ListServers() ([]Server, error) {
	return a.store.ListServers()
}

type ServerInput struct {
	Name         string  `json:"name"`
	Host         string  `json:"host"`
	Port         int     `json:"port"`
	Username     string  `json:"username"`
	AuthType     string  `json:"authType"`
	Password     *string `json:"password"`
	PrivateKey   *string `json:"privateKey"`
	Passphrase   *string `json:"passphrase"`
	HostKey      string  `json:"hostKey"`
	ClearHostKey bool    `json:"clearHostKey"`
}

func (in ServerInput) toWrite() (ServerWrite, error) {
	port := in.Port
	if port == 0 {
		port = 22
	}
	hk, err := parseHostKeyB64(in.HostKey)
	if err != nil {
		return ServerWrite{}, err
	}
	if in.ClearHostKey {
		hk = []byte{}
	}
	return ServerWrite{
		Name:       in.Name,
		Host:       in.Host,
		Port:       port,
		Username:   in.Username,
		AuthType:   in.AuthType,
		Password:   in.Password,
		PrivateKey: in.PrivateKey,
		Passphrase: in.Passphrase,
		HostKey:    hk,
	}, nil
}

func (a *App) CreateServer(in ServerInput) (int64, error) {
	w, err := in.toWrite()
	if err != nil {
		return 0, err
	}
	id, err := a.store.CreateServer(w)
	if err != nil {
		return 0, err
	}
	a.store.AddAudit(&id, "server.create", w.Name+" "+w.Host)
	return id, nil
}

func (a *App) UpdateServer(id int64, in ServerInput) error {
	if id <= 0 {
		return errors.New("invalid id")
	}
	w, err := in.toWrite()
	if err != nil {
		return err
	}
	if err := a.store.UpdateServer(id, w); err != nil {
		return err
	}
	a.mgr.Drop(id)
	a.store.AddAudit(&id, "server.update", w.Name+" "+w.Host)
	return nil
}

func (a *App) DeleteServer(id int64, confirm bool) error {
	if !confirm {
		return errors.New("confirm required")
	}
	if id <= 0 {
		return errors.New("invalid id")
	}
	if err := a.store.DeleteServer(id); err != nil {
		return err
	}
	a.mgr.Drop(id)
	a.store.AddAudit(&id, "server.delete", "deleted")
	return nil
}

type TestInput struct {
	ServerID      int64   `json:"serverId"`
	Host          string  `json:"host"`
	Port          int     `json:"port"`
	Username      string  `json:"username"`
	AuthType      string  `json:"authType"`
	Password      *string `json:"password"`
	PrivateKey    *string `json:"privateKey"`
	Passphrase    *string `json:"passphrase"`
	AcceptHostKey bool    `json:"acceptHostKey"`
}

type TestResult struct {
	OK          bool   `json:"ok"`
	Fingerprint   string `json:"fingerprint"`
	HostKey       string `json:"hostKey"`
	Error         string `json:"error"`
	NeedHostKey   bool   `json:"needHostKey"`
	HostChanged   bool   `json:"hostChanged"`
	FingerprintUI string `json:"fingerprintUI"`
}

func (a *App) TestConnection(in TestInput) TestResult {
	port := in.Port
	if port == 0 {
		port = 22
	}
	var sv Server
	var sec secrets
	saveID := in.ServerID

	if in.ServerID > 0 {
		var err error
		sv, sec, err = a.store.GetServer(in.ServerID)
		if err != nil {
			return TestResult{Error: err.Error()}
		}
		if in.Password != nil && *in.Password != "" {
			sec.Password = *in.Password
		}
		if in.PrivateKey != nil && strings.TrimSpace(*in.PrivateKey) != "" {
			sec.PrivateKey = []byte(*in.PrivateKey)
		}
		if in.Passphrase != nil {
			sec.Passphrase = *in.Passphrase
		}
	} else {
		sv = Server{
			Host:     strings.TrimSpace(in.Host),
			Port:     port,
			Username: strings.TrimSpace(in.Username),
			AuthType: in.AuthType,
		}
		if sv.Host == "" || sv.Username == "" {
			return TestResult{Error: "host and username are required"}
		}
		if in.Password != nil {
			sec.Password = *in.Password
		}
		if in.PrivateKey != nil {
			sec.PrivateKey = []byte(*in.PrivateKey)
		}
		if in.Passphrase != nil {
			sec.Passphrase = *in.Passphrase
		}
	}

	fp, hk, err := a.mgr.Test(sv, sec, in.AcceptHostKey)
	if err != nil {
		var need *HostKeyNeededError
		var mismatch *HostKeyMismatchError
		if errors.As(err, &need) {
			return TestResult{
				NeedHostKey:   true,
				FingerprintUI: need.Fingerprint,
				HostKey:       hostKeyB64(need.Key.Marshal()),
				Error:         "unrecognized host key",
			}
		}
		if errors.As(err, &mismatch) {
			return TestResult{
				HostChanged:   true,
				FingerprintUI: mismatch.Fingerprint,
				Error:         mismatch.Error(),
			}
		}
		return TestResult{Error: err.Error()}
	}
	if saveID > 0 && len(hk) > 0 {
		_ = a.store.SaveHostKey(saveID, hk)
	}
	var auditID *int64
	if saveID > 0 {
		auditID = &saveID
	}
	a.store.AddAudit(auditID, "ssh.test", sv.Host)
	return TestResult{
		OK:          true,
		Fingerprint: fp,
		HostKey:     hostKeyB64(hk),
	}
}

func (a *App) ListProjects(serverID int64) ([]Project, error) {
	if serverID <= 0 {
		return nil, errors.New("serverId is required")
	}
	return a.store.ListProjects(serverID)
}

type ProjectInput struct {
	ServerID   int64  `json:"serverId"`
	Name       string `json:"name"`
	RemotePath string `json:"remotePath"`
}

func (a *App) CreateProject(in ProjectInput) (int64, error) {
	if err := a.mgr.StatDir(in.ServerID, in.RemotePath); err != nil {
		return 0, err
	}
	id, err := a.store.CreateProject(in.ServerID, in.Name, in.RemotePath)
	if err != nil {
		return 0, err
	}
	sid := in.ServerID
	a.store.AddAudit(&sid, "project.create", in.Name+" "+absRemoteDir(in.RemotePath))
	return id, nil
}

func (a *App) UpdateProject(id int64, in ProjectInput) error {
	if id <= 0 {
		return errors.New("invalid id")
	}
	p, err := a.store.GetProject(id)
	if err != nil {
		return err
	}
	if err := a.mgr.StatDir(p.ServerID, in.RemotePath); err != nil {
		return err
	}
	if err := a.store.UpdateProject(id, in.Name, in.RemotePath); err != nil {
		return err
	}
	a.store.AddAudit(&p.ServerID, "project.update", in.Name+" "+absRemoteDir(in.RemotePath))
	return nil
}

func (a *App) DeleteProject(id int64, confirm bool) error {
	if !confirm {
		return errors.New("confirm required")
	}
	p, err := a.store.GetProject(id)
	if err != nil {
		return err
	}
	if err := a.store.DeleteProject(id); err != nil {
		return err
	}
	a.store.AddAudit(&p.ServerID, "project.delete", p.Name)
	return nil
}

type FileListResult struct {
	Root  string     `json:"root"`
	Files []FileInfo `json:"files"`
}

func (a *App) ListFiles(projectID int64, remotePath string) (FileListResult, error) {
	var out FileListResult
	p, err := a.store.GetProject(projectID)
	if err != nil {
		return out, err
	}
	list, err := a.mgr.List(p.ServerID, p.RemotePath, remotePath)
	if err != nil {
		return out, err
	}
	out.Root = p.RemotePath
	out.Files = list
	return out, nil
}

type FileContent struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func (a *App) ReadFile(projectID int64, remotePath string) (FileContent, error) {
	var out FileContent
	p, err := a.store.GetProject(projectID)
	if err != nil {
		return out, err
	}
	content, err := a.mgr.ReadFile(p.ServerID, p.RemotePath, remotePath)
	if err != nil {
		return out, err
	}
	a.store.AddAudit(&p.ServerID, "file.read", remotePath)
	out.Path = remotePath
	out.Content = content
	return out, nil
}

type FileWriteInput struct {
	ProjectID int64  `json:"projectId"`
	Path      string `json:"path"`
	Content   string `json:"content"`
}

func (a *App) WriteFile(in FileWriteInput) error {
	p, err := a.store.GetProject(in.ProjectID)
	if err != nil {
		return err
	}
	if err := a.mgr.WriteFile(p.ServerID, p.RemotePath, in.Path, []byte(in.Content)); err != nil {
		return err
	}
	a.store.AddAudit(&p.ServerID, "file.write", in.Path)
	return nil
}

type FileOpInput struct {
	ProjectID int64  `json:"projectId"`
	Path      string `json:"path"`
	Dir       string `json:"dir"`
	Name      string `json:"name"`
	Confirm   bool   `json:"confirm"`
}

func (a *App) Mkdir(in FileOpInput) error {
	p, err := a.store.GetProject(in.ProjectID)
	if err != nil {
		return err
	}
	dir := in.Dir
	if dir == "" {
		dir = in.Path
	}
	if err := a.mgr.Mkdir(p.ServerID, p.RemotePath, dir, in.Name); err != nil {
		return err
	}
	a.store.AddAudit(&p.ServerID, "file.mkdir", path.Join(dir, in.Name))
	return nil
}

func (a *App) CreateRemoteFile(in FileOpInput) error {
	p, err := a.store.GetProject(in.ProjectID)
	if err != nil {
		return err
	}
	dir := in.Dir
	if dir == "" {
		dir = in.Path
	}
	if err := a.mgr.CreateFile(p.ServerID, p.RemotePath, dir, in.Name); err != nil {
		return err
	}
	a.store.AddAudit(&p.ServerID, "file.create", path.Join(dir, in.Name))
	return nil
}

func (a *App) RenameFile(in FileOpInput) error {
	p, err := a.store.GetProject(in.ProjectID)
	if err != nil {
		return err
	}
	if err := a.mgr.Rename(p.ServerID, p.RemotePath, in.Path, in.Name); err != nil {
		return err
	}
	a.store.AddAudit(&p.ServerID, "file.rename", in.Path+" -> "+in.Name)
	return nil
}

func (a *App) DeleteFile(in FileOpInput) error {
	if !in.Confirm {
		return errors.New("confirm required")
	}
	p, err := a.store.GetProject(in.ProjectID)
	if err != nil {
		return err
	}
	if err := a.mgr.Remove(p.ServerID, p.RemotePath, in.Path); err != nil {
		return err
	}
	a.store.AddAudit(&p.ServerID, "file.delete", in.Path)
	return nil
}

func (a *App) DownloadFile(projectID int64, remotePath string) error {
	p, err := a.store.GetProject(projectID)
	if err != nil {
		return err
	}
	name, rc, _, err := a.mgr.Download(p.ServerID, p.RemotePath, remotePath)
	if err != nil {
		return err
	}
	defer rc.Close()

	save, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		DefaultFilename: name,
		Title:           "Save file",
	})
	if err != nil {
		return err
	}
	if save == "" {
		return nil
	}
	f, err := os.Create(save)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(f, rc); err != nil {
		return err
	}
	a.store.AddAudit(&p.ServerID, "file.download", remotePath)
	return nil
}

func (a *App) UploadFile(projectID int64, dir string) error {
	p, err := a.store.GetProject(projectID)
	if err != nil {
		return err
	}
	local, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Upload file",
	})
	if err != nil {
		return err
	}
	if local == "" {
		return nil
	}
	f, err := os.Open(local)
	if err != nil {
		return err
	}
	defer f.Close()
	name := path.Base(filepath.ToSlash(local))
	if err := a.mgr.Upload(p.ServerID, p.RemotePath, dir, name, f, maxUploadBytes); err != nil {
		return err
	}
	a.store.AddAudit(&p.ServerID, "file.upload", path.Join(dir, name))
	return nil
}

func (a *App) ListAudit() ([]Audit, error) {
	return a.store.ListAudit()
}

func (a *App) ClearAudit(confirm bool) error {
	if !confirm {
		return errors.New("confirm required")
	}
	return a.store.ClearAudit()
}

func (a *App) StartTerminal(projectID int64, cols, rows int) (string, error) {
	proj, err := a.store.GetProject(projectID)
	if err != nil {
		return "", err
	}
	client, err := a.mgr.OpenSession(proj.ServerID)
	if err != nil {
		return "", err
	}
	sess, stdin, stdout, err := startPTY(client, proj.RemotePath, cols, rows)
	if err != nil {
		return "", err
	}
	a.store.AddAudit(&proj.ServerID, "terminal.start", proj.Name+" "+proj.RemotePath)
	id := a.term.Register(sess, stdin)
	go a.term.pump(a.ctx, id, stdout)
	return id, nil
}

func (a *App) TerminalInput(sessionID, data string) error {
	return a.term.Write(sessionID, data)
}

func (a *App) TerminalResize(sessionID string, cols, rows int) error {
	return a.term.Resize(sessionID, cols, rows)
}

func (a *App) StopTerminal(sessionID string) {
	a.term.Stop(sessionID)
}

// TerminalOutputEvent is emitted as "terminal:output" with payload {sessionId, dataB64}.
type TerminalOutputEvent struct {
	SessionID string `json:"sessionId"`
	DataB64   string `json:"dataB64"`
} // emitted on event "terminal:output"

func encodeTermChunk(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}
