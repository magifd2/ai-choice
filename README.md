# ai-choice

**ai-choice** is a CLI tool that reads free-form text from stdin and outputs the single best-matching tag from a predefined list — powered by an OpenAI-compatible LLM. Think of it as a smart `switch` statement for natural language.

---

## 概要 / Overview

| | |
|---|---|
| **入力** / Input | 自由記述テキスト (stdin) / Free-form text on stdin |
| **出力** / Output | 設定ファイルで定義したタグ (stdout) / A tag defined in config (stdout) |
| **仕組み** / How | LLMが最適なタグを選択 / LLM selects the best-matching tag |

セキュリティのため、ユーザー入力はランダムノンスのXMLタグで囲まれ、プロンプトインジェクションを防ぎます。
For security, user input is wrapped in a random-nonce XML tag to mitigate prompt injection attacks.

---

## インストール / Installation

### ダウンロード / Download from Releases

[Releases](https://github.com/magifd2/ai-choice/releases) ページから、ご使用のOS/アーキテクチャに合ったZIPファイルをダウンロードして展開してください。

Download the ZIP for your OS/architecture from the [Releases](https://github.com/magifd2/ai-choice/releases) page and extract it.

```bash
# Example (macOS ARM64)
curl -LO https://github.com/magifd2/ai-choice/releases/latest/download/ai-choice-<version>-darwin-arm64.zip
unzip ai-choice-<version>-darwin-arm64.zip
chmod +x ai-choice-<version>-darwin-arm64
sudo mv ai-choice-<version>-darwin-arm64 /usr/local/bin/ai-choice
```

### ソースからビルド / Build from Source

```bash
git clone https://github.com/magifd2/ai-choice.git
cd ai-choice
make build
# ./ai-choice is now ready
```

**必要条件 / Requirements:** Go 1.22+

---

## 設定ファイル / Configuration

設定は2つのファイルに分かれています。
Configuration is split into two files:

| ファイル / File | 役割 / Purpose |
|---|---|
| `system.yaml` | LLM APIの接続設定 (秘密情報を含む) / LLM API settings (contains secrets) |
| `choices.yaml` | 分類の選択肢定義 (バージョン管理可) / Classification choices (safe to version-control) |

```bash
cp system.yaml.example system.yaml   # APIキーを設定 / fill in API key
cp choices.yaml.example choices.yaml # 用途に合わせて編集 / customize for your use case
```

### system.yaml

```yaml
endpoint: "https://api.openai.com/v1"

# APIキーを直接書くか、環境変数を参照 ($ プレフィックス)
# Set the API key directly, or reference an environment variable with "$"
api_key: "$OPENAI_API_KEY"

model: "gpt-4o-mini"
timeout_seconds: 30
max_retries: 3
```

| Field | Required | Default | Description |
|---|---|---|---|
| `endpoint` | Yes | — | OpenAI-compatible API base URL |
| `api_key` | Yes | — | API key or `$ENV_VAR` reference |
| `model` | Yes | — | Model identifier |
| `timeout_seconds` | No | `30` | Per-request HTTP timeout (seconds) |
| `max_retries` | No | `3` | Max retry attempts on transient errors |

### choices.yaml

```yaml
choices:
  - tag: "weather"
    description: "天気予報や気象情報に関する質問や話題"
  - tag: "time"
    description: "現在時刻や時間の経過に関する質問や話題"
  - tag: "fortune"
    description: "占いや運勢、星座に関する質問や話題"
  - tag: "default"
    description: "上記のいずれにも当てはまらない質問や話題"
```

| Field | Required | Description |
|---|---|---|
| `tag` | Yes | Value output to stdout when this choice is selected |
| `description` | Yes | Natural language description shown to the LLM for matching |

---

## 使い方 / Usage

```bash
# 基本的な使い方 / Basic usage (system.yaml + choices.yaml in current dir)
echo "今日の天気は？" | ai-choice

# 設定ファイルのパスを明示 / Specify config file paths explicitly
echo "What time is it?" | ai-choice -system /path/to/system.yaml -choices /path/to/choices.yaml

# バージョン確認 / Print version
ai-choice -version
```

### 出力例 / Example Output

```
$ echo "明日の東京の天気を教えて" | ai-choice
weather

$ echo "今何時？" | ai-choice
time

$ echo "今日の運勢は？" | ai-choice
fortune

$ echo "おすすめのレシピを教えて" | ai-choice
default
```

### シェルスクリプトでの活用 / Use in Shell Scripts

```bash
#!/bin/bash
INPUT="今日の天気は雨ですか？"
TAG=$(echo "$INPUT" | ai-choice)

case "$TAG" in
  weather)  ./handle_weather.sh "$INPUT" ;;
  time)     ./handle_time.sh "$INPUT" ;;
  fortune)  ./handle_fortune.sh "$INPUT" ;;
  *)        ./handle_default.sh "$INPUT" ;;
esac
```

### 終了コード / Exit Codes

| Code | Meaning |
|---|---|
| `0` | Success — tag printed to stdout |
| `1` | Error — details printed to stderr |

---

## ビルド / Building

```bash
# 現在のプラットフォーム向けビルド / Build for current platform
make build

# テスト実行 / Run tests
make test

# 静的解析 / Run linter (go vet)
make lint

# クロスコンパイル & ZIPリリース / Cross-compile and package ZIPs
make release
# → dist/ に各プラットフォーム用ZIPが生成される
```

### 対応プラットフォーム / Supported Platforms

| OS | Arch |
|---|---|
| macOS | amd64, arm64 |
| Linux | amd64, arm64 |
| Windows | amd64 |

---

## セキュリティ / Security

- ユーザー入力は`crypto/rand`で生成したノンスXMLタグで囲まれ、プロンプトインジェクション攻撃を緩和します。
  User input is wrapped in a nonce XML tag (generated via `crypto/rand`) to mitigate prompt injection.
- システムプロンプトでLLMに対し、ユーザー入力内の指示は無視するよう明示しています。
  The system prompt explicitly instructs the LLM to ignore any instructions embedded in user input.
- APIキーは環境変数経由での管理を推奨しています (`api_key: "$OPENAI_API_KEY"`)。
  Storing the API key in an environment variable is recommended.
- ツールコーリングを使い構造化出力を強制し、任意テキスト出力のリスクを低減しています。
  Tool calling is used to enforce structured output, reducing the risk of arbitrary text generation.

---

## ライセンス / License

MIT License — Copyright (c) 2026 magifd2

See [LICENSE](LICENSE) for details.
