package confluence

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"
)

var (
	reMacroID       = regexp.MustCompile(`\s+ac:macro-id="[^"]*"`)
	reSchemaVersion = regexp.MustCompile(`\s+ac:schema-version="[^"]*"`)
)

// NormalizeStorageHTML strips auto-generated Confluence attributes and unescapes entities for comparison.
func NormalizeStorageHTML(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = reMacroID.ReplaceAllString(s, "")
	s = reSchemaVersion.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "&quot;", `"`)
	s = strings.ReplaceAll(s, "&#34;", `"`)
	s = strings.ReplaceAll(s, "&#39;", `'`)
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	return strings.TrimSpace(s)
}

// ConflictAction represents the user's choice when remote Confluence content differs.
type ConflictAction int

const (
	ActionOverwrite ConflictAction = iota
	ActionMerge
	ActionAbort
)

// DiffLine represents a single line in unified diff formatting.
type DiffLine struct {
	Type    rune   // ' ', '-', '+'
	Content string
}

// ComputeDiff computes a simple line-by-line diff between remote string and local string.
func ComputeDiff(remoteText, localText string) []DiffLine {
	normRemote := NormalizeStorageHTML(remoteText)
	normLocal := NormalizeStorageHTML(localText)

	remoteLines := strings.Split(normRemote, "\n")
	localLines := strings.Split(normLocal, "\n")

	var diff []DiffLine

	lcs := computeLCS(remoteLines, localLines)

	i, j := 0, 0
	for i < len(remoteLines) || j < len(localLines) {
		if i < len(remoteLines) && j < len(localLines) && remoteLines[i] == localLines[j] {
			diff = append(diff, DiffLine{Type: ' ', Content: remoteLines[i]})
			i++
			j++
		} else if j < len(localLines) && (i >= len(remoteLines) || !contains(lcs, remoteLines[i])) {
			if i < len(remoteLines) && (j >= len(localLines) || !contains(lcs, localLines[j])) {
				diff = append(diff, DiffLine{Type: '-', Content: remoteLines[i]})
				diff = append(diff, DiffLine{Type: '+', Content: localLines[j]})
				i++
				j++
			} else {
				diff = append(diff, DiffLine{Type: '-', Content: remoteLines[i]})
				i++
			}
		} else {
			diff = append(diff, DiffLine{Type: '+', Content: localLines[j]})
			j++
		}
	}

	return diff
}

func computeLCS(a, b []string) []string {
	m, n := len(a), len(b)
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}

	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else if dp[i-1][j] >= dp[i][j-1] {
				dp[i][j] = dp[i-1][j]
			} else {
				dp[i][j] = dp[i][j-1]
			}
		}
	}

	var lcs []string
	i, j := m, n
	for i > 0 && j > 0 {
		if a[i-1] == b[j-1] {
			lcs = append([]string{a[i-1]}, lcs...)
			i--
			j--
		} else if dp[i-1][j] >= dp[i][j-1] {
			i--
		} else {
			j--
		}
	}
	return lcs
}

func contains(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

// PrintDiff outputs a unified diff to w with optional color formatting.
func PrintDiff(w io.Writer, color bool, diff []DiffLine) {
	reset := "\033[0m"
	red := "\033[31m"
	green := "\033[32m"
	dim := "\033[2m"
	bold := "\033[1m"

	paint := func(c bool, code, text string) string {
		if !c {
			return text
		}
		return code + text + reset
	}

	fmt.Fprintln(w, paint(color, bold, "--- Confluence Remote"))
	fmt.Fprintln(w, paint(color, bold, "+++ Lokale Datei"))
	for _, line := range diff {
		switch line.Type {
		case '-':
			fmt.Fprintln(w, paint(color, red, "- "+line.Content))
		case '+':
			fmt.Fprintln(w, paint(color, green, "+ "+line.Content))
		default:
			fmt.Fprintln(w, paint(color, dim, "  "+line.Content))
		}
	}
}

// PromptConflictChoice asks the user in an interactive terminal how to handle remote changes.
func PromptConflictChoice(r io.Reader, w io.Writer) ConflictAction {
	fmt.Fprintln(w)
	fmt.Fprintln(w, "⚠️  Remote-Seite in Confluence unterscheidet sich von der lokalen Datei!")
	fmt.Fprintln(w, "Auswahlmöglichkeiten:")
	fmt.Fprintln(w, "  [o] Overwrite  - Lokale Version in Confluence erzwingen")
	fmt.Fprintln(w, "  [m] Merge      - Confluence-Version lokal zusammenführen")
	fmt.Fprintln(w, "  [a] Abort      - Abbrechen ohne Änderungen")
	fmt.Fprint(w, "Option [o/m/a]: ")

	scanner := bufio.NewScanner(r)
	if scanner.Scan() {
		input := strings.TrimSpace(strings.ToLower(scanner.Text()))
		switch input {
		case "o", "overwrite":
			return ActionOverwrite
		case "m", "merge":
			return ActionMerge
		default:
			return ActionAbort
		}
	}
	return ActionAbort
}
