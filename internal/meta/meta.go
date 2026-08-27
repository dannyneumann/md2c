package meta

import (
	"regexp"
	"strings"
)

// Meta is the Confluence destination declared in a Markdown file.
type Meta struct {
	Space string
	Path  string
	Title string
}

var header = regexp.MustCompile(`(?s)\A\s*(?:<!--|<--)(.*?)-->[ \t]*\n?`)

// Extract reads a leading HTML metadata comment and returns the remainder.
// The comment is not part of the published body.
func Extract(markdown string) (Meta, string) {
	loc := header.FindStringSubmatchIndex(markdown)
	if loc == nil {
		return Meta{}, markdown
	}
	body := markdown[loc[2]:loc[3]]
	rest := markdown[loc[1]:]
	rest = strings.TrimLeft(rest, "\r\n")
	return parseBody(body), rest
}

// Destination is the full page path (parents + title) inside the space.
func (m Meta) Destination() string {
	path := strings.Trim(strings.TrimSpace(m.Path), "/")
	title := strings.TrimSpace(m.Title)
	switch {
	case title == "":
		return path
	case path == "":
		return title
	default:
		return path + "/" + title
	}
}

func parseBody(body string) Meta {
	var m Meta
	chunk := strings.ReplaceAll(body, "\r\n", "\n")
	chunk = strings.ReplaceAll(chunk, ";", "\n")
	chunk = strings.ReplaceAll(chunk, ",", "\n")
	for _, line := range strings.Split(chunk, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		switch normalizeKey(key) {
		case "space":
			m.Space = strings.TrimSpace(value)
		case "path":
			m.Path = strings.TrimSpace(value)
		case "title":
			m.Title = strings.TrimSpace(value)
		}
	}
	return m
}

func normalizeKey(key string) string {
	key = strings.TrimSpace(strings.ToLower(key))
	key = strings.TrimPrefix(key, "metadata.")
	return key
}
