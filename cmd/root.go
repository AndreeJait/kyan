package cmd

import (
	"fmt"
	"os"

	"github.com/AndreeJait/kyan/internal/config"
	"github.com/spf13/cobra"
)

var verbose bool

var rootCmd = &cobra.Command{
	Use:   "kyan",
	Short: "CLI tool for hexagonal Go project scaffolding",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if verbose {
			fmt.Println("verbose mode enabled")
		}
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
}

func Execute() error {
	// Load kyan config (errors are non-fatal — defaults used instead)
	if _, err := config.Load(); err != nil && verbose {
		fmt.Fprintf(os.Stderr, "warning: could not load config: %v\n", err)
	}

	return rootCmd.Execute()
}