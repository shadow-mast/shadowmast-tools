package checksum

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileSHA256(t *testing.T) {
	path := filepath.Join(t.TempDir(), "artifact.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := FileSHA256(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if got != want {
		t.Fatalf("sha256 = %q, want %q", got, want)
	}
}

func TestParseSHA256Sums(t *testing.T) {
	input := strings.NewReader(strings.Join([]string{
		strings.Repeat("a", 64) + "  setup.exe",
		strings.Repeat("b", 64) + " *portable.zip",
		"",
		"# comment",
	}, "\n"))

	entries, err := ParseSHA256Sums(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("len(entries) = %d, want 2", len(entries))
	}
	if entries[0].Filename != "setup.exe" {
		t.Fatalf("first filename = %q", entries[0].Filename)
	}
	if entries[1].Filename != "portable.zip" {
		t.Fatalf("second filename = %q", entries[1].Filename)
	}
}

func TestParseSHA256SumsRejectsMalformedLine(t *testing.T) {
	_, err := ParseSHA256Sums(strings.NewReader("abc artifact.zip\n"))
	if err == nil {
		t.Fatal("expected error")
	}
}
