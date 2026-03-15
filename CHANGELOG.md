# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

---

## [0.1.0] - 2026-03-15

### Added

- Initial release.
- Read free-form natural language from stdin; output the best-matching tag to stdout.
- Configuration split into two files:
  - `system.yaml` — LLM API connection settings (endpoint, api_key, model, timeouts).
  - `choices.yaml` — classification choices (tag + description pairs).
- OpenAI-compatible API client with configurable endpoint, supporting local LLMs (e.g. LM Studio).
- Tool calling (`select_choice`) as the primary structured-output mechanism.
- Three-tier fallback for unstable LLM output: tool call → JSON extraction → text scan → last choice.
- Exponential backoff retry on transient errors (HTTP 429, 5xx, network errors).
- Prompt injection protection:
  - User input wrapped in a `crypto/rand`-generated nonce XML tag.
  - Role separation (system / user).
  - System prompt explicitly instructs the LLM to ignore instructions in user input.
- Version string embedded at build time from Git tag via `-ldflags`.
- Cross-compilation targets: darwin/amd64, darwin/arm64, linux/amd64, linux/arm64, windows/amd64.
- `make release` produces versioned ZIP archives in `dist/`.
- Unit tests covering tool call edge cases, JSON/text fallback, request structure, and injection containment.
- Integration tests (`-tags integration`) for live LLM validation across all categories and injection patterns.
- English README (`README.md`) with Japanese translation (`README.ja.md`).
- `AGENTS.md` documenting development rules for AI coding agents and contributors.
- MIT License.

[Unreleased]: https://github.com/magifd2/ai-choice/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/magifd2/ai-choice/releases/tag/v0.1.0
