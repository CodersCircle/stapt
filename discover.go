package main

import (
	"path"
	"sort"
	"strings"
)

type discovered struct {
	Name   string
	Path   string
	Kind   string
	Status string
	Tech   string
}

var skipDomainNames = map[string]bool{
	"www": true, "public_html": true, "html": true, "httpdocs": true, "htdocs": true,
}

func isJunkDomainName(name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	n = strings.Trim(n, ".")
	if skipDomainNames[n] {
		return true
	}
	if strings.HasPrefix(n, "www.") {
		return true
	}
	first, rest, ok := strings.Cut(n, ".")
	if ok && rest != "" && skipDomainNames[first] {
		return true
	}
	return false
}

func (m *Manager) Discover(serverID int64, username string) ([]discovered, error) {
	if _, err := m.sftp(serverID); err != nil {
		return nil, err
	}
	home := m.HomeDir(serverID, username)
	if home == "" {
		return nil, nil
	}
	seen := map[string]discovered{}
	add := func(d discovered) {
		d.Path = normalizeRemote(d.Path)
		if absRemoteDir(d.Path) == "" {
			return
		}
		if isJunkDomainName(d.Name) {
			return
		}
		seen[d.Path] = d
	}

	gp := path.Join(home, "domains")
	if !m.ExistsDir(serverID, gp) {
		gp = path.Join(home, "sites")
	}
	if !m.ExistsDir(serverID, gp) {
		return nil, nil
	}
	entries, err := m.ReadDirNames(serverID, gp)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "." || e.Name() == ".." || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		orig := e.Name()
		if isJunkDomainName(orig) {
			continue
		}
		name := orig
		if strings.HasPrefix(strings.ToLower(orig), "www.") {
			continue
		}
		base := path.Join(gp, orig)
		kind := "domain"
		root := path.Join(base, "public_html")
		sub, parent := splitSubdomain(name)
		if parent != "" && m.ExistsDir(serverID, path.Join(gp, parent)) {
			kind = "subdomain"
			root = path.Join(gp, parent, "public_html", sub)
		}
		if !m.ExistsDir(serverID, root) {
			_ = m.EnsureDirRaw(serverID, root)
		}
		add(discovered{Name: name, Path: root, Kind: kind, Status: "ready"})
	}

	out := make([]discovered, 0, len(seen))
	for _, d := range seen {
		d.Tech = detectTech(m, serverID, d.Path)
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}

func splitSubdomain(name string) (sub, parent string) {
	i := strings.Index(name, ".")
	if i <= 0 {
		return name, ""
	}
	return name[:i], name[i+1:]
}

func detectTech(m *Manager, serverID int64, dir string) string {
	var tags []string
	has := func(f string) bool { return m.FileExists(serverID, path.Join(dir, f)) }
	if has("artisan") {
		tags = append(tags, "laravel")
	}
	if has("composer.json") || has("index.php") || has("wp-config.php") {
		tags = append(tags, "php")
	}
	if has("composer.json") {
		tags = append(tags, "composer")
	}
	if has("package.json") {
		tags = append(tags, "node")
	}
	if has(".git") {
		tags = append(tags, "git")
	}
	if has("requirements.txt") || has("pyproject.toml") || has("manage.py") || has("app.py") {
		tags = append(tags, "python")
	}
	if has("go.mod") {
		tags = append(tags, "go")
	}
	if has("Gemfile") {
		tags = append(tags, "ruby")
	}
	return strings.Join(tags, ",")
}
