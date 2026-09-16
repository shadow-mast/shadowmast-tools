package artifact

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/shadow-mast/shadowmast-tools/internal/checksum"
	"github.com/shadow-mast/shadowmast-tools/internal/cli"
)

type FileResult struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256,omitempty"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type Result struct {
	Status string       `json:"status"`
	Files  []FileResult `json:"files"`
}

func Verify(paths []string) Result {
	var files []FileResult
	for _, path := range paths {
		files = append(files, expandPath(path)...)
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	status := cli.StatusPass
	for i := range files {
		info, err := os.Stat(files[i].Path)
		if err != nil {
			files[i].Status = cli.StatusFail
			files[i].Error = err.Error()
			status = cli.StatusFail
			continue
		}
		if info.IsDir() {
			files[i].Status = cli.StatusFail
			files[i].Error = "expected file, got directory"
			status = cli.StatusFail
			continue
		}
		files[i].Size = info.Size()
		if info.Size() <= 0 {
			files[i].Status = cli.StatusFail
			files[i].Error = "empty file"
			status = cli.StatusFail
			continue
		}
		sum, err := checksum.FileSHA256(files[i].Path)
		if err != nil {
			files[i].Status = cli.StatusFail
			files[i].Error = err.Error()
			status = cli.StatusFail
			continue
		}
		files[i].SHA256 = sum
		files[i].Status = cli.StatusPass
	}
	if len(files) == 0 {
		return Result{
			Status: cli.StatusFail,
			Files: []FileResult{{
				Path:   "",
				Status: cli.StatusFail,
				Error:  "no artifacts found",
			}},
		}
	}
	return Result{Status: status, Files: files}
}

func expandPath(path string) []FileResult {
	info, err := os.Stat(path)
	if err != nil {
		return []FileResult{{Path: path}}
	}
	if !info.IsDir() {
		return []FileResult{{Path: path}}
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return []FileResult{{Path: path, Status: cli.StatusFail, Error: fmt.Sprintf("read directory: %v", err)}}
	}

	var files []FileResult
	for _, entry := range entries {
		if entry.Type().IsRegular() {
			files = append(files, FileResult{Path: filepath.Join(path, entry.Name())})
		}
	}
	return files
}

func Markdown(result Result) string {
	out := "# Artifact Verify\n\n"
	out += "Status: " + result.Status + "\n\n"
	out += "| File | Size | SHA256 | Status |\n"
	out += "| --- | ---: | --- | --- |\n"
	for _, file := range result.Files {
		name := file.Path
		sum := file.SHA256
		if sum == "" {
			sum = "-"
		}
		status := file.Status
		if file.Error != "" {
			status += " (" + file.Error + ")"
		}
		out += fmt.Sprintf("| `%s` | %d | `%s` | %s |\n", name, file.Size, sum, status)
	}
	return out
}
