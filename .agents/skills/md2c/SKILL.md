---
name: md2c
description: >-
  Guide and operational procedures for publishing, linting, downloading, and converting
  Markdown documentation to/from Confluence using the md2c CLI tool. Use whenever a user
  asks to deploy, convert, lint, or pull Confluence pages.
---

# md2c — Markdown to Confluence Integration Skill

`md2c` is a high-performance Go CLI utility for publishing Markdown documents to Atlassian Confluence, downloading existing Confluence pages to local Markdown, and validating Markdown syntax & custom macro compatibility.

---

## 🛠 Command Syntax & Usage Quick Reference

### 1. Publishing to Confluence (Push)

When the Markdown file contains metadata in the header comment:
```html
<!-- space:DOC,path:Guides,title:Getting started -->
```
You can publish directly using:
```bash
md2c page.md
```

If space/path/title are missing from the header comment, pass them via CLI:
```bash
md2c page.md <SPACE> <PATH>
# Example: md2c page.md DOC "Guides/Getting started"
```

#### Change Reason / Version Comment
When updating existing Confluence pages, provide a reason/comment for the version history using `--reason` or `--message`:
```bash
md2c page.md --reason "Fixed broken links and updated architecture diagram"
```

#### Skipping Pre-Publish Linting
By default, `md2c` automatically lints the Markdown before uploading. To bypass linting:
```bash
md2c --no-lint page.md
```

---

### 2. Linting & Syntax Validation

To manually lint Markdown files for syntax errors, missing local image attachments, and unsupported macro syntax without publishing:
```bash
md2c lint page.md
```

#### What the Linter Validates:
- **Heading Format**: Checks for missing space after `#` (outside code blocks).
- **GitHub Callouts**: Validates `> [!NOTE]`, `> [!TIP]`, `> [!IMPORTANT]`, `> [!WARNING]`, `> [!CAUTION]`. Flagged errors for unknown types like `> [!NOT1E]`.
- **Mermaid Diagrams**: Validates fenced ```mermaid blocks and initial diagram declarations (`graph`, `sequenceDiagram`, etc.).
- **Local Attachments**: Warns if referenced local images (`![alt](./path/image.png)`) do not exist on disk.
- **Header Metadata**: Optional check for missing `<!-- space:...,path:...,title:... -->` comments.

---

### 3. Dry Run (Preview Output)

To inspect the generated Confluence Storage Format HTML without contacting Confluence:
```bash
md2c -dry-run page.md
```

---

### 4. Pulling / Downloading from Confluence

Download a page and convert it back to Markdown (with header comments and attachments):
```bash
md2c pull <SPACE> <PATH>
# Example: md2c pull PSE "Leitplanken/Nutzung-Kalender"
```
Or pull directly using the Confluence browser URL:
```bash
md2c pull https://confluence.example.com/spaces/PSE/pages/12345/Nutzung-Kalender
```

---

## 🎨 Supported Features & Macro Conversions

| Markdown Feature | Output in Confluence |
| :--- | :--- |
| `[TOC]` or `## [TOC]` | Native Confluence Table of Contents Macro (`<ac:structured-macro ac:name="toc">`) |
| `> [!NOTE]` / `> [!INFO]` | Native Info Macro (`info`) |
| `> [!TIP]` | Native Tip Macro (`tip`) |
| `> [!IMPORTANT]` | Native Note Macro (`note`) |
| `> [!WARNING]` / `> [!CAUTION]` | Native Warning Macro (`warning`) |
| ` ```mermaid ` | Automatically converted to Native PlantUML Macro (`plantuml`) |
| `![alt](./image.png)` | Local image uploaded as Page Attachment & embedded |
| GFM Tables | Rendered as Confluence Storage Format XHTML tables |

---

## ⚙️ Installation & Management

Install or upgrade `md2c` via Homebrew:
```bash
make brew   # or make install
```
Or directly via Homebrew CLI:
```bash
brew trust --formula dannyneumann/md2c/md2c
brew tap dannyneumann/md2c https://github.com/dannyneumann/md2c.git
brew install dannyneumann/md2c/md2c
```
