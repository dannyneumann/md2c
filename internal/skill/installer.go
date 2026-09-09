package skill

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed embedded/SKILL.md
var embeddedSkill []byte

// Target represents an AI agent target directory.
type Target struct {
	Name string
	Path string
}

// GetTargets returns the target paths for skill installation.
func GetTargets(homeDir, cwd, targetName string) ([]Target, error) {
	allTargets := []Target{
		{Name: "gemini", Path: filepath.Join(homeDir, ".gemini", "skills", "md2c", "SKILL.md")},
		{Name: "agy", Path: filepath.Join(homeDir, ".gemini", "skills", "md2c", "SKILL.md")},
		{Name: "codex", Path: filepath.Join(homeDir, ".codex", "skills", "md2c", "SKILL.md")},
		{Name: "cursor", Path: filepath.Join(cwd, ".agents", "skills", "md2c", "SKILL.md")},
		{Name: "agents", Path: filepath.Join(cwd, ".agents", "skills", "md2c", "SKILL.md")},
	}

	targetName = strings.ToLower(strings.TrimSpace(targetName))
	if targetName == "" || targetName == "all" {
		// Unique by path
		seen := make(map[string]bool)
		var unique []Target
		for _, t := range allTargets {
			if !seen[t.Path] {
				seen[t.Path] = true
				unique = append(unique, t)
			}
		}
		return unique, nil
	}

	var matched []Target
	for _, t := range allTargets {
		if t.Name == targetName {
			matched = append(matched, t)
			break
		}
	}

	if len(matched) == 0 {
		return nil, fmt.Errorf("unbekanntes AI-Ziel '%s'. Erlaubt: agy, gemini, codex, cursor, agents, all", targetName)
	}

	return matched, nil
}

// Install writes the embedded SKILL.md to the specified targets.
func Install(targets []Target) ([]string, error) {
	var installed []string
	for _, target := range targets {
		dir := filepath.Dir(target.Path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return installed, fmt.Errorf("Ordner %s konnte nicht erstellt werden: %w", dir, err)
		}
		if err := os.WriteFile(target.Path, embeddedSkill, 0644); err != nil {
			return installed, fmt.Errorf("Skill konnte nicht nach %s geschrieben werden: %w", target.Path, err)
		}
		installed = append(installed, target.Path)
	}
	return installed, nil
}
