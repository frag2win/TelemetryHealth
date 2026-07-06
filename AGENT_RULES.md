# AI Agent Guidelines & Documentation Rules

This file defines the conventions for development, commits, and automated documentation updates for the **TelemetryHealth** project. It lives at repo root so AI agents and humans can find it without being told where to look. See `app/DOCS/TelemetryHealth_PRD.md` §14–§15 for the full rationale behind these rules.

## 1. Commit Message Conventions

All commits must follow these prefixes to ensure the automated documentation bot can categorize changes correctly:

- `FEATURE: {description}` - New features or capabilities.
- `BUG: {description}` - Fixes for bugs or build errors.
- `UI: {description}` - Visual changes, layout fixes, or styling updates (dashboard/).
- `PERF: {description}` - Performance optimizations.
- `SEC: {description}` - Security related changes.
- `DOCS: {description}` - Changes to documentation files.
- `REFACTOR: {description}` - Code changes that neither fix a bug nor add a feature.
- `TEST: {description}` - Adding missing tests or correcting existing tests.
- `CHORE: {description}` - Updates to build scripts, dependencies, etc.

Commits are rejected by CI's commit-lint step if they don't match `^(FEATURE|BUG|UI|PERF|SEC|DOCS|REFACTOR|TEST|CHORE):`.

**Blast-radius exception**: any non-`TEST` commit touching `processor/` or `control-plane/internal/remediation/` must reference the relevant PRD risk (e.g. `Risk: §12 row 1`) in the commit body — these are the two packages that can directly affect a customer's live telemetry pipeline.

**Linking to the PRD**: a commit may include a trailer `Closes-PRD-Section: §X.Y` to signal that a functional requirement is now complete. The docs bot uses this to update `Implementation_Status.md` automatically (see §3 below).

## 2. Documentation Updates & AI Agent Behavior

When an AI Agent is working on this project, it must:

1. **Consult Documentation First**: Always read `app/DOCS/TelemetryHealth_PRD.md` for requirements — in particular §8 (functional requirements), §10 (non-functional requirements), and §12 (risks) — and `app/DOCS/Implementation_Status.md` for current progress before starting new features.
2. **Never weaken the fail-open circuit breaker** (`processor/failopen/circuit_breaker.go`) or reduce its test coverage, regardless of the task description — this is a hard constraint from PRD Goal G6.
3. **Never mark a remediation template as safe to auto-apply** without an accompanying shadow-Collector validation test (PRD §8.5). Remediation stays propose-only until a human explicitly changes that policy in `Implementation_Status.md`.
4. **Maintenance**: The AI Agent and GitHub Bot are jointly responsible for maintaining the following files in `app/DOCS/`:
    - **CHANGELOG.md**: Must be updated after every significant change or commit that passes CI.
        - `FEATURE` maps to `### Added`
        - `BUG`, `UI` map to `### Fixed`
        - `REFACTOR`, `PERF`, `SEC`, `DOCS` map to `### Changed`
        - `TEST`, `CHORE` map to `### Internal`
    - **Build_Issue_Report.md**: Specifically for `BUG` commits that relate to compilation, CI config, `Dockerfile`, `go.mod`/`go.sum`, or Helm/Terraform (runtime environment) issues.
    - **Implementation_Status.md**: Updated when a feature mentioned in the PRD is moved to completion — in the same commit that completes it, not a later cleanup pass.
    - **commit-log/YYYY-MM-DD.md**: An append-only, per-day audit log of every commit that passed CI on `main` (SHA, author, category, description, PR number, CI run link). Never edited or rewritten after creation.

## 3. GitHub Automation

A GitHub Action (`.github/workflows/docs-bot.yml`) is configured to:

1. Trigger via `workflow_run` on completion of `ci.yml` against `main`, filtered to `conclusion == 'success'`. **A commit is only logged once CI has actually passed** — a commit that fails CI never gets a changelog, status-tracker, or commit-log entry.
2. Parse the commit message prefix and append an entry to `app/DOCS/CHANGELOG.md` under the mapped heading.
3. Append an entry to `app/DOCS/commit-log/YYYY-MM-DD.md` (created if it doesn't exist for that day).
4. If the commit is prefixed `BUG` and touches CI/build/infra files, also append to `app/DOCS/Build_Issue_Report.md`.
5. If the commit body contains `Closes-PRD-Section: §X.Y`, update the corresponding row in `Implementation_Status.md` and link the commit SHA as evidence.
6. Commit the documentation changes back to the repository as a single bot commit (`DOCS: automated changelog/status update for {sha}`), authored by the `telemetryhealth-docs-bot` machine account — never a personal token — so bot commits are clearly distinguishable from human/agent-authored changes in `git blame` and audit review.
7. On failure to push (e.g. branch protection conflict), retry once with a rebase, then open a `CHORE`-tagged issue rather than silently dropping the update. The bot never force-pushes and never rewrites existing `commit-log/` files.

---
*Note: This file is the primary source of truth for AI agent behavior in this project. Full context and rationale: `app/DOCS/TelemetryHealth_PRD.md` §14–§15.*
