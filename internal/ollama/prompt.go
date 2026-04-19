package ollama

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const commandSystemPrompt = `You are kyan, a CLI assistant that translates natural language into kyan CLI commands.

Available commands:
- kyan generate module <name> --fields="<field:type ...>" [--caching] [--auth]
  Field types: string, text, int, float, bool, uuid, time, json
- kyan init <project-name>

Examples:
"create a product module with title and price fields" -> kyan generate module product --fields="title:string price:float"
"add a caching decorator to orders" -> kyan generate module order --fields="..." --caching
"add an authenticated module for blog posts with title and content" -> kyan generate module blog --fields="title:string content:text" --auth
"initialize a new project called myapp" -> kyan init myapp

Translate the following into a kyan CLI command. Output ONLY the command, nothing else:`

// BuildCommandPrompt constructs the prompt for `kyan ai`.
func BuildCommandPrompt(userInput string) string {
	return fmt.Sprintf("%s\n\n%s", commandSystemPrompt, userInput)
}

// BuildAskPrompt constructs the prompt for `kyan ask`.
func BuildAskPrompt(question string, projectContext string, extraFiles []string) string {
	prompt := `You are an expert on hexagonal architecture in Go, specifically the conventions from go-hex-boilerplate and go-utility (v2).

`

	if projectContext != "" {
		prompt += fmt.Sprintf("Project context:\n%s\n\n", projectContext)
	}

	if len(extraFiles) > 0 {
		prompt += "Additional file context:\n"
		for _, f := range extraFiles {
			content, err := os.ReadFile(f)
			if err == nil {
				prompt += fmt.Sprintf("\n--- %s ---\n%s\n", f, string(content))
			}
		}
		prompt += "\n"
	}

	prompt += `Architecture conventions:
- Inward dependency: adapters -> ports -> domain
- Port naming: inbound at port/inbound/<feature>/, outbound at port/outbound/<feature>.go
- Constructor pattern: New<Name>() returns port interface
- Decorator pattern: caching repos wrap base repos, same port interface
- Error convention: domain errors use statusw codes with WithCustomMessage/WithError chaining
- Response convention: handlers return responsew.BaseResponse via responsew.Success/Error
- DI: uber-go/dig, organized in infra.go, service.go, router.go
- HTTP engines: echo (default), gin, mux

Answer the following question:
` + question

	return prompt
}

// BuildCodePrompt constructs the prompt for `kyan code`.
func BuildCodePrompt(layer, module, impl, method, name, modulePath, projectContext string) string {
	prompt := `You are a Go code generator for hexagonal architecture projects.
Generate Go code following these conventions:
- Package name matches directory
- Constructor returns port interface, not concrete struct
- Use statusw for errors, responsew for HTTP responses
- Import alias ` + "`domainError`" + ` for ` + "`domain/error`" + ` package
- Import alias ` + "`portOutbound`" + ` for ` + "`port/outbound`" + ` package
`

	if modulePath != "" {
		prompt += fmt.Sprintf("- Module path: %s\n", modulePath)
	}

	if projectContext != "" {
		prompt += fmt.Sprintf("\nProject context:\n%s\n", projectContext)
	}

	prompt += "\nGenerate the following:\n"
	if layer != "" {
		prompt += fmt.Sprintf("Layer: %s\n", layer)
	}
	if module != "" {
		prompt += fmt.Sprintf("Module: %s\n", module)
	}
	if method != "" {
		prompt += fmt.Sprintf("Method: %s\n", method)
	}
	if impl != "" {
		prompt += fmt.Sprintf("Implementation: %s\n", impl)
	}
	if name != "" {
		prompt += fmt.Sprintf("Name: %s\n", name)
	}

	prompt += "\nOutput ONLY valid Go source code, no markdown fences."

	return prompt
}

// GatherProjectContext scans the project directory and returns a summary.
func GatherProjectContext(projectDir string) string {
	var sb strings.Builder

	// Scan entities
	entityDir := filepath.Join(projectDir, "domain", "entity")
	if entries, err := os.ReadDir(entityDir); err == nil {
		sb.WriteString("Entities: ")
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") {
				sb.WriteString(strings.TrimSuffix(e.Name(), ".go") + ", ")
			}
		}
		sb.WriteString("\n")
	}

	// Scan outbound ports
	portDir := filepath.Join(projectDir, "port", "outbound")
	if entries, err := os.ReadDir(portDir); err == nil {
		sb.WriteString("Outbound ports: ")
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") {
				sb.WriteString(strings.TrimSuffix(e.Name(), ".go") + ", ")
			}
		}
		sb.WriteString("\n")
	}

	// Scan inbound ports
	inboundDir := filepath.Join(projectDir, "port", "inbound")
	if entries, err := os.ReadDir(inboundDir); err == nil {
		sb.WriteString("Inbound ports: ")
		for _, e := range entries {
			if e.IsDir() {
				sb.WriteString(e.Name() + ", ")
			}
		}
		sb.WriteString("\n")
	}

	// Scan use cases
	ucDir := filepath.Join(projectDir, "usecase")
	if entries, err := os.ReadDir(ucDir); err == nil {
		sb.WriteString("Use cases: ")
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") {
				sb.WriteString(strings.TrimSuffix(e.Name(), ".go") + ", ")
			}
		}
		sb.WriteString("\n")
	}

	// Read go.mod for module path
	modBytes, err := os.ReadFile(filepath.Join(projectDir, "go.mod"))
	if err == nil {
		for _, line := range strings.Split(string(modBytes), "\n") {
			if strings.HasPrefix(line, "module ") {
				sb.WriteString("Module path: " + strings.TrimPrefix(line, "module ") + "\n")
				break
			}
		}
	}

	return sb.String()
}