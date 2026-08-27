package homebrew

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const fixtureSUMS = `
# comment
aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  md2c_v0.1.0_darwin_amd64
bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb *md2c_v0.1.0_darwin_arm64
cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc  md2c_v0.1.0_linux_amd64
dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd  md2c_v0.1.0_linux_arm64
`

func TestNormalizeVersion(t *testing.T) {
	t.Parallel()
	if got := NormalizeVersion("v0.2.2"); got != "0.2.2" {
		t.Fatalf("got %q", got)
	}
	if got := NormalizeVersion("0.2.2"); got != "0.2.2" {
		t.Fatalf("got %q", got)
	}
}

func TestParseSHA256SUMS(t *testing.T) {
	t.Parallel()
	sums, err := ParseSHA256SUMS(strings.NewReader(fixtureSUMS))
	if err != nil {
		t.Fatal(err)
	}
	if sums["md2c_v0.1.0_darwin_arm64"] != "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" {
		t.Fatalf("binary-mode *file: %+v", sums)
	}
	if len(sums) != 4 {
		t.Fatalf("got %d entries", len(sums))
	}
}

func TestParseSHA256SUMSRejectsBadDigest(t *testing.T) {
	t.Parallel()
	_, err := ParseSHA256SUMS(strings.NewReader("not-a-hash  md2c_v0.1.0_darwin_amd64\n"))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGenerate(t *testing.T) {
	t.Parallel()
	sums, err := ParseSHA256SUMS(strings.NewReader(fixtureSUMS))
	if err != nil {
		t.Fatal(err)
	}
	got, err := Generate("v0.1.0", sums)
	if err != nil {
		t.Fatal(err)
	}
	wantPath := filepath.Join("testdata", "md2c.rb")
	want, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Fatalf("formula mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestGenerateMissingArtifact(t *testing.T) {
	t.Parallel()
	_, err := Generate("0.1.0", map[string]string{
		"md2c_v0.1.0_darwin_amd64": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	})
	if err == nil || !strings.Contains(err.Error(), "md2c_v0.1.0_darwin_arm64") {
		t.Fatalf("expected missing darwin_arm64, got %v", err)
	}
}

func TestWriteFormula(t *testing.T) {
	t.Parallel()
	sums, err := ParseSHA256SUMS(strings.NewReader(fixtureSUMS))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "Formula", "md2c.rb")
	if err := WriteFormula(path, "0.1.0", sums); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), `version "0.1.0"`) {
		t.Fatalf("missing version:\n%s", got)
	}
}
