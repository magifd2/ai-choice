# AGENTS.md — Development Rules for ai-choice

This file documents the conventions and rules that govern development of this project.
AI coding agents and human contributors must follow these rules when making changes.

---

## Project Overview

**ai-choice** is a CLI tool that classifies free-form natural language input into a predefined tag using an OpenAI-compatible LLM. It reads from stdin and writes a single tag to stdout.

- **Language:** Go 1.22+
- **Module:** `github.com/magifd2/ai-choice`
- **License:** MIT — Copyright (c) 2026 magifd2

---

## Core Principles

1. **Security first.** Every design decision starts with security. Do not compromise on this.
2. **Keep code small and focused.** Small functions, small files, small changesets.
3. **Fix small.** When something breaks, fix the narrowest possible scope.
4. **Tests and code ship together.** Never add or change logic without updating tests.
5. **Docs and code ship together.** Never add or change user-facing behavior without updating docs.

---

## Directory Structure

```
ai-choice/
├── main.go                          # Entry point only — wiring, flag parsing, I/O
├── internal/
│   ├── config/                      # Config loading and validation
│   │   ├── config.go
│   │   └── config_test.go
│   ├── llm/                         # LLM HTTP client and prompt construction
│   │   ├── client.go
│   │   ├── client_test.go
│   │   ├── prompt.go
│   │   └── prompt_test.go
│   └── classifier/                  # Core classification logic
│       ├── classifier.go
│       ├── classifier_test.go
│       └── classifier_integration_test.go  # live LLM tests (-tags integration)
├── system.yaml.example              # System config template
├── choices.yaml.example             # Choices config template
├── Makefile
├── CHANGELOG.md
├── README.md                        # English (primary)
└── README.ja.md                     # Japanese translation
```

`main.go` must stay thin — only flag parsing, stdin reading, config loading, client construction, and calling `classifier.Classify`. Business logic belongs in `internal/`.

---

## Configuration

Configuration is intentionally split into two files:

| File | Contents | Version-control |
|---|---|---|
| `system.yaml` | LLM API endpoint, api_key, model, timeouts | **Do not commit** (contains secrets) |
| `choices.yaml` | Classification choices (tag + description) | Safe to commit |

`system.yaml` is listed in `.gitignore`. Never add it to version control.

The `api_key` field supports `$ENV_VAR` expansion. Prefer environment variables over hardcoded keys.

---

## Security Rules

These rules must never be relaxed:

### Prompt injection protection
- **Nonce XML wrapping**: User input must always be wrapped in `<user_input_<hex_nonce>>…</user_input_<hex_nonce>>`. The nonce is generated via `crypto/rand` on every call. This is implemented in `llm.WrapUserInput`.
- **Role separation**: Classification instructions go in the `system` role. User input goes in the `user` role only. Never mix them.
- **Explicit warning in system prompt**: The system prompt must instruct the LLM to ignore any instructions found within the user input. This is implemented in `llm.BuildSystemPrompt`.

### Structured output
- Tool calling (`select_choice`) is the primary output mechanism. This prevents the model from producing arbitrary text.
- The tool's `tag` parameter must use an `enum` restricted to the configured choice tags.
- `tool_choice: "required"` enforces that the model calls the tool. Use the string form `"required"`, not the object form, for compatibility with local LLMs.

### Input handling
- Never pass raw user input directly to the LLM — always wrap it.
- Stdin is capped at **128 KB** (`maxInputBytes` in `main.go`) via `io.LimitReader`. This prevents memory exhaustion (DoS) and LLM context window overflow. Do not raise this limit without good reason.
- Validate all config fields on load; fail fast with a clear error message.

---

## LLM Compatibility

The LLM client targets OpenAI-compatible endpoints. When adding features:

- Use `tool_choice: "required"` (string) not `{"type": "function", ...}` (object) — local LLM servers (e.g. LM Studio) may reject the object form.
- Output parsing must be resilient. Implement the three-tier fallback in this order:
  1. Tool call result (`choices[0].message.tool_calls`)
  2. JSON extraction from content (`{"tag": "..."}`)
  3. Verbatim tag scan in content text
  4. Last configured choice as default
- Retry on HTTP 429 and 5xx with exponential backoff (`2^(attempt-1)` seconds). Do not retry on 4xx (except 429).
- Respect `context.Context` cancellation at every blocking point.

---

## Testing

### Unit tests (`go test ./...`)
- Every package must have unit tests.
- Tests use mocks/`httptest` — no real network calls.
- The `capturingClient` pattern in `classifier_test.go` is the standard way to assert on what is sent to the LLM. Use it when testing request structure.
- Test coverage must include:
  - Happy path
  - Each fallback tier independently
  - Edge cases (malformed input, empty responses, invalid tags)
  - Error propagation

### Integration tests (`go test -tags integration ./...`)
- Live LLM tests live in files with `//go:build integration`.
- They read from `system.yaml` / `choices.yaml` in the project root, or fall back to env vars (`AI_CHOICE_ENDPOINT`, `AI_CHOICE_API_KEY`, `AI_CHOICE_MODEL`), and call `t.Skip` if neither is available.
- Integration tests must cover: all configured categories, both Japanese and English input, prompt injection resistance patterns, and edge-case inputs.

### Running tests
```bash
make test                                                          # unit tests
go test -tags integration -v -timeout 120s ./internal/classifier/ # integration tests
```

---

## Code Style

- Follow standard Go conventions (`gofmt`, `go vet`).
- No external dependencies beyond `gopkg.in/yaml.v3`. Keep the dependency footprint minimal.
- Exported types and functions must have doc comments.
- Unexported helpers do not need comments unless the logic is non-obvious.
- Do not add error handling, fallbacks, or abstractions for hypothetical future requirements. Build for what is needed now.
- Avoid `interface{}` / `any` in public APIs except where JSON interop requires it.

---

## Build and Release

```bash
make build    # build for current platform, version from Git tag
make test     # unit tests with race detector
make lint     # go vet
make release  # cross-compile + ZIP for all platforms → dist/
make clean    # remove build artifacts
```

### Version embedding
- The version string is embedded at build time via `-ldflags "-X main.version=<version>"`.
- The version comes from `git describe --tags --always --dirty`.
- **Always create a Git tag before cutting a release.**
- Tag format: `vMAJOR.MINOR.PATCH` (e.g. `v1.0.0`).

### Supported platforms (cross-compile)
- `darwin/amd64`, `darwin/arm64`
- `linux/amd64`, `linux/arm64`
- `windows/amd64`

---

## Documentation

- `README.md` is the **English primary** documentation. This is the authoritative source.
- `README.ja.md` is the **Japanese translation**. Keep it in sync with `README.md` on every user-facing change.
- Both files must be updated whenever CLI flags, config fields, or behavior change.
- The example config files (`system.yaml.example`, `choices.yaml.example`) must stay in sync with the actual config structs.

---

## Changelog

- Follow [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) format.
- Every user-facing change (new feature, bug fix, breaking change, deprecation) must have a `CHANGELOG.md` entry.
- Add entries to `[Unreleased]` during development; move them to a versioned section when tagging a release.
- Use these categories: `Added`, `Changed`, `Deprecated`, `Removed`, `Fixed`, `Security`.

---

## Git Conventions

- Commit message format: `<type>: <short description>` (Conventional Commits style).
  - Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`, `security`
  - Example: `fix: use "required" string for tool_choice instead of object form`
- Keep commits small and focused on one concern.
- `system.yaml` must never be committed (it is in `.gitignore`).
- Tag releases with `git tag vX.Y.Z` before pushing to trigger version embedding.
- Remote: `git@github.com:magifd2/ai-choice.git`
