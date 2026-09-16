package confluence

import (
	"bytes"
	"strings"
	"testing"
)

func TestComputeDiff(t *testing.T) {
	remote := "Line 1\nLine 2\nLine 3"
	local := "Line 1\nLine 2 (modified)\nLine 3"

	diff := ComputeDiff(remote, local)
	if len(diff) == 0 {
		t.Fatalf("expected non-empty diff")
	}

	hasMinus := false
	hasPlus := false
	for _, l := range diff {
		if l.Type == '-' && strings.Contains(l.Content, "Line 2") {
			hasMinus = true
		}
		if l.Type == '+' && strings.Contains(l.Content, "Line 2 (modified)") {
			hasPlus = true
		}
	}
	if !hasMinus || !hasPlus {
		t.Errorf("diff did not correctly capture modified line: %+v", diff)
	}
}

func TestComputeDiffHandlesDifferentLineCounts(t *testing.T) {
	t.Parallel()
	diff := ComputeDiff("<p>remote content</p>", "<p>local content</p>\n<p>second line</p>")
	if len(diff) == 0 {
		t.Fatal("expected non-empty diff")
	}
	if diff[len(diff)-1].Type != '+' {
		t.Fatalf("expected trailing local line, got %+v", diff)
	}
}

func TestPrintDiff(t *testing.T) {
	diff := []DiffLine{
		{Type: ' ', Content: "same"},
		{Type: '-', Content: "old"},
		{Type: '+', Content: "new"},
	}

	var buf bytes.Buffer
	PrintDiff(&buf, false, diff)

	out := buf.String()
	if !strings.Contains(out, "--- Confluence Remote (Formatted)") || !strings.Contains(out, "+++ Lokale Datei (Formatted)") {
		t.Errorf("PrintDiff output missing formatted headers: %s", out)
	}
	if !strings.Contains(out, "- old") || !strings.Contains(out, "+ new") {
		t.Errorf("PrintDiff output invalid: %s", out)
	}
}

func TestPromptConflictChoice(t *testing.T) {
	tests := []struct {
		input    string
		expected ConflictAction
	}{
		{"o\n", ActionOverwrite},
		{"overwrite\n", ActionOverwrite},
		{"m\n", ActionMerge},
		{"merge\n", ActionMerge},
		{"a\n", ActionAbort},
		{"invalid\n", ActionAbort},
	}

	for _, tt := range tests {
		in := strings.NewReader(tt.input)
		var out bytes.Buffer
		act := PromptConflictChoice(in, &out)
		if act != tt.expected {
			t.Errorf("PromptConflictChoice(%q) = %v, expected %v", tt.input, act, tt.expected)
		}
	}
}

func TestNormalizeStorageHTML(t *testing.T) {
	remote := `<p>Nach dem Login in <strong>oneITSM</strong> wird im Menü der Punkt <strong>&quot;Change Management&quot;</strong> ausgewählt.</p>`
	local := `<p>Nach dem Login in <strong>oneITSM</strong> wird im Menü der Punkt <strong>"Change Management"</strong> ausgewählt.</p>`

	normRemote := NormalizeStorageHTML(remote)
	normLocal := NormalizeStorageHTML(local)

	if normRemote != normLocal {
		t.Errorf("NormalizeStorageHTML failed to equate remote and local:\nRemote: %s\nLocal:  %s", normRemote, normLocal)
	}

	if NormalizeStorageHTML(`<span style="color: rgb(222,49,99);">text</span>`) != NormalizeStorageHTML(`<span style="color: rgb(222, 49, 99);">text</span>`) {
		t.Fatal("NormalizeStorageHTML should ignore whitespace in rgb colors")
	}

	if got := NormalizeStorageHTML(`<p style="color: var(--accent);">Remote</p><p>Next</p>`); got != `<p style="color:var(--accent);">Remote</p>
<p>Next</p>` {
		t.Fatalf("NormalizeStorageHTML should normalize style whitespace and format tags, got %q", got)
	}
}
