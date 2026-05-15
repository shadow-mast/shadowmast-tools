package release

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shadow-mast/shadowmast-tools/internal/checksum"
	"github.com/shadow-mast/shadowmast-tools/internal/cli"
)

func TestParseReleaseAssets(t *testing.T) {
	assets, err := ParseReleaseAssets([]byte(`{"assets":[{"name":"portable.zip","size":12},{"name":"setup.exe","size":34}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(assets) != 2 {
		t.Fatalf("len(assets) = %d, want 2", len(assets))
	}
	if assets[0].Name != "portable.zip" {
		t.Fatalf("first asset = %q", assets[0].Name)
	}
}

func TestEvaluateAssetsFailsMissingRequiredAsset(t *testing.T) {
	result := EvaluateAssets("owner/repo", "v1", []Asset{{Name: "portable.zip"}}, []string{"setup.exe"})
	if result.Status != cli.StatusFail {
		t.Fatalf("status = %s, want FAIL", result.Status)
	}
}

func TestVerifyLocalDirMatchesChecksumFile(t *testing.T) {
	dir := t.TempDir()
	artifactPath := filepath.Join(dir, "portable.zip")
	if err := os.WriteFile(artifactPath, []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	sum, err := checksum.FileSHA256(artifactPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SHA256SUMS.txt"), []byte(sum+"  portable.zip\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	checks, findings, err := verifyLocalDir(context.Background(), Config{LocalDir: dir}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Fatalf("findings = %+v, want none", findings)
	}
	if len(checks) != 2 {
		t.Fatalf("len(checks) = %d, want 2", len(checks))
	}
}

func TestVerifyLocalDirReportsMalformedChecksumAsFailure(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "SHA256SUMS.txt"), []byte("not-a-valid-line\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	checks, findings, err := verifyLocalDir(context.Background(), Config{LocalDir: dir}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(checks) != 1 || checks[0].Status != cli.StatusFail {
		t.Fatalf("checks = %+v, want checksum failure", checks)
	}
	if len(findings) != 1 || findings[0].Severity != cli.StatusFail {
		t.Fatalf("findings = %+v, want failure finding", findings)
	}
}

func TestSafeLocalNameRejectsParentTraversal(t *testing.T) {
	if _, ok := safeLocalName("../artifact.zip"); ok {
		t.Fatal("expected parent traversal to be rejected")
	}
	if _, ok := safeLocalName(strings.Repeat("a", 3)); !ok {
		t.Fatal("expected plain filename to be accepted")
	}
}
