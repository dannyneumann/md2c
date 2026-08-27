# md2c

Publiziert eine Markdown-Datei als Confluence-Seite.

```bash
md2c <datei>
md2c <datei> <space> <pfad>
```

Mit Dateikopf reicht der Dateiname:

```html
<!-- space:PSE,path:Decisions+Konzepte,title:Draft: Stagingkonzept-Telematik-Anwendungen -->
```

```bash
md2c konzept.md
```

Ohne diese Zeile Space und Pfad angeben:

```bash
md2c konzept.md PSE "Decisions+Konzepte/Draft: Stagingkonzept-Telematik-Anwendungen"
```

`path` ist die Elternseite (Hierarchie mit `/`), `title` die Seite mit dem Inhalt. Derselbe Pfad aktualisiert die Seite. Der Kommentar wird nicht publiziert.

## Installation

```bash
make install
```

legt `md2c` nach `~/.local/bin/md2c`. `~/.local/bin` sollte in `PATH` stehen.

## Konfiguration

Zugang **nur** aus `~/.config/md2c/md2c.conf`. Ohne diese Datei (oder ohne die Pflichtfelder) bricht `md2c` mit Fehler ab. Keine Shell-Env, kein Hardcode im Binary.

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
md2c [-dry-run] [-version] <datei> [<space> <pfad>]
```

```bash
md2c -dry-run konzept.md
```

## Entwicklung

```bash
make test
make build
make install
```

## Herkunft

Go-Rewrite auf Basis von [md2confluence](https://github.com/jormar/md2confluence) von Jormar Arellano.

## Lizenz

MIT. Copyright 2016 Jormar Arellano; 2026 Danny Neumann.
