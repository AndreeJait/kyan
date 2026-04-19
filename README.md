# Kyan

Kyan is a CLI tool for scaffolding and managing Go projects built with [go-hex-boilerplate](https://github.com/AndreeJait/go-hex-boilerplate) and [go-utility](https://github.com/AndreeJait/go-utility). It auto-generates hexagonal architecture layers and integrates with [Ollama](https://ollama.com) for AI-assisted command translation.

## Features

### `kyan init`

Creates a new project from the go-hex-boilerplate template:

```bash
kyan init <project-name>
```

- Clones the go-hex-boilerplate repository into a new directory
- Prompts whether to keep the **todo** template module — if declined, removes `domain/entity/todo.go`, `domain/error/todo.go`, `port/inbound/todo/`, `port/outbound/todo.go`, `usecase/todo.go`, `adapter/outbound/todo/`, `adapter/inbound/*/todo.go`, and all todo-related wiring/routes
- Prompts whether to keep the **auth** template module — if declined, removes `domain/entity/user.go`, `domain/error/auth.go`, `port/inbound/auth/`, `port/outbound/user.go`, `usecase/auth.go`, `adapter/outbound/user/`, `adapter/inbound/*/auth.go`, and all auth-related wiring/routes
- Rewrites `go.mod` to use the new module path
- Cleans up vendor and re-vendors dependencies

### `kyan generate` (or `kyan g`)

Auto-generates a complete feature module following the hexagonal architecture:

```bash
kyan generate module <name> [flags]
```

Generates all layers for a new feature (e.g., `product`):

| Layer | File | Purpose |
|-------|------|---------|
| Domain entity | `domain/entity/<name>.go` | Struct definition with GORM tags |
| Domain errors | `domain/error/<name>.go` | `<Name>NotFound`, etc. using `statusw` |
| Outbound port | `port/outbound/<name>.go` | `<Name>Repository` interface |
| Inbound port | `port/inbound/<name>/usecase.go` | `<Name>.UseCase` interface |
| Inbound port input | `port/inbound/<name>/input.go` | `Create<Name>Input`, `Update<Name>Input` |
| Use case | `usecase/<name>.go` | Implements the `UseCase` interface |
| Outbound adapter | `adapter/outbound/<name>/repository.go` | GORM-backed repository |
| Inbound adapters | `adapter/inbound/echo/<name>.go`, `adapter/inbound/gin/<name>.go`, `adapter/inbound/mux/<name>.go` | HTTP handlers for all three engines |
| Migration | `files/migrations/<N>_create_<name>s.up.sql` and `.down.sql` | Database schema |

Flags:
- `--fields` — entity fields (e.g., `--fields="title:string description:text completed:bool"`)
- `--crud` — generate full CRUD methods (default: true)
- `--caching` — wrap repository with Redis caching decorator (default: false)
- `--auth` — add RBAC permission checks on routes (default: false)

After generation, kyan updates the DI wiring in `cmd/http/service.go` and `cmd/http/router.go` to register the new module.

### `kyan ai`

AI-assisted command execution powered by Ollama:

```bash
kyan ai "create a product module with title and price fields"
```

- Translates natural language into kyan CLI commands using a local Ollama model
- Choose the model via `--model` flag or configuration (default: `llama3`)
- Executes the generated command (e.g., `kyan generate module product --fields="title:string price:float"`)
- Supports `--dry-run` to preview the generated command without executing

### `kyan ask`

Ask questions about your project, architecture, or Go patterns — answers are grounded in the hexagonal conventions from go-hex-boilerplate and go-utility:

```bash
kyan ask "how should I add a caching decorator to a new repository?"
kyan ask "what's the difference between statusw.Error and domain/error?"
kyan ask "where do I register a new outbound port in the DI container?"
```

- Sends the question to Ollama along with project context (go-hex-boilerplate conventions, go-utility API references, and your project's existing domain/port/usecase structure)
- Returns a contextual answer — no files are written, no commands are executed
- Use `--context` to include specific files or packages in the prompt (e.g., `--context=port/outbound/product.go`)
- Use `--verbose` to include the full Ollama response with reasoning traces

### `kyan code`

AI-powered code generation constrained by your project's hexagonal architecture. Unlike `kyan generate` (which uses fixed templates), `kyan code` produces freeform Go source code via Ollama:

```bash
# Generate a custom use case method
kyan code --layer=usecase --module=product --method=SoftDelete

# Generate an outbound adapter for MongoDB instead of GORM
kyan code --layer=adapter/outbound --module=product --impl=mongow

# Generate a broker consumer for a new event
kyan code --layer=adapter/outbound --module=order --impl=kafkaw --method=ConsumeOrderCreated

# Generate a custom middleware
kyan code --layer=adapter/inbound/echo --name=ratelimit
```

- Sends the request to Ollama with the project's conventions, existing code structure, and go-utility API docs as context
- Generates Go source code that follows the hexagonal architecture: correct package names, import paths, interface satisfaction, constructor pattern, `statusw` errors, `responsew` responses
- Writes the generated file to the correct path and updates DI wiring in `cmd/http/service.go` / `cmd/http/router.go` when applicable
- Supports `--dry-run` to preview the generated code without writing files
- Supports `--diff` to show a diff against existing files before applying changes

How `kyan code` differs from `kyan generate`:

| | `kyan generate` | `kyan code` |
|---|---|---|
| Source | Built-in Go templates | Ollama model output |
| Scope | Full CRUD module (all layers) | Specific layer, method, or adapter |
| Customization | Flags (`--fields`, `--caching`, `--auth`) | Freeform via `--impl`, `--method`, natural language |
| Best for | Scaffolding a new feature from scratch | Adding custom logic, non-standard adapters, one-off methods |

### AI Configuration

All AI-powered commands (`ai`, `ask`, `code`) share the same Ollama configuration:

```bash
# Set default model
kyan config set ai.model codellama

# Set Ollama host (default: http://localhost:11434)
kyan config set ai.host http://localhost:11434
```

## Architecture

Kyan itself follows the hexagonal architecture from go-hex-boilerplate. The generation templates encode all conventions from the boilerplate:

- **Inward dependency rule**: adapters → ports → domain (never the reverse)
- **Port naming**: inbound ports at `port/inbound/<feature>/`, outbound ports at `port/outbound/<feature>.go`
- **Constructor pattern**: adapters return the port interface type, not the concrete struct
- **Decorator pattern**: caching repositories wrap the base implementation, both satisfying the same port interface
- **Error convention**: domain errors use `statusw` codes with `WithCustomMessage()` / `WithError()` chaining
- **Response convention**: handlers return `responsew.BaseResponse` via `responsew.Success()` / `responsew.SuccessPaginated()` / `responsew.Error()`
- **Multi-engine support**: each HTTP engine (echo, gin, mux) gets identical route structure in `adapter/inbound/<engine>/`
- **DI wiring**: `cmd/http/wire.go` orchestrates providers via uber-go/dig, organized by layer in `infra.go`, `service.go`, `router.go`
- **Config convention**: YAML with viper, three-tier override (env > `app.local.yaml` > `app.yaml`)

### Reference repositories

- **go-hex-boilerplate** — Hexagonal architecture template with todo CRUD and auth modules, multi-engine HTTP support (Echo, Gin, Gorilla Mux), and Redis caching
- **go-utility** — Companion utility library (v2) providing: `authw` (JWT/Basic/RBAC), `httpw/*` (Echo/Gin/Mux wrappers), `jwtw`, `statusw`, `responsew`, `logw`, `configw`, `gracefulw`, `sql/gormw`, `sql/sqlxw`, `sql/migratew`, `no-sql/redisw`, `no-sql/mongow`, `brokerw/*`, `storagew/*`, `botw/*`, `goroutinew`, `cronw`, `emailw`, `spanw`, `statemachinew`, and more

## Installation

```bash
go install github.com/AndreeJait/kyan@latest
```

## Contributing

PRs welcome. Please follow the hexagonal architecture conventions from go-hex-boilerplate.