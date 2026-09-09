package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"md2confluence/internal/config"
	"md2confluence/internal/confluence"
	"md2confluence/internal/convert"
	"md2confluence/internal/lint"
	"md2confluence/internal/meta"
	"md2confluence/internal/report"
	"md2confluence/internal/skill"
)

var (
	version = "dev"
)

func versionBanner() string {
	return fmt.Sprintf("md2c %s\n", version)
}

const usageText = `md2c — Markdown nach Confluence publizieren & herunterladen

Aufruf:
  md2c [flags] <datei>
  md2c [flags] <datei> <space> <pfad>

  Normalfall: Ziel steht im Dateikopf (wird nicht publiziert):
    <!-- space:DOC,path:Guides,title:Getting started -->
    md2c page.md

  Fehlt space/path/title in der Datei, auf der Kommandozeile mitgeben:
    md2c page.md DOC Guides/Getting started

Aufruf (Linting / Syntax-Prüfung):
  md2c lint <datei>

Aufruf (Download / Pull aus Confluence):
  md2c pull <space> <pfad>
  md2c pull <confluence-url>

Aufruf (Agent Skill installieren):
  md2c install-skill [target]
  Ziele: agy, gemini, codex, cursor, agents, all (Standard: all)

  Beispiele:
    md2c lint page.md
    md2c install-skill codex
    md2c pull PSE "Leitplanken/Nutzung-Kalender"
    md2c pull https://confluence.example.com/spaces/PSE/pages/123/Nutzung-Kalender

Argumente:
  datei   Markdown-Datei
  space   Space-Key (sonst space: im Dateikopf)
  pfad    Elternseiten/Seitentitel mit / (sonst path: und title: im Dateikopf)

Dateikopf (erste Zeile, wird nicht publiziert):
    <!-- space:DOC,path:Elternseite,title:Seitentitel -->

Unterstützte Formatierungen:
  - [TOC] oder ## [TOC] -> Natives Confluence-Inhaltsverzeichnis
  - GitHub Callouts (> [!NOTE], > [!TIP], > [!IMPORTANT], > [!WARNING], > [!CAUTION]) -> Confluence Info/Tip/Note/Warning-Makros
  - GFM-Tabellen:
      | Header 1 | Header 2 |
      | -------- | -------- |
      | Wert 1   | Wert 2   |
  - Mermaid-Flowcharts -> Confluence PlantUML-Makro
  - Lokale Bilder (![alt](./bild.png)) -> Automatischer Attachment-Upload & Download

Flags:
  -dry-run       Nur konvertieren, nicht publizieren (braucht keine Config)
  -no-lint       Automatischen Markdown-Linter vor dem Publizieren überspringen
  -reason        Grund/Kommentar für die Versionshistorie in Confluence angeben
  -message       Alias für -reason
  -install-skill Agent Skill installieren (Ziele: agy, gemini, codex, cursor, all)
  -version       Version, Quelle und Autor ausgeben
  -config        Conf-Datei (Standard: ~/.config/md2c/md2c.conf)
                 z. B. --config=~/.config/md2c/md2c.conf

Confluence-Zugang nur aus der Conf-Datei (MD2C_BASE_URL, MD2C_USER, MD2C_TOKEN).
Output im Terminal: farbig (angelegt = grün, aktualisiert = cyan, Fehler = rot).
NO_COLOR=1 schaltet die Farben ab.
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
	noLint := fs.Bool("no-lint", false, "Skip automatic markdown linting")
	showVersion := fs.Bool("version", false, "Print version and exit")
	configPath := fs.String("config", "", "Path to md2c.conf")
	reason := fs.String("reason", "", "Version comment/reason in Confluence history")
	message := fs.String("message", "", "Alias for -reason")
	installSkillFlag := fs.String("install-skill", "", "Install agent skill (agy, gemini, codex, cursor, all)")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	changeReason := *reason
	if changeReason == "" {
		changeReason = *message
	}

	colorOut := report.Enabled(rt.Stdout, rt.Getenv)
	colorErr := report.Enabled(rt.Stderr, rt.Getenv)
	if *showVersion {
		fmt.Fprint(rt.Stdout, versionBanner())
		return 0
	}

	if *installSkillFlag != "" {
		return handleInstallSkill([]string{*installSkillFlag}, colorOut, colorErr, rt)
	}

	rest := fs.Args()
	if len(rest) >= 1 && (rest[0] == "install-skill" || rest[0] == "install-skills") {
		return handleInstallSkill(rest[1:], colorOut, colorErr, rt)
	}
	if len(rest) >= 1 && rest[0] == "lint" {
		return handleLint(rest[1:], colorOut, colorErr, rt)
	}
	if len(rest) >= 1 && (rest[0] == "pull" || rest[0] == "download") {
		return handlePull(rest[1:], *configPath, colorOut, colorErr, rt)
	}

	if len(rest) < 1 || len(rest) > 3 {
		fmt.Fprint(rt.Stderr, usageText)
		fmt.Fprintln(rt.Stderr)
		report.Failure(rt.Stderr, colorErr, "Aufruf ungültig",
			fmt.Sprintf("erwartet <datei>, <datei> <space> <pfad>, lint <datei> oder pull <space> <pfad>, bekommen %d Argument(e)", len(rest)))
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
		report.Failure(rt.Stderr, colorErr, "Datei konnte nicht gelesen werden", fmt.Sprintf("%s: %v", filePath, err))
		return 1
	}

	if !*noLint {
		issues := lint.Lint(raw, lint.LintOptions{BaseDir: filepath.Dir(filePath)})
		hasError := false
		var issueStrs []string
		for _, iss := range issues {
			issueStrs = append(issueStrs, iss.String())
			if iss.Level == "error" {
				hasError = true
			}
		}
		if hasError {
			report.LintResult(rt.Stderr, colorErr, filePath, issueStrs, true)
			report.Failure(rt.Stderr, colorErr, "Markdown-Linting fehlgeschlagen", "Bitte Syntaxfehler vor dem Publizieren beheben (oder mit --no-lint überspringen)")
			return 1
		}
	}

	fileMeta, markdown := meta.Extract(string(raw))
	space, pagePath, err := resolveTarget(cliSpace, cliPagePath, fileMeta)
	if err != nil {
		report.Failure(rt.Stderr, colorErr, "Ziel unvollständig", err.Error())
		return 2
	}

	body, attachments, err := convert.Convert(markdown)
	if err != nil {
		report.Failure(rt.Stderr, colorErr, "Markdown konnte nicht konvertiert werden", err.Error())
		return 1
	}

	report.Target(rt.Stderr, colorErr, filepath.Base(filePath), space, pagePath)

	if *dryRun {
		fmt.Fprintln(rt.Stdout, body)
		return 0
	}

	cfg, err := config.Load(config.Sources{
		Getenv: rt.Getenv,
		Read:   rt.ReadFile,
		Home:   rt.Home,
		Path:   *configPath,
	})
	if err != nil {
		report.Failure(rt.Stderr, colorErr, "Konfiguration fehlt oder ist ungültig", err.Error())
		return 2
	}
	if cfg.Prefix != "" {
		body = convert.InfoMacro(cfg.Prefix) + body
	}

	client := confluence.New(cfg.BaseURL, cfg.User, cfg.Token)
	client.Auth = cfg.Auth
	if rt.HTTPClient != nil {
		client.HTTPClient = rt.HTTPClient
	}
	client.UserAgent = "md2c/" + version

	ctx, cancel := context.WithTimeout(context.Background(), rt.Timeout)
	defer cancel()

	totalSteps := 1 + len(attachments)
	currentStep := 1
	report.Progress(rt.Stderr, colorErr, currentStep, totalSteps, fmt.Sprintf("Publiziere Seite nach Confluence (%s / %s)...", space, pagePath))

	page, created, err := client.Publish(ctx, space, pagePath, body, changeReason)
	if err != nil {
		report.Failure(rt.Stderr, colorErr, "Publizieren fehlgeschlagen", err.Error())
		return 1
	}

	var uploadedAtts []string
	mdDir := filepath.Dir(filePath)
	for _, att := range attachments {
		currentStep++
		attBase := filepath.Base(att)
		report.Progress(rt.Stderr, colorErr, currentStep, totalSteps, fmt.Sprintf("Lade Anhang hoch (%s)...", attBase))
		attPath := filepath.Join(mdDir, att)
		if err := client.UploadAttachment(ctx, page.ID, attPath); err != nil {
			report.Failure(rt.Stderr, colorErr, fmt.Sprintf("Attachment-Upload fehlgeschlagen (%s)", att), err.Error())
		} else {
			uploadedAtts = append(uploadedAtts, attBase)
		}
	}
	if totalSteps > 0 {
		fmt.Fprintln(rt.Stderr)
	}

	report.Success(rt.Stdout, colorOut, report.Result{
		Created:     created,
		Title:       page.Title,
		Version:     page.Version.Number,
		URL:         page.WebURL(),
		ID:          page.ID,
		Attachments: uploadedAtts,
	})
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

func handlePull(args []string, configPath string, colorOut, colorErr bool, rt runtime) int {
	if len(args) == 0 {
		report.Failure(rt.Stderr, colorErr, "Pull-Aufruf unvollständig", "bitte Space und Pfad angeben: md2c pull <space> <pfad>")
		return 2
	}

	space, pagePath, rawURL := "", "", ""
	if len(args) == 1 {
		// URL or space/path
		rawURL = args[0]
		if strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://") {
			// Parse space and title or page ID from URL e.g. .../spaces/PSE/pages/572179008/DRAFT+-+Nutzung+Confluence-Kalender
			parts := strings.Split(rawURL, "/")
			for i, p := range parts {
				if strings.EqualFold(p, "spaces") && i+1 < len(parts) {
					space = parts[i+1]
				}
				if strings.EqualFold(p, "pages") && i+1 < len(parts) {
					pagePath = parts[i+1]
					if i+2 < len(parts) && parts[i+2] != "" {
						tPart := strings.ReplaceAll(parts[i+2], "+", " ")
						tPart, _ = url.QueryUnescape(tPart)
						if tPart != "" {
							pagePath = tPart
						}
					}
				}
			}
		} else {
			report.Failure(rt.Stderr, colorErr, "Pull-Aufruf ungültig", "bitte Space und Pfad angeben: md2c pull <space> <pfad>")
			return 2
		}
	} else {
		space = args[0]
		pagePath = args[1]
	}

	if space == "" || pagePath == "" {
		report.Failure(rt.Stderr, colorErr, "Pull-Ziel unvollständig", "Space oder Pfad konnte nicht ermittelt werden")
		return 2
	}

	cfg, err := config.Load(config.Sources{
		Getenv: rt.Getenv,
		Read:   rt.ReadFile,
		Home:   rt.Home,
		Path:   configPath,
	})
	if err != nil {
		report.Failure(rt.Stderr, colorErr, "Konfiguration fehlt oder ist ungültig", err.Error())
		return 2
	}

	client := confluence.New(cfg.BaseURL, cfg.User, cfg.Token)
	client.Auth = cfg.Auth
	if rt.HTTPClient != nil {
		client.HTTPClient = rt.HTTPClient
	}
	client.UserAgent = "md2c/" + version

	ctx, cancel := context.WithTimeout(context.Background(), rt.Timeout)
	defer cancel()

	page, storageHTML, parentPath, err := client.FetchPage(ctx, space, pagePath)
	if err != nil {
		report.Failure(rt.Stderr, colorErr, "Herunterladen fehlgeschlagen", err.Error())
		return 1
	}

	markdown, attachments, err := convert.ToMarkdown(storageHTML)
	if err != nil {
		report.Failure(rt.Stderr, colorErr, "Konvertierung fehlgeschlagen", err.Error())
		return 1
	}

	header := fmt.Sprintf("<!-- space:%s,path:%s,title:%s -->\n\n", page.Space.Key, parentPath, page.Title)
	fullContent := header + markdown

	outFile := page.Title + ".md"
	_, statErr := os.Stat(outFile)
	fileExisted := statErr == nil

	if err := os.WriteFile(outFile, []byte(fullContent), 0644); err != nil {
		report.Failure(rt.Stderr, colorErr, "Speichern fehlgeschlagen", err.Error())
		return 1
	}

	var downloadedAtts []string
	for _, att := range attachments {
		if err := client.DownloadAttachmentFile(ctx, page.ID, att, att); err != nil {
			report.Failure(rt.Stderr, colorErr, fmt.Sprintf("Attachment-Download fehlgeschlagen (%s)", att), err.Error())
		} else {
			downloadedAtts = append(downloadedAtts, att)
		}
	}

	pullURL := page.WebURL()
	if rawURL != "" {
		pullURL = rawURL
	}

	report.PullResult(rt.Stdout, colorOut, report.PullData{
		URL:         pullURL,
		Space:       page.Space.Key,
		Title:       page.Title,
		Path:        parentPath,
		ID:          page.ID,
		Version:     page.Version.Number,
		LocalFile:   outFile,
		FileExisted: fileExisted,
		Attachments: downloadedAtts,
	})
	return 0
}

func handleLint(args []string, colorOut, colorErr bool, rt runtime) int {
	if len(args) == 0 {
		report.Failure(rt.Stderr, colorErr, "Lint-Aufruf unvollständig", "bitte Datei angeben: md2c lint <datei>")
		return 2
	}
	filePath := args[0]
	raw, err := rt.ReadFile(filePath)
	if err != nil {
		report.Failure(rt.Stderr, colorErr, "Datei konnte nicht gelesen werden", fmt.Sprintf("%s: %v", filePath, err))
		return 1
	}

	issues := lint.Lint(raw, lint.LintOptions{BaseDir: filepath.Dir(filePath)})
	hasError := false
	var issueStrs []string
	for _, iss := range issues {
		issueStrs = append(issueStrs, iss.String())
		if iss.Level == "error" {
			hasError = true
		}
	}

	report.LintResult(rt.Stdout, colorOut, filePath, issueStrs, hasError)
	if hasError {
		return 1
	}
	return 0
}

func handleInstallSkill(args []string, colorOut, colorErr bool, rt runtime) int {
	targetName := "all"
	if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
		targetName = args[0]
	}

	targets, err := skill.GetTargets(rt.Home, rt.Cwd, targetName)
	if err != nil {
		report.Failure(rt.Stderr, colorErr, "Ungültiges Skill-Ziel", err.Error())
		return 2
	}

	installed, err := skill.Install(targets)
	if err != nil {
		report.Failure(rt.Stderr, colorErr, "Installation fehlgeschlagen", err.Error())
		return 1
	}

	fmt.Fprintln(rt.Stdout, "✓ md2c Agent Skill erfolgreich installiert:")
	fmt.Fprintln(rt.Stdout)
	for _, p := range installed {
		fmt.Fprintf(rt.Stdout, "  ➜ %s\n", p)
	}
	fmt.Fprintln(rt.Stdout)
	return 0
}
