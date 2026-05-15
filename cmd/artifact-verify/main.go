package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/shadow-mast/shadowmast-tools/internal/artifact"
	"github.com/shadow-mast/shadowmast-tools/internal/cli"
)

func main() {
	jsonOutput := flag.Bool("json", false, "print machine-readable JSON")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: artifact-verify [--json] <file-or-dir>...\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(cli.ExitUsageError)
	}

	result := artifact.Verify(flag.Args())
	if *jsonOutput {
		if err := cli.WriteJSON(os.Stdout, result); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(cli.ExitUsageError)
		}
	} else {
		fmt.Print(artifact.Markdown(result))
	}

	os.Exit(cli.ExitCodeForStatus(result.Status))
}
