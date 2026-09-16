package scripts

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPowerShellGateContract(t *testing.T) {
	content := readScript(t, "codex-delivery-gate.ps1")

	requireContains(t, content, "repo-audit")
	requireContains(t, content, "codex-status")
	requireContains(t, content, "artifact-verify")
	requireContains(t, content, "release-checker")
	requireContains(t, content, "go build")
	requireContains(t, content, "Local autonomous delivery gate")
	requireContains(t, content, "partially blocked")
	requireContains(t, content, "exit 2")
	requireContains(t, content, "-ArtifactPath")
	requireContains(t, content, "-ReleaseRepo")
	requireContains(t, content, "-ReleaseTag")
	requireContains(t, content, "CODEX_GATE_REPO_AUDIT")
	requireContains(t, content, "CODEX_GATE_CODEX_STATUS")
	if strings.Contains(content, "CODEX_GATE_BIN_DIR") {
		t.Fatal("gate must not allow overriding binary output directory")
	}
}

func TestShellGateContract(t *testing.T) {
	content := readScript(t, "codex-delivery-gate.sh")

	requireContains(t, content, "#!/usr/bin/env bash")
	requireContains(t, content, "repo-audit")
	requireContains(t, content, "codex-status")
	requireContains(t, content, "artifact-verify")
	requireContains(t, content, "release-checker")
	requireContains(t, content, "go build")
	requireContains(t, content, "Local autonomous delivery gate")
	requireContains(t, content, "partially blocked")
	requireContains(t, content, "exit 2")
	requireContains(t, content, "--artifact")
	requireContains(t, content, "--release-repo")
	requireContains(t, content, "--release-tag")
	requireContains(t, content, "requires a value")
	requireContains(t, content, "CODEX_GATE_REPO_AUDIT")
	requireContains(t, content, "CODEX_GATE_CODEX_STATUS")
	if strings.Contains(content, "CODEX_GATE_BIN_DIR") {
		t.Fatal("gate must not allow overriding binary output directory")
	}
}

func TestPowerShellGateReportsBlockedCodexStatusWithStubs(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("PowerShell behavior test uses Windows cmd stubs")
	}

	dir := t.TempDir()
	repoAudit := writeStub(t, filepath.Join(dir, "repo-audit.cmd"), "@echo off\r\necho repo-audit: PASS\r\nexit /b 0\r\n", 0o700)
	codexStatus := writeStub(t, filepath.Join(dir, "codex-status.cmd"), "@echo off\r\necho Status\r\necho Risks\r\necho - PR not found or gh unavailable\r\nexit /b 0\r\n", 0o700)

	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", "codex-delivery-gate.ps1", "-SkipBuild")
	cmd.Env = append(os.Environ(),
		"CODEX_GATE_REPO_AUDIT="+repoAudit,
		"CODEX_GATE_CODEX_STATUS="+codexStatus,
	)
	output, err := cmd.CombinedOutput()
	if code := commandExitCode(t, err); code != 2 {
		t.Fatalf("exit code = %d, want 2\n%s", code, output)
	}
	requireContains(t, string(output), "Local autonomous delivery gate: partially blocked")
	requireContains(t, string(output), "codex-status could not fully verify PR/CI")
}

func TestShellGateReportsBlockedCodexStatusWithStubs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell behavior test uses POSIX stubs")
	}

	dir := t.TempDir()
	repoAudit := writeStub(t, filepath.Join(dir, "repo-audit"), "#!/usr/bin/env sh\necho 'repo-audit: PASS'\nexit 0\n", 0o700)
	codexStatus := writeStub(t, filepath.Join(dir, "codex-status"), "#!/usr/bin/env sh\necho 'Status'\necho 'Risks'\necho '- PR not found or gh unavailable'\nexit 0\n", 0o700)

	cmd := exec.Command("bash", "codex-delivery-gate.sh", "--skip-build")
	cmd.Env = append(os.Environ(),
		"CODEX_GATE_REPO_AUDIT="+repoAudit,
		"CODEX_GATE_CODEX_STATUS="+codexStatus,
	)
	output, err := cmd.CombinedOutput()
	if code := commandExitCode(t, err); code != 2 {
		t.Fatalf("exit code = %d, want 2\n%s", code, output)
	}
	requireContains(t, string(output), "Local autonomous delivery gate: partially blocked")
	requireContains(t, string(output), "codex-status could not fully verify PR/CI")
}

func TestGateDocsAvoidManualUserHandoff(t *testing.T) {
	for _, path := range []string{"../AGENTS.md", "../docs/CODEX_USAGE.md", "../README.md"} {
		content := readScript(t, path)
		requireContains(t, content, "Codex")
		requireContains(t, content, "self")
		if strings.Contains(strings.ToLower(content), "ask the user to run") {
			requireContains(t, content, "Do not ask the user to run")
		}
	}
}

func readScript(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func requireContains(t *testing.T, content string, want string) {
	t.Helper()
	if !strings.Contains(content, want) {
		t.Fatalf("expected content to contain %q", want)
	}
}

func writeStub(t *testing.T, path string, content string, perm os.FileMode) string {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), perm); err != nil {
		t.Fatal(err)
	}
	return path
}

func commandExitCode(t *testing.T, err error) int {
	t.Helper()
	if err == nil {
		return 0
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		return exitError.ExitCode()
	}
	t.Fatal(err)
	return -1
}
