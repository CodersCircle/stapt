package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

const (
	sshDialTimeout  = 15 * time.Second
	sftpOpTimeout   = 45 * time.Second
	connIdleTimeout = 5 * time.Minute
	maxEditBytes    = 2 << 20
	maxUploadBytes  = 32 << 20
)

type HostKeyNeededError struct {
	Fingerprint string
	Key         ssh.PublicKey
}

func (e *HostKeyNeededError) Error() string {
	return "unrecognized host key " + e.Fingerprint
}

type HostKeyMismatchError struct {
	Fingerprint string
}

func (e *HostKeyMismatchError) Error() string {
	return "host key changed (" + e.Fingerprint + "); not connecting"
}

type Manager struct {
	store *Store
	mu    sync.Mutex
	conns map[int64]*liveConn
}

type liveConn struct {
	client *ssh.Client
	sftp   *sftp.Client
	last   time.Time
}

func newManager(store *Store) *Manager {
	m := &Manager{store: store, conns: map[int64]*liveConn{}}
	go m.reapLoop()
	return m
}

func (m *Manager) reapLoop() {
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	for range t.C {
		m.mu.Lock()
		for id, c := range m.conns {
			if time.Since(c.last) > connIdleTimeout {
				c.close()
				delete(m.conns, id)
			}
		}
		m.mu.Unlock()
	}
}

func (c *liveConn) close() {
	if c.sftp != nil {
		_ = c.sftp.Close()
		c.sftp = nil
	}
	if c.client != nil {
		_ = c.client.Close()
		c.client = nil
	}
}

func (m *Manager) Drop(id int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c, ok := m.conns[id]; ok {
		c.close()
		delete(m.conns, id)
	}
}

func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, c := range m.conns {
		c.close()
		delete(m.conns, id)
	}
}

func sshAuth(sv Server, sec secrets) ([]ssh.AuthMethod, error) {
	switch sv.AuthType {
	case "password":
		if sec.Password == "" {
			return nil, errors.New("password required")
		}
		return []ssh.AuthMethod{ssh.Password(sec.Password)}, nil
	case "key":
		if len(bytes.TrimSpace(sec.PrivateKey)) == 0 {
			return nil, errors.New("private key required")
		}
		var signer ssh.Signer
		var err error
		if sec.Passphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(sec.PrivateKey, []byte(sec.Passphrase))
		} else {
			signer, err = ssh.ParsePrivateKey(sec.PrivateKey)
		}
		if err != nil {
			msg := "invalid private key"
			if sec.Passphrase != "" {
				msg = "invalid private key or passphrase"
			} else if strings.Contains(strings.ToLower(err.Error()), "passphrase") {
				msg = "private key is encrypted; passphrase required"
			}
			return nil, errors.New(msg)
		}
		return []ssh.AuthMethod{ssh.PublicKeys(signer)}, nil
	default:
		return nil, errors.New("authType must be password or key")
	}
}

func hostKeyCallback(known []byte, acceptUnknown bool) (ssh.HostKeyCallback, *HostKeyNeededError) {
	needed := &HostKeyNeededError{}
	cb := func(_ string, _ net.Addr, key ssh.PublicKey) error {
		fp := ssh.FingerprintSHA256(key)
		if len(known) > 0 {
			want, err := ssh.ParsePublicKey(known)
			if err != nil {
				return errors.New("stored host key is invalid")
			}
			if subtleKeyEq(want, key) {
				return nil
			}
			return &HostKeyMismatchError{Fingerprint: fp}
		}
		if acceptUnknown {
			needed.Fingerprint = fp
			needed.Key = key
			return nil
		}
		return &HostKeyNeededError{Fingerprint: fp, Key: key}
	}
	return cb, needed
}

func subtleKeyEq(a, b ssh.PublicKey) bool {
	return bytes.Equal(a.Marshal(), b.Marshal())
}

func dialSSH(sv Server, sec secrets, acceptUnknown bool) (*ssh.Client, ssh.PublicKey, error) {
	auths, err := sshAuth(sv, sec)
	if err != nil {
		return nil, nil, err
	}
	cb, accepted := hostKeyCallback(sec.HostKey, acceptUnknown)
	cfg := &ssh.ClientConfig{
		User:            sv.Username,
		Auth:            auths,
		HostKeyCallback: cb,
		Timeout:         sshDialTimeout,
	}
	addr := net.JoinHostPort(sv.Host, strconv.Itoa(sv.Port))
	client, err := ssh.Dial("tcp", addr, cfg)
	if err != nil {
		var need *HostKeyNeededError
		var mismatch *HostKeyMismatchError
		if errors.As(err, &need) {
			return nil, need.Key, need
		}
		if errors.As(err, &mismatch) {
			return nil, nil, mismatch
		}
		if strings.Contains(strings.ToLower(err.Error()), "unable to authenticate") {
			return nil, nil, errors.New("authentication failed")
		}
		return nil, nil, fmt.Errorf("ssh connect failed: %s", sanitizeNetErr(err))
	}
	var newKey ssh.PublicKey
	if acceptUnknown && accepted.Key != nil && len(sec.HostKey) == 0 {
		newKey = accepted.Key
	}
	return client, newKey, nil
}

func sanitizeNetErr(err error) string {
	msg := err.Error()
	msg = strings.ReplaceAll(msg, "\n", " ")
	if len(msg) > 180 {
		msg = msg[:180]
	}
	return msg
}

func (m *Manager) Test(sv Server, sec secrets, acceptUnknown bool) (fingerprint string, hostKey []byte, err error) {
	client, newKey, err := dialSSH(sv, sec, acceptUnknown)
	if err != nil {
		var need *HostKeyNeededError
		if errors.As(err, &need) {
			return need.Fingerprint, need.Key.Marshal(), need
		}
		return "", nil, err
	}
	_ = client.Close()
	if newKey != nil {
		return ssh.FingerprintSHA256(newKey), newKey.Marshal(), nil
	}
	if len(sec.HostKey) > 0 {
		pk, perr := ssh.ParsePublicKey(sec.HostKey)
		if perr == nil {
			return ssh.FingerprintSHA256(pk), sec.HostKey, nil
		}
	}
	return "", nil, nil
}

func (m *Manager) client(id int64) (*ssh.Client, error) {
	sv, sec, err := m.store.GetServer(id)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	if c, ok := m.conns[id]; ok && c.client != nil {
		c.last = time.Now()
		cl := c.client
		m.mu.Unlock()
		return cl, nil
	}
	m.mu.Unlock()

	client, newKey, err := dialSSH(sv, sec, false)
	if err != nil {
		return nil, err
	}
	if newKey != nil && id > 0 {
		_ = m.store.SaveHostKey(id, newKey.Marshal())
	}
	m.mu.Lock()
	if c, ok := m.conns[id]; ok {
		c.close()
	}
	m.conns[id] = &liveConn{client: client, last: time.Now()}
	m.mu.Unlock()
	return client, nil
}

func (m *Manager) sftp(id int64) (*sftp.Client, error) {
	m.mu.Lock()
	if c, ok := m.conns[id]; ok && c.sftp != nil && c.client != nil {
		c.last = time.Now()
		cli := c.sftp
		m.mu.Unlock()
		return cli, nil
	}
	m.mu.Unlock()
	sshCli, err := m.client(id)
	if err != nil {
		return nil, err
	}
	scli, err := sftp.NewClient(sshCli)
	if err != nil {
		m.Drop(id)
		return nil, errors.New("could not start SFTP")
	}
	m.mu.Lock()
	c := m.conns[id]
	if c == nil {
		c = &liveConn{client: sshCli}
		m.conns[id] = c
	}
	c.sftp = scli
	c.last = time.Now()
	m.mu.Unlock()
	return scli, nil
}

func withTimeout(d time.Duration, fn func() error) error {
	ch := make(chan error, 1)
	go func() { ch <- fn() }()
	select {
	case err := <-ch:
		return err
	case <-time.After(d):
		return errors.New("operation timed out")
	}
}

type FileInfo struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	Dir     bool   `json:"dir"`
	Mode    string `json:"mode"`
	ModTime string `json:"modTime"`
}

func (m *Manager) List(serverID int64, root, p string) ([]FileInfo, error) {
	full, err := resolveRemote(root, p)
	if err != nil {
		return nil, err
	}
	cli, err := m.sftp(serverID)
	if err != nil {
		return nil, err
	}
	full, err = confinedPath(cli, root, full)
	if err != nil {
		return nil, err
	}
	var entries []os.FileInfo
	err = withTimeout(sftpOpTimeout, func() error {
		var e error
		entries, e = cli.ReadDir(full)
		return e
	})
	if err != nil {
		m.Drop(serverID)
		return nil, err
	}
	out := make([]FileInfo, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		if name == "." || name == ".." {
			continue
		}
		child := path.Join(full, name)
		if err := insideRoot(root, child); err != nil {
			continue
		}
		out = append(out, FileInfo{
			Name:    name,
			Path:    child,
			Size:    e.Size(),
			Dir:     e.IsDir(),
			Mode:    e.Mode().String(),
			ModTime: e.ModTime().UTC().Format(time.RFC3339),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Dir != out[j].Dir {
			return out[i].Dir
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}

func (m *Manager) ReadFile(serverID int64, root, p string) (string, error) {
	full, err := resolveRemote(root, p)
	if err != nil {
		return "", err
	}
	cli, err := m.sftp(serverID)
	if err != nil {
		return "", err
	}
	full, err = confinedPath(cli, root, full)
	if err != nil {
		return "", err
	}
	var data []byte
	err = withTimeout(sftpOpTimeout, func() error {
		st, e := cli.Stat(full)
		if e != nil {
			return e
		}
		if st.IsDir() {
			return errors.New("path is a directory")
		}
		if st.Size() > maxEditBytes {
			return fmt.Errorf("file is larger than %d bytes; download instead", maxEditBytes)
		}
		f, e := cli.Open(full)
		if e != nil {
			return e
		}
		defer f.Close()
		data, e = io.ReadAll(io.LimitReader(f, maxEditBytes+1))
		return e
	})
	if err != nil {
		return "", err
	}
	if bytes.IndexByte(data, 0) >= 0 {
		return "", errors.New("binary file; download instead")
	}
	return string(data), nil
}

func (m *Manager) WriteFile(serverID int64, root, p string, content []byte) error {
	full, err := resolveRemote(root, p)
	if err != nil {
		return err
	}
	if len(content) > maxEditBytes {
		return errors.New("file too large to save in editor")
	}
	cli, err := m.sftp(serverID)
	if err != nil {
		return err
	}
	full, err = confinedPath(cli, root, full)
	if err != nil {
		return err
	}
	return withTimeout(sftpOpTimeout, func() error {
		tmp := full + ".stapt-tmp"
		f, e := cli.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
		if e != nil {
			return e
		}
		_, e = f.Write(content)
		cerr := f.Close()
		if e != nil {
			_ = cli.Remove(tmp)
			return e
		}
		if cerr != nil {
			_ = cli.Remove(tmp)
			return cerr
		}
		_ = cli.Remove(full)
		if e = cli.PosixRename(tmp, full); e != nil {
			if e2 := cli.Rename(tmp, full); e2 != nil {
				return e2
			}
		}
		return nil
	})
}

func (m *Manager) Mkdir(serverID int64, root, dir, name string) error {
	full, err := joinUnder(root, dir, name)
	if err != nil {
		return err
	}
	cli, err := m.sftp(serverID)
	if err != nil {
		return err
	}
	full, err = confinedPath(cli, root, full)
	if err != nil {
		return err
	}
	return withTimeout(sftpOpTimeout, func() error {
		return cli.Mkdir(full)
	})
}

func (m *Manager) CreateFile(serverID int64, root, dir, name string) error {
	full, err := joinUnder(root, dir, name)
	if err != nil {
		return err
	}
	cli, err := m.sftp(serverID)
	if err != nil {
		return err
	}
	full, err = confinedPath(cli, root, full)
	if err != nil {
		return err
	}
	return withTimeout(sftpOpTimeout, func() error {
		f, e := cli.OpenFile(full, os.O_WRONLY|os.O_CREATE|os.O_EXCL)
		if e != nil {
			return e
		}
		return f.Close()
	})
}

func (m *Manager) Rename(serverID int64, root, oldPath, newName string) error {
	src, err := resolveRemote(root, oldPath)
	if err != nil {
		return err
	}
	dst, err := joinUnder(root, path.Dir(src), newName)
	if err != nil {
		return err
	}
	cli, err := m.sftp(serverID)
	if err != nil {
		return err
	}
	src, err = confinedPath(cli, root, src)
	if err != nil {
		return err
	}
	dst, err = confinedPath(cli, root, dst)
	if err != nil {
		return err
	}
	return withTimeout(sftpOpTimeout, func() error {
		return cli.Rename(src, dst)
	})
}

func (m *Manager) Remove(serverID int64, root, p string) error {
	full, err := resolveRemote(root, p)
	if err != nil {
		return err
	}
	if full == path.Clean(root) {
		return errors.New("cannot delete the project root")
	}
	cli, err := m.sftp(serverID)
	if err != nil {
		return err
	}
	full, err = confinedPath(cli, root, full)
	if err != nil {
		return err
	}
	return withTimeout(sftpOpTimeout, func() error {
		return removeAll(cli, full)
	})
}

func removeAll(cli *sftp.Client, p string) error {
	st, err := cli.Stat(p)
	if err != nil {
		return err
	}
	if st.IsDir() {
		entries, err := cli.ReadDir(p)
		if err != nil {
			return err
		}
		for _, e := range entries {
			if e.Name() == "." || e.Name() == ".." {
				continue
			}
			if err := removeAll(cli, path.Join(p, e.Name())); err != nil {
				return err
			}
		}
		return cli.RemoveDirectory(p)
	}
	return cli.Remove(p)
}

func (m *Manager) Download(serverID int64, root, p string) (name string, r io.ReadCloser, size int64, err error) {
	full, err := resolveRemote(root, p)
	if err != nil {
		return "", nil, 0, err
	}
	cli, err := m.sftp(serverID)
	if err != nil {
		return "", nil, 0, err
	}
	full, err = confinedPath(cli, root, full)
	if err != nil {
		return "", nil, 0, err
	}
	st, err := cli.Stat(full)
	if err != nil {
		return "", nil, 0, err
	}
	if st.IsDir() {
		return "", nil, 0, errors.New("cannot download a directory")
	}
	f, err := cli.Open(full)
	if err != nil {
		return "", nil, 0, err
	}
	return path.Base(full), f, st.Size(), nil
}

func (m *Manager) Upload(serverID int64, root, dir, name string, src io.Reader, maxBytes int64) error {
	full, err := joinUnder(root, dir, name)
	if err != nil {
		return err
	}
	cli, err := m.sftp(serverID)
	if err != nil {
		return err
	}
	full, err = confinedPath(cli, root, full)
	if err != nil {
		return err
	}
	if maxBytes <= 0 || maxBytes > maxUploadBytes {
		maxBytes = maxUploadBytes
	}
	return withTimeout(2*time.Minute, func() error {
		f, e := cli.OpenFile(full, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
		if e != nil {
			return e
		}
		n, e := io.Copy(f, io.LimitReader(src, maxBytes+1))
		_ = f.Close()
		if e != nil {
			_ = cli.Remove(full)
			return e
		}
		if n > maxBytes {
			_ = cli.Remove(full)
			return errors.New("file too large")
		}
		return nil
	})
}

func resizePTY(sess *ssh.Session, cols, rows int) error {
	if cols < 20 {
		cols = 80
	}
	if rows < 8 {
		rows = 24
	}
	return sess.WindowChange(rows, cols)
}

func confinedPath(cli *sftp.Client, root, full string) (string, error) {
	if err := insideRoot(root, full); err != nil {
		return "", err
	}
	rp, err := cli.RealPath(full)
	if err != nil {
		return full, nil
	}
	rp = path.Clean(filepath.ToSlash(rp))
	rootCheck := root
	if rr, e := cli.RealPath(root); e == nil {
		rootCheck = path.Clean(filepath.ToSlash(rr))
	}
	if err := insideRoot(rootCheck, rp); err != nil {
		return "", errors.New("path is outside the project")
	}
	return rp, nil
}

func (m *Manager) StatDir(serverID int64, remotePath string) error {
	remotePath = absRemoteDir(remotePath)
	if remotePath == "" {
		return errors.New("remote path must be an absolute directory")
	}
	cli, err := m.sftp(serverID)
	if err != nil {
		return err
	}
	return withTimeout(sftpOpTimeout, func() error {
		st, e := cli.Stat(remotePath)
		if e != nil {
			return fmt.Errorf("remote path not found: %s", remotePath)
		}
		if !st.IsDir() {
			return errors.New("remote path is not a directory")
		}
		return nil
	})
}

func hostKeyB64(pub []byte) string {
	return base64.StdEncoding.EncodeToString(pub)
}

func parseHostKeyB64(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, errors.New("invalid host key")
	}
	if _, err := ssh.ParsePublicKey(b); err != nil {
		return nil, errors.New("invalid host key")
	}
	return b, nil
}

func (m *Manager) OpenSession(serverID int64) (*ssh.Client, error) {
	return m.client(serverID)
}

func startPTY(client *ssh.Client, workDir string, cols, rows int) (*ssh.Session, io.WriteCloser, io.Reader, error) {
	if cols < 20 {
		cols = 80
	}
	if rows < 8 {
		rows = 24
	}
	sess, err := client.NewSession()
	if err != nil {
		return nil, nil, nil, err
	}
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := sess.RequestPty("xterm-256color", rows, cols, modes); err != nil {
		_ = sess.Close()
		return nil, nil, nil, errors.New("remote PTY not available")
	}
	stdin, err := sess.StdinPipe()
	if err != nil {
		_ = sess.Close()
		return nil, nil, nil, err
	}
	stdout, err := sess.StdoutPipe()
	if err != nil {
		_ = sess.Close()
		return nil, nil, nil, err
	}
	cmd := "exec ${SHELL:-/bin/bash} -l"
	if workDir != "" {
		cmd = "cd " + shellQuote(workDir) + " && " + cmd
	}
	if err := sess.Start(cmd); err != nil {
		_ = sess.Close()
		return nil, nil, nil, errors.New("could not start remote shell")
	}
	return sess, stdin, stdout, nil
}
