package codexstatus

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/shadow-mast/shadowmast-tools/internal/cli"
	"github.com/shadow-mast/shadowmast-tools/internal/execx"
)

type ChangedFile struct {
	Status string `json:"status"`
	Path   string `json:"path"`
}

type PullRequest struct {
	Number      int    `json:"number"`
	URL         string `json:"url"`
	State       string `json:"state"`
	HeadRefName string `json:"head_ref_name"`
	BaseRefName string `json:"base_ref_name"`
}

type Check struct {
	Name     string `json:"name"`
	State    string `json:"state,omitempty"`
	Bucket   string `json:"bucket,omitempty"`
	Link     string `json:"link,omitempty"`
	Workflow string `json:"workflow,omitempty"`
}

type Result struct {
	Status       string        `json:"status"`
	Branch       string        `json:"branch"`
	WorkingTree  string        `json:"working_tree"`
	PullRequest  *PullRequest  `json:"pull_request,omitempty"`
	ChangedFiles []ChangedFile `json:"changed_files"`
	Checks       []Check       `json:"checks"`
	Risks        []string      `json:"risks"`
	NextAction   string        `json:"next_action"`
}

func Collect(ctx context.Context, timeout time.Duration) (Result, error) {
	branch, err := gitOutput(ctx, timeout, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return Result{}, err
	}
	statusOutput, err := gitOutput(ctx, timeout, "status", "--porcelain=v1")
	if err != nil {
		return Result{}, err
	}

	result := Result{
		Branch:       strings.TrimSpace(branch),
		ChangedFiles: parseChangedFiles(statusOutput),
	}
	if len(result.ChangedFiles) == 0 {
		result.WorkingTree = "clean"
	} else {
		result.WorkingTree = "dirty"
	}

	pr, prErr := currentPR(ctx, timeout)
	if prErr == nil {
		result.PullRequest = &pr
		checks, risks := prChecks(ctx, timeout)
		result.Checks = checks
		result.Risks = append(result.Risks, risks...)
	} else {
		result.Risks = append(result.Risks, "PR not found or gh unavailable")
	}

	result.Status = deriveStatus(result)
	result.NextAction = deriveNextAction(result)
	return result, nil
}

func gitOutput(ctx context.Context, timeout time.Duration, args ...string) (string, error) {
	result, err := execx.Run(ctx, timeout, "git", args...)
	if err != nil {
		return "", err
	}
	return result.Stdout, nil
}

func parseChangedFiles(output string) []ChangedFile {
	var files []ChangedFile
	for _, line := range strings.Split(output, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if len(line) < 4 {
			files = append(files, ChangedFile{Status: strings.TrimSpace(line), Path: ""})
			continue
		}
		files = append(files, ChangedFile{
			Status: strings.TrimSpace(line[:2]),
			Path:   strings.TrimSpace(line[3:]),
		})
	}
	return files
}

func currentPR(ctx context.Context, timeout time.Duration) (PullRequest, error) {
	result, err := execx.Run(ctx, timeout, "gh", "pr", "view", "--json", "url,number,state,headRefName,baseRefName")
	if err != nil {
		return PullRequest{}, err
	}
	var payload struct {
		Number      int    `json:"number"`
		URL         string `json:"url"`
		State       string `json:"state"`
		HeadRefName string `json:"headRefName"`
		BaseRefName string `json:"baseRefName"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &payload); err != nil {
		return PullRequest{}, err
	}
	return PullRequest{
		Number:      payload.Number,
		URL:         payload.URL,
		State:       payload.State,
		HeadRefName: payload.HeadRefName,
		BaseRefName: payload.BaseRefName,
	}, nil
}

func prChecks(ctx context.Context, timeout time.Duration) ([]Check, []string) {
	result, err := execx.RunAllowExitCodes(ctx, timeout, []int{8}, "gh", "pr", "checks", "--json", "name,state,bucket,link,workflow")
	if err != nil {
		return nil, []string{"CI checks unavailable through gh"}
	}
	checks, err := parseChecks([]byte(result.Stdout))
	if err != nil {
		return nil, []string{"CI checks JSON could not be parsed"}
	}
	return checks, nil
}

func parseChecks(data []byte) ([]Check, error) {
	var checks []Check
	if err := json.Unmarshal(data, &checks); err != nil {
		return nil, err
	}
	return checks, nil
}

func deriveStatus(result Result) string {
	status := cli.StatusPass
	if result.PullRequest == nil || result.WorkingTree == "dirty" {
		status = cli.StatusWarn
	}
	if len(result.Risks) > 0 && status == cli.StatusPass {
		status = cli.StatusWarn
	}
	for _, check := range result.Checks {
		bucket := strings.ToLower(check.Bucket)
		state := strings.ToLower(check.State)
		switch bucket {
		case "fail", "cancel":
			return cli.StatusFail
		case "pending":
			status = cli.StatusWarn
		}
		switch state {
		case "failure", "error", "cancelled", "timed_out", "action_required":
			return cli.StatusFail
		case "pending", "queued", "in_progress", "waiting":
			status = cli.StatusWarn
		}
	}
	return status
}

func deriveNextAction(result Result) string {
	if result.Status == cli.StatusFail {
		return "Fix failing checks."
	}
	if result.WorkingTree == "dirty" {
		return "Commit or intentionally discard local changes."
	}
	if result.PullRequest == nil {
		return "Open a draft PR."
	}
	for _, check := range result.Checks {
		if strings.ToLower(check.Bucket) == "pending" {
			return "Wait for checks or inspect pending CI."
		}
		state := strings.ToLower(check.State)
		if state == "pending" || state == "queued" || state == "in_progress" || state == "waiting" {
			return "Wait for checks or inspect pending CI."
		}
	}
	return "Ready for review or merge according to repository policy."
}

func Human(result Result) string {
	var out strings.Builder
	out.WriteString("Status\n")
	fmt.Fprintf(&out, "Branch: %s\n", result.Branch)
	fmt.Fprintf(&out, "Working tree: %s\n", result.WorkingTree)
	if result.PullRequest == nil {
		out.WriteString("PR: not found\n")
	} else {
		fmt.Fprintf(&out, "PR: #%d %s (%s)\n", result.PullRequest.Number, result.PullRequest.URL, result.PullRequest.State)
	}
	fmt.Fprintf(&out, "Overall: %s\n\n", result.Status)

	out.WriteString("Changed files\n")
	if len(result.ChangedFiles) == 0 {
		out.WriteString("- none\n")
	} else {
		for _, file := range result.ChangedFiles {
			fmt.Fprintf(&out, "- %s %s\n", file.Status, file.Path)
		}
	}

	out.WriteString("\nChecks\n")
	if len(result.Checks) == 0 {
		out.WriteString("- none\n")
	} else {
		for _, check := range result.Checks {
			fmt.Fprintf(&out, "- %s: %s", check.Name, valueOr(check.Bucket, check.State, "unknown"))
			if check.Link != "" {
				fmt.Fprintf(&out, " (%s)", check.Link)
			}
			out.WriteString("\n")
		}
	}

	out.WriteString("\nRisks\n")
	if len(result.Risks) == 0 {
		out.WriteString("- none\n")
	} else {
		for _, risk := range result.Risks {
			fmt.Fprintf(&out, "- %s\n", risk)
		}
	}

	out.WriteString("\nNext action\n")
	fmt.Fprintf(&out, "%s\n", result.NextAction)
	return out.String()
}

func valueOr(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
