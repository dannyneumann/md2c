package skill

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetTargetsAll(t *testing.T) {
	home := "/home/user"
	cwd := "/work/project"

	targets, err := GetTargets(home, cwd, "all")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(targets) != 3 { // gemini, codex, cursor/agents (unique paths)
		t.Fatalf("expected 3 unique target paths for 'all', got %d: %+v", len(targets), targets)
	}
}

func TestGetTargetsSpecific(t *testing.T) {
	home := "/home/user"
	cwd := "/work/project"

	targets, err := GetTargets(home, cwd, "codex")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(targets) != 1 || targets[0].Name != "codex" {
		t.Fatalf("expected 1 codex target, got %+v", targets)
	}
}

func TestGetTargetsInvalid(t *testing.T) {
	_, err := GetTargets("/home", "/work", "unknown_agent")
	if err == nil {
		t.Fatal("expected error for unknown agent target, got nil")
	}
}

func TestInstall(t *testing.T) {
	dir := t.TempDir()
	targetPath := filepath.Join(dir, "skills", "md2c", "SKILL.md")

	targets := []Target{{Name: "test", Path: targetPath}}
	installed, err := Install(targets)
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	if len(installed) != 1 || installed[0] != targetPath {
		t.Fatalf("expected installed path %q, got %v", targetPath, installed)
	}

	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		t.Fatalf("expected file to exist at %s", targetPath)
	}
}
