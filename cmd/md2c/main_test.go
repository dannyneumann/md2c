package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunVersion(t *testing.T) {
	t.Parallel()
	stdout, stderr := &strings.Builder{}, &strings.Builder{}
	code := run([]string{"-version"}, runtime{Stdout: stdout, Stderr: stderr})
	if code != 0 {
		t.Fatalf("exit %d stderr %s", code, stderr)
	}
	got := stdout.String()
	if got != versionBanner() {
		t.Fatalf("stdout %q", got)
	}
	if !strings.Contains(got, "source "+source) {
		t.Fatalf("missing source in %q", got)
	}
	if !strings.Contains(got, "optimized by "+author) {
		t.Fatalf("missing author line in %q", got)
	}
}

func TestRunUsage(t *testing.T) {
	t.Parallel()
	stderr := &strings.Builder{}
	code := run(nil, runtime{Stdout: io.Discard, Stderr: stderr})
	if code != 2 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stderr.String(), "Aufruf:") {
		t.Fatalf("stderr %s", stderr)
	}
	if !strings.Contains(stderr.String(), "<datei>") {
		t.Fatalf("missing primary usage: %s", stderr)
	}
	if !strings.Contains(stderr.String(), "space:") {
		t.Fatalf("missing file metadata: %s", stderr)
	}
	if !strings.Contains(stderr.String(), "-config") {
		t.Fatalf("missing config flag: %s", stderr)
	}
}

func TestRunDryRun(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	if err := os.WriteFile(path, []byte("# Hello\n\nWorld **bold**.\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	stdout, stderr := &strings.Builder{}, &strings.Builder{}
	code := run([]string{"-dry-run", path, "DEV", "Docs/Hello"}, runtime{
		Stdout: stdout,
		Stderr: stderr,
		Getenv: func(string) string { return "" },
		Cwd:    dir,
	})
	if code != 0 {
		t.Fatalf("exit %d stderr %s", code, stderr)
	}
	got := stdout.String()
	if !strings.Contains(got, "<h1>Hello</h1>") {
		t.Fatalf("missing heading: %s", got)
	}
	if !strings.Contains(got, "<strong>bold</strong>") {
		t.Fatalf("missing bold: %s", got)
	}
}

func TestRunDryRunFromMetadata(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	src := `<!-- space:DOC,path:Guides,title:Getting started -->
# Inhalt

Hallo **Welt**.
`
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}

	stdout, stderr := &strings.Builder{}, &strings.Builder{}
	code := run([]string{"-dry-run", path}, runtime{
		Stdout: stdout,
		Stderr: stderr,
		Getenv: func(string) string { return "" },
		Cwd:    dir,
	})
	if code != 0 {
		t.Fatalf("exit %d stderr %s", code, stderr)
	}
	if !strings.Contains(stderr.String(), "ziel:") {
		t.Fatalf("missing target: %s", stderr)
	}
	if !strings.Contains(stderr.String(), "DOC") || !strings.Contains(stderr.String(), "Getting started") {
		t.Fatalf("wrong target %s", stderr)
	}
	got := stdout.String()
	if strings.Contains(got, "space:DOC") {
		t.Fatalf("metadata leaked into body: %s", got)
	}
	if !strings.Contains(got, "<h1>Inhalt</h1>") {
		t.Fatalf("missing heading: %s", got)
	}
}

func TestRunMissingTargetWithoutMetadata(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	if err := os.WriteFile(path, []byte("# Hi\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	stderr := &strings.Builder{}
	code := run([]string{"-dry-run", path}, runtime{
		Stdout: io.Discard,
		Stderr: stderr,
		Getenv: func(string) string { return "" },
		Cwd:    dir,
	})
	if code != 2 {
		t.Fatalf("exit %d stderr %s", code, stderr)
	}
	if !strings.Contains(stderr.String(), "md2c <datei> <space> <pfad>") {
		t.Fatalf("stderr %s", stderr)
	}
}

func TestRunPublishFromMetadata(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	if err := os.WriteFile(path, []byte(`<!-- space:DOC,path:Guides,title:Getting started -->
Hello
`), 0o600); err != nil {
		t.Fatal(err)
	}

	var titles []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_, _ = w.Write([]byte(`{"results":[],"size":0}`))
		case http.MethodPost:
			raw, _ := io.ReadAll(r.Body)
			var payload map[string]any
			_ = json.Unmarshal(raw, &payload)
			title, _ := payload["title"].(string)
			titles = append(titles, title)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":    "7",
				"type":  "page",
				"title": title,
				"space": map[string]string{"key": "DOC"},
				"version": map[string]int{
					"number": 1,
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	home := t.TempDir()
	writeConf(t, home, "MD2C_BASE_URL="+srv.URL+"\nMD2C_USER=me\nMD2C_TOKEN=token\n")

	stdout, stderr := &strings.Builder{}, &strings.Builder{}
	code := run([]string{"--config=~/.config/md2c/md2c.conf", path}, runtime{
		Stdout:     stdout,
		Stderr:     stderr,
		HTTPClient: srv.Client(),
		Getenv:     func(string) string { return "" },
		Home:       home,
		Cwd:        dir,
	})
	if code != 0 {
		t.Fatalf("exit %d stderr %s", code, stderr)
	}
	if len(titles) < 2 {
		t.Fatalf("expected parent + leaf create, got %v", titles)
	}
	if titles[len(titles)-1] != "Getting started" {
		t.Fatalf("leaf title %q in %v", titles[len(titles)-1], titles)
	}
	if !strings.Contains(stdout.String(), "DOC") {
		t.Fatalf("stdout %s", stdout)
	}
}

func TestRunMissingFile(t *testing.T) {
	t.Parallel()
	stderr := &strings.Builder{}
	code := run([]string{"-dry-run", "/no/such/file.md", "DEV", "Page"}, runtime{
		Stdout: io.Discard,
		Stderr: stderr,
		Getenv: func(string) string { return "" },
	})
	if code != 1 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stderr.String(), "read") {
		t.Fatalf("stderr %s", stderr)
	}
}

func TestRunPublish(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	if err := os.WriteFile(path, []byte("Hello\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_, _ = w.Write([]byte(`{"results":[],"size":0}`))
		case http.MethodPost:
			raw, _ := io.ReadAll(r.Body)
			var payload map[string]any
			_ = json.Unmarshal(raw, &payload)
			title, _ := payload["title"].(string)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":    "99",
				"type":  "page",
				"title": title,
				"space": map[string]string{"key": "DEV"},
				"version": map[string]int{
					"number": 1,
				},
				"_links": map[string]string{
					"base":  "https://acme.atlassian.net/wiki",
					"webui": "/spaces/DEV/pages/99/" + title,
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	home := t.TempDir()
	writeConf(t, home, "MD2C_BASE_URL="+srv.URL+"\nMD2C_USER=me\nMD2C_TOKEN=token\n")

	stdout, stderr := &strings.Builder{}, &strings.Builder{}
	code := run([]string{path, "DEV", "Hello"}, runtime{
		Stdout:     stdout,
		Stderr:     stderr,
		HTTPClient: srv.Client(),
		Getenv:     func(string) string { return "" },
		Home:       home,
		Cwd:        dir,
	})
	if code != 0 {
		t.Fatalf("exit %d stderr %s", code, stderr)
	}
	if !strings.Contains(stdout.String(), "published") {
		t.Fatalf("stdout %s", stdout)
	}
	if !strings.Contains(stdout.String(), "https://acme.atlassian.net/wiki/spaces/DEV/pages/99/Hello") {
		t.Fatalf("missing url: %s", stdout)
	}
}

func TestRunMissingCredentials(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	if err := os.WriteFile(path, []byte("Hi\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	stderr := &strings.Builder{}
	code := run([]string{path, "DEV", "Hi"}, runtime{
		Stdout: io.Discard,
		Stderr: stderr,
		Getenv: func(string) string { return "" },
		Home:   t.TempDir(),
		Cwd:    dir,
	})
	if code != 2 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stderr.String(), "keine Config") {
		t.Fatalf("stderr %s", stderr)
	}
}

func TestRunPublishWithConfigFlag(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	if err := os.WriteFile(path, []byte("Hello\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	conf := filepath.Join(home, "alt.conf")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_, _ = w.Write([]byte(`{"results":[],"size":0}`))
		case http.MethodPost:
			_, _ = w.Write([]byte(`{"id":"1","type":"page","title":"Hello","space":{"key":"DEV"},"version":{"number":1}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	if err := os.WriteFile(conf, []byte("MD2C_BASE_URL="+srv.URL+"\nMD2C_USER=me\nMD2C_TOKEN=token\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	stdout, stderr := &strings.Builder{}, &strings.Builder{}
	code := run([]string{"--config=~/alt.conf", path, "DEV", "Hello"}, runtime{
		Stdout:     stdout,
		Stderr:     stderr,
		HTTPClient: srv.Client(),
		Getenv:     func(string) string { return "" },
		Home:       home,
		Cwd:        dir,
	})
	if code != 0 {
		t.Fatalf("exit %d stderr %s", code, stderr)
	}
	if !strings.Contains(stdout.String(), "published") {
		t.Fatalf("stdout %s", stdout)
	}
}

func TestRunPublishFromHomeEnv(t *testing.T) {
	t.Parallel()
	work := t.TempDir()
	home := t.TempDir()
	path := filepath.Join(work, "note.md")
	if err := os.WriteFile(path, []byte("Hello\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfgDir := filepath.Join(home, ".config", "md2c")
	if err := os.MkdirAll(cfgDir, 0o700); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_, _ = w.Write([]byte(`{"results":[],"size":0}`))
		case http.MethodPost:
			_, _ = w.Write([]byte(`{"id":"1","type":"page","title":"Hello","space":{"key":"DEV"},"version":{"number":1}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	if err := os.WriteFile(filepath.Join(cfgDir, "md2c.conf"), []byte(
		"MD2C_BASE_URL="+srv.URL+"\nMD2C_USER=me\nMD2C_TOKEN=token\n",
	), 0o600); err != nil {
		t.Fatal(err)
	}

	stdout, stderr := &strings.Builder{}, &strings.Builder{}
	code := run([]string{path, "DEV", "Hello"}, runtime{
		Stdout:     stdout,
		Stderr:     stderr,
		HTTPClient: srv.Client(),
		Getenv:     func(string) string { return "" },
		Home:       home,
		Cwd:        work,
	})
	if code != 0 {
		t.Fatalf("exit %d stderr %s", code, stderr)
	}
	if !strings.Contains(stdout.String(), "published") {
		t.Fatalf("stdout %s", stdout)
	}
}

func writeConf(t *testing.T, home, body string) {
	t.Helper()
	dir := filepath.Join(home, ".config", "md2c")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "md2c.conf"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}
