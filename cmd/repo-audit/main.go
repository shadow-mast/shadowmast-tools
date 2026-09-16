package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/shadow-mast/shadowmast-tools/internal/audit"
	"github.com/shadow-mast/shadowmast-tools/internal/cli"
)

func main() {
	root := flag.String("root", ".", "repository root to audit")
	denylist := flag.String("denylist", ".shadowmast-denylist", "line-based denylist path relative to root")
	jsonOutput := flag.Bool("json", false, "print machine-readable JSON")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: repo-audit [--root path] [--denylist file] [--json]\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	result := audit.Run(*root, *denylist)
	if *jsonOutput {
		if err := cli.WriteJSON(os.Stdout, result); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(cli.ExitUsageError)
		}
	} else {
		fmt.Print(audit.Human(result))
	}
	os.Exit(cli.ExitCodeForStatus(result.Status))
}
