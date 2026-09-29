package main

import "testing"

func TestParseSSHCommand(t *testing.T) {
	p, err := parseSSHCommand("ssh -p 65002 alice@10.0.0.8")
	if err != nil {
		t.Fatal(err)
	}
	if p.Host != "10.0.0.8" || p.Port != 65002 || p.Username != "alice" {
		t.Fatalf("%+v", p)
	}
	p, err = parseSSHCommand("bob@example.test")
	if err != nil {
		t.Fatal(err)
	}
	if p.Host != "example.test" || p.Port != 22 || p.Username != "bob" {
		t.Fatalf("%+v", p)
	}
	p, err = parseSSHCommand("bob@example.test:2200")
	if err != nil {
		t.Fatal(err)
	}
	if p.Host != "example.test" || p.Port != 2200 || p.Username != "bob" {
		t.Fatalf("%+v", p)
	}
}

func TestJunkDomainNames(t *testing.T) {
	junk := []string{"www", "public_html", "html", "WWW", "www.example.com", "public_html.example.com", "html.site.com"}
	for _, n := range junk {
		if !isJunkDomainName(n) {
			t.Fatalf("expected junk %q", n)
		}
	}
	keep := []string{"example.com", "blog.example.com", "example.co.uk", "Staging"}
	for _, n := range keep {
		if isJunkDomainName(n) {
			t.Fatalf("expected keep %q", n)
		}
	}
}
