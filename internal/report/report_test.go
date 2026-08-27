package report

import (
	"os"
	"strings"
	"testing"
)

func TestTarget(t *testing.T) {
	t.Parallel()
	var b strings.Builder
	Target(&b, false, "note.md", "PSE", "Decisions/Draft")
	got := b.String()
	wantLines := []string{
		"Ziel",
		"  Datei:   note.md",
		"  Space:   PSE",
		"  Pfad:    Decisions/Draft",
	}
	for _, line := range wantLines {
		if !strings.Contains(got, line) {
			t.Fatalf("missing %q in:\n%s", line, got)
		}
	}
	if strings.Count(got, "\n") < 5 {
		t.Fatalf("expected multi-line target, got %q", got)
	}
}

func TestSuccessCreated(t *testing.T) {
	t.Parallel()
	var b strings.Builder
	Success(&b, false, Result{
		Created: true,
		Title:   "Getting started",
		Version: 1,
		URL:     "https://wiki.example/page",
	})
	got := b.String()
	if !strings.Contains(got, "Seite angelegt") {
		t.Fatalf("missing create headline:\n%s", got)
	}
	if strings.Contains(got, "Seite aktualisiert") {
		t.Fatalf("create report looks like an update:\n%s", got)
	}
	if !strings.Contains(got, "  Titel:   Getting started") {
		t.Fatalf("missing title:\n%s", got)
	}
	if !strings.Contains(got, "  Version: 1") {
		t.Fatalf("missing version:\n%s", got)
	}
	if !strings.Contains(got, "  URL:     https://wiki.example/page") {
		t.Fatalf("missing url:\n%s", got)
	}
	if strings.Count(strings.TrimSpace(got), "\n") < 4 {
		t.Fatalf("expected multi-line success, got %q", got)
	}
}

func TestSuccessUpdated(t *testing.T) {
	t.Parallel()
	var b strings.Builder
	Success(&b, false, Result{
		Created: false,
		Title:   "Draft",
		Version: 12,
		URL:     "https://wiki.example/draft",
	})
	got := b.String()
	if !strings.Contains(got, "Seite aktualisiert") {
		t.Fatalf("missing update headline:\n%s", got)
	}
	if strings.Contains(got, "Seite angelegt") {
		t.Fatalf("update report looks like a create:\n%s", got)
	}
	if !strings.Contains(got, "  Version: 12") {
		t.Fatalf("missing version:\n%s", got)
	}
}

func TestSuccessFallsBackToID(t *testing.T) {
	t.Parallel()
	var b strings.Builder
	Success(&b, false, Result{Title: "T", ID: "99"})
	got := b.String()
	if !strings.Contains(got, "  ID:      99") {
		t.Fatalf("missing id fallback:\n%s", got)
	}
	if strings.Contains(got, "URL:") {
		t.Fatalf("unexpected url:\n%s", got)
	}
}

func TestFailure(t *testing.T) {
	t.Parallel()
	var b strings.Builder
	Failure(&b, false, "Publizieren fehlgeschlagen", `create page "Hello": HTTP 500: boom`)
	got := b.String()
	if !strings.HasPrefix(strings.TrimSpace(got), "Fehler: Publizieren fehlgeschlagen") {
		t.Fatalf("headline:\n%s", got)
	}
	if !strings.Contains(got, `create page "Hello": HTTP 500: boom`) {
		t.Fatalf("missing detail:\n%s", got)
	}
	if strings.Count(got, "\n") < 3 {
		t.Fatalf("expected multi-line error, got %q", got)
	}
}

func TestColoredOutput(t *testing.T) {
	t.Parallel()
	var created, updated, failed strings.Builder
	Success(&created, true, Result{Created: true, Title: "N", Version: 1, URL: "https://x"})
	Success(&updated, true, Result{Created: false, Title: "N", Version: 2, URL: "https://x"})
	Failure(&failed, true, "Datei lesen", "no such file")

	if !strings.Contains(created.String(), green) || !strings.Contains(created.String(), "Seite angelegt") {
		t.Fatalf("create should be green:\n%s", created.String())
	}
	if !strings.Contains(updated.String(), cyan) || !strings.Contains(updated.String(), "Seite aktualisiert") {
		t.Fatalf("update should be cyan:\n%s", updated.String())
	}
	if !strings.Contains(failed.String(), red) || !strings.Contains(failed.String(), "Fehler:") {
		t.Fatalf("error should be red:\n%s", failed.String())
	}
	if !strings.Contains(created.String(), reset) {
		t.Fatalf("missing reset in colored output")
	}
}

func TestEnabledNOCOLOR(t *testing.T) {
	t.Parallel()
	if Enabled(os.Stdout, func(string) string { return "1" }) {
		t.Fatal("NO_COLOR must disable color")
	}
	if Enabled(&strings.Builder{}, func(string) string { return "" }) {
		t.Fatal("non-terminal writer must not use color")
	}
}
