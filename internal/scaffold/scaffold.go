package scaffold

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/AndreeJait/kyan/internal/di"
	"github.com/go-git/go-git/v5"
)

const boilerplateRepo = "https://github.com/AndreeJait/go-hex-boilerplate"

type InitOptions struct {
	ProjectName string
	KeepTodo    bool
	KeepAuth    bool
	CurrentDir  bool // when true, init in current directory instead of creating a new one
}

func Init(opts InitOptions) error {
	var projectDir string

	if opts.CurrentDir {
		// Clone into a temp directory, then move contents into current dir
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("could not determine current directory: %w", err)
		}
		projectDir = cwd
		opts.ProjectName = filepath.Base(cwd)

		// Clone into temp dir first
		tmpDir, err := os.MkdirTemp("", "kyan-*")
		if err != nil {
			return fmt.Errorf("could not create temp directory: %w", err)
		}
		defer os.RemoveAll(tmpDir)

		fmt.Printf("Cloning %s...\n", boilerplateRepo)
		_, err = git.PlainClone(tmpDir, false, &git.CloneOptions{
			URL: boilerplateRepo,
		})
		if err != nil {
			return fmt.Errorf("could not clone boilerplate: %w", err)
		}

		// Remove .git from clone
		if err := os.RemoveAll(filepath.Join(tmpDir, ".git")); err != nil {
			return fmt.Errorf("could not remove .git: %w", err)
		}

		// Move all contents from tmpDir to projectDir (current dir)
		entries, err := os.ReadDir(tmpDir)
		if err != nil {
			return fmt.Errorf("could not read cloned directory: %w", err)
		}
		for _, entry := range entries {
			src := filepath.Join(tmpDir, entry.Name())
			dst := filepath.Join(projectDir, entry.Name())
			if err := os.Rename(src, dst); err != nil {
				return fmt.Errorf("could not move %s: %w", entry.Name(), err)
			}
		}
	} else {
		projectDir = opts.ProjectName

		// 1. Clone boilerplate
		fmt.Printf("Cloning %s into %s...\n", boilerplateRepo, projectDir)
		_, err := git.PlainClone(projectDir, false, &git.CloneOptions{
			URL: boilerplateRepo,
		})
		if err != nil {
			return fmt.Errorf("could not clone boilerplate: %w", err)
		}

		// 2. Remove .git so the new project starts fresh
		if err := os.RemoveAll(filepath.Join(projectDir, ".git")); err != nil {
			return fmt.Errorf("could not remove .git: %w", err)
		}
	}

	// 3. Optionally remove template modules
	if !opts.KeepTodo {
		fmt.Println("Removing todo template module...")
		if err := RemoveTodo(projectDir); err != nil {
			return fmt.Errorf("could not remove todo module: %w", err)
		}
	}

	if !opts.KeepAuth {
		fmt.Println("Removing auth template module...")
		if err := RemoveAuth(projectDir); err != nil {
			return fmt.Errorf("could not remove auth module: %w", err)
		}
	}

	// 4. Rewrite module path
	oldModule := "github.com/AndreeJait/go-hex-boilerplate"
	newModule := "github.com/AndreeJait/" + opts.ProjectName
	fmt.Printf("Rewriting module path: %s -> %s\n", oldModule, newModule)
	if err := rewriteModulePath(projectDir, oldModule, newModule); err != nil {
		return fmt.Errorf("could not rewrite module path: %w", err)
	}

	// 5. Rewrite config references
	if err := rewriteConfigReferences(projectDir, "go-hex-boilerplate", opts.ProjectName); err != nil {
		return fmt.Errorf("could not rewrite config references: %w", err)
	}

	// 6. Inject DI markers
	if err := injectDIMarkers(projectDir); err != nil {
		return fmt.Errorf("could not inject DI markers: %w", err)
	}

	// 7. Run go mod tidy && go mod vendor
	fmt.Println("Running go mod tidy...")
	if err := runInDir(projectDir, "go", "mod", "tidy"); err != nil {
		fmt.Printf("warning: go mod tidy failed: %v\n", err)
	}

	fmt.Println("Running go mod vendor...")
	if err := runInDir(projectDir, "go", "mod", "vendor"); err != nil {
		fmt.Printf("warning: go mod vendor failed: %v\n", err)
	}

	// 8. Regenerate swagger docs (needed when modules were removed)
	if !opts.KeepTodo || !opts.KeepAuth {
		if err := RegenerateSwagger(projectDir); err != nil {
			fmt.Printf("warning: swagger regeneration failed: %v\n", err)
			fmt.Println("You can regenerate manually with: kyan generate swagger")
		}
	}

	fmt.Printf("\nProject %s initialized successfully!\n", opts.ProjectName)
	fmt.Printf("cd %s to get started.\n", projectDir)
	return nil
}

func rewriteModulePath(dir, oldPath, newPath string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Skip vendor and .git directories
			if info.Name() == "vendor" || info.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, ".yaml") && filepath.Base(path) != "go.mod" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		if !bytes.Contains(content, []byte(oldPath)) {
			return nil
		}

		newContent := bytes.ReplaceAll(content, []byte(oldPath), []byte(newPath))
		return os.WriteFile(path, newContent, info.Mode())
	})
}

func rewriteConfigReferences(dir, oldName, newName string) error {
	configPath := filepath.Join(dir, "files", "config", "app.yaml")
	content, err := os.ReadFile(configPath)
	if err != nil {
		return nil // non-fatal
	}
	content = bytes.ReplaceAll(content, []byte(oldName), []byte(newName))
	return os.WriteFile(configPath, content, 0o644)
}

func injectDIMarkers(dir string) error {
	// Inject markers into wiring files
	wiringFiles := []string{
		filepath.Join(dir, "cmd", "http", "service.go"),
		filepath.Join(dir, "cmd", "http", "router.go"),
		filepath.Join(dir, "adapter", "inbound", "echo", "router.go"),
		filepath.Join(dir, "adapter", "inbound", "gin", "router.go"),
		filepath.Join(dir, "adapter", "inbound", "mux", "router.go"),
	}

	for _, f := range wiringFiles {
		content, err := os.ReadFile(f)
		if err != nil {
			fmt.Printf("warning: could not read %s for marker injection: %v\n", f, err)
			continue
		}
		content = di.InjectMarkers(content)
		if err := os.WriteFile(f, content, 0o644); err != nil {
			fmt.Printf("warning: could not write markers to %s: %v\n", f, err)
		}
	}

	return nil
}
func runInDir(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
