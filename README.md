# CLI Proxy API

CLI Proxy API is an API-key upstream gateway for OpenAI, Gemini, Claude, OpenAI Responses, and OpenAI-compatible model providers. It exposes client-compatible API surfaces while routing requests to configured upstream API keys with load balancing, retry, cooldown, model aliasing, and protocol translation.

## Overview

- OpenAI-compatible endpoints: `/v1/models`, `/v1/chat/completions`, `/v1/completions`, and `/v1/responses`
- Claude-compatible endpoints: `/v1/messages` and `/v1/messages/count_tokens`
- Gemini-compatible endpoints: `/v1beta/models` and model action routes
- Config-backed upstream API keys for Gemini, Vertex Gemini, Anthropic, OpenAI Responses, and OpenAI-compatible providers
- Round-robin and fill-first load balancing strategies
- Retry and cooldown handling across configured upstream credentials
- Configurable model aliases, prefixes, priorities, and per-key excluded models
- Management APIs for config, API keys, request logs, providers, model definitions, routing, retry, and Vertex key import
- Bubbletea TUI for local configuration, API key management, and logs
- Docker deployment with persistent config and logs
- Reusable Go SDK for embedding the gateway

Personal account-pool sign-in flows, tool-specific reverse proxy routes, credential-file management, realtime relay providers, and tool-specific compatibility routes are not part of this gateway profile.

## Quick Start

1. Add upstream credentials to `config.yaml`.
2. Start the gateway:

```bash
go run ./cmd/server --config config.yaml
```

3. Send client requests to `http://127.0.0.1:8317` with one of the configured gateway API keys.

Example OpenAI-compatible request:

```bash
curl http://127.0.0.1:8317/v1/chat/completions \
  -H 'Authorization: Bearer sk-123456' \
  -H 'Content-Type: application/json' \
  -d '{
    "model": "gpt-5.5",
    "messages": [{"role": "user", "content": "hello"}]
  }'
```

## Configuration

The default config file is `config.yaml`. The gateway reads `.env` from the working directory when present.

Supported upstream sections include:

- `gemini-api-key`
- `vertex-api-key`
- `anthropic`
- `openai-responses`
- `openai-compatibility`

Each upstream entry can define its API key, base URL, models, aliases, prefixes, priority, and excluded models where supported by that provider type.

## Docker Deployment

```bash
docker compose up -d --build
docker compose build
docker compose logs -f
docker compose restart
docker compose down
```

Environment variables:

| Variable | Purpose | Default |
|---|---|---|
| `CLI_PROXY_IMAGE` | Image tag | `eceasy/cli-proxy-api:latest` |
| `CLI_PROXY_CONFIG_PATH` | Config file mount path | `./config.yaml` |
| `CLI_PROXY_AUTH_PATH` | Legacy writable state mount path | `./auths` |
| `CLI_PROXY_LOG_PATH` | Log mount path | `./logs` |

Ports:

| Port | Purpose |
|---|---|
| `8317` | Main API service |
| `8085` | Management UI and management API |
| `1455` | Reserved |
| `54545` | Reserved |
| `51121` | Reserved |
| `11451` | Reserved |

Production notes:

- `config.yaml` is mounted into the container at `/CLIProxyAPI/config.yaml`.
- Restart the container after editing mounted config directly.
- Logs are persisted under the mounted `logs/` directory.
- The default timezone is `Asia/Shanghai`; set `TZ` to override it.
- The Dockerfile uses a multi-stage Go build and an Alpine runtime image.

## Local Development

```bash
gofmt -w .
go build -o cli-proxy-api ./cmd/server
go run ./cmd/server --config config.yaml
go test ./...
go test -v -run TestName ./path/to/pkg
go build -o test-output ./cmd/server && rm test-output
docker compose build
```

Common flags:

- `--config <path>`
- `--tui`
- `--standalone`
- `--local-model`
- `--vertex-import <path>`
- `--vertex-import-prefix <prefix>`

## SDK

The `sdk/cliproxy` package exposes the embeddable service builder, watcher wrapper, access manager integration, and runtime execution pipeline used by the server.

## Contributing

Contributions are welcome.

1. Fork the repository.
2. Create a feature branch.
3. Run tests and `docker compose build`.
4. Open a pull request.

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE).
