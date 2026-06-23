# AGENTS.md

Go 1.26+ API-key upstream gateway providing OpenAI/Gemini/Claude/OpenAI Responses compatible APIs with management APIs, protocol translation, retries, cooldowns, and round-robin/fill-first load balancing.

This project is deployed to production with Docker. Every committed code change must verify that the Docker build passes.

## Repository

- GitHub: https://github.com/router-for-me/CLIProxyAPI

## Docker Deployment (Production)

```bash
docker compose up -d --build
docker compose build
docker compose logs -f
docker compose restart
docker compose down
```

### Docker Configuration

| Environment variable | Purpose | Default |
|---|---|---|
| `CLI_PROXY_IMAGE` | Image tag | `eceasy/cli-proxy-api:latest` |
| `CLI_PROXY_CONFIG_PATH` | Config file mount path | `./config.yaml` |
| `CLI_PROXY_AUTH_PATH` | Legacy writable state mount path | `./auths` |
| `CLI_PROXY_LOG_PATH` | Log mount path | `./logs` |

### Exposed Ports

| Port | Purpose |
|---|---|
| `8317` | Main API service |
| `8085` | Management UI / management API |
| `1455` | Reserved |
| `54545` | Reserved |
| `51121` | Reserved |
| `11451` | Reserved |

### Production Notes

- `config.yaml` is mounted into the container at `/CLIProxyAPI/config.yaml`; restart the container after direct config edits.
- Upstream credentials are config-backed API keys.
- Logs are persisted in the mounted `logs/` directory.
- The default timezone is `Asia/Shanghai`; override it with `TZ` when needed.
- Dockerfile uses a multi-stage build (`golang:1.26-alpine` to `alpine:3.23`).

## Local Development

```bash
gofmt -w .                                           # Format after Go changes
go build -o cli-proxy-api ./cmd/server              # Build
go run ./cmd/server --config config.yaml            # Run dev server
go test ./...                                       # Run all tests
go test -v -run TestName ./path/to/pkg              # Run one test
go build -o test-output ./cmd/server && rm test-output
docker compose build                                # Required before committing
```

Common flags: `--config <path>`, `--tui`, `--standalone`, `--local-model`, `--vertex-import <path>`, `--vertex-import-prefix <prefix>`.

## Config

- Default config: `config.yaml`.
- `.env` is auto-loaded from the working directory.
- Supported config-backed upstreams: `gemini-api-key`, `vertex-api-key`, `anthropic`, `openai-responses`, and `openai-compatible` (legacy alias `openai-compatibility`).
- All config-backed upstreams support an `api-key-entries: [{api-key, proxy-url, priority}]` list for multi-key mode. The flat `api-key` field is kept as a backward-compat shim that expands to a single entry; both forms produce identical Auth IDs.
- Storage backends remain available for config/state persistence when enabled with `PGSTORE_*`, `GITSTORE_*`, or `OBJECTSTORE_*`.

## Architecture

- `cmd/server/` - Server entrypoint and CLI flags.
- `internal/api/` - Gin HTTP API, management routes, middleware, and protocol entrypoints.
- `internal/thinking/` - Thinking/reasoning pipeline. Preserve the canonical `ThinkingConfig` to provider-specific translation architecture.
- `internal/runtime/executor/` - Runtime executors for retained API-key upstream providers and their unit tests.
- `internal/translator/` - Provider protocol translators and shared translator helpers.
- `internal/registry/` - Model registry and remote updater (`StartModelsUpdater`); `--local-model` disables remote updates.
- `internal/store/` - Storage implementations and secret resolution.
- `internal/managementasset/` - Config snapshots and management assets.
- `internal/cache/` - Request signature caching.
- `internal/watcher/` - Config hot reload and API-key auth synthesis.
- `internal/usage/` - Usage and token accounting.
- `internal/tui/` - Bubbletea terminal UI (`--tui`, `--standalone`).
- `sdk/cliproxy/` - Embeddable SDK entrypoint, service builder, watcher wrapper, and runtime pipeline.
- `test/` - Cross-module integration tests.
- `Dockerfile` - Multi-stage production image build.
- `docker-compose.yml` - Production deployment orchestration with config/log mounts.

## Code Conventions

- Keep changes small and simple.
- Comments in English only.
- If editing code that already contains non-English comments, translate them to English.
- For user-visible strings, keep the existing language used in that file or area.
- New Markdown docs should be in English unless the file is explicitly language-specific.
- As a rule, do not make standalone changes to `internal/translator/`. If a task requires changing only `internal/translator/`, run `gh repo view --json viewerPermission -q .viewerPermission` and proceed only with `WRITE`, `MAINTAIN`, or `ADMIN`; otherwise file a GitHub issue and stop.
- `internal/runtime/executor/` should contain executors and their unit tests only. Place helper/supporting files under `internal/runtime/executor/helps/`.
- Follow `gofmt`; keep imports goimports-style.
- Wrap errors with context where helpful.
- Do not use `log.Fatal` or `log.Fatalf`; return errors and log through logrus.
- Avoid shadowed variables; use method suffixes such as `errStart := server.Start()`.
- Wrap defer errors: `defer func() { if err := f.Close(); err != nil { log.Errorf(...) } }()`.
- Use logrus structured logging and avoid leaking secrets or tokens.
- Avoid panics in HTTP handlers; prefer logged errors and meaningful HTTP status codes.
- Docker build verification is required before committing: `docker compose build`.
- Timeouts are allowed only during credential acquisition. After an upstream connection is established, do not set timeouts for subsequent network behavior. Intentional exceptions that must remain allowed are the OpenAI Responses websocket liveness deadlines in `internal/runtime/executor/codex_websockets_executor.go`, the management APICall timeout in `internal/api/handlers/management/api_tools.go`, and utility-only fetch command timeouts.
