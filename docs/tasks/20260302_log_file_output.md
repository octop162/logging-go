# ログファイル出力対応

## Context

現在 `main.go` の `collect()` は `json.NewEncoder(os.Stdout)` で標準出力にJSONLを書き出している。
これをファイル出力に変更し、ファイル名に日付を含める。日付が変わったら新しいファイルに切り替える。

## 方針

- ログファイル名: `YYYY-MM-DD.jsonl`（例: `2026-03-02.jsonl`）
- 出力先ディレクトリは `--logdir` フラグで指定（デフォルト: 実行ファイルと同階層の `logs/`）
- 日付が変わったタイミングで自動的に新ファイルへローテーション
- ファイルは追記モード（`os.O_APPEND`）で開く
- 標準出力への出力は廃止（ファイルのみ）

## チェックリスト

- [x] `config.toml` と `config.go` に `log_dir` 追加
  - `Config` 構造体に `LogDir string` フィールド追加
  - `config.toml` に `log_dir = "logs"` を追加
- [x] `main.go` を修正
  - `--logdir` フラグ追加（設定ファイルより優先）
  - ポーリングループ内で日付チェック → 日付変更時にファイルを閉じて新ファイルを開く
  - ログディレクトリが存在しなければ `os.MkdirAll` で作成
- [x] CLAUDE.md の使い方セクション更新
  - `--logdir` フラグと出力先の説明を追記
- [x] テスト・lint
  - `gofmt -w .`
  - `go test ./...`
  - `golangci-lint run`
  - `go build -o logging.exe .`

## 修正対象ファイル

| ファイル | 操作 |
|---|---|
| `main.go` | ファイル出力ロジック追加 |
| `internal/config/config.go` | `LogDir` フィールド追加 |
| `config.toml` | `log_dir` 追加 |
| `CLAUDE.md` | 使い方セクション更新 |

## 検証

1. `go build -o logging.exe . && ./logging.exe --interval 5` で `logs/2026-03-02.jsonl` にJSONLが追記されること
2. `golangci-lint run` がクリーン
3. `go test ./...` が通ること
