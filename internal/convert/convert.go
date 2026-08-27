package convert

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// Convert turns Markdown into Confluence storage format (XHTML + macros).
func Convert(markdown string) (string, error) {
	source := []byte(markdown)
	md := goldmark.New(goldmark.WithExtensions(extension.GFM))
	doc := md.Parser().Parse(text.NewReader(source))
	r := &renderer{source: source}
	if err := ast.Walk(doc, r.walk); err != nil {
		return "", err
	}
	out := r.buf.String()
	if strings.TrimSpace(out) == "" {
		return "<p></p>", nil
	}
	return out, nil
}

// InfoMacro wraps text in a Confluence info panel.
func InfoMacro(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	return `<ac:structured-macro ac:name="info"><ac:rich-text-body><p>` +
		escapeXML(text) +
		`</p></ac:rich-text-body></ac:structured-macro>`
}

type renderer struct {
	source   []byte
	buf      strings.Builder
	inHeader bool
}

func (r *renderer) walk(n ast.Node, entering bool) (ast.WalkStatus, error) {
	switch n := n.(type) {
	case *ast.Document:
		return ast.WalkContinue, nil
	case *ast.Heading:
		if isTOCMarker(r.textContent(n)) {
			if entering {
				r.writeTOC()
			}
			return ast.WalkSkipChildren, nil
		}
		r.toggle(entering, fmt.Sprintf("h%d", n.Level))
	case *ast.Paragraph:
		if isTOCMarker(r.textContent(n)) {
			if entering {
				r.writeTOC()
			}
			return ast.WalkSkipChildren, nil
		}
		r.toggle(entering, "p")
	case *ast.Emphasis:
		if n.Level >= 2 {
			r.toggle(entering, "strong")
		} else {
			r.toggle(entering, "em")
		}
	case *east.Strikethrough:
		r.toggle(entering, "del")
	case *ast.Link:
		if entering {
			fmt.Fprintf(&r.buf, `<a href="%s">`, escapeAttr(string(n.Destination)))
		} else {
			r.buf.WriteString("</a>")
		}
	case *ast.AutoLink:
		if entering {
			u := string(n.URL(r.source))
			fmt.Fprintf(&r.buf, `<a href="%s">%s</a>`, escapeAttr(u), escapeXML(u))
			return ast.WalkSkipChildren, nil
		}
	case *ast.List:
		tag := "ul"
		attrs := ""
		if n.IsOrdered() {
			tag = "ol"
			if n.Start > 1 {
				attrs = fmt.Sprintf(` start="%d"`, n.Start)
			}
		}
		if entering {
			fmt.Fprintf(&r.buf, "<%s%s>", tag, attrs)
		} else {
			fmt.Fprintf(&r.buf, "</%s>", tag)
		}
	case *ast.ListItem:
		r.toggle(entering, "li")
	case *east.TaskCheckBox:
		if entering {
			if n.IsChecked {
				r.buf.WriteString("☑ ")
			} else {
				r.buf.WriteString("☐ ")
			}
		}
	case *ast.TextBlock:
		return ast.WalkContinue, nil
	case *ast.Blockquote:
		r.toggle(entering, "blockquote")
	case *ast.ThematicBreak:
		if entering {
			r.buf.WriteString("<hr />")
		}
	case *ast.FencedCodeBlock, *ast.CodeBlock:
		if entering {
			r.writeCode(n)
			return ast.WalkSkipChildren, nil
		}
	case *ast.CodeSpan:
		r.toggle(entering, "code")
	case *ast.Image:
		if entering {
			r.writeImage(string(n.Destination), r.textContent(n))
			return ast.WalkSkipChildren, nil
		}
	case *east.Table:
		if entering {
			r.buf.WriteString("<table><tbody>")
		} else {
			r.buf.WriteString("</tbody></table>")
		}
	case *east.TableHeader:
		r.inHeader = entering
		r.toggle(entering, "tr")
	case *east.TableRow:
		r.toggle(entering, "tr")
	case *east.TableCell:
		if r.inHeader {
			r.toggle(entering, "th")
		} else {
			r.toggle(entering, "td")
		}
	case *ast.Text:
		if entering {
			r.buf.WriteString(escapeXML(string(n.Segment.Value(r.source))))
			if n.HardLineBreak() {
				r.buf.WriteString("<br />")
			} else if n.SoftLineBreak() {
				r.buf.WriteByte('\n')
			}
		}
	case *ast.String:
		if entering {
			r.buf.WriteString(escapeXML(string(n.Value)))
		}
	case *ast.RawHTML:
		if entering {
			r.buf.WriteString(escapeXML(r.segmentsText(n.Segments)))
			return ast.WalkSkipChildren, nil
		}
	case *ast.HTMLBlock:
		if entering {
			r.buf.WriteString(escapeXML(r.linesText(n)))
			return ast.WalkSkipChildren, nil
		}
	}
	return ast.WalkContinue, nil
}

func (r *renderer) toggle(entering bool, tag string) {
	if entering {
		fmt.Fprintf(&r.buf, "<%s>", tag)
	} else {
		fmt.Fprintf(&r.buf, "</%s>", tag)
	}
}

func (r *renderer) writeCode(n ast.Node) {
	lang := "none"
	if fenced, ok := n.(*ast.FencedCodeBlock); ok {
		if l := strings.TrimSpace(string(fenced.Language(r.source))); l != "" {
			lang = mapLanguage(l)
		}
	}
	var code strings.Builder
	lines := n.Lines()
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		code.Write(line.Value(r.source))
	}
	src := code.String()
	switch strings.ToLower(lang) {
	case "mermaid":
		if puml, ok := mermaidToPlantUML(src); ok {
			r.writePlantUML(puml)
			return
		}
	case "plantuml", "puml":
		r.writePlantUML(src)
		return
	}
	body := strings.ReplaceAll(src, "]]>", "]]]]><![CDATA[>")
	fmt.Fprintf(&r.buf,
		`<ac:structured-macro ac:name="code"><ac:parameter ac:name="language">%s</ac:parameter><ac:plain-text-body><![CDATA[%s]]></ac:plain-text-body></ac:structured-macro>`,
		escapeAttr(lang), body)
}

func (r *renderer) writeTOC() {
	r.buf.WriteString(`<ac:structured-macro ac:name="toc"></ac:structured-macro>`)
}

func isTOCMarker(s string) bool {
	s = strings.TrimSpace(s)
	s = strings.TrimLeft(s, "#")
	s = strings.TrimSpace(s)
	switch strings.ToLower(s) {
	case "[toc]", "[[toc]]":
		return true
	default:
		return false
	}
}

func (r *renderer) writePlantUML(src string) {
	src = strings.TrimSpace(src)
	if !strings.Contains(strings.ToLower(src), "@startuml") {
		src = "@startuml\n" + src + "\n@enduml\n"
	}
	body := strings.ReplaceAll(src, "]]>", "]]]]><![CDATA[>")
	fmt.Fprintf(&r.buf,
		`<ac:structured-macro ac:name="plantuml" ac:schema-version="1"><ac:parameter ac:name="atlassian-macro-output-type">INLINE</ac:parameter><ac:plain-text-body><![CDATA[%s]]></ac:plain-text-body></ac:structured-macro>`,
		body)
}

func (r *renderer) writeImage(dest, alt string) {
	if isRemoteURL(dest) {
		fmt.Fprintf(&r.buf, `<ac:image ac:alt="%s"><ri:url ri:value="%s" /></ac:image>`,
			escapeAttr(alt), escapeAttr(dest))
		return
	}
	label := dest
	if alt != "" {
		label = alt + " (" + dest + ")"
	}
	fmt.Fprintf(&r.buf, "[image: %s]", escapeXML(label))
}

func (r *renderer) textContent(n ast.Node) string {
	var b strings.Builder
	_ = ast.Walk(n, func(child ast.Node, entering bool) (ast.WalkStatus, error) {
		if child == n {
			return ast.WalkContinue, nil
		}
		if t, ok := child.(*ast.Text); ok && entering {
			b.Write(t.Segment.Value(r.source))
		}
		if s, ok := child.(*ast.String); ok && entering {
			b.Write(s.Value)
		}
		return ast.WalkContinue, nil
	})
	return b.String()
}

func (r *renderer) segmentsText(segs *text.Segments) string {
	if segs == nil {
		return ""
	}
	var b strings.Builder
	for i := 0; i < segs.Len(); i++ {
		seg := segs.At(i)
		b.Write(seg.Value(r.source))
	}
	return b.String()
}

func (r *renderer) linesText(n ast.Node) string {
	var b strings.Builder
	lines := n.Lines()
	if lines == nil {
		return ""
	}
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		b.Write(line.Value(r.source))
	}
	return b.String()
}

func mapLanguage(lang string) string {
	switch strings.ToLower(lang) {
	case "js", "javascript", "node":
		return "javascript"
	case "ts", "typescript":
		return "typescript"
	case "py", "python":
		return "python"
	case "sh", "zsh", "shell", "bash":
		return "bash"
	case "yml":
		return "yaml"
	case "golang":
		return "go"
	case "cs", "csharp":
		return "csharp"
	case "c++", "cpp":
		return "cpp"
	default:
		return lang
	}
}

func isRemoteURL(dest string) bool {
	u, err := url.Parse(dest)
	if err != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func escapeAttr(s string) string {
	s = escapeXML(s)
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}
