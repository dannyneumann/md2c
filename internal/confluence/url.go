package confluence

import "strings"

// NormalizeAPIBase turns a site, wiki, or REST URL into the /rest/api root.
func NormalizeAPIBase(raw string) string {
	s := strings.TrimRight(strings.TrimSpace(raw), "/")
	if s == "" {
		return ""
	}
	if i := strings.Index(s, "/rest/api"); i >= 0 {
		return s[:i+len("/rest/api")]
	}
	if strings.Contains(s, "atlassian.net") && !strings.Contains(s, "/wiki") {
		return s + "/wiki/rest/api"
	}
	return s + "/rest/api"
}
