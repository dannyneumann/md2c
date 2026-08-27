package convert

import (
	"fmt"
	"strings"
	"unicode"
)

func mermaidToPlantUML(src string) (string, bool) {
	src = strings.ReplaceAll(src, "\r\n", "\n")
	src = strings.TrimSpace(src)
	if src == "" {
		return "", false
	}

	d := &flowDiagram{root: &flowGroup{id: "_root"}}
	cur := d.root
	stack := []*flowGroup{d.root}

	for _, raw := range strings.Split(src, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "%%") || strings.HasPrefix(line, "click ") {
			continue
		}
		low := strings.ToLower(line)
		if strings.HasPrefix(low, "flowchart ") || strings.HasPrefix(low, "graph ") {
			d.lr = strings.Contains(low, " lr") || strings.HasSuffix(low, " lr") || strings.Contains(low, " rl")
			continue
		}
		if strings.HasPrefix(low, "subgraph ") {
			id, label := parseSubgraph(line)
			if id == "" {
				return "", false
			}
			g := &flowGroup{id: id, label: label}
			cur.children = append(cur.children, g)
			stack = append(stack, g)
			cur = g
			continue
		}
		if low == "end" {
			if len(stack) < 2 {
				return "", false
			}
			stack = stack[:len(stack)-1]
			cur = stack[len(stack)-1]
			continue
		}
		if strings.HasPrefix(low, "click ") {
			continue
		}
		line, inline := extractInlineNodes(line)
		if len(inline) > 0 {
			cur.nodes = append(cur.nodes, inline...)
		}
		if isEdgeLine(line) {
			edges, ok := parseEdges(line)
			if !ok {
				return "", false
			}
			d.edges = append(d.edges, edges...)
			continue
		}
		if ident := strings.TrimSpace(line); ident == "" || isIdent(ident) {
			if ident != "" && len(inline) == 0 {
				cur.nodes = append(cur.nodes, flowNode{id: ident, label: ident})
			}
			continue
		}
		return "", false
	}

	if len(stack) != 1 {
		return "", false
	}
	if len(d.root.nodes) == 0 && len(d.root.children) == 0 && len(d.edges) == 0 {
		return "", false
	}

	var b strings.Builder
	b.WriteString("@startuml\n")
	b.WriteString("!pragma layout smetana\n")
	b.WriteString("skinparam shadowing false\n")
	if d.lr {
		b.WriteString("left to right direction\n")
	} else {
		b.WriteString("top to bottom direction\n")
	}
	writeGroup(&b, d.root)
	for _, e := range d.edges {
		arrow := "-->"
		if e.dotted {
			arrow = "..>"
		}
		if e.label != "" {
			fmt.Fprintf(&b, "%s %s %s : %s\n", e.from, arrow, e.to, plantUMLText(e.label))
		} else {
			fmt.Fprintf(&b, "%s %s %s\n", e.from, arrow, e.to)
		}
	}
	b.WriteString("@enduml\n")
	return b.String(), true
}

type flowDiagram struct {
	lr    bool
	root  *flowGroup
	edges []flowEdge
}

type flowGroup struct {
	id, label string
	nodes     []flowNode
	children  []*flowGroup
}

type flowNode struct {
	id, label string
}

type flowEdge struct {
	from, to, label string
	dotted          bool
}

func writeGroup(b *strings.Builder, g *flowGroup) {
	nested := g.id != "_root"
	if nested {
		fmt.Fprintf(b, "rectangle \"%s\" {\n", plantUMLText(g.label))
	}
	for _, n := range g.nodes {
		label := n.label
		if label == "" {
			label = n.id
		}
		fmt.Fprintf(b, "rectangle \"%s\" as %s\n", plantUMLText(label), n.id)
	}
	for _, child := range g.children {
		writeGroup(b, child)
	}
	if nested {
		b.WriteString("}\n")
	}
}

func extractInlineNodes(line string) (string, []flowNode) {
	var nodes []flowNode
	var b strings.Builder
	i := 0
	for i < len(line) {
		if id, n := readIdent(line[i:]); n > 0 {
			ws := 0
			for i+n+ws < len(line) && (line[i+n+ws] == ' ' || line[i+n+ws] == '\t') {
				ws++
			}
			rest := line[i+n+ws:]
			if strings.HasPrefix(rest, "[") {
				end := strings.IndexByte(rest, ']')
				if end > 0 {
					label := strings.TrimSpace(rest[1:end])
					label = strings.Trim(label, `"`)
					nodes = append(nodes, flowNode{id: id, label: label})
					b.WriteString(id)
					i = i + n + ws + end + 1
					continue
				}
			}
		}
		b.WriteByte(line[i])
		i++
	}
	return b.String(), nodes
}

func parseSubgraph(line string) (id, label string) {
	rest := strings.TrimSpace(line[len("subgraph"):])
	id, label, ok := parseNodeDef(rest)
	if ok {
		if label == "" {
			label = id
		}
		return id, label
	}
	if strings.HasPrefix(rest, `"`) {
		label = strings.Trim(rest, `"`)
		return sanitizeID(label), label
	}
	if fields := strings.Fields(rest); len(fields) == 1 {
		return fields[0], fields[0]
	}
	return "", ""
}

func parseNodeDef(s string) (id, label string, ok bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", "", false
	}
	lb := strings.IndexByte(s, '[')
	if lb < 0 {
		if isIdent(s) {
			return s, s, true
		}
		return "", "", false
	}
	id = strings.TrimSpace(s[:lb])
	if !isIdent(id) {
		return "", "", false
	}
	rest := strings.TrimSpace(s[lb+1:])
	rest = strings.TrimSuffix(rest, "]")
	rest = strings.TrimSpace(rest)
	label = strings.Trim(rest, `"`)
	return id, label, true
}

func isEdgeLine(line string) bool {
	return strings.Contains(line, "-->") || strings.Contains(line, "-.->") || strings.Contains(line, "---")
}

func parseEdges(line string) ([]flowEdge, bool) {
	rest := strings.TrimSpace(line)
	from, n := readIdent(rest)
	if from == "" {
		return nil, false
	}
	rest = strings.TrimSpace(rest[n:])

	var out []flowEdge
	for rest != "" {
		dotted := false
		switch {
		case strings.HasPrefix(rest, "-.->"):
			dotted = true
			rest = strings.TrimSpace(rest[4:])
		case strings.HasPrefix(rest, "-->"):
			rest = strings.TrimSpace(rest[3:])
		case strings.HasPrefix(rest, "---"):
			rest = strings.TrimSpace(rest[3:])
		default:
			return nil, false
		}
		label := ""
		if strings.HasPrefix(rest, "|") {
			rest = rest[1:]
			end := strings.IndexByte(rest, '|')
			if end < 0 {
				return nil, false
			}
			label = strings.TrimSpace(rest[:end])
			rest = strings.TrimSpace(rest[end+1:])
		}
		to, n := readIdent(rest)
		if to == "" {
			return nil, false
		}
		out = append(out, flowEdge{from: from, to: to, label: label, dotted: dotted})
		rest = strings.TrimSpace(rest[n:])
		from = to
	}
	return out, len(out) > 0
}

func readIdent(s string) (string, int) {
	i := 0
	for i < len(s) {
		r := rune(s[i])
		if i == 0 && !isIdentStart(r) {
			return "", 0
		}
		if !isIdentPart(r) {
			break
		}
		i++
	}
	if i == 0 {
		return "", 0
	}
	return s[:i], i
}

func isIdent(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 && !isIdentStart(r) {
			return false
		}
		if !isIdentPart(r) {
			return false
		}
	}
	return true
}

func isIdentStart(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}

func isIdentPart(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

func sanitizeID(s string) string {
	var b strings.Builder
	b.WriteByte('_')
	for _, r := range s {
		if isIdentPart(r) {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	return b.String()
}

func plantUMLText(s string) string {
	s = strings.ReplaceAll(s, `"`, "'")
	s = strings.ReplaceAll(s, "<", "‹")
	s = strings.ReplaceAll(s, ">", "›")
	s = strings.ReplaceAll(s, "\n", `\n`)
	return s
}
