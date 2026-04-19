package scaffold

import (
	"os"
	"os/exec"
	"path/filepath"
)

// RemoveTodo deletes all todo-related files and wiring from the project.
func RemoveTodo(dir string) error {
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

	// Remove empty directories
	os.Remove(filepath.Join(dir, "port", "inbound", "todo"))
	os.Remove(filepath.Join(dir, "adapter", "outbound", "todo"))

	return nil
}

// RemoveAuth deletes all auth-related files and wiring from the project.
func RemoveAuth(dir string) error {
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

	// Remove empty directories
	os.Remove(filepath.Join(dir, "port", "inbound", "auth"))
	os.Remove(filepath.Join(dir, "adapter", "outbound", "user"))

	return nil
}

func runInDir(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}