package lint

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLintValidMarkdown(t *testing.T) {
	md := []byte("<!-- space:DOC,path:Guides,title:Test -->\n# Test Page\n\nHere is a paragraph with [a link](https://example.com) and a callout:\n\n> [!NOTE]\n> This is a callout note.\n\n| Header 1 | Header 2 |\n| -------- | -------- |\n| Val 1    | Val 2    |\n\n```mermaid\ngraph TD\n    A --> B\n```\n")

	issues := Lint(md, LintOptions{})
	if len(issues) > 0 {
		t.Errorf("Expected 0 issues for valid markdown, got %d: %v", len(issues), issues)
	}
}

func TestLintUnclosedCodeBlock(t *testing.T) {
	md := []byte("# Heading\n\n```go\nfunc main() {\n}\n")

	issues := Lint(md, LintOptions{})
	found := false
	for _, issue := range issues {
		if issue.Rule == "unclosed-code-block" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected 'unclosed-code-block' issue, got: %v", issues)
	}
}

func TestLintHeadingSpace(t *testing.T) {
	md := []byte(`#InvalidHeading`)

	issues := Lint(md, LintOptions{})
	found := false
	for _, issue := range issues {
		if issue.Rule == "heading-space" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected 'heading-space' issue, got: %v", issues)
	}
}

func TestLintInvalidMermaid(t *testing.T) {
	md := []byte("```mermaid\ninvalid_diagram_type A --> B\n```")

	issues := Lint(md, LintOptions{})
	found := false
	for _, issue := range issues {
		if issue.Rule == "mermaid-syntax" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected 'mermaid-syntax' issue, got: %v", issues)
	}
}

func TestLintMissingLocalImage(t *testing.T) {
	tmpDir := t.TempDir()
	md := []byte(`![Missing Image](./nonexistent_image.png)`)

	issues := Lint(md, LintOptions{BaseDir: tmpDir})
	found := false
	for _, issue := range issues {
		if issue.Rule == "missing-local-image" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected 'missing-local-image' issue, got: %v", issues)
	}

	// Create the image and test again
	_ = os.WriteFile(filepath.Join(tmpDir, "nonexistent_image.png"), []byte("fake"), 0644)
	issuesFixed := Lint(md, LintOptions{BaseDir: tmpDir})
	for _, issue := range issuesFixed {
		if issue.Rule == "missing-local-image" {
			t.Errorf("Did not expect 'missing-local-image' issue after file creation, got: %v", issue)
		}
	}
}

func TestLintEmptyLinkDest(t *testing.T) {
	md := []byte(`[Empty Link]()`)

	issues := Lint(md, LintOptions{})
	found := false
	for _, issue := range issues {
		if issue.Rule == "empty-link-dest" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected 'empty-link-dest' issue, got: %v", issues)
	}
}

func TestLintUnknownCalloutType(t *testing.T) {
	md := []byte("> [!NOT1E]\n> Sample content\n")

	issues := Lint(md, LintOptions{})
	found := false
	for _, issue := range issues {
		if issue.Rule == "unknown-callout-type" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected 'unknown-callout-type' issue for [!NOT1E], got: %v", issues)
	}
}
