package audit

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/shadow-mast/shadowmast-tools/internal/cli"
)

type Check struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type Finding struct {
	Severity string `json:"severity"`
	Path     string `json:"path,omitempty"`
	Message  string `json:"message"`
}

type Result struct {
	Status   string    `json:"status"`
	Checks   []Check   `json:"checks"`
	Findings []Finding `json:"findings"`
}

func Run(root string, denylistPath string) Result {
	root = filepath.Clean(root)
	denylistAbs, denylistRel := denylistPaths(root, denylistPath)
	checks := []Check{
		checkFile(root, "README.md"),
		checkFile(root, "AGENTS.md"),
		checkDir(root, filepath.Join(".github", "workflows")),
	}

	findings := make([]Finding, 0)
	terms, denylistExists, err := loadDenylist(denylistAbs)
	if err != nil {
		findings = append(findings, Finding{Severity: cli.StatusFail, Path: denylistRel, Message: err.Error()})
	}
	if !denylistExists {
		findings = append(findings, Finding{Severity: cli.StatusWarn, Path: denylistRel, Message: "denylist file not found"})
	}

	findings = append(findings, scanDenylist(root, denylistRel, terms)...)
	findings = append(findings, scanWorkflowLinks(root)...)

	status := cli.StatusPass
	for _, check := range checks {
		if check.Status == cli.StatusFail {
			status = cli.StatusFail
		}
	}
	for _, finding := range findings {
		if finding.Severity == cli.StatusFail {
			status = cli.StatusFail
			break
		}
		if finding.Severity == cli.StatusWarn && status == cli.StatusPass {
			status = cli.StatusWarn
		}
	}

	return Result{Status: status, Checks: checks, Findings: findings}
}

func denylistPaths(root, configured string) (string, string) {
	if filepath.IsAbs(configured) {
		return configured, normalizeRelativePath(relative(root, configured))
	}
	rel := normalizeRelativePath(configured)
	return filepath.Join(root, filepath.FromSlash(rel)), rel
}

func normalizeRelativePath(value string) string {
	value = strings.ReplaceAll(value, "\\", "/")
	return path.Clean(value)
}

func checkFile(root, path string) Check {
	full := filepath.Join(root, path)
	info, err := os.Stat(full)
	if err != nil {
		return Check{Name: path, Status: cli.StatusFail, Message: "missing"}
	}
	if info.IsDir() {
		return Check{Name: path, Status: cli.StatusFail, Message: "expected file"}
	}
	return Check{Name: path, Status: cli.StatusPass, Message: "present"}
}

func checkDir(root, path string) Check {
	full := filepath.Join(root, path)
	info, err := os.Stat(full)
	if err != nil {
		return Check{Name: path, Status: cli.StatusFail, Message: "missing"}
	}
	if !info.IsDir() {
		return Check{Name: path, Status: cli.StatusFail, Message: "expected directory"}
	}
	return Check{Name: path, Status: cli.StatusPass, Message: "present"}
}

func loadDenylist(path string) ([]string, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	defer file.Close()

	var terms []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		terms = append(terms, strings.ToLower(line))
	}
	if err := scanner.Err(); err != nil {
		return nil, true, err
	}
	slices.Sort(terms)
	terms = slices.Compact(terms)
	return terms, true, nil
}

func scanDenylist(root, denylistPath string, terms []string) []Finding {
	if len(terms) == 0 {
		return nil
	}

	var findings []Finding
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			findings = append(findings, Finding{Severity: cli.StatusWarn, Path: relative(root, path), Message: err.Error()})
			return nil
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".codex-tools", ".git", "bin", "dist":
				return filepath.SkipDir
			}
			return nil
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		rel := relative(root, path)
		if normalizeRelativePath(rel) == denylistPath {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			findings = append(findings, Finding{Severity: cli.StatusWarn, Path: rel, Message: err.Error()})
			return nil
		}
		lower := strings.ToLower(string(content))
		for _, term := range terms {
			if strings.Contains(lower, term) {
				findings = append(findings, Finding{
					Severity: cli.StatusFail,
					Path:     rel,
					Message:  fmt.Sprintf("denylist term found: %s", term),
				})
			}
		}
		return nil
	})
	return findings
}

func scanWorkflowLinks(root string) []Finding {
	workflowDir := filepath.Join(root, ".github", "workflows")
	var findings []Finding
	entries, err := os.ReadDir(workflowDir)
	if err != nil {
		return nil
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".yml") && !strings.HasSuffix(name, ".yaml") {
			continue
		}
		path := filepath.Join(workflowDir, name)
		content, err := os.ReadFile(path)
		if err != nil {
			findings = append(findings, Finding{Severity: cli.StatusWarn, Path: relative(root, path), Message: err.Error()})
			continue
		}
		for lineNumber, line := range strings.Split(string(content), "\n") {
			ref, ok := workflowUseRef(line)
			if !ok {
				continue
			}
			if isForeignReusableWorkflow(ref) {
				findings = append(findings, Finding{
					Severity: cli.StatusFail,
					Path:     relative(root, path),
					Message:  fmt.Sprintf("foreign reusable workflow reference on line %d: %s", lineNumber+1, ref),
				})
			}
		}
	}
	return findings
}

func workflowUseRef(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "uses:") {
		return "", false
	}
	ref := strings.TrimSpace(strings.TrimPrefix(trimmed, "uses:"))
	ref = strings.Trim(ref, `"'`)
	return ref, ref != ""
}

func isForeignReusableWorkflow(ref string) bool {
	lower := strings.ToLower(ref)
	if strings.HasPrefix(lower, "./") {
		return false
	}
	if !strings.Contains(lower, "/.github/workflows/") {
		return false
	}
	return !strings.HasPrefix(lower, "shadow-mast/")
}

func relative(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}

func Human(result Result) string {
	var out strings.Builder
	fmt.Fprintf(&out, "repo-audit: %s\n\n", result.Status)
	out.WriteString("Checks:\n")
	for _, check := range result.Checks {
		fmt.Fprintf(&out, "- %s: %s", check.Name, check.Status)
		if check.Message != "" {
			fmt.Fprintf(&out, " (%s)", check.Message)
		}
		out.WriteString("\n")
	}
	out.WriteString("\nFindings:\n")
	if len(result.Findings) == 0 {
		out.WriteString("- none\n")
		return out.String()
	}
	for _, finding := range result.Findings {
		if finding.Path != "" {
			fmt.Fprintf(&out, "- %s %s: %s\n", finding.Severity, finding.Path, finding.Message)
		} else {
			fmt.Fprintf(&out, "- %s: %s\n", finding.Severity, finding.Message)
		}
	}
	return out.String()
}
