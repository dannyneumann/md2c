package lint

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// Issue represents a linting error or warning.
type Issue struct {
	Line    int
	Rule    string
	Message string
	Level   string // "error" or "warning"
}

func (i Issue) String() string {
	if i.Line > 0 {
		return fmt.Sprintf("Zeile %d [%s]: %s (%s)", i.Line, i.Rule, i.Message, i.Level)
	}
	return fmt.Sprintf("[%s]: %s (%s)", i.Rule, i.Message, i.Level)
}

// LintOptions defines options for linting.
type LintOptions struct {
	BaseDir       string // Base directory to resolve local image files
	RequireHeader bool   // If true, warns if Dateikopf (<!-- space:...,path:... -->) is missing
}

// Lint parses and checks the Markdown source for syntax issues and md2c custom rules.
func Lint(markdown []byte, opts LintOptions) []Issue {
	var issues []Issue

	// 1. Line-by-line checks (e.g. unclosed code fences, malformed headers)
	issues = append(issues, checkLineByLine(markdown, opts)...)

	// 2. AST checks with GFM extension
	md := goldmark.New(goldmark.WithExtensions(extension.GFM))
	reader := text.NewReader(markdown)
	doc := md.Parser().Parse(reader)

	astIssues := checkAST(doc, markdown, opts)
	issues = append(issues, astIssues...)

	return issues
}

func checkLineByLine(source []byte, opts LintOptions) []Issue {
	var issues []Issue
	lines := strings.Split(string(source), "\n")

	inFence := false
	fenceStartLine := 0
	fenceChar := ""

	for idx, line := range lines {
		lineNum := idx + 1
		trimmed := strings.TrimSpace(line)

		// Check fenced code block open/close
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			char := trimmed[:3]
			if !inFence {
				inFence = true
				fenceStartLine = lineNum
				fenceChar = char
			} else {
				if char == fenceChar {
					inFence = false
				}
			}
			continue
		}

		// Check heading syntax (e.g. #Heading without space) - only outside code blocks
		if !inFence && strings.HasPrefix(trimmed, "#") {
			hashes := 0
			for _, ch := range trimmed {
				if ch == '#' {
					hashes++
				} else {
					break
				}
			}
			if hashes <= 6 && len(trimmed) > hashes && trimmed[hashes] != ' ' {
				issues = append(issues, Issue{
					Line:    lineNum,
					Rule:    "heading-space",
					Message: fmt.Sprintf("Fehlendes Leerzeichen nach '%s' in Überschrift", trimmed[:hashes]),
					Level:   "error",
				})
			}
		}
	}

	if inFence {
		issues = append(issues, Issue{
			Line:    fenceStartLine,
			Rule:    "unclosed-code-block",
			Message: "Code-Block (```) wurde nicht geschlossen",
			Level:   "error",
		})
	}

	// Check for md2c header metadata (e.g. <!-- space:...,path:...,title:... -->)
	if opts.RequireHeader {
		hasHeader := false
		for _, line := range lines {
			t := strings.TrimSpace(line)
			if t == "" {
				continue
			}
			if strings.HasPrefix(t, "<!--") && strings.Contains(t, "space:") {
				hasHeader = true
			}
			break
		}
		if !hasHeader {
			issues = append(issues, Issue{
				Line:    1,
				Rule:    "missing-header-metadata",
				Message: "Fehlender md2c-Dateikopf in der ersten Zeile (z. B. <!-- space:DOC,path:Elternseite,title:Seitentitel -->)",
				Level:   "warning",
			})
		}
	}

	return issues
}

func checkAST(doc ast.Node, source []byte, opts LintOptions) []Issue {
	var issues []Issue

	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch node := n.(type) {
		case *ast.FencedCodeBlock:
			lang := strings.TrimSpace(string(node.Language(source)))
			if strings.EqualFold(lang, "mermaid") {
				var code strings.Builder
				lines := node.Lines()
				for i := 0; i < lines.Len(); i++ {
					line := lines.At(i)
					code.Write(line.Value(source))
				}
				lineNum := getLineNum(node, source)
				if err := validateMermaid(code.String()); err != nil {
					issues = append(issues, Issue{
						Line:    lineNum,
						Rule:    "mermaid-syntax",
						Message: fmt.Sprintf("Invalides Mermaid-Diagramm: %v", err),
						Level:   "error",
					})
				}
			}

		case *ast.Image:
			dest := string(node.Destination)
			lineNum := getLineNum(node, source)
			if dest == "" {
				issues = append(issues, Issue{
					Line:    lineNum,
					Rule:    "empty-image-src",
					Message: "Bild-Quelle (URL/Pfad) ist leer",
					Level:   "error",
				})
			} else if !isRemoteURL(dest) {
				// Local attachment check
				if opts.BaseDir != "" {
					imgPath := filepath.Join(opts.BaseDir, dest)
					if _, err := os.Stat(imgPath); os.IsNotExist(err) {
						issues = append(issues, Issue{
							Line:    lineNum,
							Rule:    "missing-local-image",
							Message: fmt.Sprintf("Lokale Bild-Datei nicht gefunden: '%s'", dest),
							Level:   "warning",
						})
					}
				}
			}

		case *ast.Link:
			dest := string(node.Destination)
			lineNum := getLineNum(node, source)
			if dest == "" {
				issues = append(issues, Issue{
					Line:    lineNum,
					Rule:    "empty-link-dest",
					Message: "Link-Ziel (URL/Pfad) ist leer",
					Level:   "error",
				})
			}

		case *ast.Blockquote:
			lineNum := getLineNum(node, source)
			if child := node.FirstChild(); child != nil {
				var p ast.Node = child
				if _, ok := p.(*ast.Paragraph); ok || isTextBlock(p) {
					txt := strings.TrimSpace(nodeTextContent(node, source))
					if strings.HasPrefix(txt, "[!") {
						idx := strings.Index(txt, "]")
						if idx != -1 {
							marker := strings.ToUpper(txt[2:idx])
							validMarkers := map[string]bool{
								"NOTE": true, "INFO": true, "TIP": true,
								"IMPORTANT": true, "WARNING": true, "CAUTION": true, "ALERT": true,
							}
							if !validMarkers[marker] {
								issues = append(issues, Issue{
									Line:    lineNum,
									Rule:    "unknown-callout-type",
									Message: fmt.Sprintf("Unbekannter Callout-Typ '[!%s]' (Erlaubt: NOTE, TIP, IMPORTANT, WARNING, CAUTION)", marker),
									Level:   "error",
								})
							}
						} else {
							issues = append(issues, Issue{
								Line:    lineNum,
								Rule:    "malformed-callout",
								Message: "Fehlerhafter Callout-Marker (schließendes ']' fehlt)",
								Level:   "error",
							})
						}
					}
				}
			}

		case *east.Table:
			lineNum := getLineNum(node, source)
			if node.FirstChild() == nil {
				issues = append(issues, Issue{
					Line:    lineNum,
					Rule:    "malformed-table",
					Message: "Tabelle ist leer oder fehlerhaft formatiert",
					Level:   "error",
				})
			}
		}

		return ast.WalkContinue, nil
	})

	return issues
}

func getLineNum(node ast.Node, source []byte) int {
	for p := node; p != nil; p = p.Parent() {
		if block, ok := p.(interface{ Lines() *text.Segments }); ok {
			// Goldmark inline types panic on Lines(), check if node type is NodeKindBlock or not inline
			if p.Type() == ast.TypeBlock {
				lines := block.Lines()
				if lines != nil && lines.Len() > 0 {
					offset := lines.At(0).Start
					return strings.Count(string(source[:offset]), "\n") + 1
				}
			}
		}
	}
	return 1
}

func isRemoteURL(dest string) bool {
	return strings.HasPrefix(dest, "http://") || strings.HasPrefix(dest, "https://")
}

func validateMermaid(src string) error {
	trimmed := strings.TrimSpace(src)
	if trimmed == "" {
		return fmt.Errorf("Mermaid-Block ist leer")
	}
	firstLine := strings.Split(trimmed, "\n")[0]
	firstLine = strings.TrimSpace(firstLine)

	validHeader := false
	knownHeaders := []string{"graph", "flowchart", "sequencediagram", "classdiagram", "statediagram", "erdiagram", "gantt", "pie", "gitgraph", "mindmap", "timeline"}
	lowerFirst := strings.ToLower(firstLine)
	for _, kh := range knownHeaders {
		if strings.HasPrefix(lowerFirst, kh) {
			validHeader = true
			break
		}
	}

	if !validHeader {
		return fmt.Errorf("unbekannter oder fehlender Mermaid-Typ in der ersten Zeile '%s'", firstLine)
	}

	return nil
}

func isTextBlock(n ast.Node) bool {
	_, ok := n.(*ast.TextBlock)
	return ok
}

func nodeTextContent(n ast.Node, source []byte) string {
	var b strings.Builder
	_ = ast.Walk(n, func(child ast.Node, entering bool) (ast.WalkStatus, error) {
		if t, ok := child.(*ast.Text); ok && entering {
			b.Write(t.Segment.Value(source))
		}
		if s, ok := child.(*ast.String); ok && entering {
			b.Write(s.Value)
		}
		return ast.WalkContinue, nil
	})
	return b.String()
}
