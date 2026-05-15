# AGENTS

## Role

Act as a pragmatic engineering team for this repository. Keep changes small, deterministic, and GitHub-only.

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
- Prefer squash merge after review and green CI.

## Safety

- Use only GitHub, `git`, `gh`, and local repository tooling by default.
- Do not add assumptions about non-GitHub trackers, private hosts, office chat, or deployment systems.
- Never commit secrets, tokens, private keys, credentials, copied chat logs, or large generated binaries.
- Do not delete, move, overwrite, or destructively rename user files without explicit approval.
- For file operations, prefer read-only checks, dry-run behavior, manifests, logs, and undo strategy.

## Go Rules

- Target the Go version declared in `go.mod`.
- Prefer the standard library.
- Keep command packages thin and put reusable behavior under `internal/`.
- Preserve machine-readable JSON output and stable exit codes for CI.
