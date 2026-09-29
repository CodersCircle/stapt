package main

import (
	"bytes"
	"errors"
	"path"
	"strings"
)

const pathTestMarker = "stapt-path-test"

const pathTestHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Path Test Successful</title>
  <!-- stapt-path-test -->
  <style>
    body { margin: 0; min-height: 100vh; display: flex; align-items: center; justify-content: center;
      font-family: ui-sans-serif, system-ui, sans-serif; background: #14120a; color: #fef9c3; }
    .card { text-align: center; padding: 2.5rem; border: 1px solid rgba(250,204,21,.45); border-radius: 16px;
      background: #1c1910; max-width: 28rem; }
    h1 { margin: 0 0 .5rem; color: #facc15; font-size: 1.5rem; }
    p { margin: 0; color: #d6d3d1; font-size: .95rem; }
  </style>
</head>
<body>
  <div class="card">
    <h1>Path Test Successful ✓</h1>
    <p>This folder is the active remote path. Uploaded by STAPT / Keynest.</p>
  </div>
</body>
</html>
`

type PathTestResult struct {
	OK     bool   `json:"ok"`
	Dest   string `json:"dest"`
	URL    string `json:"url"`
	Error  string `json:"error"`
	Marker string `json:"marker"`
}

func (a *App) resolveActive(p Project, dir string) (string, error) {
	root := absRemoteDir(p.RemotePath)
	if root == "" {
		return "", errors.New("invalid project path")
	}
	if strings.TrimSpace(dir) == "" {
		dir = p.ActivePath()
	}
	return resolveRemote(root, dir)
}

func guessLiveURL(p Project, dest string) string {
	name := strings.TrimSpace(p.Name)
	if !strings.Contains(name, ".") {
		return ""
	}
	dest = normalizeRemote(dest)
	rel := ""
	if i := strings.Index(dest, "/public_html"); i >= 0 {
		rel = strings.TrimPrefix(dest[i+len("/public_html"):], "/")
	}
	u := "https://" + name
	if rel != "" {
		u += "/" + rel
	}
	return u + "/index.html"
}

func (a *App) SetActivePath(projectID int64, dir string) (string, error) {
	p, err := a.store.GetProject(projectID)
	if err != nil {
		return "", err
	}
	full, err := a.resolveActive(p, dir)
	if err != nil {
		return "", err
	}
	if err := a.mgr.StatDir(p.ServerID, full); err != nil {
		return "", err
	}
	if err := a.store.SetWorkPath(p.ID, full); err != nil {
		return "", err
	}
	a.store.AddAudit(&p.ServerID, "path.active", full)
	return full, nil
}

type SavePathInput struct {
	ProjectID int64  `json:"projectId"`
	Name      string `json:"name"`
	Dir       string `json:"dir"`
}

func (a *App) SaveRemotePath(in SavePathInput) (int64, error) {
	p, err := a.store.GetProject(in.ProjectID)
	if err != nil {
		return 0, err
	}
	full, err := a.resolveActive(p, in.Dir)
	if err != nil {
		return 0, err
	}
	if err := a.mgr.StatDir(p.ServerID, full); err != nil {
		return 0, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = path.Base(full)
	}
	id, err := a.store.SavePath(p.ServerID, p.ID, name, full)
	if err != nil {
		return 0, err
	}
	_ = a.store.SetWorkPath(p.ID, full)
	a.store.AddAudit(&p.ServerID, "path.save", name+" "+full)
	return id, nil
}

func (a *App) ListSavedPaths(serverID, projectID int64) ([]SavedPath, error) {
	if serverID <= 0 {
		return nil, errors.New("serverId is required")
	}
	return a.store.ListSavedPaths(serverID, projectID)
}

func (a *App) DeleteSavedPath(id int64, confirm bool) error {
	if !confirm {
		return errors.New("confirm required")
	}
	return a.store.DeleteSavedPath(id)
}

func (a *App) TestPath(projectID int64, dir string) PathTestResult {
	var out PathTestResult
	p, err := a.store.GetProject(projectID)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	full, err := a.resolveActive(p, dir)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	src := strings.NewReader(pathTestHTML)
	if err := a.mgr.Upload(p.ServerID, p.RemotePath, full, "index.html", src, int64(len(pathTestHTML)+1)); err != nil {
		out.Error = err.Error()
		a.store.RecordPathTest(p.ID, "failed", "")
		return out
	}
	out.OK = true
	out.Dest = full
	out.URL = guessLiveURL(p, full)
	out.Marker = pathTestMarker
	a.store.RecordPathTest(p.ID, "ok", out.URL)
	a.store.AddAudit(&p.ServerID, "path.test", full)
	return out
}

func (a *App) RemovePathTest(projectID int64, dir string) error {
	p, err := a.store.GetProject(projectID)
	if err != nil {
		return err
	}
	full, err := a.resolveActive(p, dir)
	if err != nil {
		return err
	}
	target := path.Join(full, "index.html")
	content, err := a.mgr.ReadFile(p.ServerID, p.RemotePath, target)
	if err != nil {
		return err
	}
	if !bytes.Contains([]byte(content), []byte(pathTestMarker)) && !strings.Contains(content, pathTestMarker) {
		return errors.New("index.html is not a STAPT path test file")
	}
	if err := a.mgr.Remove(p.ServerID, p.RemotePath, target); err != nil {
		return err
	}
	a.store.RecordPathTest(p.ID, "removed", "")
	a.store.AddAudit(&p.ServerID, "path.test.remove", target)
	return nil
}

func (a *App) DisconnectServer(serverID int64) error {
	if serverID <= 0 {
		return errors.New("invalid id")
	}
	a.term.CloseAll()
	a.mgr.Drop(serverID)
	a.store.AddAudit(&serverID, "server.disconnect", "disconnected")
	return nil
}
