package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AndreeJait/kyan/internal/config"
	"github.com/AndreeJait/kyan/internal/ollama"
	"github.com/spf13/cobra"
)

var (
	codeLayer   string
	codeModule  string
	codeImpl    string
	codeMethod  string
	codeName    string
	codeModel   string
	codeDryRun  bool
	codeDiff    bool
)

var codeCmd = &cobra.Command{
	Use:   "code",
	Short: "Generate Go code via Ollama constrained by hex conventions",
	Run: func(cmd *cobra.Command, args []string) {
		if codeLayer == "" && codeName == "" {
			fmt.Fprintf(cmd.ErrOrStderr(), "error: at least one of --layer or --name is required\n")
			return
		}

		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: could not load config, using defaults: %v\n", err)
			cfg = config.DefaultConfig()
		}

		model := cfg.AI.Model
		if codeModel != "" {
			model = codeModel
		}

		projectDir, _ := os.Getwd()
		projectContext := ollama.GatherProjectContext(projectDir)

		// Extract module path from project context
		modulePath := extractModulePath(projectDir)

		prompt := ollama.BuildCodePrompt(codeLayer, codeModule, codeImpl, codeMethod, codeName, modulePath, projectContext)

		client := ollama.NewClient(cfg.AI.Host, model, cfg.AI.Key)
		fmt.Printf("Generating code with %s...\n", model)

		result, err := client.Generate(cmd.Context(), prompt)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "error: %v\n", err)
			return
		}

		result = stripMarkdownFences(result)

		if codeDryRun {
			fmt.Println(result)
			return
		}

		// Determine output file path
		outPath := determineOutputPath(codeLayer, codeModule, codeName)
		if outPath == "" {
			fmt.Fprintln(cmd.ErrOrStderr(), "error: could not determine output file path. Use --layer and --module or --name explicitly.")
			fmt.Println(result)
			return
		}

		absPath := filepath.Join(projectDir, outPath)

		// Check if file exists for diff
		if codeDiff {
			existing, err := os.ReadFile(absPath)
			if err == nil {
				fmt.Printf("Diff for %s:\n", outPath)
				fmt.Printf("--- existing\n+++ generated\n")
				printSimpleDiff(string(existing), result)
				return
			}
		}

		// Write file
		if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "error: could not create directory: %v\n", err)
			return
		}

		if err := os.WriteFile(absPath, []byte(result), 0o644); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "error: could not write file: %v\n", err)
			return
		}

		fmt.Printf("Generated: %s\n", outPath)
	},
}

func extractModulePath(projectDir string) string {
	modBytes, err := os.ReadFile(filepath.Join(projectDir, "go.mod"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(modBytes), "\n") {
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}

func determineOutputPath(layer, module, name string) string {
	// Map layer to file path
	switch {
	case strings.HasPrefix(layer, "usecase"):
		if module != "" {
			return filepath.Join("usecase", module+".go")
		}
	case strings.HasPrefix(layer, "adapter/outbound") || strings.HasPrefix(layer, "repository"):
		if module != "" {
			return filepath.Join("adapter", "outbound", module, "repository.go")
		}
	case strings.HasPrefix(layer, "adapter/inbound/echo"):
		if name != "" {
			return filepath.Join("adapter", "inbound", "echo", name+".go")
		}
		if module != "" {
			return filepath.Join("adapter", "inbound", "echo", module+".go")
		}
	case strings.HasPrefix(layer, "adapter/inbound/gin"):
		if name != "" {
			return filepath.Join("adapter", "inbound", "gin", name+".go")
		}
		if module != "" {
			return filepath.Join("adapter", "inbound", "gin", module+".go")
		}
	case strings.HasPrefix(layer, "adapter/inbound/mux"):
		if name != "" {
			return filepath.Join("adapter", "inbound", "mux", name+".go")
		}
		if module != "" {
			return filepath.Join("adapter", "inbound", "mux", module+".go")
		}
	case strings.HasPrefix(layer, "domain/entity"):
		if module != "" {
			return filepath.Join("domain", "entity", module+".go")
		}
	case strings.HasPrefix(layer, "domain/error"):
		if module != "" {
			return filepath.Join("domain", "error", module+".go")
		}
	case strings.HasPrefix(layer, "port/outbound"):
		if module != "" {
			return filepath.Join("port", "outbound", module+".go")
		}
	case strings.HasPrefix(layer, "port/inbound"):
		if module != "" {
			return filepath.Join("port", "inbound", module, "usecase.go")
		}
	case name != "":
		// Use name as the file name in the current directory
		if strings.HasSuffix(name, ".go") {
			return name
		}
		return name + ".go"
	}

	return ""
}

func stripMarkdownFences(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		// Remove opening fence (possibly with language tag like ```go)
		newlineIdx := strings.Index(s, "\n")
		if newlineIdx != -1 {
			s = s[newlineIdx+1:]
		}
	}
	if strings.HasSuffix(s, "```") {
		s = s[:len(s)-3]
	}
	return strings.TrimSpace(s)
}

func printSimpleDiff(existing, generated string) {
	existingLines := strings.Split(existing, "\n")
	generatedLines := strings.Split(generated, "\n")

	maxLen := len(existingLines)
	if len(generatedLines) > maxLen {
		maxLen = len(generatedLines)
	}

	for i := 0; i < maxLen; i++ {
		var oldLine, newLine string
		if i < len(existingLines) {
			oldLine = existingLines[i]
		}
		if i < len(generatedLines) {
			newLine = generatedLines[i]
		}

		if oldLine != newLine {
			if oldLine != "" {
				fmt.Printf("-%s\n", oldLine)
			}
			if newLine != "" {
				fmt.Printf("+%s\n", newLine)
			}
		}
	}
}

func init() {
	codeCmd.Flags().StringVar(&codeLayer, "layer", "", "Target layer (e.g., usecase, adapter/outbound, adapter/inbound/echo, domain/entity)")
	codeCmd.Flags().StringVar(&codeModule, "module", "", "Module name (e.g., product)")
	codeCmd.Flags().StringVar(&codeImpl, "impl", "", "Implementation type (e.g., mongow, kafkaw)")
	codeCmd.Flags().StringVar(&codeMethod, "method", "", "Method name to generate (e.g., SoftDelete)")
	codeCmd.Flags().StringVar(&codeName, "name", "", "Custom file name")
	codeCmd.Flags().StringVar(&codeModel, "model", "", "Ollama model to use (overrides config)")
	codeCmd.Flags().BoolVar(&codeDryRun, "dry-run", false, "Preview the generated code without writing")
	codeCmd.Flags().BoolVar(&codeDiff, "diff", false, "Show diff against existing file before applying")
	rootCmd.AddCommand(codeCmd)
}