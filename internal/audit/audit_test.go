package audit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shadow-mast/shadowmast-tools/internal/cli"
)

func TestRunPassesMinimalRepository(t *testing.T) {
	root := minimalRepo(t)
	writeFile(t, filepath.Join(root, ".shadowmast-denylist"), "legacycrm\n")

	result := Run(root, ".shadowmast-denylist")
	if result.Status != cli.StatusPass {
		t.Fatalf("status = %s, want PASS; findings=%+v", result.Status, result.Findings)
	}
}

func TestRunFailsOnDenylistTerm(t *testing.T) {
	root := minimalRepo(t)
	writeFile(t, filepath.Join(root, ".shadowmast-denylist"), "legacycrm\n")
	writeFile(t, filepath.Join(root, "README.md"), "legacycrm should not appear\n")

	result := Run(root, ".shadowmast-denylist")
	if result.Status != cli.StatusFail {
		t.Fatalf("status = %s, want FAIL", result.Status)
	}
}

func TestRunFailsOnForeignReusableWorkflow(t *testing.T) {
	root := minimalRepo(t)
	writeFile(t, filepath.Join(root, ".shadowmast-denylist"), "legacycrm\n")
	writeFile(t, filepath.Join(root, ".github", "workflows", "ci.yml"), "jobs:\n  call:\n    uses: outside/repo/.github/workflows/ci.yml@main\n")

	result := Run(root, ".shadowmast-denylist")
	if result.Status != cli.StatusFail {
		t.Fatalf("status = %s, want FAIL", result.Status)
	}
}

func TestRunWarnsOnMissingDenylist(t *testing.T) {
	root := minimalRepo(t)

	result := Run(root, ".shadowmast-denylist")
	if result.Status != cli.StatusWarn {
		t.Fatalf("status = %s, want WARN", result.Status)
	}
}

func TestRunSkipsNestedDenylistItself(t *testing.T) {
	root := minimalRepo(t)
	writeFile(t, filepath.Join(root, "config", "denylist.txt"), "legacycrm\n")

	result := Run(root, "config\\denylist.txt")
	if result.Status != cli.StatusPass {
		t.Fatalf("status = %s, want PASS; findings=%+v", result.Status, result.Findings)
	}
}

func minimalRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "README.md"), "ok\n")
	writeFile(t, filepath.Join(root, "AGENTS.md"), "ok\n")
	if err := os.MkdirAll(filepath.Join(root, ".github", "workflows"), 0o700); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, ".github", "workflows", "ci.yml"), "name: CI\n")
	return root
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
