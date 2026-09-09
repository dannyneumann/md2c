package report

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

const (
	reset   = "\033[0m"
	bold    = "\033[1m"
	dim     = "\033[2m"
	red     = "\033[31m"
	green   = "\033[32m"
	yellow  = "\033[33m"
	blue    = "\033[34m"
	magenta = "\033[35m"
	cyan    = "\033[36m"
)

// Result holds the data for a publish summary.
type Result struct {
	Created     bool
	Title       string
	Version     int
	Reason      string
	URL         string
	ID          string
	Attachments []string
}

// PullData holds details for a pull summary.
type PullData struct {
	URL          string
	Space        string
	Title        string
	Path         string
	ID           string
	Version      int
	LocalFile    string
	FileExisted  bool
	Attachments  []string
}

// Enabled reports whether ANSI colors should be used for w.
// Colors are on by default for terminals. NO_COLOR disables them.
func Enabled(w io.Writer, getenv func(string) string) bool {
	if getenv != nil && strings.TrimSpace(getenv("NO_COLOR")) != "" {
		return false
	}
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// Target writes the destination before the Confluence call.
func Target(w io.Writer, color bool, file, space, pagePath string) {
	fmt.Fprintln(w, paint(color, bold+magenta, "ZIEL"))
	fmt.Fprintln(w)
	kv(w, color, "Datei", file)
	kv(w, color, "Space", space)
	kv(w, color, "Pfad", pagePath)
	fmt.Fprintln(w)
}

// Progress writes an active step or progressbar line for uploads/publishing.
func Progress(w io.Writer, color bool, current, total int, label string) {
	if total <= 0 {
		fmt.Fprintf(w, "  %s %s...\n", paint(color, bold+cyan, "➜"), label)
		return
	}
	width := 20
	percent := float64(current) / float64(total)
	filled := int(percent * float64(width))
	if filled > width {
		filled = width
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	pctText := fmt.Sprintf("%3.0f%%", percent*100)
	stepText := fmt.Sprintf("[%d/%d]", current, total)
	fmt.Fprintf(w, "  %s %s [%s] %s %s\n",
		paint(color, bold+cyan, "➜"),
		paint(color, bold+cyan, bar),
		paint(color, yellow, pctText),
		paint(color, dim, stepText),
		label,
	)
}

// Success writes a multi-line create or update summary for publish.
func Success(w io.Writer, color bool, r Result) {
	headline := "Seite aktualisiert"
	style := bold + cyan
	if r.Created {
		headline = "Seite angelegt"
		style = bold + green
	}
	fmt.Fprintln(w, paint(color, style, headline))
	fmt.Fprintln(w)
	kv(w, color, "Titel", r.Title)
	if r.Version > 0 {
		kv(w, color, "Version", strconv.Itoa(r.Version))
	}
	if r.Reason != "" {
		kv(w, color, "Grund", r.Reason)
	}
	if r.URL != "" {
		kv(w, color, "URL", paint(color, blue, r.URL))
	} else if r.ID != "" {
		kv(w, color, "ID", r.ID)
	}
	if len(r.Attachments) > 0 {
		kv(w, color, "Anhänge", strings.Join(r.Attachments, ", "))
	}
	fmt.Fprintln(w)
}

// PullResult writes a clean summary after pulling a Confluence page.
func PullResult(w io.Writer, color bool, p PullData) {
	headline := "PULL CONFLUENCE SEITE"
	if p.URL != "" {
		headline = fmt.Sprintf("PULL %s", p.URL)
	}
	fmt.Fprintln(w, paint(color, bold+cyan, headline))
	fmt.Fprintln(w)

	fmt.Fprintln(w, paint(color, bold+yellow, "REMOTE"))
	kv(w, color, "Space", p.Space)
	kv(w, color, "Titel", p.Title)
	if p.Path != "" {
		kv(w, color, "Pfad", p.Path)
	}
	if p.ID != "" {
		kv(w, color, "ID", p.ID)
	}
	if p.Version > 0 {
		kv(w, color, "Version", strconv.Itoa(p.Version))
	}
	if len(p.Attachments) > 0 {
		kv(w, color, "Anhänge", strings.Join(p.Attachments, ", "))
	}
	fmt.Fprintln(w)

	fmt.Fprintln(w, paint(color, bold+green, "LOKAL"))
	kv(w, color, "Datei", p.LocalFile)
	status := "Neu erstellt"
	if p.FileExisted {
		status = "Aktualisiert"
	}
	kv(w, color, "Status", status)
	fmt.Fprintln(w)
}

// LintResult writes a clean summary of lint issues found.
func LintResult(w io.Writer, color bool, file string, issues []string, hasError bool) {
	if len(issues) == 0 {
		fmt.Fprintln(w, paint(color, bold+green, fmt.Sprintf("✓ Linting erfolgreich: %s ist valide", file)))
		fmt.Fprintln(w)
		return
	}

	headline := fmt.Sprintf("LINT SYNTAX / FEHLER (%s)", file)
	style := bold + yellow
	if hasError {
		style = bold + red
	}
	fmt.Fprintln(w, paint(color, style, headline))
	fmt.Fprintln(w)
	for _, issue := range issues {
		fmt.Fprintf(w, "  %s\n", issue)
	}
	fmt.Fprintln(w)
}

// Failure writes a multi-line error report.
func Failure(w io.Writer, color bool, summary string, details ...string) {
	fmt.Fprintln(w, paint(color, bold+red, "Fehler: "+summary))
	fmt.Fprintln(w)
	for _, d := range details {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		fmt.Fprintf(w, "  %s\n", d)
	}
	fmt.Fprintln(w)
}

func kv(w io.Writer, color bool, key, value string) {
	label := paint(color, dim, fmt.Sprintf("%-8s", key+":"))
	fmt.Fprintf(w, "  %s %s\n", label, value)
}

func paint(color bool, code, s string) string {
	if !color || s == "" {
		return s
	}
	return code + s + reset
}
