# ai-choice

**ai-choice** is a CLI tool that reads free-form text from stdin and outputs the single best-matching tag from a predefined list — powered by an OpenAI-compatible LLM.

Think of it as a smart `switch` statement for natural language:

```bash
echo "What's the weather like tomorrow?" | ai-choice
# → weather
```

> **Japanese documentation:** [README.ja.md](README.ja.md)

---

## Overview

| | |
|---|---|
| **Input** | Free-form text on stdin |
| **Output** | A tag defined in `choices.yaml` (stdout) |
| **How** | LLM selects the best-matching tag via tool calling |

User input is wrapped in a random-nonce XML tag on every call to mitigate prompt injection attacks.

---

## Installation

### Download from Releases

Download the ZIP for your OS/architecture from the [Releases](https://github.com/magifd2/ai-choice/releases) page and extract it.

```bash
# Example: macOS ARM64
curl -LO https://github.com/magifd2/ai-choice/releases/latest/download/ai-choice-v0.1.0-darwin-arm64.zip
unzip ai-choice-v0.1.0-darwin-arm64.zip
chmod +x ai-choice-v0.1.0-darwin-arm64
sudo mv ai-choice-v0.1.0-darwin-arm64 /usr/local/bin/ai-choice
```

**Supported platforms:**

| OS | Arch |
|---|---|
| macOS | amd64, arm64 |
| Linux | amd64, arm64 |
| Windows | amd64 |

### Build from Source

```bash
git clone https://github.com/magifd2/ai-choice.git
cd ai-choice
make build
# ./ai-choice is ready
```

**Requirements:** Go 1.22+

---

## Configuration

Configuration is split into two files:

| File | Purpose |
|---|---|
| `system.yaml` | LLM API connection settings — **keep this private** (contains API key) |
| `choices.yaml` | Classification choices — safe to version-control |

```bash
cp system.yaml.example system.yaml   # fill in your API key
cp choices.yaml.example choices.yaml # customize for your use case
```

### system.yaml

```yaml
# OpenAI-compatible API endpoint
endpoint: "https://api.openai.com/v1"

# API key — set directly or reference an environment variable with "$"
api_key: "$OPENAI_API_KEY"

model: "gpt-4o-mini"
timeout_seconds: 30   # default: 30
max_retries: 3        # default: 3
```

| Field | Required | Default | Description |
|---|---|---|---|
| `endpoint` | Yes | — | Base URL of the OpenAI-compatible API |
| `api_key` | Yes | — | API key or `$ENV_VAR` reference |
| `model` | Yes | — | Model identifier |
| `timeout_seconds` | No | `30` | Per-request HTTP timeout (seconds) |
| `max_retries` | No | `3` | Max retry attempts on transient errors (429, 5xx) |

### choices.yaml

```yaml
choices:
  - tag: "weather"
    description: "Questions or topics about weather forecasts and meteorological information"
  - tag: "time"
    description: "Questions or topics about the current time or elapsed time"
  - tag: "fortune"
    description: "Questions or topics about fortune telling, horoscopes, or luck"
  - tag: "default"
    description: "Any topic not covered by the above choices"
```

| Field | Required | Description |
|---|---|---|
| `tag` | Yes | Value printed to stdout when this choice is selected |
| `description` | Yes | Natural language description shown to the LLM for matching |

---

## Usage

```bash
# Basic usage — reads system.yaml and choices.yaml from the current directory
echo "Will it rain tomorrow?" | ai-choice

# Specify config file paths explicitly
echo "What time is it?" | ai-choice -system /path/to/system.yaml -choices /path/to/choices.yaml

# Print version
ai-choice -version
```

### Example output

```
$ echo "What will the weather be like in Tokyo tomorrow?" | ai-choice
weather

$ echo "What time is it now?" | ai-choice
time

$ echo "What does my horoscope say today?" | ai-choice
fortune

$ echo "Recommend a pasta recipe" | ai-choice
default
```

### Use in shell scripts

```bash
#!/bin/bash
INPUT="Will it rain tomorrow?"
TAG=$(echo "$INPUT" | ai-choice)

case "$TAG" in
  weather)  ./handle_weather.sh "$INPUT" ;;
  time)     ./handle_time.sh "$INPUT" ;;
  fortune)  ./handle_fortune.sh "$INPUT" ;;
  *)        ./handle_default.sh "$INPUT" ;;
esac
```

### Exit codes

| Code | Meaning |
|---|---|
| `0` | Success — tag printed to stdout |
| `1` | Error — details printed to stderr |

---

## Model Benchmark

Classification accuracy and per-request latency measured against 36 test cases
(weather / time / fortune / default categories, Japanese and English input, prompt injection patterns).

**Test environment**

| | |
|---|---|
| Backend | LM Studio local server |
| Host | Mac Studio (Z17Z000QJJ/A) |
| Chip | Apple M2 Max — 12 cores (8P + 4E) |
| Memory | 64 GB |

**Results**

| Model | Accuracy | Steady-state latency | Total (36 req) |
|---|---|---|---|
| llama-3.2-1b-instruct | 18/36 (50%) | ~0.22s | 10.1s |
| llama-3.2-3b-instruct | 28/36 (78%) | ~0.21s | 9.7s |
| qwen2.5-7b-instruct-mlx | 35/36 (97%) | ~0.48s | 19.0s |
| openai/gpt-oss-20b | **36/36 (100%)** | ~1.22s | 46.1s |

"Steady-state latency" excludes the first request (model warm-up).

**Recommendations**

- Speed priority → **Qwen2.5-7B-Instruct** (0.48s/req, 97%)
- Accuracy priority → **gpt-oss-20B** (1.22s/req, 100%)
- The 1B/3B models are too inaccurate for production use with Japanese input.

---

## Building

```bash
make build    # build for the current platform
make test     # run unit tests
make lint     # run go vet
make release  # cross-compile and package ZIP archives in dist/
make clean    # remove build artifacts
```

The version string is embedded at build time from the latest Git tag:

```bash
git tag v1.0.0
make build
./ai-choice -version  # → ai-choice v1.0.0
```

If no tag exists, the version falls back to the output of `git describe --tags --always --dirty`.

---

## Security

- **Nonce XML wrapping** — User input is enclosed in `<user_input_<random_hex>>…</user_input_<random_hex>>` on every call, preventing injected instructions from being interpreted as system-level commands. The nonce is generated via `crypto/rand`.
- **Role separation** — Classification instructions live in the `system` role; user input is confined to the `user` role.
- **Explicit injection warning** — The system prompt instructs the LLM to ignore any instructions found inside the user input.
- **Tool calling** — Structured output is enforced via tool calling (`select_choice`), so the model cannot produce arbitrary text as a response.
- **API key via environment variable** — Storing the key as `api_key: "$OPENAI_API_KEY"` avoids hardcoding secrets in config files.

---

## License

MIT License — Copyright (c) 2026 magifd2

See [LICENSE](LICENSE) for details.
