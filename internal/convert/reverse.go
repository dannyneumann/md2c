package convert

import (
	"fmt"
	"strings"

	"golang.org/x/net/html"
)

// ToMarkdown converts Confluence Storage Format (XHTML + macros) back to Markdown.
// It returns the generated Markdown string and a slice of referenced attachment filenames.
func ToMarkdown(xhtml string) (string, []string, error) {
	doc, err := html.Parse(strings.NewReader(xhtml))
	if err != nil {
		return "", nil, fmt.Errorf("parse xhtml: %w", err)
	}

	w := &reverseWriter{}
	w.walk(doc)

	res := strings.TrimSpace(w.buf.String())
	if res != "" {
		res += "\n"
	}
	return res, w.attachments, nil
}

type reverseWriter struct {
	buf          strings.Builder
	attachments  []string
	inCode       bool
	inBlockquote bool
	inTable      bool
	tableRows    [][]string
	currentRow   []string
	listDepth    int
	listTypes    []string // "ul" or "ol"
}

func (w *reverseWriter) walk(n *html.Node) {
	if n == nil {
		return
	}

	switch n.Type {
	case html.DocumentNode:
		w.walkChildren(n)

	case html.TextNode:
		text := n.Data
		if w.inCode {
			w.buf.WriteString(text)
		} else {
			if w.inBlockquote {
				text = strings.ReplaceAll(text, "\n", "\n> ")
			}
			w.buf.WriteString(text)
		}

	case html.ElementNode:
		tag := strings.ToLower(n.Data)
		switch tag {
		case "h1", "h2", "h3", "h4", "h5", "h6":
			level := int(tag[1] - '0')
			w.ensureNewline()
			w.buf.WriteString(strings.Repeat("#", level) + " ")
			w.walkChildren(n)
			w.buf.WriteString("\n\n")

		case "p":
			w.ensureNewline()
			w.walkChildren(n)
			w.buf.WriteString("\n\n")

		case "strong", "b":
			w.buf.WriteString("**")
			w.walkChildren(n)
			w.buf.WriteString("**")

		case "em", "i":
			w.buf.WriteString("*")
			w.walkChildren(n)
			w.buf.WriteString("*")

		case "del", "strike":
			w.buf.WriteString("~~")
			w.walkChildren(n)
			w.buf.WriteString("~~")

		case "code":
			if !w.inCode {
				w.buf.WriteString("`")
				w.walkChildren(n)
				w.buf.WriteString("`")
			} else {
				w.walkChildren(n)
			}

		case "a":
			href := getAttr(n, "href")
			w.buf.WriteString("[")
			w.walkChildren(n)
			w.buf.WriteString("]")
			if href != "" {
				fmt.Fprintf(&w.buf, "(%s)", href)
			}

		case "hr":
			w.ensureNewline()
			w.buf.WriteString("---\n\n")

		case "br":
			w.buf.WriteString("\n")

		case "blockquote":
			w.ensureNewline()
			w.buf.WriteString("> ")
			oldInBQ := w.inBlockquote
			w.inBlockquote = true
			w.walkChildren(n)
			w.inBlockquote = oldInBQ
			w.buf.WriteString("\n\n")

		case "ul", "ol":
			w.ensureNewline()
			w.listDepth++
			w.listTypes = append(w.listTypes, tag)
			w.walkChildren(n)
			w.listTypes = w.listTypes[:len(w.listTypes)-1]
			w.listDepth--
			w.ensureNewline()

		case "li":
			w.ensureNewline()
			indent := strings.Repeat("  ", w.listDepth-1)
			marker := "- "
			if len(w.listTypes) > 0 && w.listTypes[len(w.listTypes)-1] == "ol" {
				marker = "1. "
			}
			w.buf.WriteString(indent + marker)
			w.walkChildren(n)
			w.buf.WriteString("\n")

		case "table":
			w.ensureNewline()
			oldInTable := w.inTable
			w.inTable = true
			w.tableRows = nil
			w.walkChildren(n)
			w.renderTable()
			w.inTable = oldInTable
			w.buf.WriteString("\n")

		case "tr":
			w.currentRow = nil
			w.walkChildren(n)
			if len(w.currentRow) > 0 {
				w.tableRows = append(w.tableRows, w.currentRow)
			}

		case "th", "td":
			cellBuf := &strings.Builder{}
			oldBuf := w.buf
			w.buf = strings.Builder{}
			w.walkChildren(n)
			cellStr := strings.TrimSpace(w.buf.String())
			w.buf = oldBuf
			w.currentRow = append(w.currentRow, cellStr)
			_ = cellBuf

		case "ac:structured-macro":
			name := getAttr(n, "ac:name")
			switch name {
			case "toc":
				w.ensureNewline()
				w.buf.WriteString("[TOC]\n\n")
				return

			case "info", "tip", "note", "warning":
				w.ensureNewline()
				marker := "NOTE"
				switch name {
				case "info":
					marker = "NOTE"
				case "tip":
					marker = "TIP"
				case "note":
					marker = "IMPORTANT"
				case "warning":
					marker = "WARNING"
				}
				w.buf.WriteString("> [!" + marker + "]\n> ")
				oldInBQ := w.inBlockquote
				w.inBlockquote = true
				bodyNode := findChild(n, "ac:rich-text-body")
				if bodyNode != nil {
					w.walkChildren(bodyNode)
				}
				w.inBlockquote = oldInBQ
				w.buf.WriteString("\n\n")
				return

			case "code":
				w.ensureNewline()
				lang := findParam(n, "language")
				body := findText(n, "ac:plain-text-body")
				fmt.Fprintf(&w.buf, "```%s\n%s\n```\n\n", lang, strings.TrimSuffix(body, "\n"))
				return

			case "plantuml":
				w.ensureNewline()
				body := findText(n, "ac:plain-text-body")
				body = strings.TrimPrefix(body, "@startuml\n")
				body = strings.TrimSuffix(body, "\n@enduml\n")
				body = strings.TrimSuffix(body, "@enduml\n")
				body = strings.TrimSuffix(body, "@enduml")
				fmt.Fprintf(&w.buf, "```plantuml\n%s\n```\n\n", strings.TrimSpace(body))
				return

			default:
				w.walkChildren(n)
			}

		case "ac:image":
			alt := getAttr(n, "ac:alt")
			urlVal := findAttrValue(n, "ri:url", "ri:value")
			attVal := findAttrValue(n, "ri:attachment", "ri:filename")

			if urlVal != "" {
				fmt.Fprintf(&w.buf, "![%s](%s)", alt, urlVal)
			} else if attVal != "" {
				fmt.Fprintf(&w.buf, "![%s](%s)", alt, attVal)
				w.addAttachment(attVal)
			}

		default:
			w.walkChildren(n)
		}
	}
}

func (w *reverseWriter) walkChildren(n *html.Node) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		w.walk(c)
	}
}

func (w *reverseWriter) ensureNewline() {
	s := w.buf.String()
	if len(s) > 0 && !strings.HasSuffix(s, "\n") {
		w.buf.WriteString("\n")
	}
}

func (w *reverseWriter) addAttachment(name string) {
	for _, a := range w.attachments {
		if a == name {
			return
		}
	}
	w.attachments = append(w.attachments, name)
}

func (w *reverseWriter) renderTable() {
	if len(w.tableRows) == 0 {
		return
	}
	headers := w.tableRows[0]
	fmt.Fprintf(&w.buf, "| %s |\n", strings.Join(headers, " | "))
	sep := make([]string, len(headers))
	for i := range sep {
		sep[i] = "---"
	}
	fmt.Fprintf(&w.buf, "| %s |\n", strings.Join(sep, " | "))
	for _, row := range w.tableRows[1:] {
		fmt.Fprintf(&w.buf, "| %s |\n", strings.Join(row, " | "))
	}
}

func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return a.Val
		}
	}
	return ""
}

func findChild(n *html.Node, tag string) *html.Node {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode {
			tName := c.Data
			if c.Namespace != "" {
				tName = c.Namespace + ":" + c.Data
			}
			if strings.EqualFold(tName, tag) || strings.EqualFold(c.Data, tag) {
				return c
			}
		}
	}
	return nil
}

func findParam(n *html.Node, paramName string) string {
	var val string
	var search func(*html.Node)
	search = func(node *html.Node) {
		if node.Type == html.ElementNode && strings.EqualFold(node.Data, "ac:parameter") {
			if getAttr(node, "ac:name") == paramName {
				val = nodeText(node)
				return
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			search(c)
		}
	}
	search(n)
	return val
}

func findText(n *html.Node, tag string) string {
	target := findChild(n, tag)
	if target != nil {
		return nodeText(target)
	}
	return ""
}

func findAttrValue(n *html.Node, elementTag, attrKey string) string {
	var val string
	var search func(*html.Node)
	search = func(node *html.Node) {
		if node.Type == html.ElementNode && strings.EqualFold(node.Data, elementTag) {
			val = getAttr(node, attrKey)
			return
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			search(c)
		}
	}
	search(n)
	return val
}

func nodeText(n *html.Node) string {
	var b strings.Builder
	var walkText func(*html.Node)
	walkText = func(node *html.Node) {
		if node.Type == html.TextNode || node.Type == html.CommentNode || node.Type == html.RawNode {
			b.WriteString(node.Data)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walkText(c)
		}
	}
	walkText(n)
	return b.String()
}
