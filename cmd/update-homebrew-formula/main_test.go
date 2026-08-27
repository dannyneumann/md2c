package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const fixtureSUMS = `
aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  md2c_v0.1.0_darwin_amd64
bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb  md2c_v0.1.0_darwin_arm64
cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc  md2c_v0.1.0_linux_amd64
dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd  md2c_v0.1.0_linux_arm64
`

func TestRunWritesFormula(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	sums := filepath.Join(dir, "SHA256SUMS")
	if err := os.WriteFile(sums, []byte(fixtureSUMS), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "Formula", "md2c.rb")
	if code := run([]string{"-version", "v0.1.0", "-sums", sums, "-out", out}, io.Discard); code != 0 {
		t.Fatalf("exit %d", code)
	}
	body, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `version "0.1.0"`) {
		t.Fatalf("missing version:\n%s", body)
	}
}

func TestRunUsage(t *testing.T) {
	t.Parallel()
	if code := run(nil, io.Discard); code != 2 {
		t.Fatalf("got %d", code)
	}
}

func TestRunMissingSums(t *testing.T) {
	t.Parallel()
	if code := run([]string{"-version", "v0.1.0", "-sums", filepath.Join(t.TempDir(), "nope")}, io.Discard); code != 1 {
		t.Fatalf("got %d", code)
	}
}
