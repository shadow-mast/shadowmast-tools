#!/usr/bin/env bash
set -u

ARTIFACTS=()
REQUIRE_ASSETS=()
RELEASE_REPO=""
RELEASE_TAG=""
RELEASE_LOCAL_DIR=""
SKIP_BUILD=0

require_arg() {
    local option="$1"
    local value_count="$2"
    if [[ "$value_count" -lt 2 ]]; then
        echo "$option requires a value" >&2
        exit 2
    fi
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --artifact)
            require_arg "$1" "$#"
            ARTIFACTS+=("$2")
            shift 2
            ;;
        --release-repo)
            require_arg "$1" "$#"
            RELEASE_REPO="$2"
            shift 2
            ;;
        --release-tag)
            require_arg "$1" "$#"
            RELEASE_TAG="$2"
            shift 2
            ;;
        --require-asset)
            require_arg "$1" "$#"
            REQUIRE_ASSETS+=("$2")
            shift 2
            ;;
        --release-local-dir)
            require_arg "$1" "$#"
            RELEASE_LOCAL_DIR="$2"
            shift 2
            ;;
        --skip-build)
            SKIP_BUILD=1
            shift
            ;;
        --help|-h)
            cat <<'USAGE'
Usage: scripts/codex-delivery-gate.sh [options]

Options:
  --artifact PATH          Run artifact-verify for PATH. Repeat for multiple paths.
  --release-repo OWNER/REPO
  --release-tag TAG
  --require-asset NAME    Required release asset. Repeat for multiple assets.
  --release-local-dir DIR Verify release SHA256SUMS.txt against local files.
  --skip-build            Do not build missing binaries.

Exit codes:
  0 pass
  1 check failure
  2 blocked gate/tool error
USAGE
            exit 0
            ;;
        *)
            echo "unknown option: $1" >&2
            exit 2
            ;;
    esac
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BIN_DIR="$REPO_ROOT/.codex-tools/bin"
FINAL_EXIT=0
LAST_TOOL_OUTPUT=""

set_gate_exit_code() {
    local code="$1"
    if [[ "$code" -eq 0 ]]; then
        return
    fi
    if [[ "$code" -eq 1 && "$FINAL_EXIT" -eq 0 ]]; then
        FINAL_EXIT=1
        return
    fi
    if [[ "$code" -ge 2 ]]; then
        FINAL_EXIT=2
    fi
}

tool_path() {
    case "$1" in
        repo-audit)
            printf '%s' "${CODEX_GATE_REPO_AUDIT:-$BIN_DIR/repo-audit}"
            ;;
        codex-status)
            printf '%s' "${CODEX_GATE_CODEX_STATUS:-$BIN_DIR/codex-status}"
            ;;
        artifact-verify)
            printf '%s' "${CODEX_GATE_ARTIFACT_VERIFY:-$BIN_DIR/artifact-verify}"
            ;;
        release-checker)
            printf '%s' "${CODEX_GATE_RELEASE_CHECKER:-$BIN_DIR/release-checker}"
            ;;
    esac
}

tool_package() {
    printf './cmd/%s' "$1"
}

ensure_tool() {
    local name="$1"
    local package="$2"
    local path
    path="$(tool_path "$name")"
    local override=0
    case "$name" in
        repo-audit)
            [[ -n "${CODEX_GATE_REPO_AUDIT:-}" ]] && override=1
            ;;
        codex-status)
            [[ -n "${CODEX_GATE_CODEX_STATUS:-}" ]] && override=1
            ;;
        artifact-verify)
            [[ -n "${CODEX_GATE_ARTIFACT_VERIFY:-}" ]] && override=1
            ;;
        release-checker)
            [[ -n "${CODEX_GATE_RELEASE_CHECKER:-}" ]] && override=1
            ;;
    esac

    if [[ -f "$path" ]]; then
        return 0
    fi
    if [[ "$override" -eq 1 ]]; then
        echo "$name override path does not exist: $path"
        return 2
    fi
    if [[ "$SKIP_BUILD" -eq 1 ]]; then
        echo "$name binary is missing and --skip-build was set"
        return 2
    fi
    if ! command -v go >/dev/null 2>&1; then
        echo "$name binary is missing and Go is not available in PATH"
        return 2
    fi

    mkdir -p "$BIN_DIR"
    (cd "$REPO_ROOT" && go build -o "$path" "$package")
}

run_gate_tool() {
    local name="$1"
    local path="$2"
    shift 2

    echo
    echo "## $name"
    echo
    echo '```text'
    "$path" "$@" 2>&1
    local code=$?
    echo '```'
    echo
    echo "- exit code: $code"
    set_gate_exit_code "$code"
}

run_gate_tool_capture() {
    local name="$1"
    local path="$2"
    shift 2

    echo
    echo "## $name"
    echo
    echo '```text'
    LAST_TOOL_OUTPUT="$("$path" "$@" 2>&1)"
    local code=$?
    printf '%s\n' "$LAST_TOOL_OUTPUT"
    echo '```'
    echo
    echo "- exit code: $code"
    set_gate_exit_code "$code"
}

codex_status_blocked() {
    [[ "$LAST_TOOL_OUTPUT" == *"PR not found or gh unavailable"* ]] ||
        [[ "$LAST_TOOL_OUTPUT" == *"CI checks unavailable through gh"* ]] ||
        [[ "$LAST_TOOL_OUTPUT" == *"CI checks JSON could not be parsed"* ]]
}

echo "# Codex Delivery Gate"
echo
echo "- repo: $REPO_ROOT"
echo "- platform: POSIX shell"
echo "- exit codes: 0 pass, 1 check failure, 2 blocked gate/tool error"

required_tools=(repo-audit codex-status)
if [[ "${#ARTIFACTS[@]}" -gt 0 ]]; then
    required_tools+=(artifact-verify)
fi
if [[ -n "$RELEASE_REPO" || -n "$RELEASE_TAG" ]]; then
    required_tools+=(release-checker)
fi

for tool in "${required_tools[@]}"; do
    if ! output="$(ensure_tool "$tool" "$(tool_package "$tool")" 2>&1)"; then
        echo
        echo "## Build"
        echo
        echo '```text'
        echo "$output"
        echo '```'
        echo
        echo "## Summary"
        echo
        echo "- Local autonomous delivery gate: partially blocked"
        echo "- Reason: $output"
        exit 2
    fi
done

cd "$REPO_ROOT" || exit 2
run_gate_tool "repo-audit" "$(tool_path repo-audit)" --root "$REPO_ROOT"
run_gate_tool_capture "codex-status" "$(tool_path codex-status)"
if codex_status_blocked; then
    echo "- gate note: codex-status could not fully verify PR/CI"
    set_gate_exit_code 2
fi

if [[ "${#ARTIFACTS[@]}" -gt 0 ]]; then
    run_gate_tool "artifact-verify" "$(tool_path artifact-verify)" "${ARTIFACTS[@]}"
fi

if [[ -n "$RELEASE_REPO" || -n "$RELEASE_TAG" ]]; then
    if [[ -z "$RELEASE_REPO" || -z "$RELEASE_TAG" ]]; then
        echo
        echo "## release-checker"
        echo
        echo '```text'
        echo "Release checks require both --release-repo and --release-tag."
        echo '```'
        echo
        echo "- exit code: 2"
        set_gate_exit_code 2
    else
        release_args=(--repo "$RELEASE_REPO" --tag "$RELEASE_TAG")
        for asset in "${REQUIRE_ASSETS[@]}"; do
            release_args+=(--require-asset "$asset")
        done
        if [[ -n "$RELEASE_LOCAL_DIR" ]]; then
            release_args+=(--local-dir "$RELEASE_LOCAL_DIR")
        fi
        run_gate_tool "release-checker" "$(tool_path release-checker)" "${release_args[@]}"
    fi
fi

echo
echo "## Summary"
echo
if [[ "$FINAL_EXIT" -eq 0 ]]; then
    echo "- Local autonomous delivery gate: PASS"
elif [[ "$FINAL_EXIT" -eq 1 ]]; then
    echo "- Local autonomous delivery gate: FAIL"
else
    echo "- Local autonomous delivery gate: partially blocked"
fi
echo "- Final exit code: $FINAL_EXIT"

exit "$FINAL_EXIT"
