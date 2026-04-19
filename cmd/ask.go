package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/AndreeJait/kyan/internal/config"
	"github.com/AndreeJait/kyan/internal/ollama"
	"github.com/spf13/cobra"
)

var (
	askModel   string
	askContext  string
	askVerbose bool
)

var askCmd = &cobra.Command{
	Use:   "ask <question>",
	Short: "Ask questions about your project architecture via Ollama",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: could not load config, using defaults: %v\n", err)
			cfg = config.DefaultConfig()
		}

		model := cfg.AI.Model
		if askModel != "" {
			model = askModel
		}

		// Gather project context
		projectDir, _ := os.Getwd()
		projectContext := ollama.GatherProjectContext(projectDir)

		// Parse extra context files
		var extraFiles []string
		if askContext != "" {
			extraFiles = strings.Split(askContext, ",")
		}

		prompt := ollama.BuildAskPrompt(args[0], projectContext, extraFiles)

		client := ollama.NewClient(cfg.AI.Host, model, cfg.AI.Key)
		fmt.Printf("Asking %s...\n\n", model)

		if err := client.GenerateStream(cmd.Context(), prompt, func(token string) {
			fmt.Print(token)
		}); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "\nerror: %v\n", err)
			return
		}

		fmt.Println()
	},
}

func init() {
	askCmd.Flags().StringVar(&askModel, "model", "", "Ollama model to use (overrides config)")
	askCmd.Flags().StringVar(&askContext, "context", "", "Comma-separated file paths to include in prompt")
	askCmd.Flags().BoolVar(&askVerbose, "verbose", false, "Include reasoning traces in output")
	rootCmd.AddCommand(askCmd)
}