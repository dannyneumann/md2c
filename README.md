# md2c

Publiziert eine Markdown-Datei als Confluence-Seite.

```bash
md2c <datei>
md2c <datei> <space> <pfad>
```

Mit Dateikopf reicht der Dateiname:

```html
<!-- space:DOC,path:Guides,title:Getting started -->
```

```bash
md2c page.md
```

Ohne diese Zeile Space und Pfad angeben:

```bash
md2c page.md DOC "Guides/Getting started"
```

`path` ist die Elternseite (Hierarchie mit `/`), `title` die Seite mit dem Inhalt. Derselbe Pfad aktualisiert die Seite. Der Kommentar wird nicht publiziert.

Eine eigene Zeile `[TOC]` (oder `## [TOC]`) wird zum nativen Confluence-Inhaltsverzeichnis.

## Installation

```bash
make install
```

legt `md2c` nach `~/.local/bin/md2c`. `~/.local/bin` sollte in `PATH` stehen.

## Konfiguration

Zugang **nur** aus einer Conf-Datei. Standard: `~/.config/md2c/md2c.conf`. Anderer Pfad per `-config` / `--config=~/.config/md2c/md2c.conf`. Ohne Datei (oder ohne die Pflichtfelder) bricht `md2c` mit Fehler ab. Keine Shell-Env, kein Hardcode im Binary.

```bash
mkdir -p ~/.config/md2c
cp md2c.conf.example ~/.config/md2c/md2c.conf
```

```bash
MD2C_BASE_URL=https://acme.atlassian.net
MD2C_USER=you@acme.com
MD2C_TOKEN=dein-api-token
```

Für Confluence Cloud: Atlassian-E-Mail plus [API-Token](https://id.atlassian.com/manage-profile/security/api-tokens).

`MD2C_BASE_URL` akzeptiert die Site-URL, die Wiki-URL oder `.../rest/api`.

## Aufruf

```text
md2c [-dry-run] [-version] [-config pfad] <datei> [<space> <pfad>]
```

```bash
md2c -dry-run page.md
```

## Entwicklung

```bash
make test
make build
make install
make hooks
```

`make hooks` installiert einen pre-push-Hook: `git push` läuft nur, wenn `go test -race ./...` grün ist. Confluence wird in den Tests gemockt, es geht kein Netzverkehr raus.

## CI und Releases

Jeder Push und jeder PR auf GitHub führt die Unit-Tests aus. Ist `main` grün, entsteht automatisch:

- ein Semver-Tag `v0.x.y` (eigene Reihe, nicht die alten npm-Tags)
- ein GitHub Release
- Binaries für macOS und Linux (amd64/arm64) plus `SHA256SUMS`

Releases: https://github.com/dannyneumann/md2c/releases

## FAQ

**Nutzt md2c KI?**

Nein. md2c ist ein lokales CLI. Es schickt Markdown nicht an ein Sprachmodell, sondern nur an deine Confluence-Instanz.

**Wohin gehen meine Daten?**

Zugang liegt nur in `~/.config/md2c/md2c.conf` auf deinem Rechner. Publizierter Inhalt geht an Confluence. Es gibt kein Konto bei md2c und keinen Upload zu einem KI-Dienst.

**Wie ist der Code entstanden?**

Die Go-Rewrite wurde mit KI-Assistenten in [Cursor](https://cursor.com) geschrieben und von Danny Neumann geprüft und gepflegt.

**Built with**

- Go CLI, Markdown-Konvertierung mit [goldmark](https://github.com/yuin/goldmark)
- Entwicklung mit KI-Unterstützung in Cursor
- Ausgangspunkt: [md2confluence](https://github.com/jormar/md2confluence) von Jormar Arellano

## Herkunft

Go-Rewrite auf Basis von [md2confluence](https://github.com/jormar/md2confluence) von Jormar Arellano.

## Lizenz

MIT. Copyright 2016 Jormar Arellano; 2026 Danny Neumann.
