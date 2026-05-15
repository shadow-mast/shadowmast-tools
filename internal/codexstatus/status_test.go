package codexstatus

import (
	"strings"
	"testing"

	"github.com/shadow-mast/shadowmast-tools/internal/cli"
)

func TestParseChangedFiles(t *testing.T) {
	files := parseChangedFiles(" M README.md\nA  cmd/tool/main.go\n?? new.txt\n")
	if len(files) != 3 {
		t.Fatalf("len(files) = %d, want 3", len(files))
	}
	if files[0].Status != "M" || files[0].Path != "README.md" {
		t.Fatalf("unexpected first file: %+v", files[0])
	}
	if files[2].Status != "??" || files[2].Path != "new.txt" {
		t.Fatalf("unexpected third file: %+v", files[2])
	}
}

func TestDeriveStatusFailsOnFailedCheck(t *testing.T) {
	result := Result{
		WorkingTree: "clean",
		PullRequest: &PullRequest{
			Number: 1,
			URL:    "https://example.invalid/pr/1",
		},
		Checks: []Check{{Name: "CI", Bucket: "fail"}},
	}
	if got := deriveStatus(result); got != cli.StatusFail {
		t.Fatalf("status = %s, want FAIL", got)
	}
}

func TestDeriveStatusWarnsOnPendingCheck(t *testing.T) {
	result := Result{
		WorkingTree: "clean",
		PullRequest: &PullRequest{
			Number: 1,
			URL:    "https://example.invalid/pr/1",
		},
		Checks: []Check{{Name: "CI", Bucket: "pending"}},
	}
	if got := deriveStatus(result); got != cli.StatusWarn {
		t.Fatalf("status = %s, want WARN", got)
	}
}

func TestDeriveStatusWarnsOnRisk(t *testing.T) {
	result := Result{
		WorkingTree: "clean",
		PullRequest: &PullRequest{
			Number: 1,
			URL:    "https://example.invalid/pr/1",
		},
		Risks: []string{"CI checks unavailable through gh"},
	}
	if got := deriveStatus(result); got != cli.StatusWarn {
		t.Fatalf("status = %s, want WARN", got)
	}
}

func TestParseChecksUsesBucketShape(t *testing.T) {
	checks, err := parseChecks([]byte(`[{"name":"CI","state":"PENDING","bucket":"pending","link":"https://example.invalid/check"}]`))
	if err != nil {
		t.Fatal(err)
	}
	if len(checks) != 1 {
		t.Fatalf("len(checks) = %d, want 1", len(checks))
	}
	if checks[0].Bucket != "pending" || checks[0].State != "PENDING" {
		t.Fatalf("unexpected check: %+v", checks[0])
	}
}

func TestHumanIncludesRequiredSections(t *testing.T) {
	result := Result{
		Status:      cli.StatusWarn,
		Branch:      "codex/test",
		WorkingTree: "dirty",
		Risks:       []string{"PR not found or gh unavailable"},
		NextAction:  "Open a draft PR.",
	}
	output := Human(result)
	for _, section := range []string{"Status", "Changed files", "Checks", "Risks", "Next action"} {
		if !strings.Contains(output, section) {
			t.Fatalf("output missing section %q:\n%s", section, output)
		}
	}
}
