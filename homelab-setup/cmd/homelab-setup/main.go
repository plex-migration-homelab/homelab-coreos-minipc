package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/cli"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/pkg/version"
)

func main() {
	// Define flags
	showVersion := flag.Bool("version", false, "Print version information")
	flag.Parse()

	// Handle version flag
	if *showVersion {
		fmt.Println(version.Info())
		return
	}

	// Initialize setup context
	ctx, err := cli.NewSetupContext()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to initialize setup context: %v\n", err)
		os.Exit(1)
	}

	// Launch interactive menu
	menu := cli.NewMenu(ctx)
	if err := menu.Show(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
