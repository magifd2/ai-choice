# ai-choice

**ai-choice** は、標準入力から受け取った自然言語テキストを、設定ファイルに定義された選択肢の中から最も適合するタグに分類して標準出力に出力するCLIツールです。LLMを使ったスマートな `switch` 文と考えてください。

```bash
echo "明日の天気はどうですか？" | ai-choice
# → weather
```

> **English documentation:** [README.md](README.md)

---

## 概要

| | |
|---|---|
| **入力** | 自由記述テキスト (stdin) |
| **出力** | `choices.yaml` で定義したタグ (stdout) |
| **仕組み** | LLMがツールコーリングで最適なタグを選択 |

ユーザー入力は毎回ランダムなノンスXMLタグで囲まれ、プロンプトインジェクション攻撃を緩和します。

---

## インストール

### リリースからダウンロード

[Releases](https://github.com/magifd2/ai-choice/releases) ページからご使用のOS/アーキテクチャに合ったZIPファイルをダウンロードして展開してください。

```bash
# 例: macOS ARM64
curl -LO https://github.com/magifd2/ai-choice/releases/latest/download/ai-choice-v0.1.0-darwin-arm64.zip
unzip ai-choice-v0.1.0-darwin-arm64.zip
chmod +x ai-choice-v0.1.0-darwin-arm64
sudo mv ai-choice-v0.1.0-darwin-arm64 /usr/local/bin/ai-choice
```

**対応プラットフォーム:**

| OS | Arch |
|---|---|
| macOS | amd64, arm64 |
| Linux | amd64, arm64 |
| Windows | amd64 |

### ソースからビルド

```bash
git clone https://github.com/magifd2/ai-choice.git
cd ai-choice
make build
# ./ai-choice が生成されます
```

**必要条件:** Go 1.22+

---

## 設定ファイル

設定は2つのファイルに分かれています。

| ファイル | 役割 |
|---|---|
| `system.yaml` | LLM API接続設定 — **非公開にしてください**（APIキーを含む） |
| `choices.yaml` | 分類の選択肢定義 — バージョン管理可 |

```bash
cp system.yaml.example system.yaml   # APIキーを設定
cp choices.yaml.example choices.yaml # 用途に合わせて編集
```

### system.yaml

```yaml
# OpenAI互換APIエンドポイント
endpoint: "https://api.openai.com/v1"

# APIキー — 直接書くか "$" プレフィックスで環境変数を参照
api_key: "$OPENAI_API_KEY"

model: "gpt-4o-mini"
timeout_seconds: 30   # デフォルト: 30
max_retries: 3        # デフォルト: 3
```

| フィールド | 必須 | デフォルト | 説明 |
|---|---|---|---|
| `endpoint` | Yes | — | OpenAI互換APIのベースURL |
| `api_key` | Yes | — | APIキーまたは `$ENV_VAR` 参照 |
| `model` | Yes | — | 使用するモデル識別子 |
| `timeout_seconds` | No | `30` | リクエストのHTTPタイムアウト（秒） |
| `max_retries` | No | `3` | 一時的エラー時の最大リトライ回数（429, 5xx） |

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

| フィールド | 必須 | 説明 |
|---|---|---|
| `tag` | Yes | この選択肢が選ばれたときに stdout に出力される値 |
| `description` | Yes | LLMに渡すマッチング用の自然言語説明文 |

---

## 使い方

```bash
# 基本的な使い方（カレントディレクトリの system.yaml と choices.yaml を使用）
echo "明日は雨が降りますか？" | ai-choice

# 設定ファイルのパスを明示
echo "今何時？" | ai-choice -system /path/to/system.yaml -choices /path/to/choices.yaml

# バージョン確認
ai-choice -version
```

### 出力例

```
$ echo "明日の東京の天気を教えて" | ai-choice
weather

$ echo "今何時ですか？" | ai-choice
time

$ echo "今日の運勢は？" | ai-choice
fortune

$ echo "おすすめのレシピを教えて" | ai-choice
default
```

### シェルスクリプトでの活用

```bash
#!/bin/bash
INPUT="明日の天気は？"
TAG=$(echo "$INPUT" | ai-choice)

case "$TAG" in
  weather)  ./handle_weather.sh "$INPUT" ;;
  time)     ./handle_time.sh "$INPUT" ;;
  fortune)  ./handle_fortune.sh "$INPUT" ;;
  *)        ./handle_default.sh "$INPUT" ;;
esac
```

### 終了コード

| コード | 意味 |
|---|---|
| `0` | 成功 — タグを stdout に出力 |
| `1` | エラー — 詳細を stderr に出力 |

---

## モデルベンチマーク

36問のテストケース（天気・時刻・占い・defaultカテゴリ、日本語・英語入力、プロンプトインジェクションパターン）で分類精度とレイテンシを計測。

**計測環境**

| | |
|---|---|
| バックエンド | LM Studio ローカルサーバー |
| ホスト | Mac Studio (Z17Z000QJJ/A) |
| チップ | Apple M2 Max — 12コア（パフォーマンス: 8、効率性: 4）|
| メモリ | 64 GB |

**結果**

| モデル | 正答率 | 定常レイテンシ | 合計 (36問) |
|---|---|---|---|
| llama-3.2-1b-instruct | 18/36 (50%) | ~0.22s | 10.1s |
| llama-3.2-3b-instruct | 28/36 (78%) | ~0.21s | 9.7s |
| qwen2.5-7b-instruct-mlx | 35/36 (97%) | ~0.48s | 19.0s |
| openai/gpt-oss-20b | **36/36 (100%)** | ~1.22s | 46.1s |

「定常レイテンシ」は初回リクエスト（モデルウォームアップ）を除いた値。

**推奨モデル**

- 速度優先 → **Qwen2.5-7B-Instruct**（0.48s/req、97%）
- 精度優先 → **gpt-oss-20B**（1.22s/req、100%）
- 1B・3Bモデルは日本語入力での精度が低く、実用には不向き。

---

## ビルド

```bash
make build    # 現在のプラットフォーム向けにビルド
make test     # ユニットテストを実行
make lint     # go vet を実行
make release  # クロスコンパイルして dist/ にZIPを生成
make clean    # ビルド成果物を削除
```

バージョン文字列はビルド時にGitタグから埋め込まれます:

```bash
git tag v1.0.0
make build
./ai-choice -version  # → ai-choice v1.0.0
```

タグが存在しない場合は `git describe --tags --always --dirty` の出力にフォールバックします。

---

## セキュリティ

- **ノンスXMLラッピング** — ユーザー入力は毎回 `<user_input_<ランダムhex>>…</user_input_<ランダムhex>>` で囲まれます。ノンスは `crypto/rand` で生成され、注入された指示がシステムレベルのコマンドとして解釈されることを防ぎます。
- **ロール分離** — 分類指示は `system` ロールに、ユーザー入力は `user` ロールに限定されます。
- **インジェクション警告** — システムプロンプトでLLMに対し、ユーザー入力内の指示は無視するよう明示しています。
- **ツールコーリング** — `select_choice` ツールを通じて構造化出力を強制し、LLMが任意のテキストを返すリスクを低減します。
- **APIキーの環境変数管理** — `api_key: "$OPENAI_API_KEY"` の形式で設定することで、設定ファイルへの秘密情報のハードコードを避けられます。

---

## ライセンス

MIT License — Copyright (c) 2026 magifd2

詳細は [LICENSE](LICENSE) を参照してください。
