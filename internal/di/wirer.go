package di

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/AndreeJait/kyan/internal/template"
)

const (
	markerProviderStart = "// kyan:provider:start"
	markerProviderEnd   = "// kyan:provider:end"
	markerServiceStart  = "// kyan:service:start"
	markerServiceEnd    = "// kyan:service:end"
	markerParamStart    = "// kyan:param:start"
	markerParamEnd      = "// kyan:param:end"
	markerRegisterStart = "// kyan:register:start"
	markerRegisterEnd   = "// kyan:register:end"
)

// WireModule updates the DI wiring files to register a new module.
func WireModule(projectDir string, vars template.ModuleVars) error {
	if err := wireService(projectDir, vars); err != nil {
		return fmt.Errorf("wire service.go: %w", err)
	}
	if err := wireRouter(projectDir, vars); err != nil {
		return fmt.Errorf("wire router.go: %w", err)
	}
	if err := wireAdapterRouter(projectDir, "echo", vars); err != nil {
		return fmt.Errorf("wire echo router.go: %w", err)
	}
	if err := wireAdapterRouter(projectDir, "gin", vars); err != nil {
		return fmt.Errorf("wire gin router.go: %w", err)
	}
	if err := wireAdapterRouter(projectDir, "mux", vars); err != nil {
		return fmt.Errorf("wire mux router.go: %w", err)
	}
	return nil
}

// wireService updates cmd/http/service.go to add provider and constructor functions.
func wireService(projectDir string, vars template.ModuleVars) error {
	path := projectDir + "/cmd/http/service.go"
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}

	// Build the new provider lines
	providerLines := fmt.Sprintf("\tc.Provide(new%sRepository)\n\tc.Provide(new%sUseCase)",
		vars.ModuleName, vars.ModuleName)

	// Build the new constructor functions
	repoFunc := buildRepoConstructor(vars)
	ucFunc := buildUseCaseConstructor(vars)
	serviceBlock := "\n" + repoFunc + "\n\n" + ucFunc + "\n"

	// Build new import lines
	importAlias := vars.ModuleNameLower + "Outbound"
	importPath := vars.ModulePath + "/adapter/outbound/" + vars.ModuleNameLower
	inboundImport := vars.ModulePath + "/port/inbound/" + vars.ModuleNameLower
	newImports := fmt.Sprintf("\n\t%s \"%s\"", importAlias, importPath)
	newImports += fmt.Sprintf("\n\t\"%s\"", inboundImport)

	content = insertAfterMarker(content, markerProviderEnd, providerLines)
	content = insertAfterMarker(content, markerServiceEnd, serviceBlock)
	content = addImport(content, newImports)

	return os.WriteFile(path, content, 0o644)
}

// wireRouter updates cmd/http/router.go to add use case parameter and RegisterRoutes calls.
func wireRouter(projectDir string, vars template.ModuleVars) error {
	path := projectDir + "/cmd/http/router.go"
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}

	// Add use case parameter to newRouter
	paramLine := fmt.Sprintf("\t%sUC %s.UseCase,", vars.ModuleNameLower, vars.ModuleNameLower)

	// Add import for inbound port
	inboundImport := vars.ModulePath + "/port/inbound/" + vars.ModuleNameLower
	importLine := fmt.Sprintf("\n\t\"%s\"", inboundImport)

	// Add RegisterRoutes calls for each engine
	// We need to add the use case param to each RegisterRoutes call
	ucParam := fmt.Sprintf("%sUC, ", vars.ModuleNameLower)

	// Add to each engine's RegisterRoutes call
	content = insertAfterMarker(content, markerParamEnd, paramLine)

	// Insert the usecase parameter into each RegisterRoutes call
	// Pattern: RegisterRoutes(e, healthUC, todoUC, rbac, authenticator)
	// We add the new use case before rbac
	content = addUCToRegisterRoutes(content, ucParam)

	content = addImport(content, importLine)

	return os.WriteFile(path, content, 0o644)
}

// wireAdapterRouter updates adapter/inbound/<engine>/router.go to add the new routes.
func wireAdapterRouter(projectDir, engine string, vars template.ModuleVars) error {
	path := projectDir + "/adapter/inbound/" + engine + "/router.go"
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}

	// Add use case parameter to RegisterRoutes
	paramLine := fmt.Sprintf("\t%sUC %s.UseCase,", vars.ModuleNameLower, vars.ModuleNameLower)

	// Add registerRoutes call
	registerCall := fmt.Sprintf("\tregister%sRoutes(protected, %sUC, rbac)", vars.ModuleName, vars.ModuleNameLower)

	// Add import for inbound port
	inboundImport := vars.ModulePath + "/port/inbound/" + vars.ModuleNameLower
	importLine := fmt.Sprintf("\n\t\"%s\"", inboundImport)

	content = insertAfterMarker(content, markerParamEnd, paramLine)
	content = insertAfterMarker(content, markerRegisterEnd, registerCall)
	content = addImport(content, importLine)

	return os.WriteFile(path, content, 0o644)
}

// buildRepoConstructor builds the repository provider function.
func buildRepoConstructor(vars template.ModuleVars) string {
	importAlias := vars.ModuleNameLower + "Outbound"

	if vars.WithCaching {
		return fmt.Sprintf(`func new%sRepository(gormDB *gorm.DB, redisClient *redis.Client) portOutbound.%sRepository {
	baseRepo := %s.NewRepository(gormDB)
	return %s.NewCachingRepository(baseRepo, redisClient)
}`, vars.ModuleName, vars.ModuleName, importAlias, importAlias)
	}

	return fmt.Sprintf(`func new%sRepository(gormDB *gorm.DB) portOutbound.%sRepository {
	return %s.NewRepository(gormDB)
}`, vars.ModuleName, vars.ModuleName, importAlias)
}

// buildUseCaseConstructor builds the use case provider function.
func buildUseCaseConstructor(vars template.ModuleVars) string {
	return fmt.Sprintf(`func new%sUseCase(repo portOutbound.%sRepository) %s.UseCase {
	return usecase.New%sUseCase(repo)
}`, vars.ModuleName, vars.ModuleName, vars.ModuleNameLower, vars.ModuleName)
}

// insertAfterMarker inserts content after the marker line.
func insertAfterMarker(content []byte, marker, insertion string) []byte {
	lines := bytes.Split(content, []byte("\n"))
	var result [][]byte
	inserted := false

	for _, line := range lines {
		result = append(result, line)
		if bytes.Contains(line, []byte(marker)) && !inserted {
			result = append(result, []byte(insertion))
			inserted = true
		}
	}

	return bytes.Join(result, []byte("\n"))
}

// addImport adds an import line before the closing import paren.
func addImport(content []byte, importLine string) []byte {
	// Find the closing paren of the import block
	idx := bytes.LastIndex(content, []byte(")"))
	if idx == -1 {
		return content
	}

	// Check if we're in the import block by looking for "import" before the closing paren
	importIdx := bytes.Index(content, []byte("import"))
	if importIdx == -1 || importIdx > idx {
		return content
	}

	// Insert before the closing paren of the import block
	result := make([]byte, 0, len(content)+len(importLine))
	result = append(result, content[:idx]...)
	result = append(result, []byte(importLine+"\n")...)
	result = append(result, content[idx:]...)

	return result
}

// addUCToRegisterRoutes inserts the use case param into each RegisterRoutes call.
// Pattern: finds "RegisterRoutes(" and adds the UC param after the last existing UC param.
func addUCToRegisterRoutes(content []byte, ucParam string) []byte {
	// Find all occurrences of "RegisterRoutes(" and add the UC param
	// This is a simple approach: find the pattern and add before "rbac"
	return bytes.ReplaceAll(content, []byte("rbac,"), []byte(ucParam+"rbac,"))
}

// UnwireModule removes a module's wiring from all DI files.
func UnwireModule(projectDir string, vars template.ModuleVars) error {
	// Remove from service.go
	if err := unwireService(projectDir, vars); err != nil {
		return err
	}
	// Remove from router.go
	if err := unwireRouter(projectDir, vars); err != nil {
		return err
	}
	// Remove from adapter routers
	for _, engine := range []string{"echo", "gin", "mux"} {
		if err := unwireAdapterRouter(projectDir, engine, vars); err != nil {
			return err
		}
	}
	return nil
}

func unwireService(projectDir string, vars template.ModuleVars) error {
	path := projectDir + "/cmd/http/service.go"
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	content = bytes.ReplaceAll(content, []byte(fmt.Sprintf("\tc.Provide(new%sRepository)\n", vars.ModuleName)), nil)
	content = bytes.ReplaceAll(content, []byte(fmt.Sprintf("\tc.Provide(new%sUseCase)\n", vars.ModuleName)), nil)
	content = removeFunction(content, fmt.Sprintf("new%sRepository", vars.ModuleName))
	content = removeFunction(content, fmt.Sprintf("new%sUseCase", vars.ModuleName))

	// Remove imports
	importAlias := vars.ModuleNameLower + "Outbound"
	importPath := vars.ModulePath + "/adapter/outbound/" + vars.ModuleNameLower
	content = bytes.ReplaceAll(content, []byte(fmt.Sprintf("\n\t%s \"%s\"", importAlias, importPath)), nil)
	inboundImport := vars.ModulePath + "/port/inbound/" + vars.ModuleNameLower
	content = bytes.ReplaceAll(content, []byte(fmt.Sprintf("\n\t\"%s\"", inboundImport)), nil)

	return os.WriteFile(path, content, 0o644)
}

func unwireRouter(projectDir string, vars template.ModuleVars) error {
	path := projectDir + "/cmd/http/router.go"
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Remove use case parameter
	content = bytes.ReplaceAll(content, []byte(fmt.Sprintf("\t%sUC %s.UseCase,\n", vars.ModuleNameLower, vars.ModuleNameLower)), nil)

	// Remove from RegisterRoutes calls
	content = bytes.ReplaceAll(content, []byte(fmt.Sprintf("%sUC, ", vars.ModuleNameLower)), nil)

	// Remove import
	inboundImport := vars.ModulePath + "/port/inbound/" + vars.ModuleNameLower
	content = bytes.ReplaceAll(content, []byte(fmt.Sprintf("\n\t\"%s\"", inboundImport)), nil)

	return os.WriteFile(path, content, 0o644)
}

func unwireAdapterRouter(projectDir, engine string, vars template.ModuleVars) error {
	path := projectDir + "/adapter/inbound/" + engine + "/router.go"
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Remove use case parameter
	content = bytes.ReplaceAll(content, []byte(fmt.Sprintf("\t%sUC %s.UseCase,\n", vars.ModuleNameLower, vars.ModuleNameLower)), nil)

	// Remove registerRoutes call
	content = bytes.ReplaceAll(content, []byte(fmt.Sprintf("\tregister%sRoutes(protected, %sUC, rbac)\n", vars.ModuleName, vars.ModuleNameLower)), nil)

	// Remove import
	inboundImport := vars.ModulePath + "/port/inbound/" + vars.ModuleNameLower
	content = bytes.ReplaceAll(content, []byte(fmt.Sprintf("\n\t\"%s\"", inboundImport)), nil)

	return os.WriteFile(path, content, 0o644)
}

// removeFunction removes a top-level function by name from Go source.
func removeFunction(content []byte, funcName string) []byte {
	// Find "func <name>" and remove until the matching closing brace
	start := bytes.Index(content, []byte("func "+funcName))
	if start == -1 {
		return content
	}

	// Find the opening brace
	braceStart := bytes.Index(content[start:], []byte("{"))
	if braceStart == -1 {
		return content
	}
	braceStart += start

	// Count braces to find the matching close
	depth := 0
	i := braceStart
	for i < len(content) {
		if content[i] == '{' {
			depth++
		} else if content[i] == '}' {
			depth--
			if depth == 0 {
				// Include the trailing newline
				end := i + 1
				if end < len(content) && content[end] == '\n' {
					end++
				}
				// Also remove leading newline before the function
				if start > 0 && content[start-1] == '\n' {
					start--
				}
				return append(content[:start], content[end:]...)
			}
		}
		i++
	}

	return content
}

// InjectMarkers adds kyan markers to a Go source file for reliable insertion.
// This is called by kyan init to prepare the wiring files.
func InjectMarkers(content []byte) []byte {
	// Add markers to provideServices function
	content = injectMarkerAround(content, "provideServices", markerProviderStart, markerProviderEnd)
	// Add markers after provideServices for constructor functions
	content = injectMarkerAtEndOfFile(content, markerServiceStart, markerServiceEnd)
	// Add markers to newRouter function params
	content = injectMarkerAroundParams(content, markerParamStart, markerParamEnd)
	// Add markers for route registration calls
	content = injectMarkerAroundRegister(content, markerRegisterStart, markerRegisterEnd)

	return content
}

func injectMarkerAround(content []byte, funcName, startMarker, endMarker string) []byte {
	// Find all c.Provide lines within the function and add markers around them
	lines := strings.Split(string(content), "\n")
	var result []string
	inFunc := false
	firstProvide := true
	lastProvide := -1

	for i, line := range lines {
		if strings.Contains(line, "func "+funcName) {
			inFunc = true
		}
		if inFunc && strings.Contains(line, "c.Provide(") {
			if firstProvide {
				result = append(result, "\t"+startMarker)
				firstProvide = false
			}
			lastProvide = i
		}
		result = append(result, line)

		if inFunc && strings.Contains(line, "}") && len(strings.TrimSpace(line)) == 1 {
			if lastProvide > 0 && !firstProvide {
				// Insert end marker before the closing brace
				result = append(result[:len(result)-1], "\t"+endMarker, line)
			}
			inFunc = false
		}
	}

	return []byte(strings.Join(result, "\n"))
}

func injectMarkerAtEndOfFile(content []byte, startMarker, endMarker string) []byte {
	return append(content, []byte("\n\n"+startMarker+"\n"+endMarker+"\n")...)
}

func injectMarkerAroundParams(content []byte, startMarker, endMarker string) []byte {
	lines := strings.Split(string(content), "\n")
	var result []string
	inNewRouter := false
	firstParam := true

	for _, line := range lines {
		result = append(result, line)
		if strings.Contains(line, "func newRouter(") {
			inNewRouter = true
			continue
		}
		if inNewRouter {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" && !strings.HasPrefix(trimmed, "//") && trimmed != "{" {
				if firstParam {
					// Insert start marker before the first parameter
					result = append(result[:len(result)-1], "\t"+startMarker, line)
					firstParam = false
				}
			}
			if strings.Contains(line, "{") {
				// Insert end marker before the opening brace
				result = append(result[:len(result)-1], "\t"+endMarker, line)
				inNewRouter = false
			}
		}
	}

	return []byte(strings.Join(result, "\n"))
}

func injectMarkerAroundRegister(content []byte, startMarker, endMarker string) []byte {
	// Simple approach: find "registerTodoRoutes" or similar lines in adapter routers
	// and add markers around them
	lines := strings.Split(string(content), "\n")
	var result []string
	inProtected := false
	firstRegister := true

	for _, line := range lines {
		result = append(result, line)

		if strings.Contains(line, "protected :=") || strings.Contains(line, "protected=") {
			inProtected = true
			continue
		}

		if inProtected && strings.Contains(line, "register") && strings.Contains(line, "Routes") {
			if firstRegister {
				result = append(result[:len(result)-1], "\t"+startMarker, line)
				firstRegister = false
			}
			// Add end marker after this line
			result = append(result, "\t"+endMarker)
			inProtected = false
		}
	}

	return []byte(strings.Join(result, "\n"))
}