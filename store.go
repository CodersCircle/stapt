package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const maxAuditRows = 500

type Store struct {
	db  *sql.DB
	key []byte
	Dir string
}

type Server struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	Host           string    `json:"host"`
	Port           int       `json:"port"`
	Username       string    `json:"username"`
	AuthType       string    `json:"authType"`
	HasPassword    bool      `json:"hasPassword"`
	HasKey         bool      `json:"hasKey"`
	HasHostKey     bool      `json:"hasHostKey"`
	LastConnected  string    `json:"lastConnected"`
	ProjectCount   int       `json:"projectCount"`
	PathCount      int       `json:"pathCount"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type Project struct {
	ID               int64  `json:"id"`
	ServerID         int64  `json:"serverId"`
	Name             string `json:"name"`
	RemotePath       string `json:"remotePath"`
	WorkPath         string `json:"workPath"`
	Kind             string `json:"kind"`
	Status           string `json:"status"`
	LastChecked      string `json:"lastChecked"`
	Tech             string `json:"tech"`
	LastUploadAt     string `json:"lastUploadAt"`
	LastUploadFile   string `json:"lastUploadFile"`
	LastUploadSize   int64  `json:"lastUploadSize"`
	LastUploadStatus string `json:"lastUploadStatus"`
	PathTestStatus   string `json:"pathTestStatus"`
	PathTestURL      string `json:"pathTestUrl"`
	PathTestAt       string `json:"pathTestAt"`
}

type SavedPath struct {
	ID         int64  `json:"id"`
	ServerID   int64  `json:"serverId"`
	ProjectID  int64  `json:"projectId"`
	Name       string `json:"name"`
	RemotePath string `json:"remotePath"`
	CreatedAt  string `json:"createdAt"`
}

type Note struct {
	ID        int64  `json:"id"`
	ProjectID int64  `json:"projectId"`
	Category  string `json:"category"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type Audit struct {
	ID       int64     `json:"id"`
	TS       time.Time `json:"ts"`
	ServerID *int64    `json:"serverId"`
	Action   string    `json:"action"`
	Detail   string    `json:"detail"`
}

type secrets struct {
	Password   string
	PrivateKey []byte
	Passphrase string
	HostKey    []byte
}

type ServerWrite struct {
	Name       string
	Host       string
	Port       int
	Username   string
	AuthType   string
	Password   *string
	PrivateKey *string
	Passphrase *string
	HostKey    []byte
}

func openStore() (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(home, ".stapt")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	key, err := loadOrCreateKey(filepath.Join(dir, "master.key"))
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", filepath.Join(dir, "stapt.db"))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		_ = db.Close()
		return nil, err
	}
	s := &Store{db: db, key: key, Dir: dir}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func loadOrCreateKey(p string) ([]byte, error) {
	b, err := os.ReadFile(p)
	if err == nil {
		if len(b) != 32 {
			return nil, errors.New("invalid local encryption key")
		}
		return b, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	if err := os.WriteFile(p, key, 0o600); err != nil {
		return nil, err
	}
	return key, nil
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS servers (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  host TEXT NOT NULL,
  port INTEGER NOT NULL DEFAULT 22,
  username TEXT NOT NULL,
  auth_type TEXT NOT NULL,
  password_enc BLOB,
  key_enc BLOB,
  passphrase_enc BLOB,
  host_key BLOB,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS projects (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  server_id INTEGER NOT NULL,
  name TEXT NOT NULL,
  remote_path TEXT NOT NULL,
  FOREIGN KEY(server_id) REFERENCES servers(id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS audit (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  ts TEXT NOT NULL,
  server_id INTEGER,
  action TEXT NOT NULL,
  detail TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS notes (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  project_id INTEGER,
  category TEXT NOT NULL DEFAULT '',
  title TEXT NOT NULL,
  content_enc BLOB,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
`)
	if err != nil {
		return err
	}
	for _, q := range []string{
		`ALTER TABLE projects ADD COLUMN kind TEXT NOT NULL DEFAULT 'project'`,
		`ALTER TABLE projects ADD COLUMN status TEXT NOT NULL DEFAULT 'ready'`,
		`ALTER TABLE projects ADD COLUMN last_checked TEXT`,
		`ALTER TABLE projects ADD COLUMN tech TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE projects ADD COLUMN work_path TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE projects ADD COLUMN last_upload_at TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE projects ADD COLUMN last_upload_file TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE projects ADD COLUMN last_upload_size INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE projects ADD COLUMN last_upload_status TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE projects ADD COLUMN path_test_status TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE projects ADD COLUMN path_test_url TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE projects ADD COLUMN path_test_at TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE notes ADD COLUMN category TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE servers ADD COLUMN last_connected TEXT NOT NULL DEFAULT ''`,
		`CREATE TABLE IF NOT EXISTS saved_paths (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  server_id INTEGER NOT NULL,
  project_id INTEGER NOT NULL DEFAULT 0,
  name TEXT NOT NULL,
  remote_path TEXT NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY(server_id) REFERENCES servers(id) ON DELETE CASCADE
)`,
	} {
		_, _ = s.db.Exec(q)
	}
	return nil
}

func (s *Store) encrypt(plain []byte) ([]byte, error) {
	if len(plain) == 0 {
		return nil, nil
	}
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plain, nil), nil
}

func (s *Store) decrypt(blob []byte) ([]byte, error) {
	if len(blob) == 0 {
		return nil, nil
	}
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	ns := gcm.NonceSize()
	if len(blob) < ns {
		return nil, errors.New("corrupt credential data")
	}
	return gcm.Open(nil, blob[:ns], blob[ns:], nil)
}

func (s *Store) ListServers() ([]Server, error) {
	rows, err := s.db.Query(`
SELECT id, name, host, port, username, auth_type,
       password_enc, key_enc, host_key, created_at, updated_at, COALESCE(last_connected,'')
FROM servers ORDER BY name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Server{}
	for rows.Next() {
		var sv Server
		var pw, key, hk []byte
		var created, updated string
		if err := rows.Scan(&sv.ID, &sv.Name, &sv.Host, &sv.Port, &sv.Username, &sv.AuthType,
			&pw, &key, &hk, &created, &updated, &sv.LastConnected); err != nil {
			return nil, err
		}
		sv.HasPassword = len(pw) > 0
		sv.HasKey = len(key) > 0
		sv.HasHostKey = len(hk) > 0
		sv.CreatedAt, _ = time.Parse(time.RFC3339, created)
		sv.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
		_ = s.db.QueryRow(`SELECT COUNT(*) FROM projects WHERE server_id=?`, sv.ID).Scan(&sv.ProjectCount)
		_ = s.db.QueryRow(`SELECT COUNT(*) FROM saved_paths WHERE server_id=?`, sv.ID).Scan(&sv.PathCount)
		out = append(out, sv)
	}
	return out, rows.Err()
}

func (s *Store) GetServer(id int64) (Server, secrets, error) {
	var sv Server
	var sec secrets
	var pwEnc, keyEnc, passEnc, hk []byte
	var created, updated string
	err := s.db.QueryRow(`
SELECT id, name, host, port, username, auth_type,
       password_enc, key_enc, passphrase_enc, host_key, created_at, updated_at
FROM servers WHERE id = ?`, id).Scan(
		&sv.ID, &sv.Name, &sv.Host, &sv.Port, &sv.Username, &sv.AuthType,
		&pwEnc, &keyEnc, &passEnc, &hk, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return sv, sec, errors.New("server not found")
	}
	if err != nil {
		return sv, sec, err
	}
	sv.HasPassword = len(pwEnc) > 0
	sv.HasKey = len(keyEnc) > 0
	sv.HasHostKey = len(hk) > 0
	sv.CreatedAt, _ = time.Parse(time.RFC3339, created)
	sv.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
	pw, err := s.decrypt(pwEnc)
	if err != nil {
		return sv, sec, errors.New("could not decrypt stored password")
	}
	key, err := s.decrypt(keyEnc)
	if err != nil {
		return sv, sec, errors.New("could not decrypt stored key")
	}
	pass, err := s.decrypt(passEnc)
	if err != nil {
		return sv, sec, errors.New("could not decrypt stored passphrase")
	}
	sec.Password = string(pw)
	sec.PrivateKey = key
	sec.Passphrase = string(pass)
	sec.HostKey = hk
	return sv, sec, nil
}

func (s *Store) CreateServer(in ServerWrite) (int64, error) {
	if err := validateServerWrite(in, true); err != nil {
		return 0, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	pw, key, pass, err := s.encSecrets(in)
	if err != nil {
		return 0, err
	}
	res, err := s.db.Exec(`
INSERT INTO servers (name, host, port, username, auth_type, password_enc, key_enc, passphrase_enc, host_key, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		strings.TrimSpace(in.Name), strings.TrimSpace(in.Host), in.Port, strings.TrimSpace(in.Username),
		in.AuthType, pw, key, pass, in.HostKey, now, now)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) UpdateServer(id int64, in ServerWrite) error {
	_, sec, err := s.GetServer(id)
	if err != nil {
		return err
	}
	if err := validateServerWrite(in, false); err != nil {
		return err
	}
	if in.Password == nil {
		p := sec.Password
		in.Password = &p
	}
	if in.PrivateKey == nil {
		k := string(sec.PrivateKey)
		in.PrivateKey = &k
	}
	if in.Passphrase == nil {
		p := sec.Passphrase
		in.Passphrase = &p
	}
	if in.HostKey == nil {
		in.HostKey = sec.HostKey
	}
	pw, key, pass, err := s.encSecrets(in)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = s.db.Exec(`
UPDATE servers SET name=?, host=?, port=?, username=?, auth_type=?,
 password_enc=?, key_enc=?, passphrase_enc=?, host_key=?, updated_at=?
WHERE id=?`,
		strings.TrimSpace(in.Name), strings.TrimSpace(in.Host), in.Port, strings.TrimSpace(in.Username),
		in.AuthType, pw, key, pass, in.HostKey, now, id)
	return err
}

func (s *Store) SaveHostKey(id int64, hostKey []byte) error {
	_, err := s.db.Exec(`UPDATE servers SET host_key=?, updated_at=? WHERE id=?`,
		hostKey, time.Now().UTC().Format(time.RFC3339), id)
	return err
}

func (s *Store) TouchLastConnected(id int64) {
	now := time.Now().UTC().Format(time.RFC3339)
	_, _ = s.db.Exec(`UPDATE servers SET last_connected=?, updated_at=? WHERE id=?`, now, now, id)
}

func (s *Store) DeleteServer(id int64) error {
	res, err := s.db.Exec(`DELETE FROM servers WHERE id=?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("server not found")
	}
	return nil
}

func (s *Store) encSecrets(in ServerWrite) (pw, key, pass []byte, err error) {
	if in.Password != nil {
		pw, err = s.encrypt([]byte(*in.Password))
		if err != nil {
			return
		}
	}
	if in.PrivateKey != nil {
		key, err = s.encrypt([]byte(*in.PrivateKey))
		if err != nil {
			return
		}
	}
	if in.Passphrase != nil {
		pass, err = s.encrypt([]byte(*in.Passphrase))
		if err != nil {
			return
		}
	}
	return
}

func validateServerWrite(in ServerWrite, creating bool) error {
	if strings.TrimSpace(in.Name) == "" {
		return errors.New("name is required")
	}
	if strings.TrimSpace(in.Host) == "" {
		return errors.New("host is required")
	}
	if in.Port <= 0 || in.Port > 65535 {
		return errors.New("port must be 1-65535")
	}
	if strings.TrimSpace(in.Username) == "" {
		return errors.New("username is required")
	}
	if in.AuthType != "password" && in.AuthType != "key" {
		return errors.New("authType must be password or key")
	}
	if creating {
		if in.AuthType == "password" && (in.Password == nil || *in.Password == "") {
			return errors.New("password is required")
		}
		if in.AuthType == "key" && (in.PrivateKey == nil || strings.TrimSpace(*in.PrivateKey) == "") {
			return errors.New("private key is required")
		}
	}
	return nil
}

func scanProject(scan func(dest ...any) error) (Project, error) {
	var p Project
	var last sql.NullString
	err := scan(&p.ID, &p.ServerID, &p.Name, &p.RemotePath, &p.Kind, &p.Status, &last, &p.Tech,
		&p.WorkPath, &p.LastUploadAt, &p.LastUploadFile, &p.LastUploadSize, &p.LastUploadStatus,
		&p.PathTestStatus, &p.PathTestURL, &p.PathTestAt)
	if last.Valid {
		p.LastChecked = last.String
	}
	if p.Kind == "" {
		p.Kind = "project"
	}
	if p.Status == "" || (p.Status == "needs_root" && p.RemotePath != "") {
		p.Status = "ready"
	}
	return p, err
}

const projectSelect = `id, server_id, name, remote_path, COALESCE(kind,'project'), COALESCE(status,'ready'), last_checked, COALESCE(tech,''),
 COALESCE(work_path,''), COALESCE(last_upload_at,''), COALESCE(last_upload_file,''), COALESCE(last_upload_size,0), COALESCE(last_upload_status,''),
 COALESCE(path_test_status,''), COALESCE(path_test_url,''), COALESCE(path_test_at,'')`

func (p Project) ActivePath() string {
	if w := absRemoteDir(p.WorkPath); w != "" {
		return w
	}
	return p.RemotePath
}

func (s *Store) PruneJunkProjects(serverID int64) {
	q := `SELECT id, name, COALESCE(kind,'project') FROM projects`
	args := []any{}
	if serverID > 0 {
		q += ` WHERE server_id=?`
		args = append(args, serverID)
	}
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return
	}
	defer rows.Close()
	var drop []int64
	for rows.Next() {
		var id int64
		var name, kind string
		if err := rows.Scan(&id, &name, &kind); err != nil {
			return
		}
		if kind == "document_root" || isJunkDomainName(name) {
			drop = append(drop, id)
		}
	}
	for _, id := range drop {
		_, _ = s.db.Exec(`DELETE FROM projects WHERE id=?`, id)
	}
}

func (s *Store) ListProjects(serverID int64) ([]Project, error) {
	s.PruneJunkProjects(serverID)
	rows, err := s.db.Query(`
SELECT `+projectSelect+`
FROM projects WHERE server_id=? ORDER BY name COLLATE NOCASE`, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Project{}
	for rows.Next() {
		p, err := scanProject(rows.Scan)
		if err != nil {
			return nil, err
		}
		if isJunkDomainName(p.Name) || p.Kind == "document_root" {
			continue
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) GetProject(id int64) (Project, error) {
	p, err := scanProject(s.db.QueryRow(`
SELECT `+projectSelect+`
FROM projects WHERE id=?`, id).Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return p, errors.New("project not found")
	}
	return p, err
}

func (s *Store) CreateProject(serverID int64, name, remotePath string) (int64, error) {
	return s.CreateProjectFull(serverID, name, remotePath, "project", "ready", "")
}

func (s *Store) CreateProjectFull(serverID int64, name, remotePath, kind, status, tech string) (int64, error) {
	if _, _, err := s.GetServer(serverID); err != nil {
		return 0, err
	}
	name = strings.TrimSpace(name)
	remotePath = absRemoteDir(remotePath)
	if name == "" {
		return 0, errors.New("project name is required")
	}
	if remotePath == "" {
		return 0, errors.New("remote path must be an absolute directory")
	}
	if kind == "" {
		kind = "project"
	}
	if status == "" {
		status = "ready"
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.Exec(`
INSERT INTO projects (server_id, name, remote_path, kind, status, last_checked, tech)
VALUES (?, ?, ?, ?, ?, ?, ?)`, serverID, name, remotePath, kind, status, now, tech)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) UpsertDiscovered(serverID int64, name, remotePath, kind, status, tech string) (int64, error) {
	remotePath = absRemoteDir(remotePath)
	if remotePath == "" {
		return 0, errors.New("remote path must be an absolute directory")
	}
	var id int64
	err := s.db.QueryRow(`SELECT id FROM projects WHERE server_id=? AND remote_path=?`, serverID, remotePath).Scan(&id)
	if err == nil {
		now := time.Now().UTC().Format(time.RFC3339)
		_, err = s.db.Exec(`UPDATE projects SET name=?, kind=?, status=?, last_checked=?, tech=? WHERE id=?`,
			strings.TrimSpace(name), kind, status, now, tech, id)
		return id, err
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	return s.CreateProjectFull(serverID, name, remotePath, kind, status, tech)
}

func (s *Store) UpdateProject(id int64, name, remotePath string) error {
	return s.UpdateProjectMeta(id, name, remotePath, "", "")
}

func (s *Store) UpdateProjectMeta(id int64, name, remotePath, kind, status string) error {
	name = strings.TrimSpace(name)
	remotePath = absRemoteDir(remotePath)
	if name == "" {
		return errors.New("project name is required")
	}
	if remotePath == "" {
		return errors.New("remote path must be an absolute directory")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	q := `UPDATE projects SET name=?, remote_path=?, last_checked=?`
	args := []any{name, remotePath, now}
	if kind != "" {
		q += `, kind=?`
		args = append(args, kind)
	}
	if status != "" {
		q += `, status=?`
		args = append(args, status)
	}
	q += ` WHERE id=?`
	args = append(args, id)
	res, err := s.db.Exec(q, args...)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("project not found")
	}
	return nil
}

func (s *Store) DeleteProject(id int64) error {
	res, err := s.db.Exec(`DELETE FROM projects WHERE id=?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("project not found")
	}
	return nil
}

func (s *Store) SetWorkPath(id int64, workPath string) error {
	workPath = absRemoteDir(workPath)
	if workPath == "" {
		return errors.New("remote path must be an absolute directory")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.Exec(`UPDATE projects SET work_path=?, last_checked=? WHERE id=?`, workPath, now, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("project not found")
	}
	return nil
}

func (s *Store) RecordUpload(id int64, file string, size int64, status string) {
	now := time.Now().UTC().Format(time.RFC3339)
	_, _ = s.db.Exec(`UPDATE projects SET last_upload_at=?, last_upload_file=?, last_upload_size=?, last_upload_status=?, last_checked=? WHERE id=?`,
		now, file, size, status, now, id)
}

func (s *Store) RecordPathTest(id int64, status, url string) {
	now := time.Now().UTC().Format(time.RFC3339)
	_, _ = s.db.Exec(`UPDATE projects SET path_test_status=?, path_test_url=?, path_test_at=?, last_checked=? WHERE id=?`,
		status, url, now, now, id)
}

func (s *Store) ListSavedPaths(serverID, projectID int64) ([]SavedPath, error) {
	q := `SELECT id, server_id, project_id, name, remote_path, created_at FROM saved_paths WHERE server_id=?`
	args := []any{serverID}
	if projectID > 0 {
		q += ` AND project_id=?`
		args = append(args, projectID)
	}
	q += ` ORDER BY name COLLATE NOCASE`
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SavedPath{}
	for rows.Next() {
		var p SavedPath
		if err := rows.Scan(&p.ID, &p.ServerID, &p.ProjectID, &p.Name, &p.RemotePath, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) SavePath(serverID, projectID int64, name, remotePath string) (int64, error) {
	name = strings.TrimSpace(name)
	remotePath = absRemoteDir(remotePath)
	if name == "" {
		return 0, errors.New("path name is required")
	}
	if remotePath == "" {
		return 0, errors.New("remote path must be an absolute directory")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	var id int64
	err := s.db.QueryRow(`SELECT id FROM saved_paths WHERE server_id=? AND remote_path=?`, serverID, remotePath).Scan(&id)
	if err == nil {
		_, err = s.db.Exec(`UPDATE saved_paths SET name=?, project_id=? WHERE id=?`, name, projectID, id)
		return id, err
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	res, err := s.db.Exec(`INSERT INTO saved_paths (server_id, project_id, name, remote_path, created_at) VALUES (?,?,?,?,?)`,
		serverID, projectID, name, remotePath, now)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) DeleteSavedPath(id int64) error {
	res, err := s.db.Exec(`DELETE FROM saved_paths WHERE id=?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("path not found")
	}
	return nil
}

func absRemoteDir(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	p = path.Clean(filepath.ToSlash(p))
	if !path.IsAbs(p) || p == "/" {
		return ""
	}
	return p
}

func (s *Store) AddAudit(serverID *int64, action, detail string) {
	action = strings.TrimSpace(action)
	if action == "" {
		return
	}
	detail = strings.TrimSpace(detail)
	if len(detail) > 300 {
		detail = detail[:300]
	}
	if looksSecret(detail) {
		detail = "[redacted]"
	}
	_, _ = s.db.Exec(`INSERT INTO audit (ts, server_id, action, detail) VALUES (?, ?, ?, ?)`,
		time.Now().UTC().Format(time.RFC3339), serverID, action, detail)
	_, _ = s.db.Exec(`DELETE FROM audit WHERE id NOT IN (SELECT id FROM (SELECT id FROM audit ORDER BY id DESC LIMIT ?))`, maxAuditRows)
}

func (s *Store) ListAudit() ([]Audit, error) {
	rows, err := s.db.Query(`SELECT id, ts, server_id, action, detail FROM audit ORDER BY id DESC LIMIT 200`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Audit{}
	for rows.Next() {
		var a Audit
		var ts string
		var sid sql.NullInt64
		if err := rows.Scan(&a.ID, &ts, &sid, &a.Action, &a.Detail); err != nil {
			return nil, err
		}
		a.TS, _ = time.Parse(time.RFC3339, ts)
		if sid.Valid {
			v := sid.Int64
			a.ServerID = &v
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) ClearAudit() error {
	_, err := s.db.Exec(`DELETE FROM audit`)
	return err
}

func (s *Store) ListNotes(projectID int64) ([]Note, error) {
	return s.queryNotes(`SELECT id, project_id, COALESCE(category,''), title, content_enc, created_at, updated_at FROM notes WHERE project_id=? ORDER BY id DESC`, projectID)
}

func (s *Store) ListAllNotes() ([]Note, error) {
	return s.queryNotes(`SELECT id, project_id, COALESCE(category,''), title, content_enc, created_at, updated_at FROM notes ORDER BY category COLLATE NOCASE, id DESC`)
}

func (s *Store) queryNotes(q string, args ...any) ([]Note, error) {
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Note{}
	for rows.Next() {
		n, err := scanNote(rows.Scan, s)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func scanNote(scan func(dest ...any) error, s *Store) (Note, error) {
	var n Note
	var enc []byte
	var pid sql.NullInt64
	err := scan(&n.ID, &pid, &n.Category, &n.Title, &enc, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		return n, err
	}
	if pid.Valid {
		n.ProjectID = pid.Int64
	}
	plain, err := s.decrypt(enc)
	if err != nil {
		return n, errors.New("could not decrypt note")
	}
	n.Content = string(plain)
	return n, nil
}

func (s *Store) GetNote(id int64) (Note, error) {
	n, err := scanNote(s.db.QueryRow(`SELECT id, project_id, COALESCE(category,''), title, content_enc, created_at, updated_at FROM notes WHERE id=?`, id).Scan, s)
	if errors.Is(err, sql.ErrNoRows) {
		return n, errors.New("note not found")
	}
	return n, err
}

func (s *Store) CreateNote(projectID int64, category, title, content string) (int64, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return 0, errors.New("title is required")
	}
	category = strings.TrimSpace(category)
	if category == "" {
		category = "General"
	}
	enc, err := s.encrypt([]byte(content))
	if err != nil {
		return 0, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	var pid any
	if projectID > 0 {
		pid = projectID
	}
	res, err := s.db.Exec(`INSERT INTO notes (project_id, category, title, content_enc, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		pid, category, title, enc, now, now)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) UpdateNote(id int64, category, title, content string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return errors.New("title is required")
	}
	category = strings.TrimSpace(category)
	if category == "" {
		category = "General"
	}
	enc, err := s.encrypt([]byte(content))
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.Exec(`UPDATE notes SET category=?, title=?, content_enc=?, updated_at=? WHERE id=?`, category, title, enc, now, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("note not found")
	}
	return nil
}

func (s *Store) DeleteNote(id int64) error {
	res, err := s.db.Exec(`DELETE FROM notes WHERE id=?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("note not found")
	}
	return nil
}

func looksSecret(s string) bool {
	l := strings.ToLower(s)
	for _, n := range []string{"password", "passwd", "secret", "private key", "begin openssh", "begin rsa", "passphrase", "authorization:"} {
		if strings.Contains(l, n) {
			return true
		}
	}
	return false
}
