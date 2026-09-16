# Codex Usage

These tools exist so Codex can verify its own GitHub delivery work before handing status back to the user.

Do not ask the user to run these commands manually when Codex has access to the repository, shell, and required local tools. Run the relevant checks yourself, then report what passed and what was blocked.

## Default Gate

Use the delivery gate before final handoff:

```powershell
.\scripts\codex-delivery-gate.ps1
```

```sh
./scripts/codex-delivery-gate.sh
```

The gate builds missing binaries and runs:

- `repo-audit`
- `codex-status`

If the gate is partially blocked, keep the PR draft and report the exact reason.

## When To Run Tools

Run `repo-audit` before final handoff for repository, workflow, documentation, or policy changes. It checks required guidance files, workflow directory presence, denylist terms, and foreign reusable workflow references.

Run `codex-status` before final handoff for GitHub delivery tasks. It summarizes branch, working tree, PR, CI checks, risks, and next action.

Run `artifact-verify` when the task creates or modifies local build artifacts. Pass each artifact path, or a directory containing artifacts:

```powershell
.\scripts\codex-delivery-gate.ps1 -ArtifactPath dist
```

```sh
./scripts/codex-delivery-gate.sh --artifact dist
```

Run `release-checker` when the task creates, reviews, or updates a GitHub release:

```powershell
.\scripts\codex-delivery-gate.ps1 -ReleaseRepo shadow-mast/shadowmast-tools -ReleaseTag v0.1.0 -RequireAsset SHA256SUMS.txt
```

```sh
./scripts/codex-delivery-gate.sh --release-repo shadow-mast/shadowmast-tools --release-tag v0.1.0 --require-asset SHA256SUMS.txt
```

## Final Handoff Examples

Use this shape when all relevant checks passed:

```text
Status:
- Phase/task: <task>
- PR: <draft/ready link>
- CI: green
- Local autonomous delivery gate: PASS
- Manual user action required: review/merge decision only
```

Use this shape when the local environment blocks a check:

```text
Status:
- Phase/task: <task>
- PR: <draft/ready link>
- CI: green
- Local autonomous delivery gate: partially blocked: <exact reason>
- Manual user action required: review/merge decision only
```

Never replace a blocked check with a request for the user to run it manually. Report the blocker and continue with checks that are available.
