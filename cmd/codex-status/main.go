package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/shadow-mast/shadowmast-tools/internal/cli"
	"github.com/shadow-mast/shadowmast-tools/internal/codexstatus"
)

func main() {
	timeout := flag.Duration("timeout", 30*time.Second, "timeout for git and gh commands")
	jsonOutput := flag.Bool("json", false, "print machine-readable JSON")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: codex-status [--timeout duration] [--json]\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	result, err := codexstatus.Collect(context.Background(), *timeout)
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
		fmt.Print(codexstatus.Human(result))
	}
	os.Exit(cli.ExitCodeForStatus(result.Status))
}
