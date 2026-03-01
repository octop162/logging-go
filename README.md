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

# 実行
./logging.exe
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
- タグ `v*`（例: `v0.1.0`）をpushすると、GitHub Actionsが自動でReleaseを作成し `logging.exe` を添付する
