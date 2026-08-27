package report

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

const (
	reset = "\033[0m"
	bold  = "\033[1m"
	dim   = "\033[2m"
	red   = "\033[31m"
	green = "\033[32m"
	cyan  = "\033[36m"
	blue  = "\033[34m"
)

// Result holds the data for a publish summary.
type Result struct {
	Created bool
	Title   string
	Version int
	URL     string
	ID      string
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
	fmt.Fprintln(w, paint(color, bold, "Ziel"))
	fmt.Fprintln(w)
	kv(w, color, "Datei", file)
	kv(w, color, "Space", space)
	kv(w, color, "Pfad", pagePath)
	fmt.Fprintln(w)
}

// Success writes a multi-line create or update summary.
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
	if r.URL != "" {
		kv(w, color, "URL", paint(color, blue, r.URL))
	} else if r.ID != "" {
		kv(w, color, "ID", r.ID)
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
