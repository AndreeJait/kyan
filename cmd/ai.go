package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/AndreeJait/kyan/internal/config"
	"github.com/AndreeJait/kyan/internal/ollama"
	"github.com/spf13/cobra"
)

var aiModel string
var aiDryRun bool

var aiCmd = &cobra.Command{
	Use:   "ai <prompt>",
	Short: "Translate natural language into kyan CLI commands via Ollama",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: could not load config, using defaults: %v\n", err)
			cfg = config.DefaultConfig()
		}

		model := cfg.AI.Model
		if aiModel != "" {
			model = aiModel
		}

		client := ollama.NewClient(cfg.AI.Host, model, cfg.AI.Key)
		prompt := ollama.BuildCommandPrompt(args[0])

		fmt.Printf("Asking %s...\n", model)
		result, err := client.Generate(cmd.Context(), prompt)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "error: %v\n", err)
			return
		}

		result = sanitizeCommand(result)
		fmt.Printf("Command: %s\n", result)

		if aiDryRun {
			return
		}

		// Execute the command
		execCmd := exec.Command("sh", "-c", result)
		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr
		execCmd.Stdin = os.Stdin

		if err := execCmd.Run(); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "error executing command: %v\n", err)
		}
	},
}

func sanitizeCommand(s string) string {
	// Remove markdown code fences if present
	s = trimFences(s)
	// Trim whitespace
	return trimWhitespace(s)
}

func trimFences(s string) string {
	lines := s
	if len(lines) >= 6 && lines[:3] == "```" {
		// Find closing fence
		end := len(lines) - 1
		for end > 0 && (lines[end] == '\n' || lines[end] == '`') {
			if end >= 3 && lines[end-2:end+1] == "```" {
				lines = lines[3 : end-2]
				break
			}
			end--
		}
	}
	return lines
}

func trimWhitespace(s string) string {
	result := s
	for len(result) > 0 && (result[0] == ' ' || result[0] == '\n' || result[0] == '\r' || result[0] == '\t') {
		result = result[1:]
	}
	for len(result) > 0 && (result[len(result)-1] == ' ' || result[len(result)-1] == '\n' || result[len(result)-1] == '\r' || result[len(result)-1] == '\t') {
		result = result[:len(result)-1]
	}
	return result
}

func init() {
	aiCmd.Flags().StringVar(&aiModel, "model", "", "Ollama model to use (overrides config)")
	aiCmd.Flags().BoolVar(&aiDryRun, "dry-run", false, "Preview the generated command without executing")
	rootCmd.AddCommand(aiCmd)
}