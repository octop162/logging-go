# logging-rs

Windows PC活動ロガー。アクティブウィンドウ・ブラウザURL・バックグラウンドプロセスを記録する。

## 必要環境

- Go 1.26+
- Node.js v18+（Huskyのgitフック用）
- [golangci-lint](https://golangci-lint.run/usage/install/)

## セットアップ（clone後）

```bash
# Go依存パッケージの取得
go mod download

# Huskyのセットアップ（git pre-commitフック）
npm install
npx husky init
```

### Huskyとは

`npm install` + `npx husky init` を実行すると、`git commit` のたびに以下が自動チェックされる。

| チェック | 内容 |
|---|---|
| `gofmt -l .` | フォーマット未適用ファイルがあればコミットをブロック |
| `golangci-lint run` | lintエラーがあればコミットをブロック |

フックの内容は `.husky/pre-commit` に定義されている。

## ビルド・実行

```bash
# ビルド
go build -o logging.exe .

# デフォルト設定で起動（interval: 60秒）
./logging.exe

# ポーリング間隔を10秒に変更
./logging.exe --interval 10

# カスタム設定ファイルを指定
./logging.exe --config /path/to/config.toml

# ログ出力ディレクトリを指定
./logging.exe --logdir /path/to/logs
```

### コマンドラインフラグ

| フラグ | 説明 | デフォルト |
|---|---|---|
| `--interval` | ポーリング間隔（秒）。設定ファイルより優先 | 60（設定ファイル未指定時） |
| `--config` | 設定ファイルのパス | 実行ファイルと同階層の `config.toml` |
| `--logdir` | ログ出力ディレクトリ。設定ファイルより優先 | 実行ファイルと同階層の `logs` |

### 設定ファイル（config.toml）

```toml
# ポーリング間隔（秒）
interval = 60

# ログ出力ディレクトリ（デフォルト: 実行ファイルと同階層の logs）
log_dir = "logs"

# 除外するプロセス名（完全一致、大文字小文字無視）
exclude_processes = [
    "TextInputHost.exe",
    "Widgets.exe",
    "SystemSettings.exe",
]
```

### 出力形式

1行1JSONレコード（JSONL）で日付ごとのファイルに書き出す。ファイルは `<log_dir>/YYYY-MM-DD.jsonl` に追記される。日付が変わると自動的に新しいファイルにローテーションする。

出力例:

```json
{
  "timestamp": "2026-03-02T12:00:00.000000+09:00",
  "active_window": {
    "title": "GitHub - Google Chrome",
    "process_name": "chrome.exe",
    "url": "https://github.com"
  },
  "background_windows": [
    {"title": "Visual Studio Code", "process_name": "Code.exe"},
    {"title": "Windows Terminal", "process_name": "WindowsTerminal.exe"}
  ]
}
```

## テスト・Lint

```bash
go test ./...
golangci-lint run
```

## ブランチ戦略

| ブランチ | 用途 |
|---|---|
| `main` | デフォルトブランチ。常にビルド・テストが通る状態を維持する |
| `feature/*` | 新機能の開発用。`main` から分岐し、完了後にPRで `main` へマージ |
| `fix/*` | バグ修正用。`main` から分岐し、完了後にPRで `main` へマージ |

### ルール

- `main` への直接pushは避け、Pull Requestを経由する
- PRマージ前にCI（lint・テスト・ビルド）が全て通っていること
- タグ `v*`（例: `v0.1.0`）をpushすると、GitHub Actionsが自動でReleaseを作成し `logging-rs.zip`（`logging.exe` + `config.toml`）を添付する
