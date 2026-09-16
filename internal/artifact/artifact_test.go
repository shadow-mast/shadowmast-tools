package artifact

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shadow-mast/shadowmast-tools/internal/cli"
)

func TestVerifyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "artifact.bin")
	if err := os.WriteFile(path, []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}

	result := Verify([]string{path})
	if result.Status != cli.StatusPass {
		t.Fatalf("status = %s, want PASS", result.Status)
	}
	if len(result.Files) != 1 || result.Files[0].SHA256 == "" {
		t.Fatalf("expected one file with checksum: %+v", result.Files)
	}
}

func TestVerifyRejectsEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.bin")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	result := Verify([]string{path})
	if result.Status != cli.StatusFail {
		t.Fatalf("status = %s, want FAIL", result.Status)
	}
}

func TestVerifyDirectoryUsesRegularFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "b.bin"), []byte("b"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.bin"), []byte("a"), 0o600); err != nil {
		t.Fatal(err)
	}

	result := Verify([]string{dir})
	if result.Status != cli.StatusPass {
		t.Fatalf("status = %s, want PASS", result.Status)
	}
	if len(result.Files) != 2 {
		t.Fatalf("len(files) = %d, want 2", len(result.Files))
	}
	if result.Files[0].Path > result.Files[1].Path {
		t.Fatalf("files are not sorted: %+v", result.Files)
	}
}
