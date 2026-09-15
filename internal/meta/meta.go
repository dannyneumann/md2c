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

	// Split by newline first
	lines := strings.Split(chunk, "\n")
	var items []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		// If line contains multiple key:value pairs separated by comma (e.g. "space:DOC,path:Guides,title:Getting started")
		// split only on commas that precede a known key name.
		parts := splitCommaKeys(l)
		items = append(items, parts...)
	}

	for _, item := range items {
		key, value, ok := strings.Cut(item, ":")
		if !ok {
			continue
		}
		val := strings.TrimSpace(value)
		val = strings.TrimRight(val, ",;\r\n\t ")
		switch normalizeKey(key) {
		case "space":
			m.Space = val
		case "path":
			m.Path = val
		case "title":
			m.Title = val
		}
	}
	return m
}

var keyPrefixRegex = regexp.MustCompile(`,\s*(?i)(space|path|title|metadata\.[a-z]+)\s*:`)

func splitCommaKeys(s string) []string {
	locs := keyPrefixRegex.FindAllStringIndex(s, -1)
	if len(locs) == 0 {
		return []string{s}
	}
	var res []string
	start := 0
	for _, loc := range locs {
		res = append(res, strings.TrimSpace(s[start:loc[0]]))
		start = loc[0] + 1 // skip the comma
	}
	res = append(res, strings.TrimSpace(s[start:]))
	return res
}

func normalizeKey(key string) string {
	key = strings.TrimSpace(strings.ToLower(key))
	key = strings.TrimPrefix(key, "metadata.")
	return key
}
