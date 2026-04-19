package scaffold

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// RemoveTodo deletes all todo-related files and rewrites wiring to remove todo references.
func RemoveTodo(dir string) error {
	// Delete todo files
	files := []string{
		filepath.Join(dir, "domain", "entity", "todo.go"),
		filepath.Join(dir, "domain", "error", "todo.go"),
		filepath.Join(dir, "port", "outbound", "todo.go"),
		filepath.Join(dir, "port", "inbound", "todo", "usecase.go"),
		filepath.Join(dir, "port", "inbound", "todo", "input.go"),
		filepath.Join(dir, "usecase", "todo.go"),
		filepath.Join(dir, "adapter", "outbound", "todo", "repository.go"),
		filepath.Join(dir, "adapter", "outbound", "todo", "caching.go"),
		filepath.Join(dir, "adapter", "inbound", "echo", "todo.go"),
		filepath.Join(dir, "adapter", "inbound", "gin", "todo.go"),
		filepath.Join(dir, "adapter", "inbound", "mux", "todo.go"),
		filepath.Join(dir, "files", "migrations", "000001_create_todos.up.sql"),
		filepath.Join(dir, "files", "migrations", "000001_create_todos.down.sql"),
	}
	for _, f := range files {
		os.Remove(f)
	}
	os.RemoveAll(filepath.Join(dir, "port", "inbound", "todo"))
	os.RemoveAll(filepath.Join(dir, "adapter", "outbound", "todo"))

	// Rewrite wiring files to remove todo references
	if err := removeTodoFromService(dir); err != nil {
		return fmt.Errorf("service.go: %w", err)
	}
	if err := removeTodoFromRouter(dir); err != nil {
		return fmt.Errorf("router.go: %w", err)
	}
	for _, engine := range []string{"echo", "gin", "mux"} {
		if err := removeTodoFromAdapterRouter(dir, engine); err != nil {
			return fmt.Errorf("adapter/inbound/%s/router.go: %w", engine, err)
		}
	}

	return nil
}

// RemoveAuth deletes all auth-related files and rewrites wiring to remove auth references.
func RemoveAuth(dir string) error {
	// Delete auth files
	files := []string{
		filepath.Join(dir, "domain", "entity", "user.go"),
		filepath.Join(dir, "domain", "error", "auth.go"),
		filepath.Join(dir, "port", "outbound", "user.go"),
		filepath.Join(dir, "port", "inbound", "auth", "usecase.go"),
		filepath.Join(dir, "port", "inbound", "auth", "input.go"),
		filepath.Join(dir, "usecase", "auth.go"),
		filepath.Join(dir, "adapter", "outbound", "user", "repository.go"),
		filepath.Join(dir, "adapter", "inbound", "echo", "auth.go"),
		filepath.Join(dir, "adapter", "inbound", "gin", "auth.go"),
		filepath.Join(dir, "adapter", "inbound", "mux", "auth.go"),
		filepath.Join(dir, "files", "migrations", "000002_create_users.up.sql"),
		filepath.Join(dir, "files", "migrations", "000002_create_users.down.sql"),
	}
	for _, f := range files {
		os.Remove(f)
	}
	os.RemoveAll(filepath.Join(dir, "port", "inbound", "auth"))
	os.RemoveAll(filepath.Join(dir, "adapter", "outbound", "user"))

	// Rewrite wiring files to remove auth references
	if err := removeAuthFromInfra(dir); err != nil {
		return fmt.Errorf("infra.go: %w", err)
	}
	if err := removeAuthFromService(dir); err != nil {
		return fmt.Errorf("service.go: %w", err)
	}
	if err := removeAuthFromRouter(dir); err != nil {
		return fmt.Errorf("router.go: %w", err)
	}
	for _, engine := range []string{"echo", "gin", "mux"} {
		if err := removeAuthFromAdapterRouter(dir, engine); err != nil {
			return fmt.Errorf("adapter/inbound/%s/router.go: %w", engine, err)
		}
	}
	if err := removeAuthFromConfig(dir); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if err := removeAuthFromMain(dir); err != nil {
		return fmt.Errorf("main.go: %w", err)
	}

	return nil
}

// removeTodoFromService removes todo references from cmd/http/service.go.
func removeTodoFromService(dir string) error {
	path := filepath.Join(dir, "cmd", "http", "service.go")
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Remove todo-related imports (split and filter approach for reliable substring matching)
	lines := strings.Split(string(content), "\n")
	var filtered []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, `/adapter/outbound/todo"`) {
			continue
		}
		if strings.Contains(trimmed, `/port/inbound/todo"`) {
			continue
		}
		filtered = append(filtered, line)
	}
	content = []byte(strings.Join(filtered, "\n"))

	// Remove todo provider lines
	content = removeLineContaining(content, "c.Provide(newTodoRepository)")
	content = removeLineContaining(content, "c.Provide(newTodoUseCase)")

	// Remove todo constructor functions
	content = removeFunction(content, "newTodoRepository")
	content = removeFunction(content, "newTodoUseCase")

	return writeFile(path, content)
}

// removeTodoFromRouter removes todo references from cmd/http/router.go.
func removeTodoFromRouter(dir string) error {
	path := filepath.Join(dir, "cmd", "http", "router.go")
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Remove todo imports (split and filter approach)
	lines := strings.Split(string(content), "\n")
	var filtered []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, `"github.com/AndreeJait/`) && strings.Contains(trimmed, `/port/inbound/todo"`) {
			continue
		}
		filtered = append(filtered, line)
	}
	content = []byte(strings.Join(filtered, "\n"))

	// Remove todoUC parameter from newRouter
	content = removeLineContaining(content, "todoUC todo.UseCase,")

	// Remove todoUC from RegisterRoutes calls — handle patterns like:
	// echoAdapter.RegisterRoutes(e, healthUC, todoUC, rbac, authenticator)
	// → echoAdapter.RegisterRoutes(e, healthUC, rbac, authenticator)
	content = bytes.ReplaceAll(content, []byte("healthUC, todoUC, rbac, authenticator)"), []byte("healthUC, rbac, authenticator)"))

	return writeFile(path, content)
}

// removeTodoFromAdapterRouter removes todo references from adapter/inbound/<engine>/router.go.
func removeTodoFromAdapterRouter(dir, engine string) error {
	path := filepath.Join(dir, "adapter", "inbound", engine, "router.go")
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Remove todo import
	lines := strings.Split(string(content), "\n")
	var filtered []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, `/port/inbound/todo"`) {
			continue
		}
		filtered = append(filtered, line)
	}
	content = []byte(strings.Join(filtered, "\n"))

	// Remove todoUC parameter from RegisterRoutes
	content = bytes.ReplaceAll(content, []byte("todoUC todo.UseCase, "), nil)
	content = bytes.ReplaceAll(content, []byte(", todoUC todo.UseCase"), nil)
	content = bytes.ReplaceAll(content, []byte("todoUC todo.UseCase"), nil)

	// Remove registerTodoRoutes call
	content = removeLineContaining(content, "registerTodoRoutes")

	// Remove todoUC from function call args
	content = bytes.ReplaceAll(content, []byte(", todoUC, rbac"), []byte(", rbac"))
	content = bytes.ReplaceAll(content, []byte("todoUC, rbac"), []byte("rbac"))

	return writeFile(path, content)
}

// removeAuthFromInfra removes auth references from cmd/http/infra.go.
func removeAuthFromInfra(dir string) error {
	path := filepath.Join(dir, "cmd", "http", "infra.go")
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Remove auth-related imports
	lines := strings.Split(string(content), "\n")
	var filtered []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, `/adapter/outbound/user"`) {
			continue
		}
		if strings.Contains(trimmed, `/go-utility/v2/authw"`) {
			continue
		}
		if strings.Contains(trimmed, `/go-utility/v2/jwtw"`) {
			continue
		}
		if strings.Contains(trimmed, `/golang-jwt/jwt/v5"`) {
			continue
		}
		filtered = append(filtered, line)
	}
	content = []byte(strings.Join(filtered, "\n"))

	// Remove auth provider lines
	content = removeLineContaining(content, "c.Provide(newJWTManager)")
	content = removeLineContaining(content, "c.Provide(newAuthenticator)")
	content = removeLineContaining(content, "c.Provide(newUserRepository)")
	content = removeLineContaining(content, "c.Provide(newRBAC)")

	// Remove auth constructor functions
	content = removeFunction(content, "newJWTManager")
	content = removeFunction(content, "newAuthenticator")
	content = removeFunction(content, "newUserRepository")
	content = removeFunction(content, "newRBAC")

	return writeFile(path, content)
}

// removeAuthFromService removes auth references from cmd/http/service.go.
func removeAuthFromService(dir string) error {
	path := filepath.Join(dir, "cmd", "http", "service.go")
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Remove auth-related imports
	lines := strings.Split(string(content), "\n")
	var filtered []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, `/port/inbound/auth"`) {
			continue
		}
		if strings.Contains(trimmed, `/go-utility/v2/jwtw"`) {
			continue
		}
		filtered = append(filtered, line)
	}
	content = []byte(strings.Join(filtered, "\n"))

	// Remove auth provider lines
	content = removeLineContaining(content, "c.Provide(newAuthUseCase)")

	// Remove auth constructor function
	content = removeFunction(content, "newAuthUseCase")

	return writeFile(path, content)
}

// removeAuthFromRouter removes auth references from cmd/http/router.go.
func removeAuthFromRouter(dir string) error {
	path := filepath.Join(dir, "cmd", "http", "router.go")
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Remove auth-related imports
	lines := strings.Split(string(content), "\n")
	var filtered []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, `/port/inbound/auth"`) {
			continue
		}
		if strings.Contains(trimmed, `/go-utility/v2/authw"`) {
			continue
		}
		filtered = append(filtered, line)
	}
	content = []byte(strings.Join(filtered, "\n"))

	// Remove auth params from newRouter
	content = removeLineContaining(content, "authenticator authw.Authenticator,")
	content = removeLineContaining(content, "rbac *authw.RBAC,")
	content = removeLineContaining(content, "authUC auth.UseCase,")

	// Remove RegisterAuthRoutes calls
	content = removeLineContaining(content, "RegisterAuthRoutes")

	// Remove auth params from RegisterRoutes calls
	content = bytes.ReplaceAll(content, []byte(", rbac, authenticator)"), []byte(")"))
	content = bytes.ReplaceAll(content, []byte("healthUC, todoUC, rbac, authenticator)"), []byte("healthUC)"))
	content = bytes.ReplaceAll(content, []byte("healthUC, rbac, authenticator)"), []byte("healthUC)"))

	// Remove auth import
	lines = strings.Split(string(content), "\n")
	filtered = nil
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, `/port/inbound/auth"`) {
			continue
		}
		if strings.Contains(trimmed, `/go-utility/v2/authw"`) && !strings.Contains(trimmed, "httpw") {
			continue
		}
		filtered = append(filtered, line)
	}
	content = []byte(strings.Join(filtered, "\n"))

	return writeFile(path, content)
}

// removeAuthFromAdapterRouter removes auth references from adapter/inbound/<engine>/router.go.
func removeAuthFromAdapterRouter(dir, engine string) error {
	path := filepath.Join(dir, "adapter", "inbound", engine, "router.go")
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Remove auth-related imports
	lines := strings.Split(string(content), "\n")
	var filtered []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, `/go-utility/v2/authw"`) {
			continue
		}
		filtered = append(filtered, line)
	}
	content = []byte(strings.Join(filtered, "\n"))

	// Remove auth middleware group/subrouter setup
	content = removeLineContaining(content, "protected :=")
	content = removeLineContaining(content, "protected =")
	content = removeLineContaining(content, "protected.Use(httpw.AuthMiddleware(authenticator))")
	content = removeLineContaining(content, `protected := e.Group("", httpw.AuthMiddleware(authenticator))`)
	content = removeLineContaining(content, `protected.Use(httpw.AuthMiddleware(authenticator))`)

	// Remove rbac and authenticator from RegisterRoutes params
	content = bytes.ReplaceAll(content, []byte(", rbac *authw.RBAC"), nil)
	content = bytes.ReplaceAll(content, []byte(", rbac"), []byte(""))
	content = bytes.ReplaceAll(content, []byte(", authenticator authw.Authenticator"), nil)
	content = bytes.ReplaceAll(content, []byte(", authenticator"), []byte(""))
	content = bytes.ReplaceAll(content, []byte("rbac *authw.RBAC, "), nil)
	content = bytes.ReplaceAll(content, []byte("authenticator authw.Authenticator"), nil)

	// Remove RegisterAuthRoutes calls
	content = removeLineContaining(content, "RegisterAuthRoutes")

	// Remove protected group/subrouter declarations
	content = removeLineContaining(content, `e.Group("", httpw.AuthMiddleware(authenticator))`)
	content = removeLineContaining(content, `r.Group("", httpw.AuthMiddleware(authenticator))`)
	content = removeLineContaining(content, `protected.Use(httpw.AuthMiddleware(authenticator))`)

	// If engine is echo, replace protected group with direct registration on e
	// Change "registerXxxRoutes(protected, ...)" to direct calls
	// Actually, we need to leave the kyan:register markers intact for future module generation
	// Just remove the protected group setup lines and auth middleware

	// For mux: remove the subrouter line
	content = removeLineContaining(content, `protected := r.PathPrefix("").Subrouter()`)
	content = removeLineContaining(content, `protected.Use(httpw.AuthMiddleware(authenticator))`)

	// Remove auth import if it still exists
	lines = strings.Split(string(content), "\n")
	filtered = nil
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, `/go-utility/v2/authw"`) {
			continue
		}
		filtered = append(filtered, line)
	}
	content = []byte(strings.Join(filtered, "\n"))

	return writeFile(path, content)
}

// removeAuthFromConfig removes auth config from config.go and app.yaml.
func removeAuthFromConfig(dir string) error {
	// Remove Auth struct from config.go
	configPath := filepath.Join(dir, "config", "config.go")
	content, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	// Remove Auth struct field
	content = removeBlockContaining(content, "Auth struct {")
	// Remove Auth defaults
	content = removeLineContaining(content, "cfg.Auth.JWTTTL")
	content = removeLineContaining(content, "cfg.Auth.JWTIssuer")

	if err := writeFile(configPath, content); err != nil {
		return err
	}

	// Remove auth section from app.yaml
	yamlPath := filepath.Join(dir, "files", "config", "app.yaml")
	yamlContent, err := os.ReadFile(yamlPath)
	if err != nil {
		return nil // non-fatal
	}

	lines := strings.Split(string(yamlContent), "\n")
	var filtered []string
	inAuthSection := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "auth:" || strings.HasPrefix(trimmed, "auth:") {
			inAuthSection = true
			continue
		}
		if inAuthSection {
			// Auth section lines are indented under "auth:"
			if line != "" && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
				inAuthSection = false
			} else {
				continue
			}
		}
		filtered = append(filtered, line)
	}

	// Remove trailing blank lines
	for len(filtered) > 0 && filtered[len(filtered)-1] == "" {
		filtered = filtered[:len(filtered)-1]
	}

	return os.WriteFile(yamlPath, []byte(strings.Join(filtered, "\n")+"\n"), 0o644)
}

// removeAuthFromMain removes auth-related swagger annotations from main.go.
func removeAuthFromMain(dir string) error {
	mainPath := filepath.Join(dir, "cmd", "http", "main.go")
	content, err := os.ReadFile(mainPath)
	if err != nil {
		return err
	}

	// Remove swagger auth annotations
	content = removeLineContaining(content, "@securityDefinitions.basic BasicAuth")
	content = removeLineContaining(content, "@securityDefinitions.apikey BearerAuth")
	content = removeLineContaining(content, "@in header")
	content = removeLineContaining(content, "@name Authorization")

	return writeFile(mainPath, content)
}

// Helper functions

func removeImportLine(content []byte, pattern string) []byte {
	lines := strings.Split(string(content), "\n")
	var filtered []string
	for _, line := range lines {
		if strings.Contains(line, pattern) {
			continue
		}
		filtered = append(filtered, line)
	}
	return []byte(strings.Join(filtered, "\n"))
}

func removeLineContaining(content []byte, substr string) []byte {
	lines := strings.Split(string(content), "\n")
	var filtered []string
	for _, line := range lines {
		if strings.Contains(line, substr) {
			continue
		}
		filtered = append(filtered, line)
	}
	return []byte(strings.Join(filtered, "\n"))
}

func removeBlockContaining(content []byte, startMarker string) []byte {
	lines := strings.Split(string(content), "\n")
	var filtered []string
	skip := false
	for _, line := range lines {
		if strings.Contains(line, startMarker) {
			skip = true
		}
		if skip {
			// Find closing brace for the struct field (may have struct tags after })
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "}") {
				skip = false
				continue
			}
			continue
		}
		filtered = append(filtered, line)
	}
	return []byte(strings.Join(filtered, "\n"))
}

func removeFunction(content []byte, funcName string) []byte {
	str := string(content)

	// Find "func <funcName>"
	idx := strings.Index(str, "func "+funcName)
	if idx == -1 {
		return content
	}

	// Find the opening brace
	braceStart := strings.Index(str[idx:], "{")
	if braceStart == -1 {
		return content
	}
	braceStart += idx

	// Count braces to find matching close
	depth := 0
	i := braceStart
	for i < len(str) {
		if str[i] == '{' {
			depth++
		} else if str[i] == '}' {
			depth--
			if depth == 0 {
				// Include trailing newline
				end := i + 1
				if end < len(str) && str[end] == '\n' {
					end++
				}
				// Also remove leading blank line before the function
				start := idx
				if start > 0 && str[start-1] == '\n' {
					start--
				}
				return []byte(str[:start] + str[end:])
			}
		}
		i++
	}

	return content
}

func writeFile(path string, content []byte) error {
	// Clean up multiple blank lines
	str := string(content)
	for strings.Contains(str, "\n\n\n") {
		str = strings.ReplaceAll(str, "\n\n\n", "\n\n")
	}
	return os.WriteFile(path, []byte(str), 0o644)
}

// RegenerateSwagger deletes stale docs/ and regenerates via swag init.
func RegenerateSwagger(dir string) error {
	docsDir := filepath.Join(dir, "docs")

	// Remove existing stale docs
	if err := os.RemoveAll(docsDir); err != nil {
		return fmt.Errorf("could not remove stale docs: %w", err)
	}

	// Ensure swag is installed
	if err := ensureSwag(); err != nil {
		return fmt.Errorf("could not install swag: %w", err)
	}

	// Run swag init
	fmt.Println("Regenerating swagger docs...")
	if err := runInDir(dir, "swag", "init", "-g", "cmd/http/main.go", "-o", "./docs", "--parseDependency", "--parseInternal"); err != nil {
		return fmt.Errorf("swag init failed: %w", err)
	}

	return nil
}

// ensureSwag checks if swag is on PATH; if not, installs it.
func ensureSwag() error {
	_, err := exec.LookPath("swag")
	if err == nil {
		return nil
	}
	fmt.Println("Installing swag...")
	return runInDir("", "go", "install", "github.com/swaggo/swag/cmd/swag@latest")
}