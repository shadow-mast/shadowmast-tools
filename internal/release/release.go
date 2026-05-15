package release

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/shadow-mast/shadowmast-tools/internal/checksum"
	"github.com/shadow-mast/shadowmast-tools/internal/cli"
	"github.com/shadow-mast/shadowmast-tools/internal/execx"
)

type Asset struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
	URL  string `json:"url,omitempty"`
}

type Finding struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type Check struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type Result struct {
	Status   string    `json:"status"`
	Repo     string    `json:"repo"`
	Tag      string    `json:"tag"`
	Assets   []Asset   `json:"assets"`
	Checks   []Check   `json:"checks"`
	Findings []Finding `json:"findings"`
}

type Config struct {
	Repo           string
	Tag            string
	RequiredAssets []string
	LocalDir       string
	Timeout        time.Duration
}

func CheckRelease(ctx context.Context, cfg Config) (Result, error) {
	assets, err := fetchAssets(ctx, cfg)
	if err != nil {
		return Result{}, err
	}
	result := EvaluateAssets(cfg.Repo, cfg.Tag, assets, cfg.RequiredAssets)
	if cfg.LocalDir != "" {
		checks, findings, err := verifyLocalDir(ctx, cfg, assets)
		if err != nil {
			return result, err
		}
		result.Checks = append(result.Checks, checks...)
		result.Findings = append(result.Findings, findings...)
		result.Status = statusFrom(result.Checks, result.Findings)
	}
	return result, nil
}

func EvaluateAssets(repo, tag string, assets []Asset, required []string) Result {
	slices.SortFunc(assets, func(a, b Asset) int {
		return strings.Compare(a.Name, b.Name)
	})
	result := Result{
		Status: cli.StatusPass,
		Repo:   repo,
		Tag:    tag,
		Assets: assets,
	}
	for _, name := range required {
		check := Check{Name: "asset:" + name, Status: cli.StatusPass, Message: "present"}
		if !hasAsset(assets, name) {
			check.Status = cli.StatusFail
			check.Message = "missing"
			result.Findings = append(result.Findings, Finding{Severity: cli.StatusFail, Message: "missing required asset: " + name})
		}
		result.Checks = append(result.Checks, check)
	}
	result.Status = statusFrom(result.Checks, result.Findings)
	return result
}

func ParseReleaseAssets(data []byte) ([]Asset, error) {
	var payload struct {
		Assets []Asset `json:"assets"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	return payload.Assets, nil
}

func fetchAssets(ctx context.Context, cfg Config) ([]Asset, error) {
	result, err := execx.Run(ctx, cfg.Timeout, "gh", "release", "view", cfg.Tag, "--repo", cfg.Repo, "--json", "assets")
	if err != nil {
		return nil, err
	}
	return ParseReleaseAssets([]byte(result.Stdout))
}

func verifyLocalDir(ctx context.Context, cfg Config, assets []Asset) ([]Check, []Finding, error) {
	sumPath := filepath.Join(cfg.LocalDir, "SHA256SUMS.txt")
	if _, err := os.Stat(sumPath); err != nil {
		if !os.IsNotExist(err) {
			return nil, nil, err
		}
		if !hasAsset(assets, "SHA256SUMS.txt") {
			return []Check{{Name: "checksum:file", Status: cli.StatusFail, Message: "missing SHA256SUMS.txt"}}, []Finding{{Severity: cli.StatusFail, Message: "checksum file not found locally or in release assets"}}, nil
		}
		downloaded, err := downloadChecksum(ctx, cfg)
		if err != nil {
			return nil, nil, err
		}
		sumPath = downloaded
		defer os.RemoveAll(filepath.Dir(downloaded))
	}

	file, err := os.Open(sumPath)
	if err != nil {
		return []Check{{Name: "checksum:file", Status: cli.StatusFail, Message: err.Error()}}, []Finding{{Severity: cli.StatusFail, Message: "checksum file could not be opened"}}, nil
	}
	defer file.Close()

	entries, err := checksum.ParseSHA256Sums(file)
	if err != nil {
		return []Check{{Name: "checksum:file", Status: cli.StatusFail, Message: err.Error()}}, []Finding{{Severity: cli.StatusFail, Message: "checksum file is malformed"}}, nil
	}

	checks := []Check{{Name: "checksum:file", Status: cli.StatusPass, Message: "parsed"}}
	var findings []Finding
	for _, entry := range entries {
		name, ok := safeLocalName(entry.Filename)
		check := Check{Name: "checksum:" + entry.Filename, Status: cli.StatusPass, Message: "matched"}
		if !ok {
			check.Status = cli.StatusFail
			check.Message = "unsafe filename"
			findings = append(findings, Finding{Severity: cli.StatusFail, Message: "unsafe checksum filename: " + entry.Filename})
			checks = append(checks, check)
			continue
		}
		localPath := filepath.Join(cfg.LocalDir, name)
		actual, err := checksum.FileSHA256(localPath)
		if err != nil {
			check.Status = cli.StatusFail
			check.Message = err.Error()
			findings = append(findings, Finding{Severity: cli.StatusFail, Message: "checksum target failed: " + entry.Filename})
			checks = append(checks, check)
			continue
		}
		if actual != strings.ToLower(entry.SHA256) {
			check.Status = cli.StatusFail
			check.Message = "mismatch"
			findings = append(findings, Finding{Severity: cli.StatusFail, Message: "checksum mismatch: " + entry.Filename})
		}
		checks = append(checks, check)
	}
	return checks, findings, nil
}

func downloadChecksum(ctx context.Context, cfg Config) (string, error) {
	dir, err := os.MkdirTemp("", "shadowmast-release-checker-*")
	if err != nil {
		return "", err
	}
	_, err = execx.Run(ctx, cfg.Timeout, "gh", "release", "download", cfg.Tag, "--repo", cfg.Repo, "--pattern", "SHA256SUMS.txt", "--dir", dir, "--clobber")
	if err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}
	return filepath.Join(dir, "SHA256SUMS.txt"), nil
}

func safeLocalName(name string) (string, bool) {
	clean := filepath.Clean(filepath.FromSlash(name))
	if clean == "." || clean == string(filepath.Separator) || filepath.IsAbs(clean) {
		return "", false
	}
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", false
	}
	return clean, true
}

func hasAsset(assets []Asset, name string) bool {
	for _, asset := range assets {
		if asset.Name == name {
			return true
		}
	}
	return false
}

func statusFrom(checks []Check, findings []Finding) string {
	status := cli.StatusPass
	for _, check := range checks {
		if check.Status == cli.StatusFail {
			status = cli.StatusFail
		}
	}
	for _, finding := range findings {
		if finding.Severity == cli.StatusFail {
			return cli.StatusFail
		}
		if finding.Severity == cli.StatusWarn && status == cli.StatusPass {
			status = cli.StatusWarn
		}
	}
	return status
}

func Human(result Result) string {
	var out strings.Builder
	fmt.Fprintf(&out, "release-checker: %s\n", result.Status)
	fmt.Fprintf(&out, "Repo: %s\n", result.Repo)
	fmt.Fprintf(&out, "Tag: %s\n\n", result.Tag)

	out.WriteString("Assets:\n")
	if len(result.Assets) == 0 {
		out.WriteString("- none\n")
	} else {
		for _, asset := range result.Assets {
			fmt.Fprintf(&out, "- %s (%d bytes)\n", asset.Name, asset.Size)
		}
	}

	out.WriteString("\nChecks:\n")
	if len(result.Checks) == 0 {
		out.WriteString("- none\n")
	} else {
		for _, check := range result.Checks {
			fmt.Fprintf(&out, "- %s: %s", check.Name, check.Status)
			if check.Message != "" {
				fmt.Fprintf(&out, " (%s)", check.Message)
			}
			out.WriteString("\n")
		}
	}

	out.WriteString("\nFindings:\n")
	if len(result.Findings) == 0 {
		out.WriteString("- none\n")
	} else {
		for _, finding := range result.Findings {
			fmt.Fprintf(&out, "- %s: %s\n", finding.Severity, finding.Message)
		}
	}
	return out.String()
}
