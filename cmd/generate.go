package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AndreeJait/kyan/internal/generator"
	"github.com/AndreeJait/kyan/internal/template"
	"github.com/spf13/cobra"
)

var (
	fieldsStr string
	withCaching bool
	withAuth   bool
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate hexagonal architecture code",
	Aliases: []string{"g"},
}

var generateModuleCmd = &cobra.Command{
	Use:   "module <name>",
	Short: "Generate a complete feature module",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		fields, err := template.ParseFields(fieldsStr)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "error: %v\n", err)
			return
		}

		// Detect project directory (current working directory)
		projectDir, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "error: could not determine working directory: %v\n", err)
			return
		}

		// Verify this is a hex project
		if err := validateHexProject(projectDir); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "error: %v\n", err)
			return
		}

		gen, err := generator.New(projectDir)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "error: %v\n", err)
			return
		}

		vars := template.ModuleVars{
			ModuleName:      pascalCase(name),
			ModuleNameLower: lowerCase(name),
			Fields:          fields,
			WithCaching:     withCaching,
			WithAuth:        withAuth,
		}

		fmt.Printf("Generating module %s...\n", vars.ModuleName)
		if err := gen.GenerateModule(vars); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "error: %v\n", err)
			return
		}

		fmt.Printf("\nModule %s generated successfully!\n", vars.ModuleName)
		fmt.Println("Run 'go mod tidy' to ensure dependencies are up to date.")
	},
}

func validateHexProject(dir string) error {
	checks := []string{
		filepath.Join(dir, "go.mod"),
		filepath.Join(dir, "domain"),
		filepath.Join(dir, "port"),
		filepath.Join(dir, "usecase"),
		filepath.Join(dir, "adapter"),
	}
	for _, check := range checks {
		if _, err := os.Stat(check); os.IsNotExist(err) {
			return fmt.Errorf("not a hex project: missing %s (run 'kyan init' first)", check)
		}
	}
	return nil
}

func pascalCase(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func lowerCase(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

func init() {
	generateModuleCmd.Flags().StringVar(&fieldsStr, "fields", "", `entity fields (e.g., --fields="title:string description:text completed:bool")`)
	generateModuleCmd.Flags().BoolVar(&withCaching, "caching", false, "wrap repository with Redis caching decorator")
	generateModuleCmd.Flags().BoolVar(&withAuth, "auth", false, "add RBAC permission checks on routes")

	generateCmd.AddCommand(generateModuleCmd)
	rootCmd.AddCommand(generateCmd)
}