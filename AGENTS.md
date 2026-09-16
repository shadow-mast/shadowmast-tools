# AGENTS

## Role

Act as a pragmatic engineering team for this repository. Keep changes small, deterministic, and GitHub-only.
The CLI tools in this repository are primarily for Codex self-check automation, not for shifting manual work to the user.

## Language

- Speak to the user in Russian.
- Code, paths, identifiers, branches, commits, and PR text may stay in English.
- Keep final status reports short.

## Required Codex Skills

- Use installed skill `github-codex-delivery` for GitHub, PR, branch, CI, release, and status work.
- Use installed skill `use-modern-go` for all Go implementation, review, and modernization work.
- Do not vendor skill packages or skill zip files into this repository.

## Workflow

- Use `develop` as the integration branch.
- Never work directly on `develop` or `main` for implementation work.
- Create one focused feature branch per task.
- Push the branch and open a draft PR for non-trivial changes.
- Put technical details and validation commands in the PR body.
- Run tests before requesting review.
- Before a final handoff, Codex must run the relevant local checks itself when the environment allows it.
- Do not ask the user to run `repo-audit`, `codex-status`, `release-checker`, `artifact-verify`, or delivery gate wrappers manually when Codex can run them.
- If a check cannot run because a local tool, auth, network access, or build prerequisite is missing, report that exact blocker instead of delegating the command to the user.
- Prefer squash merge after review and green CI.

## Safety

- Use only GitHub, `git`, `gh`, and local repository tooling by default.
- Do not add assumptions about non-GitHub trackers, private hosts, office chat, or deployment systems.
- Never commit secrets, tokens, private keys, credentials, copied chat logs, or large generated binaries.
- Preserve unrelated user files and changes. Project edits within the task use
  standing authorization; destructive changes to user data need a specific decision.
- For file operations, prefer read-only checks, dry-run behavior, manifests, logs, and undo strategy.

## Go Rules

- Target the Go version declared in `go.mod`.
- Prefer the standard library.
- Keep command packages thin and put reusable behavior under `internal/`.
- Preserve machine-readable JSON output and stable exit codes for CI.

## Codex Self-Check Gate

- Use `scripts/codex-delivery-gate.ps1` on Windows and `scripts/codex-delivery-gate.sh` on Linux/macOS for implementation/release handoffs. Documentation-only edits need diff/link checks; do not rebuild tools solely for Markdown.
- The gate always runs `repo-audit` and `codex-status`.
- Add `artifact-verify` to the gate only when the task produced local build artifacts.
- Add `release-checker` to the gate only when the task touches a GitHub release or release artifacts.
- Final reports must say what passed, what was partially blocked, and what was not applicable.
