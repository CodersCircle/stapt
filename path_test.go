package main

import "testing"

func TestResolveRemoteBlocksTraversal(t *testing.T) {
	root := "/home/u/public_html/app"
	ok, err := resolveRemote(root, "/home/u/public_html/app/index.php")
	if err != nil || ok != "/home/u/public_html/app/index.php" {
		t.Fatalf("got %q %v", ok, err)
	}
	if _, err := resolveRemote(root, "/home/u/public_html/app/../secrets"); err == nil {
		t.Fatal("expected traversal to fail")
	}
	if _, err := resolveRemote(root, "/etc/passwd"); err == nil {
		t.Fatal("expected outside path to fail")
	}
	if _, err := resolveRemote(root, root+"-other/file"); err == nil {
		t.Fatal("expected prefix sibling to fail")
	}
}

func TestJoinUnderRejectsBadNames(t *testing.T) {
	root := "/var/www/html/site"
	if _, err := joinUnder(root, root, ".."); err == nil {
		t.Fatal("expected .. to fail")
	}
	if _, err := joinUnder(root, root, "a/b"); err == nil {
		t.Fatal("expected slash to fail")
	}
	got, err := joinUnder(root, root, ".env")
	if err != nil || got != "/var/www/html/site/.env" {
		t.Fatalf("got %q %v", got, err)
	}
}
