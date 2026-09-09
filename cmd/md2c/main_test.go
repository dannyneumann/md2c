package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
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
	if !strings.HasPrefix(got, "md2c ") {
		t.Fatalf("missing name in %q", got)
	}
	if strings.Contains(got, "source") || strings.Contains(got, "optimized by") {
		t.Fatalf("banner should be name and version only: %q", got)
	}
	if strings.Count(got, "\n") != 1 {
		t.Fatalf("expected a single line, got %q", got)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}

func TestVersionScript(t *testing.T) {
	t.Parallel()
	out, err := exec.Command("sh", filepath.Join(repoRoot(t), "scripts", "version.sh")).Output()
	if err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(string(out))
	if !strings.HasPrefix(got, "v0.") {
		t.Fatalf("version %q", got)
	}
	if strings.Contains(got, "-g") {
		t.Fatalf("must not use git describe --always: %q", got)
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
	if !strings.Contains(stderr.String(), "NO_COLOR") {
		t.Fatalf("missing color note: %s", stderr)
	}
	if !strings.Contains(stderr.String(), "[TOC]") {
		t.Fatalf("missing toc note: %s", stderr)
	}
	if !strings.Contains(stderr.String(), "[!NOTE]") {
		t.Fatalf("missing callout note: %s", stderr)
	}
}

func TestRunPull(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/content") {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"results": []map[string]any{
					{
						"id":    "123",
						"type":  "page",
						"title": "Onboarding",
						"space": map[string]string{"key": "DEV"},
						"version": map[string]int{
							"number": 1,
						},
						"body": map[string]any{
							"storage": map[string]string{
								"value": "<h1>Onboarding</h1><p>Welcome to the team.</p>",
							},
						},
					},
				},
				"size": 1,
			})
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	confPath := filepath.Join(dir, "md2c.conf")
	if err := os.WriteFile(confPath, []byte("MD2C_BASE_URL="+srv.URL+"\nMD2C_USER=me\nMD2C_TOKEN=token\n"), 0644); err != nil {
		t.Fatal(err)
	}

	stdout, stderr := &strings.Builder{}, &strings.Builder{}
	code := run([]string{"--config=" + confPath, "pull", "DEV", "Onboarding"}, runtime{
		Stdout:     stdout,
		Stderr:     stderr,
		HTTPClient: srv.Client(),
		Home:       dir,
	})
	if code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, stderr)
	}
	defer os.Remove("Onboarding.md")

	if !strings.Contains(stdout.String(), "PULL") {
		t.Fatalf("stdout output missing summary: %s", stdout)
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
	if !strings.Contains(stderr.String(), "ZIEL") {
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
	if !strings.Contains(stderr.String(), "DOC") {
		t.Fatalf("stderr %s", stderr)
	}
	if !strings.Contains(stdout.String(), "Seite angelegt") {
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
	if !strings.Contains(stderr.String(), "Datei konnte nicht gelesen werden") {
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
	if !strings.Contains(stdout.String(), "Seite angelegt") {
		t.Fatalf("stdout %s", stdout)
	}
	if !strings.Contains(stdout.String(), "https://acme.atlassian.net/wiki/spaces/DEV/pages/99/Hello") {
		t.Fatalf("missing url: %s", stdout)
	}
	if strings.Contains(stdout.String(), "Seite aktualisiert") {
		t.Fatalf("create looks like update: %s", stdout)
	}
}

func TestRunPublishUpdate(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	if err := os.WriteFile(path, []byte("Hello\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"results": []map[string]any{{
					"id":    "99",
					"type":  "page",
					"title": "Hello",
					"space": map[string]string{"key": "DEV"},
					"version": map[string]int{
						"number": 3,
					},
					"_links": map[string]string{
						"base":  "https://acme.atlassian.net/wiki",
						"webui": "/spaces/DEV/pages/99/Hello",
					},
				}},
				"size": 1,
			})
		case http.MethodPut:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":    "99",
				"type":  "page",
				"title": "Hello",
				"space": map[string]string{"key": "DEV"},
				"version": map[string]int{
					"number": 4,
				},
				"_links": map[string]string{
					"base":  "https://acme.atlassian.net/wiki",
					"webui": "/spaces/DEV/pages/99/Hello",
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
	got := stdout.String()
	if !strings.Contains(got, "Seite aktualisiert") {
		t.Fatalf("stdout %s", got)
	}
	if strings.Contains(got, "Seite angelegt") {
		t.Fatalf("update looks like create: %s", got)
	}
	if !strings.Contains(got, "Version: 4") {
		t.Fatalf("missing version: %s", got)
	}
	if !strings.Contains(stderr.String(), "ZIEL") || !strings.Contains(stderr.String(), "DEV") {
		t.Fatalf("missing target on stderr: %s", stderr)
	}
}

func TestRunPublishWithReason(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	if err := os.WriteFile(path, []byte("Hello\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var capturedReason string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"results": []map[string]any{{
					"id":      "99",
					"type":    "page",
					"title":   "Hello",
					"space":   map[string]string{"key": "DEV"},
					"version": map[string]int{"number": 1},
				}},
				"size": 1,
			})
		case http.MethodPut:
			raw, _ := io.ReadAll(r.Body)
			var payload map[string]any
			_ = json.Unmarshal(raw, &payload)
			if ver, ok := payload["version"].(map[string]any); ok {
				capturedReason, _ = ver["message"].(string)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":      "99",
				"type":    "page",
				"title":   "Hello",
				"space":   map[string]string{"key": "DEV"},
				"version": map[string]int{"number": 2},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	home := t.TempDir()
	writeConf(t, home, "MD2C_BASE_URL="+srv.URL+"\nMD2C_USER=me\nMD2C_TOKEN=token\n")

	stdout, stderr := &strings.Builder{}, &strings.Builder{}
	code := run([]string{"--reason=Fix formatting and typos", path, "DEV", "Hello"}, runtime{
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
	if capturedReason != "Fix formatting and typos" {
		t.Fatalf("expected version message 'Fix formatting and typos', got %q", capturedReason)
	}
}

func TestRunPublishWithTrailingReason(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	if err := os.WriteFile(path, []byte("Hello\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var capturedReason string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"results": []map[string]any{{
					"id":      "99",
					"type":    "page",
					"title":   "Hello",
					"space":   map[string]string{"key": "DEV"},
					"version": map[string]int{"number": 1},
				}},
				"size": 1,
			})
		case http.MethodPut:
			raw, _ := io.ReadAll(r.Body)
			var payload map[string]any
			_ = json.Unmarshal(raw, &payload)
			if ver, ok := payload["version"].(map[string]any); ok {
				capturedReason, _ = ver["message"].(string)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":      "99",
				"type":    "page",
				"title":   "Hello",
				"space":   map[string]string{"key": "DEV"},
				"version": map[string]int{"number": 2},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	home := t.TempDir()
	writeConf(t, home, "MD2C_BASE_URL="+srv.URL+"\nMD2C_USER=me\nMD2C_TOKEN=token\n")

	stdout, stderr := &strings.Builder{}, &strings.Builder{}
	// Pass --reason AFTER the filename and arguments
	code := run([]string{path, "DEV", "Hello", "--reason", "Trailing flag reason"}, runtime{
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
	if capturedReason != "Trailing flag reason" {
		t.Fatalf("expected version message 'Trailing flag reason', got %q", capturedReason)
	}
}

func TestRunPublishError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	if err := os.WriteFile(path, []byte("Hello\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"storage backend down"}`))
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
	if code != 1 {
		t.Fatalf("exit %d stdout %s stderr %s", code, stdout, stderr)
	}
	errOut := stderr.String()
	if !strings.Contains(errOut, "Fehler: Publizieren fehlgeschlagen") {
		t.Fatalf("stderr %s", errOut)
	}
	if !strings.Contains(errOut, "storage backend down") {
		t.Fatalf("missing cause: %s", errOut)
	}
	if stdout.String() != "" {
		t.Fatalf("stdout should be empty on error, got %q", stdout)
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
	if !strings.Contains(stdout.String(), "Seite angelegt") {
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
	if !strings.Contains(stdout.String(), "Seite angelegt") {
		t.Fatalf("stdout %s", stdout)
	}
}

func TestRunLintCommand(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "invalid.md")
	if err := os.WriteFile(path, []byte("#InvalidHeading\n\n```go\nunclosed block\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	stdout, stderr := &strings.Builder{}, &strings.Builder{}
	code := run([]string{"lint", path}, runtime{
		Stdout: stdout,
		Stderr: stderr,
		Cwd:    dir,
	})
	if code != 1 {
		t.Fatalf("expected exit code 1 for invalid markdown, got %d", code)
	}
	if !strings.Contains(stdout.String(), "LINT SYNTAX / FEHLER") {
		t.Fatalf("stdout missing lint header: %s", stdout)
	}
	if !strings.Contains(stdout.String(), "heading-space") || !strings.Contains(stdout.String(), "unclosed-code-block") {
		t.Fatalf("stdout missing expected rules: %s", stdout)
	}
}

func TestRunPublishFailsOnLintError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "invalid.md")
	if err := os.WriteFile(path, []byte("#NoSpaceHeading\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	stdout, stderr := &strings.Builder{}, &strings.Builder{}
	code := run([]string{"-dry-run", path, "DEV", "Page"}, runtime{
		Stdout: stdout,
		Stderr: stderr,
		Cwd:    dir,
	})
	if code != 1 {
		t.Fatalf("expected exit code 1 when linting fails before dry-run/publish, got %d", code)
	}
	if !strings.Contains(stderr.String(), "Markdown-Linting fehlgeschlagen") {
		t.Fatalf("stderr missing lint failure report: %s", stderr)
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

func TestRunPublishDiffOnly(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/content") {
			resp := map[string]any{
				"results": []map[string]any{
					{
						"id":    "123",
						"title": "Page",
						"space": map[string]string{"key": "DEV"},
						"body": map[string]any{
							"storage": map[string]string{"value": "<h1>Remote Header</h1><p>Remote Content</p>"},
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	home := t.TempDir()
	writeConf(t, home, "MD2C_BASE_URL="+srv.URL+"\nMD2C_USER=u\nMD2C_TOKEN=t\n")

	dir := t.TempDir()
	path := filepath.Join(dir, "doc.md")
	_ = os.WriteFile(path, []byte("# Local Header\n\nLocal Content\n"), 0o600)

	stdout, stderr := &strings.Builder{}, &strings.Builder{}
	code := run([]string{path, "DEV", "Page", "--diff-only"}, runtime{
		Home:   home,
		Stdout: stdout,
		Stderr: stderr,
		Cwd:    dir,
	})

	if code != 0 {
		t.Fatalf("expected exit code 0 for diff-only, got %d. stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout.String(), "--- Confluence Remote") || !strings.Contains(stdout.String(), "+++ Lokale Datei") {
		t.Fatalf("stdout missing diff header: %s", stdout)
	}
}

func TestRunPublishFailOnRemoteChange(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/content") {
			resp := map[string]any{
				"results": []map[string]any{
					{
						"id":    "123",
						"title": "Page",
						"space": map[string]string{"key": "DEV"},
						"body": map[string]any{
							"storage": map[string]string{"value": "<h1>Remote Header</h1>"},
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	home := t.TempDir()
	writeConf(t, home, "MD2C_BASE_URL="+srv.URL+"\nMD2C_USER=u\nMD2C_TOKEN=t\n")

	dir := t.TempDir()
	path := filepath.Join(dir, "doc.md")
	_ = os.WriteFile(path, []byte("# Local Header\n"), 0o600)

	stdout, stderr := &strings.Builder{}, &strings.Builder{}
	code := run([]string{path, "DEV", "Page", "--fail-on-remote-change"}, runtime{
		Home:   home,
		Stdout: stdout,
		Stderr: stderr,
		Cwd:    dir,
	})

	if code != 1 {
		t.Fatalf("expected exit code 1 when remote change detected, got %d", code)
	}
	if !strings.Contains(stderr.String(), "Publizieren fehlgeschlagen") {
		t.Fatalf("stderr missing failure text: %s", stderr)
	}
}

