package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AndreeJait/kyan/internal/scaffold"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init [project-name]",
	Short: "Create a new hexagonal Go project from go-hex-boilerplate",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var projectName string
		currentDir := false

		if len(args) == 1 {
			projectName = args[0]
		} else {
			// No argument: init in current directory
			currentDir = true
			dir, err := os.Getwd()
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "error: could not determine current directory: %v\n", err)
				return
			}
			projectName = filepath.Base(dir)
		}

		// Validate project name
		if strings.Contains(projectName, "/") || strings.Contains(projectName, " ") {
			fmt.Fprintf(cmd.ErrOrStderr(), "error: invalid project name: %q\n", projectName)
			return
		}

		// Prompt for todo module
		keepTodo := promptYesNo("Keep the todo template module? (y/N)", false)
		keepAuth := promptYesNo("Keep the auth template module? (y/N)", false)

		opts := scaffold.InitOptions{
			ProjectName: projectName,
			KeepTodo:    keepTodo,
			KeepAuth:    keepAuth,
			CurrentDir:  currentDir,
		}

		if err := scaffold.Init(opts); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "error: %v\n", err)
			return
		}
	},
}

func promptYesNo(prompt string, defaultVal bool) bool {
	reader := bufio.NewReader(os.Stdin)
	suffix := "(y/N)"
	if defaultVal {
		suffix = "(Y/n)"
	}

	fmt.Printf("%s %s: ", prompt, suffix)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	if input == "" {
		return defaultVal
	}
	return input == "y" || input == "yes"
}

func init() {
	rootCmd.AddCommand(initCmd)
}