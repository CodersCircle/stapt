package main

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx     context.Context
	store   *Store
	mgr     *Manager
	term    *termHub
	stageMu sync.Mutex
	staged  *stagedUpload
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
	if saveID > 0 {
		a.store.TouchLastConnected(saveID)
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
	a.store.PruneJunkProjects(serverID)
	return a.store.ListProjects(serverID)
}

type ProjectInput struct {
	ServerID   int64  `json:"serverId"`
	Name       string `json:"name"`
	RemotePath string `json:"remotePath"`
	Kind       string `json:"kind"`
	Status     string `json:"status"`
}

func (a *App) CreateProject(in ProjectInput) (int64, error) {
	if err := a.mgr.StatDir(in.ServerID, in.RemotePath); err != nil {
		return 0, err
	}
	kind := in.Kind
	if kind == "" {
		kind = "project"
	}
	status := in.Status
	if status == "" {
		status = "ready"
	}
	tech := detectTech(a.mgr, in.ServerID, absRemoteDir(in.RemotePath))
	id, err := a.store.CreateProjectFull(in.ServerID, in.Name, in.RemotePath, kind, status, tech)
	if err != nil {
		return 0, err
	}
	sid := in.ServerID
	a.store.AddAudit(&sid, "project.create", in.Name)
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
	if p.RemotePath == "" {
		return errors.New("project root is missing")
	}
	if err := a.mgr.StatDir(p.ServerID, in.RemotePath); err != nil {
		return err
	}
	status := in.Status
	if status == "" {
		status = "ready"
	}
	if err := a.store.UpdateProjectMeta(id, in.Name, in.RemotePath, in.Kind, status); err != nil {
		return err
	}
	a.store.AddAudit(&p.ServerID, "project.update", in.Name)
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
	if strings.TrimSpace(remotePath) == "" {
		remotePath = p.ActivePath()
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
	sess, stdin, stdout, err := startPTY(client, proj.ActivePath(), cols, rows)
	if err != nil {
		return "", err
	}
	a.store.AddAudit(&proj.ServerID, "terminal.start", proj.Name+" "+proj.ActivePath())
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

func (a *App) ParseSSHCommand(raw string) (ParsedSSH, error) {
	return parseSSHCommand(raw)
}

func (a *App) DiscoverProjects(serverID int64) ([]Project, error) {
	if serverID <= 0 {
		return nil, errors.New("invalid id")
	}
	sv, _, err := a.store.GetServer(serverID)
	if err != nil {
		return nil, err
	}
	found, err := a.mgr.Discover(serverID, sv.Username)
	if err != nil {
		return nil, err
	}
	for _, d := range found {
		if isJunkDomainName(d.Name) {
			continue
		}
		_, _ = a.store.UpsertDiscovered(serverID, d.Name, d.Path, d.Kind, d.Status, d.Tech)
	}
	a.store.PruneJunkProjects(serverID)
	a.store.AddAudit(&serverID, "discover", "scan complete")
	return a.store.ListProjects(serverID)
}

type ConnectDiscoverResult struct {
	Test     TestResult `json:"test"`
	Projects []Project  `json:"projects"`
}

func (a *App) ConnectAndDiscover(in TestInput) ConnectDiscoverResult {
	res := a.TestConnection(in)
	out := ConnectDiscoverResult{Test: res}
	if !res.OK {
		return out
	}
	sid := in.ServerID
	if sid > 0 {
		list, err := a.DiscoverProjects(sid)
		if err != nil {
			out.Test.Error = err.Error()
			out.Test.OK = false
			return out
		}
		out.Projects = list
	}
	return out
}

func (a *App) ListQuickCommands(projectID int64) ([]QuickCommand, error) {
	p, err := a.store.GetProject(projectID)
	if err != nil {
		return nil, err
	}
	tech := p.Tech
	if tech == "" {
		tech = detectTech(a.mgr, p.ServerID, p.ActivePath())
		_, _ = a.store.UpsertDiscovered(p.ServerID, p.Name, p.RemotePath, p.Kind, p.Status, tech)
	}
	return commandsForTech(tech), nil
}

func (a *App) RunQuickCommand(projectID int64, key string, confirm bool) CommandResult {
	p, err := a.store.GetProject(projectID)
	if err != nil {
		return CommandResult{Error: err.Error()}
	}
	cmd, destructive, err := commandLine(key)
	if err != nil {
		return CommandResult{Error: err.Error()}
	}
	if destructive && !confirm {
		return CommandResult{Error: "confirm required"}
	}
	out, err := a.mgr.Exec(p.ServerID, p.ActivePath(), cmd)
	if err != nil {
		msg := err.Error()
		if looksSecret(msg) {
			msg = "command failed"
		}
		return CommandResult{Output: out, Error: msg}
	}
	a.store.AddAudit(&p.ServerID, "quick."+key, p.Name)
	return CommandResult{OK: true, Output: out}
}

func (a *App) ListNotes(projectID int64) ([]Note, error) {
	if projectID <= 0 {
		return a.store.ListAllNotes()
	}
	return a.store.ListNotes(projectID)
}

func (a *App) ListAllNotes() ([]Note, error) {
	return a.store.ListAllNotes()
}

type NoteInput struct {
	ProjectID int64  `json:"projectId"`
	Category  string `json:"category"`
	Title     string `json:"title"`
	Content   string `json:"content"`
}

func (a *App) CreateNote(in NoteInput) (int64, error) {
	id, err := a.store.CreateNote(in.ProjectID, in.Category, in.Title, in.Content)
	if err != nil {
		return 0, err
	}
	if in.ProjectID > 0 {
		if p, e := a.store.GetProject(in.ProjectID); e == nil {
			a.store.AddAudit(&p.ServerID, "note.create", in.Title)
		}
	}
	return id, nil
}

func (a *App) UpdateNote(id int64, in NoteInput) error {
	if err := a.store.UpdateNote(id, in.Category, in.Title, in.Content); err != nil {
		return err
	}
	return nil
}

func (a *App) DeleteNote(id int64, confirm bool) error {
	if !confirm {
		return errors.New("confirm required")
	}
	n, err := a.store.GetNote(id)
	if err != nil {
		return err
	}
	if err := a.store.DeleteNote(id); err != nil {
		return err
	}
	if n.ProjectID > 0 {
		if p, e := a.store.GetProject(n.ProjectID); e == nil {
			a.store.AddAudit(&p.ServerID, "note.delete", n.Title)
		}
	}
	return nil
}
