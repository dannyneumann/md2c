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

func TestPrintDiff(t *testing.T) {
	diff := []DiffLine{
		{Type: ' ', Content: "same"},
		{Type: '-', Content: "old"},
		{Type: '+', Content: "new"},
	}

	var buf bytes.Buffer
	PrintDiff(&buf, false, diff)

	out := buf.String()
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
	remote := `<ac:structured-macro ac:name="info" ac:schema-version="1" ac:macro-id="810ad198-980c-4713-b8a2-7f54b4a83558"><p>Maske &quot;Neuer Change&quot;</p></ac:structured-macro>`
	local := `<ac:structured-macro ac:name="info"><p>Maske "Neuer Change"</p></ac:structured-macro>`

	normRemote := NormalizeStorageHTML(remote)
	normLocal := NormalizeStorageHTML(local)

	if normRemote != normLocal {
		t.Errorf("NormalizeStorageHTML failed to equate remote and local:\nRemote: %s\nLocal:  %s", normRemote, normLocal)
	}
}
