# shadowmast-tools

Small GitHub-only CLI tools for Codex and repository handoff workflows.

## Tools

- `repo-audit`: checks repository guidance, workflows, denylist terms, and foreign reusable workflow references.
- `release-checker`: checks GitHub release assets through `gh` and can verify local files against `SHA256SUMS.txt`.
- `codex-status`: prints a compact branch, PR, changed files, checks, risks, and next action handoff.
- `artifact-verify`: verifies local build artifacts, size, and SHA256 checksums.

All commands support:

- `--help`
- `--json`
- human-readable output by default
- CI-oriented exit codes: `0` pass, `1` check failure, `2` usage or command error

## For Codex Automation

These tools are built for Codex self-checks inside GitHub delivery work. The user is not expected to run them by hand during normal tasks.

Before a final handoff, Codex should run the local delivery gate itself:

PowerShell:

```powershell
.\scripts\codex-delivery-gate.ps1
```

POSIX shells:

```sh
./scripts/codex-delivery-gate.sh
```

The gate builds missing local tool binaries, runs `repo-audit` and `codex-status`, and prints a markdown handoff. Codex should add optional artifact or release checks only when the task actually produced artifacts or touched a release.

## Build And Test

PowerShell:

```powershell
go test ./...
go vet ./...
go build -o .\bin\repo-audit.exe .\cmd\repo-audit
go build -o .\bin\release-checker.exe .\cmd\release-checker
go build -o .\bin\codex-status.exe .\cmd\codex-status
go build -o .\bin\artifact-verify.exe .\cmd\artifact-verify
```

POSIX shells:

```sh
go test ./...
go vet ./...
mkdir -p bin
go build -o bin/repo-audit ./cmd/repo-audit
go build -o bin/release-checker ./cmd/release-checker
go build -o bin/codex-status ./cmd/codex-status
go build -o bin/artifact-verify ./cmd/artifact-verify
```

## Examples

Audit the current repository:

```sh
repo-audit
repo-audit --json
repo-audit --denylist .shadowmast-denylist
```

Check a release and require desktop artifacts explicitly:

```sh
release-checker --repo shadow-mast/shadowmast-tools --tag v0.1.0 --require-asset setup.exe --require-asset portable.zip --require-asset SHA256SUMS.txt
release-checker --repo shadow-mast/shadowmast-tools --tag v0.1.0 --local-dir dist --json
```

Prepare a Codex handoff:

```sh
codex-status
codex-status --json
```

Verify artifacts:

```sh
artifact-verify dist
artifact-verify dist/setup.exe dist/portable.zip --json
```

## Codex Workflow

Codex should use the installed `github-codex-delivery` skill for branch, PR, CI, release, and status work, and `use-modern-go` for Go code. The repository uses `develop` as the integration branch and feature branches for implementation. Keep changes GitHub-only, small, tested, and free of secrets.

Skill source archives are not vendored here; installed Codex skills are operating instructions, not runtime dependencies.
