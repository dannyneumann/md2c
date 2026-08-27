package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"md2confluence/internal/config"
	"md2confluence/internal/confluence"
	"md2confluence/internal/convert"
	"md2confluence/internal/meta"
)

var (
	version = "dev"
	source  = "https://github.com/dannyneumann/md2c.git"
	author  = "Danny Neumann"
)

func versionBanner() string {
	return fmt.Sprintf("md2c %s - source %s\noptimized by %s\n", version, source, author)
}

const usageText = `md2c — Markdown nach Confluence publizieren

Aufruf:
  md2c [flags] <datei>
  md2c [flags] <datei> <space> <pfad>

Normalfall: Ziel steht in der Datei, dann nur den Dateinamen angeben.

    <!-- space:PSE,path:Decisions+Konzepte,title:Draft: Stagingkonzept -->
    md2c konzept.md

Fehlt space/path/title in der Datei, auf der Kommandozeile mitgeben:

    md2c konzept.md PSE Decisions+Konzepte/Draft: Stagingkonzept

Argumente:
  datei   Markdown-Datei
  space   Space-Key (sonst space: im Dateikopf)
  pfad    Elternseiten/Seitentitel mit / (sonst path: und title: im Dateikopf)

Dateikopf (erste Zeile, wird nicht publiziert):
    <!-- space:PSE,path:Elternseite,title:Seitentitel -->
  path = Elternseite oder Hierarchie (a/b). title = Seite mit dem Inhalt.

Flags:
  -dry-run    Nur konvertieren, nicht publizieren (braucht keine Config)
  -version    Version, Quelle und Autor ausgeben

Confluence-Zugang nur aus ~/.config/md2c/md2c.conf
  (MD2C_BASE_URL, MD2C_USER, MD2C_TOKEN). Fehlt die Datei, bricht md2c ab.
`

type runtime struct {
	Getenv     func(string) string
	ReadFile   func(string) ([]byte, error)
	Stdout     io.Writer
	Stderr     io.Writer
	HTTPClient *http.Client
	Cwd        string
	Home       string
	Timeout    time.Duration
}

func main() {
	cwd, _ := os.Getwd()
	home, _ := os.UserHomeDir()
	os.Exit(run(os.Args[1:], runtime{
		Getenv:   os.Getenv,
		ReadFile: os.ReadFile,
		Stdout:   os.Stdout,
		Stderr:   os.Stderr,
		Cwd:      cwd,
		Home:     home,
		Timeout:  60 * time.Second,
	}))
}

func run(args []string, rt runtime) int {
	if rt.Getenv == nil {
		rt.Getenv = os.Getenv
	}
	if rt.ReadFile == nil {
		rt.ReadFile = os.ReadFile
	}
	if rt.Stdout == nil {
		rt.Stdout = os.Stdout
	}
	if rt.Stderr == nil {
		rt.Stderr = os.Stderr
	}
	if rt.Timeout == 0 {
		rt.Timeout = 60 * time.Second
	}

	fs := flag.NewFlagSet("md2c", flag.ContinueOnError)
	fs.SetOutput(rt.Stderr)
	fs.Usage = func() {
		fmt.Fprint(rt.Stderr, usageText)
	}

	dryRun := fs.Bool("dry-run", false, "Convert only; do not publish")
	showVersion := fs.Bool("version", false, "Print version and exit")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *showVersion {
		fmt.Fprint(rt.Stdout, versionBanner())
		return 0
	}

	rest := fs.Args()
	if len(rest) < 1 || len(rest) > 3 {
		fmt.Fprint(rt.Stderr, usageText)
		fmt.Fprintf(rt.Stderr, "\nerror: erwartet <datei> oder <datei> <space> <pfad>, bekommen %d Argument(e)\n", len(rest))
		return 2
	}

	filePath := rest[0]
	cliSpace, cliPagePath := "", ""
	if len(rest) >= 2 {
		cliSpace = rest[1]
	}
	if len(rest) == 3 {
		cliPagePath = rest[2]
	}

	raw, err := rt.ReadFile(filePath)
	if err != nil {
		fmt.Fprintf(rt.Stderr, "error: read %s: %v\n", filePath, err)
		return 1
	}

	fileMeta, markdown := meta.Extract(string(raw))
	space, pagePath, err := resolveTarget(cliSpace, cliPagePath, fileMeta)
	if err != nil {
		fmt.Fprintf(rt.Stderr, "error: %v\n", err)
		return 2
	}

	body, err := convert.Convert(markdown)
	if err != nil {
		fmt.Fprintf(rt.Stderr, "error: convert markdown: %v\n", err)
		return 1
	}

	if *dryRun {
		fmt.Fprintln(rt.Stdout, body)
		return 0
	}

	cfg, err := config.Load(config.Sources{
		Getenv: rt.Getenv,
		Read:   rt.ReadFile,
		Home:   rt.Home,
	})
	if err != nil {
		fmt.Fprintf(rt.Stderr, "error: %v\n", err)
		return 2
	}
	if cfg.Prefix != "" {
		body = convert.InfoMacro(cfg.Prefix) + body
	}

	client := confluence.New(cfg.BaseURL, cfg.User, cfg.Token)
	if rt.HTTPClient != nil {
		client.HTTPClient = rt.HTTPClient
	}
	client.UserAgent = "md2c/" + version

	ctx, cancel := context.WithTimeout(context.Background(), rt.Timeout)
	defer cancel()

	page, err := client.Publish(ctx, space, pagePath, body)
	if err != nil {
		fmt.Fprintf(rt.Stderr, "error: publish: %v\n", err)
		return 1
	}

	loc := page.WebURL()
	if loc == "" {
		loc = page.ID
	}
	fmt.Fprintf(rt.Stdout, "published %q to %s (%s)\n", page.Title, space, loc)
	return 0
}

func resolveTarget(cliSpace, cliPath string, m meta.Meta) (space, pagePath string, err error) {
	space = cliSpace
	if space == "" {
		space = m.Space
	}
	pagePath = cliPath
	if pagePath == "" {
		pagePath = m.Destination()
	}

	var missing []string
	if space == "" {
		missing = append(missing, "space")
	}
	if len(confluence.SplitPath(pagePath)) == 0 {
		missing = append(missing, "path/title")
	}
	if len(missing) == 0 {
		return space, pagePath, nil
	}
	return "", "", fmt.Errorf("%s fehlt in der Datei — bitte angeben: md2c <datei> <space> <pfad>", strings.Join(missing, " und "))
}
