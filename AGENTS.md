<!-- template-version: dev-2026-08-25 -->
# 🤖 AGENTS.md — Application Development & Engineering Guide

**Template** for application/library repos. Stored in the dotfiles repo for
distribution — **not** the working policy for the dotfiles tree itself (that is
the root `AGENTS.md`).

This document defines mandatory operational guidelines for automated AI agents (**Codex**, **Claude Code**, **Gemini / Agy**, and **Cursor**) working on software development, applications, and library code in a target repository.

---

## 📚 1. Documentation Hierarchy & Tool Integration

| File / Source | Target Agent | Scope & Content |
| :--- | :--- | :--- |
| **`README.md`** | *All Agents* | Setup, architectural decisions, local development steps, and usage examples. |
| **`AGENTS.md`** | **Codex** (canonical in target repos) | Operational guidelines, IDD task policy, Git workflows, and engineering quality gates. |
| **`CLAUDE.md`** | **Claude Code** | Symlink to `AGENTS.md` (`ln -sfn AGENTS.md CLAUDE.md`). |
| **`.cursorrules`** / **`.cursor/rules/agents.mdc`** | **Cursor (CLI & IDE)** | Symlinks to `AGENTS.md`. |
| **`GEMINI.md`** | **Gemini / Agy** | Symlink to `AGENTS.md` (`ln -sfn AGENTS.md GEMINI.md`). |

---

## 🎯 2. Issue-Driven Development (IDD) Policy

> **Core Law:** *No Ticket, No Work.* Never edit files, refactor, or write code without an associated task reference (`<REF>`).

### 2.1 Task Tracking & Dynamic Project Scope
* **Aven Primacy:** Aven is the **single source of truth** for all tasks (work, follow-ups, bugs, debt). Do not create or mirror tasks in Gitea/GitHub/GitLab issues. Command details live in the Aven skill (`aven skill` / `aven skill install`), not in this file.
* **Dynamic Scope:** Prefer the mapped Aven project for the current directory. Derive from folder name (`$(basename "$PWD")`) when that matches an existing project. Ops-related checkouts (including services under the homelab tree and the personal `dotfiles` repo) use Aven workspace **`ops`**.
* **Git Forge:** PRs and code review stay on the connected remote (Gitea/`tea`, GitHub/`gh`, GitLab, …). The forge is not the task tracker.
* **Workspace Initialization:**
  * Verify project scope before starting work:
    ```bash
    aven list --open
    aven list --project "$(basename "$PWD")" --json
    ```
  * If uninitialized, set up the project non-interactively (and map the path):
    ```bash
    aven project create "$(basename "$PWD")" --path "$PWD"
    aven add "Initial workspace setup" --project "$(basename "$PWD")"
    ```

### 2.2 Task Lifecycle & Documentation
1. **Before Starting Work:**
   * Identify task reference (`<REF>`, e.g., `APP-101`) or create a new task:
     ```bash
     aven add "<TITLE>" --project "$(basename "$PWD")" --description "<BODY>"
     ```
   * Set task status to active:
     ```bash
     aven edit <REF> --status active
     ```
2. **Decision & Architecture Notes:**
   * Document architectural choices, framework evaluations, and trade-offs directly in task notes:
     ```bash
     aven note <REF> "Decision: <RATIONALE>"
     ```
3. **Sub-Tasks & Bug Spinoffs:**
   * Do not expand the scope of the current task. Create dedicated subtasks:
     ```bash
     aven add "Fix <ISSUE_DESCRIPTION>" --project "$(basename "$PWD")" --description "Discovered while working on <REF>"
     ```
4. **Completion:**
   * Run full test suites & linters, open the MR/PR, and log progress in notes.
     **Do not** set the Aven task to `done` yet.
   * Close the Aven task **only after the MR/PR is merged** (not after a local
     commit, not after pushing the feature branch, not when the PR is opened):
     ```bash
     aven note <REF> "Completed: <SUMMARY_OF_CHANGES> (merged PR #<n>)"
     aven edit <REF> --status done
     ```
   * Direct pushes to `main`/`master` are not the default. If the user
     explicitly asked for that, mark `done` only once `origin/main` has the
     commit.

---

## 🌿 3. Git & Branching Hygiene

1. **Branch-per-Task Workflow:**
   * For every non-trivial task or subtask, switch to a dedicated branch before editing code:
     ```bash
     git checkout -b task/<REF>-<short-description>
     ```
   * Never commit directly to `main` or `master`.
2. **Commit Hygiene:**
   * Prefix all commit messages with the task reference:
     ```bash
     git commit -m "[<REF>] <Imperative changes of summary>"
     ```
   * **Atomic Commits:** Implement model/schema -> Tests pass -> Commit -> Implement handler -> Tests pass -> Commit.
3. **Sub-Agent Isolation:**
   * When running parallel sessions, ensure each agent operates strictly within its own branch and file boundaries to prevent merge collisions.

---

## 🧪 4. Testing, Quality & Engineering Mandate

> **Core Law:** *Unverified code is broken code.* Everything developed must be covered by automated tests and validated by static analysis.

### 4.1 Automated Test Suites
* **Go Codebases:**
  * Every new package, service, or handler requires unit/integration tests (`*_test.go`).
  * Run tests with race detection and coverage:
    ```bash
    go test -v -race -cover ./...
    ```
* **Python Codebases:**
  * Run test suites with `pytest` (e.g., `pytest -v --cov`).
* **TypeScript / JavaScript:**
  * Run unit tests and type verification:
    ```bash
    npm run test && npx tsc --noEmit
    ```
* **Mocking & External I/O:**
  * Unit tests must never depend on live network connections or remote services. Use interface mocking.

### 4.2 Linting & Static Analysis
* **Go:** `golangci-lint run` (or `make lint`). Code must be free of warnings.
* **Python:** `ruff check .` and `mypy .` for static typing.
* **Frontend:** `npm run lint` / `eslint` and `prettier --check .`.
* **Database Schemas:** Schema changes MUST include reversible migration scripts. Never alter raw schemas manually.

### 4.3 Bug Fixing Standard
1. Write a failing test reproducing the reported bug.
2. Inspect logs and find the root cause (no quick-fix hacks).
3. Fix the implementation until the test passes.
4. Verify the entire regression suite remains green.

---

## 🏗 5. Runtime & Execution Standards

* **Container-First Execution:** Do not install runtime tools directly on the host OS. All builds, tests, and linters run inside Docker containers (e.g., via `docker compose exec` or `make`).
* **Dependency Hygiene:** Adding third-party packages requires explicit justification in task notes. Clean up dependency files immediately (`go mod tidy`, lockfile sync).
* **Makefile Authority:** Targets in `Makefile` (e.g., `make test`, `make lint`, `make build`) are authoritative.
* **Pre-Approved Tools:** `grep -n`, `ls`, `cat`, `find`, `make <target>`, `aven <cmd>`, `tea <cmd>`.
