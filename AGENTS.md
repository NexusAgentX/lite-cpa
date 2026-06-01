# AGENTS.md

Go 1.26+ proxy server providing OpenAI/Gemini/Claude/Codex compatible APIs with OAuth and round-robin load balancing.

**本项目已通过 Docker 部署到生产环境。所有代码改动提交后必须验证 Docker 构建通过。**

## Repository
- GitHub: https://github.com/router-for-me/CLIProxyAPI

## Docker Deployment (Production)

```bash
# 构建并启动（推荐）
docker compose up -d --build

# 仅构建镜像
docker compose build

# 查看日志
docker compose logs -f

# 重启服务
docker compose restart

# 停止服务
docker compose down
```

### Docker 关键配置

| 环境变量 | 用途 | 默认值 |
|---|---|---|
| `CLI_PROXY_IMAGE` | 镜像标签 | `eceasy/cli-proxy-api:latest` |
| `CLI_PROXY_CONFIG_PATH` | 配置文件挂载路径 | `./config.yaml` |
| `CLI_PROXY_AUTH_PATH` | OAuth 认证数据挂载路径 | `./auths` |
| `CLI_PROXY_LOG_PATH` | 日志挂载路径 | `./logs` |

### 暴露端口

| 端口 | 用途 |
|---|---|
| `8317` | 主 API 服务 |
| `8085` | 管理后台 / 转发 |
| `1455` | 预留 |
| `54545` | 预留 |
| `51121` | 预留 |
| `11451` | 预留 |

### 生产注意事项
- `config.yaml` 通过 Docker volume 挂载，修改后需 `docker compose restart`
- OAuth 认证数据持久化在 `auths/` 目录，通过 volume 挂载
- 日志持久化在 `logs/` 目录
- 时区默认 `Asia/Shanghai`，通过 `TZ` 环境变量配置
- Dockerfile 使用多阶段构建（`golang:1.26-alpine` → `alpine:3.23`），最终镜像约 20MB

## Local Development

```bash
gofmt -w .                    # Format (required after Go changes)
go build -o cli-proxy-api ./cmd/server              # Build
go run ./cmd/server                                  # Run dev server
go test ./...                                        # Run all tests
go test -v -run TestName ./path/to/pkg               # Run single test
go build -o test-output ./cmd/server && rm test-output  # Verify compile
```
- Common flags: `--config <path>`, `--tui`, `--standalone`, `--local-model`, `--no-browser`, `--oauth-callback-port <port>`

## Config
- Default config: `config.yaml` (挂载到容器内的 `/CLIProxyAPI/config.yaml`)
- `.env` is auto-loaded from the working directory
- Auth material defaults under `auths/` (挂载到容器内的 `/root/.cli-proxy-api`)
- Storage backends: file-based default; optional Postgres/git/object store (`PGSTORE_*`, `GITSTORE_*`, `OBJECTSTORE_*`)

## Architecture
- `cmd/server/` — Server entrypoint
- `internal/api/` — Gin HTTP API (routes, middleware, modules)
- `internal/api/modules/amp/` — Amp integration (Amp-style routes + reverse proxy)
- `internal/thinking/` — Main thinking/reasoning pipeline. `ApplyThinking()` (apply.go) parses suffixes (`suffix.go`, suffix overrides body), normalizes config to canonical `ThinkingConfig` (`types.go`), normalizes and validates centrally (`validate.go`/`convert.go`), then applies provider-specific output via `ProviderApplier`. Do not break this "canonical representation → per-provider translation" architecture.
- `internal/runtime/executor/` — Per-provider runtime executors (incl. Codex WebSocket)
- `internal/translator/` — Provider protocol translators (and shared `common`)
- `internal/registry/` — Model registry + remote updater (`StartModelsUpdater`); `--local-model` disables remote updates
- `internal/store/` — Storage implementations and secret resolution
- `internal/managementasset/` — Config snapshots and management assets
- `internal/cache/` — Request signature caching
- `internal/watcher/` — Config hot-reload and watchers
- `internal/wsrelay/` — WebSocket relay sessions
- `internal/usage/` — Usage and token accounting
- `internal/tui/` — Bubbletea terminal UI (`--tui`, `--standalone`)
- `sdk/cliproxy/` — Embeddable SDK entry (service/builder/watchers/pipeline)
- `test/` — Cross-module integration tests
- `Dockerfile` — 多阶段构建，最终基于 `alpine:3.23`，约 20MB
- `docker-compose.yml` — 生产部署编排，含 config/auths/logs 卷挂载

## Code Conventions
- Keep changes small and simple (KISS)
- Comments in English only
- If editing code that already contains non-English comments, translate them to English (don't add new non-English comments)
- For user-visible strings, keep the existing language used in that file/area
- New Markdown docs should be in English unless the file is explicitly language-specific (e.g. `README_CN.md`)
- As a rule, do not make standalone changes to `internal/translator/`. You may modify it only as part of broader changes elsewhere.
- If a task requires changing only `internal/translator/`, run `gh repo view --json viewerPermission -q .viewerPermission` to confirm you have `WRITE`, `MAINTAIN`, or `ADMIN`. If you do, you may proceed; otherwise, file a GitHub issue including the goal, rationale, and the intended implementation code, then stop further work.
- `internal/runtime/executor/` should contain executors and their unit tests only. Place any helper/supporting files under `internal/runtime/executor/helps/`.
- Follow `gofmt`; keep imports goimports-style; wrap errors with context where helpful
- Do not use `log.Fatal`/`log.Fatalf` (terminates the process); prefer returning errors and logging via logrus
- Shadowed variables: use method suffix (`errStart := server.Start()`)
- Wrap defer errors: `defer func() { if err := f.Close(); err != nil { log.Errorf(...) } }()`
- Use logrus structured logging; avoid leaking secrets/tokens in logs
- Avoid panics in HTTP handlers; prefer logged errors and meaningful HTTP status codes
- **Docker 构建验证：`docker compose build` 必须通过后才能提交代码**
- Timeouts are allowed only during credential acquisition; after an upstream connection is established, do not set timeouts for any subsequent network behavior. Intentional exceptions that must remain allowed are the Codex websocket liveness deadlines in `internal/runtime/executor/codex_websockets_executor.go`, the wsrelay session deadlines in `internal/wsrelay/session.go`, the management APICall timeout in `internal/api/handlers/management/api_tools.go`, and the `cmd/fetch_antigravity_models` utility timeouts
