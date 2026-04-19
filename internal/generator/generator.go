package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	tmpl "text/template"

	"github.com/AndreeJait/kyan/internal/template"
	"golang.org/x/mod/modfile"
)

type Generator struct {
	ModulePath string
	ProjectDir string
}

func New(projectDir string) (*Generator, error) {
	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return nil, fmt.Errorf("could not resolve project directory: %w", err)
	}

	modPath, err := readModulePath(absDir)
	if err != nil {
		return nil, err
	}

	return &Generator{
		ModulePath: modPath,
		ProjectDir: absDir,
	}, nil
}

func readModulePath(dir string) (string, error) {
	modBytes, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return "", fmt.Errorf("could not read go.mod: %w", err)
	}

	mf, err := modfile.Parse("go.mod", modBytes, nil)
	if err != nil {
		return "", fmt.Errorf("could not parse go.mod: %w", err)
	}

	return mf.Module.Mod.Path, nil
}

func (g *Generator) GenerateModule(vars template.ModuleVars) error {
	vars.ModulePath = g.ModulePath

	// Ensure required directories exist
	dirs := []string{
		filepath.Join(g.ProjectDir, "domain", "entity"),
		filepath.Join(g.ProjectDir, "domain", "error"),
		filepath.Join(g.ProjectDir, "port", "outbound"),
		filepath.Join(g.ProjectDir, "port", "inbound", vars.ModuleNameLower),
		filepath.Join(g.ProjectDir, "usecase"),
		filepath.Join(g.ProjectDir, "adapter", "outbound", vars.ModuleNameLower),
		filepath.Join(g.ProjectDir, "adapter", "inbound", "echo"),
		filepath.Join(g.ProjectDir, "adapter", "inbound", "gin"),
		filepath.Join(g.ProjectDir, "adapter", "inbound", "mux"),
		filepath.Join(g.ProjectDir, "files", "migrations"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("could not create directory %s: %w", dir, err)
		}
	}

	// Template name -> output file path
	type tmplOutput struct {
		tmplName string
		outPath  string
		cond     bool // only generate if true
	}

	migrationNum := g.nextMigrationNumber()
	outputs := []tmplOutput{
		{"entity.tmpl", filepath.Join("domain", "entity", vars.ModuleNameLower+".go"), true},
		{"error.tmpl", filepath.Join("domain", "error", vars.ModuleNameLower+".go"), true},
		{"outbound_port.tmpl", filepath.Join("port", "outbound", vars.ModuleNameLower+".go"), true},
		{"inbound_usecase.tmpl", filepath.Join("port", "inbound", vars.ModuleNameLower, "usecase.go"), true},
		{"inbound_input.tmpl", filepath.Join("port", "inbound", vars.ModuleNameLower, "input.go"), true},
		{"usecase.tmpl", filepath.Join("usecase", vars.ModuleNameLower+".go"), true},
		{"repository.tmpl", filepath.Join("adapter", "outbound", vars.ModuleNameLower, "repository.go"), true},
		{"caching.tmpl", filepath.Join("adapter", "outbound", vars.ModuleNameLower, "caching.go"), vars.WithCaching},
		{"echo_handler.tmpl", filepath.Join("adapter", "inbound", "echo", vars.ModuleNameLower+".go"), true},
		{"gin_handler.tmpl", filepath.Join("adapter", "inbound", "gin", vars.ModuleNameLower+".go"), true},
		{"mux_handler.tmpl", filepath.Join("adapter", "inbound", "mux", vars.ModuleNameLower+".go"), true},
		{"migration_up.tmpl", filepath.Join("files", "migrations", fmt.Sprintf("%06d_create_%ss.up.sql", migrationNum, vars.ModuleNameLower)), true},
		{"migration_down.tmpl", filepath.Join("files", "migrations", fmt.Sprintf("%06d_create_%ss.down.sql", migrationNum, vars.ModuleNameLower)), true},
	}

	for _, o := range outputs {
		if !o.cond {
			continue
		}
		if err := g.renderTemplate(o.tmplName, vars, filepath.Join(g.ProjectDir, o.outPath)); err != nil {
			return fmt.Errorf("could not generate %s: %w", o.outPath, err)
		}
		fmt.Printf("  created %s\n", o.outPath)
	}

	return nil
}

func (g *Generator) renderTemplate(tmplName string, vars template.ModuleVars, outPath string) error {
	tmplData, err := template.TemplateFS.ReadFile("templates/" + tmplName)
	if err != nil {
		return fmt.Errorf("could not read template %s: %w", tmplName, err)
	}

	t, err := tmpl.New(tmplName).Parse(string(tmplData))
	if err != nil {
		return fmt.Errorf("could not parse template %s: %w", tmplName, err)
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("could not create directory for %s: %w", outPath, err)
	}

	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("could not create file %s: %w", outPath, err)
	}
	defer f.Close()

	if err := t.Execute(f, vars); err != nil {
		return fmt.Errorf("could not execute template %s: %w", tmplName, err)
	}

	return nil
}

func (g *Generator) nextMigrationNumber() int {
	migrationsDir := filepath.Join(g.ProjectDir, "files", "migrations")
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return 1
	}

	maxNum := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Extract leading digits from filenames like "000001_create_todos.up.sql"
		numStr := strings.TrimRightFunc(name, func(r rune) bool {
			return r < '0' || r > '9'
		})
		parts := strings.SplitN(name, "_", 2)
		if len(parts) > 0 {
			numStr = parts[0]
		}

		var num int
		if _, err := fmt.Sscanf(numStr, "%d", &num); err == nil && num > maxNum {
			maxNum = num
		}
	}

	return maxNum + 1
}