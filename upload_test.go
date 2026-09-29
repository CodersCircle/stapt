package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSplitSubdomain(t *testing.T) {
	sub, parent := splitSubdomain("blog.example.com")
	if sub != "blog" || parent != "example.com" {
		t.Fatalf("got %q %q", sub, parent)
	}
	sub, parent = splitSubdomain("example.com")
	if sub != "example" || parent != "com" {
		t.Fatalf("got %q %q", sub, parent)
	}
}

func TestSkipUploadRel(t *testing.T) {
	if !skipUploadRel("__MACOSX/foo") || !skipUploadRel("a/._bar") || !skipUploadRel("../x") {
		t.Fatal("expected skip")
	}
	if skipUploadRel("index.php") || skipUploadRel("css/app.css") {
		t.Fatal("expected keep")
	}
}

func TestScanLocalFolderCounts(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "css"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.php"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "css", "app.css"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".DS_Store"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := scanLocalFolder(1, "/home/u/domains/example.com/public_html", dir)
	if err != nil {
		t.Fatal(err)
	}
	if st.Folders != 1 || len(st.Files) != 2 {
		t.Fatalf("folders=%d files=%d", st.Folders, len(st.Files))
	}
	dirs := uniqueParentDirs(st.Files)
	if len(dirs) != 1 || dirs[0] != "css" {
		t.Fatalf("dirs=%v", dirs)
	}
}
