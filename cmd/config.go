package cmd

import (
	"fmt"

	"github.com/AndreeJait/kyan/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage kyan configuration",
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		key, value := args[0], args[1]

		if err := config.Set(key, value); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "error: %v\n", err)
			return
		}

		fmt.Printf("set %s = %s\n", key, value)
	},
}

func init() {
	configCmd.AddCommand(configSetCmd)
	rootCmd.AddCommand(configCmd)
}