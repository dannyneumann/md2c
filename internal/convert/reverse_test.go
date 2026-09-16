package convert

import (
	"strings"
	"testing"
)

func TestToMarkdownHeadingsAndParagraphs(t *testing.T) {
	t.Parallel()
	xhtml := `<h1>Title</h1><p>Hello <strong>world</strong> and <em>italics</em>.</p>`
	got, atts, err := ToMarkdown(xhtml)
	if err != nil {
		t.Fatalf("ToMarkdown: %v", err)
	}
	if len(atts) != 0 {
		t.Fatalf("expected 0 attachments, got %v", atts)
	}
	if !strings.Contains(got, "# Title") {
		t.Fatalf("missing heading in:\n%s", got)
	}
	if !strings.Contains(got, "Hello **world** and *italics*.") {
		t.Fatalf("missing formatted paragraph in:\n%s", got)
	}
}

func TestToMarkdownCallouts(t *testing.T) {
	t.Parallel()
	xhtml := `<ac:structured-macro ac:name="info"><ac:rich-text-body><p>Useful info</p></ac:rich-text-body></ac:structured-macro>`
	got, _, err := ToMarkdown(xhtml)
	if err != nil {
		t.Fatalf("ToMarkdown: %v", err)
	}
	if !strings.Contains(got, "> [!NOTE]") {
		t.Fatalf("missing note callout in:\n%s", got)
	}
	if !strings.Contains(got, "Useful info") {
		t.Fatalf("missing text in:\n%s", got)
	}
}

func TestToMarkdownCalloutSkipsEmptyParagraphs(t *testing.T) {
	t.Parallel()
	xhtml := `<ac:structured-macro ac:name="info"><ac:rich-text-body><p></p><p>Die Bearbeitung erfolgt im Ticket.</p></ac:rich-text-body></ac:structured-macro>`

	got, _, err := ToMarkdown(xhtml)
	if err != nil {
		t.Fatalf("ToMarkdown: %v", err)
	}
	want := "> [!NOTE]\n> Die Bearbeitung erfolgt im Ticket.\n"
	if got != want {
		t.Fatalf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestToMarkdownCodeBlock(t *testing.T) {
	t.Parallel()
	xhtml := `<ac:structured-macro ac:name="code"><ac:parameter ac:name="language">go</ac:parameter><ac:plain-text-body><![CDATA[fmt.Println("hi")]]></ac:plain-text-body></ac:structured-macro>`
	got, _, err := ToMarkdown(xhtml)
	if err != nil {
		t.Fatalf("ToMarkdown: %v", err)
	}
	if !strings.Contains(got, "```go") || !strings.Contains(got, `fmt.Println("hi")`) {
		t.Fatalf("unexpected code block:\n%s", got)
	}
}

func TestToMarkdownImageAttachment(t *testing.T) {
	t.Parallel()
	xhtml := `<ac:image ac:alt="Logo"><ri:attachment ri:filename="logo.png" /></ac:image>`
	got, atts, err := ToMarkdown(xhtml)
	if err != nil {
		t.Fatalf("ToMarkdown: %v", err)
	}
	if !strings.Contains(got, "![Logo](logo.png)") {
		t.Fatalf("missing image in:\n%s", got)
	}
	if len(atts) != 1 || atts[0] != "logo.png" {
		t.Fatalf("unexpected attachments: %v", atts)
	}
}

func TestToMarkdownJiraMacro(t *testing.T) {
	t.Parallel()
	xhtml := `<ac:structured-macro ac:name="jira"><ac:parameter ac:name="server">Example Jira</ac:parameter><ac:parameter ac:name="columns">issuekey,summary,status</ac:parameter><ac:parameter ac:name="key">PROJ-123</ac:parameter></ac:structured-macro>`

	got, _, err := ToMarkdown(xhtml)
	if err != nil {
		t.Fatalf("ToMarkdown: %v", err)
	}

	for _, want := range []string{
		"> **Jira-Makro** (Example Jira)",
		"> Spalten: `issuekey,summary,status`",
		"> Key: `PROJ-123`",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Jiraissuekey") {
		t.Fatalf("Jira storage parameters leaked into Markdown:\n%s", got)
	}
}
