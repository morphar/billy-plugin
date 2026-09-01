package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cortexium-io/api-mcp/apimcp"
	"github.com/morphar/billy-plugin/internal/billyworkflow"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		fmt.Fprintf(os.Stdout, "billy-mcp %s (commit %s, built %s; api-mcp %s)\n", version, commit, buildDate, apimcp.DefaultVersion)
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "keychain" {
		if err := apimcp.RunKeychainCommand(os.Args[2:], os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	cfg, err := apimcp.LoadConfigFromArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := apimcp.RunStdio(context.Background(), cfg, billyworkflow.Tools()...); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
