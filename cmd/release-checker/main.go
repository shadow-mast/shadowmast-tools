package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/shadow-mast/shadowmast-tools/internal/cli"
	"github.com/shadow-mast/shadowmast-tools/internal/flags"
	"github.com/shadow-mast/shadowmast-tools/internal/release"
)

func main() {
	var required flags.StringList
	repo := flag.String("repo", "", "GitHub repository in owner/name format")
	tag := flag.String("tag", "", "release tag or version")
	localDir := flag.String("local-dir", "", "directory with local files to verify against SHA256SUMS.txt")
	timeout := flag.Duration("timeout", 30*time.Second, "timeout for gh commands")
	jsonOutput := flag.Bool("json", false, "print machine-readable JSON")
	flag.Var(&required, "require-asset", "required release asset name; repeat for multiple assets")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: release-checker --repo owner/name --tag vX.Y.Z [--require-asset name] [--local-dir path] [--json]\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *repo == "" || *tag == "" {
		flag.Usage()
		os.Exit(cli.ExitUsageError)
	}

	result, err := release.CheckRelease(context.Background(), release.Config{
		Repo:           *repo,
		Tag:            *tag,
		RequiredAssets: []string(required),
		LocalDir:       *localDir,
		Timeout:        *timeout,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(cli.ExitUsageError)
	}

	if *jsonOutput {
		if err := cli.WriteJSON(os.Stdout, result); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(cli.ExitUsageError)
		}
	} else {
		fmt.Print(release.Human(result))
	}
	os.Exit(cli.ExitCodeForStatus(result.Status))
}
