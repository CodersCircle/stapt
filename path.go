package main

import (
	"errors"
	"path"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

func normalizeRemote(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	p = path.Clean(filepath.ToSlash(p))
	if p == "." {
		return ""
	}
	return p
}

func insideRoot(root, full string) error {
	root = strings.TrimSuffix(normalizeRemote(root), "/")
	full = normalizeRemote(full)
	if root == "" || root == "/" || full == "" {
		return errors.New("path is outside the project")
	}
	if full == root {
		return nil
	}
	prefix := root + "/"
	if !strings.HasPrefix(full, prefix) {
		return errors.New("path is outside the project")
	}
	rel := strings.TrimPrefix(full, prefix)
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return errors.New("path is outside the project")
	}
	return nil
}

func resolveRemote(root, p string) (string, error) {
	root = normalizeRemote(root)
	if root == "" || root == "/" || !path.IsAbs(root) {
		return "", errors.New("invalid project path")
	}
	if strings.IndexByte(p, 0) >= 0 || !utf8.ValidString(p) {
		return "", errors.New("invalid path")
	}
	p = strings.TrimSpace(p)
	if p == "" || p == "." {
		return root, nil
	}
	p = normalizeRemote(filepath.ToSlash(p))
	if !path.IsAbs(p) {
		p = normalizeRemote(path.Join(root, p))
	}
	if err := insideRoot(root, p); err != nil {
		return "", err
	}
	return p, nil
}

func validBaseName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return errors.New("invalid name")
	}
	if strings.ContainsAny(name, "/\\:\x00") {
		return errors.New("invalid name")
	}
	if !utf8.ValidString(name) {
		return errors.New("invalid name")
	}
	return nil
}

func joinUnder(root, dir, name string) (string, error) {
	if err := validBaseName(name); err != nil {
		return "", err
	}
	base, err := resolveRemote(root, dir)
	if err != nil {
		return "", err
	}
	return resolveRemote(root, path.Join(base, strings.TrimSpace(name)))
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}
