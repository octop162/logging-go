# CLAUDE.md

このファイルはClaude Code（claude.ai/code）がこのリポジトリで作業する際のガイダンスを提供します。

## プロジェクト概要

Goで書かれたWindows PC活動ロガー。設定可能なポーリング間隔でアクティブウィンドウ、ブラウザURL、バックグラウンドプロセスを記録し、時間追跡分析のための構造化ログを書き出す。

## セットアップ（clone後）

```bash
# 1. Go依存パッケージの取得
go mod download

# 2. Huskyのセットアップ（git pre-commitフック）
npm install        # node_modules/ に husky をインストール
npx husky init     # .husky/_ を生成し core.hooksPath を設定
```

> **Node.js が必要**（v18以上推奨）。インストール済みか確認: `node -v`

### pre-commitで自動実行されるチェック

| チェック | 内容 |
|---|---|
| `gofmt -l .` | フォーマット未適用ファイルがあればブロック |
| `golangci-lint run` | lintエラーがあればブロック |

## コマンド

```bash
# ビルド
go build -o logging.exe .

# 実行
./logging.exe

# テスト実行
go test ./...

# 単一テスト実行
go test ./... -run TestFunctionName

# Lint（golangci-lint が必要）
golangci-lint run
```

## アーキテクチャ

WindowsのみをターゲットとしたGoアプリケーション。外部ランタイム不要な単一の `.exe` として動作するよう設計されている。

**コア監視機能:**

- **アクティブウィンドウ:** Win32 API（`golang.org/x/sys/windows`）を使用してフォアグラウンドウィンドウのアプリケーション名とタイトルを取得する。
- **Chrome URL:** アクティブウィンドウがChromeの場合、COM/UI Automation（`github.com/go-ole/go-ole`）を使用してアドレスバーから現在のページURLを取得する。
- **バックグラウンドウィンドウ:** `github.com/shirou/gopsutil/v3` を使用してプロセス名と共にすべての可視（最小化されていない）デスクトップウィンドウを列挙する。
- **ポーリングループ:** 設定可能な間隔（デフォルト: 1分）で上記3つのデータを収集し、日付別のJSONLログファイル（`<log_dir>/YYYY-MM-DD.jsonl`）に構造化レコードを書き出す。日付が変わると自動ローテーション。

**主要依存パッケージ:**

| パッケージ | 用途 |
|---|---|
| `golang.org/x/sys/windows` | Win32 API呼び出し（フォアグラウンドウィンドウ、ウィンドウ列挙） |
| `github.com/go-ole/go-ole` | Chrome URL取得のためのCOM / UI Automation |
| `github.com/shirou/gopsutil/v3` | プロセス名取得 |
| `github.com/BurntSushi/toml` | TOML設定ファイル読み込み |

**パフォーマンス目標:** アイドルポーリング中のCPU使用率を最小限に抑える。メモリは数十MB以下に維持する。バイナリはWindowsスタートアップ項目、タスクスケジューラ、またはWindowsサービスとして動作すること。

## 実装ルール

### タスクファイルに従った実施
- 該当タスクファイル（`docs/tasks/YYYYMMDD_taskname.md`）のチェックリストに従い、**上から順に**作業すること。
- 現在のステップが完了・検証されるまで、次のステップに進まないこと。

### コード品質（gofmt / golangci-lint）
- コードを書いたら**必ず** `gofmt -w .` を実行してフォーマットを整えること。
- 各ステップの完了条件として `golangci-lint run` を実行し、エラーが**0件**であることを確認すること。
- lintエラーは無視・抑制せず、コードを修正して解消すること（`//nolint` は最終手段）。

### チェックリストの管理
- タスクが完了したら、該当タスクファイルのチェックボックスを**即座に** `[ ]` から `[x]` に更新すること。
- ビルド通過・テスト通過などの完了条件が実際に満たされていない限り、タスクを完了とマークしないこと。

### 完了時の承認
- タスクのすべてのステップが完了したら、**必ずユーザーに報告し、承認を得てから**次の作業に進むこと。
- 報告内容: 完了したステップの一覧、完了条件の達成状況（ビルド・lint結果など）。

## ワークフロー設計
### 1. Planモードを基本とする
- 3ステップ以上 or アーキテクチャに関わるタスクは必ずPlanモードで開始する
- 途中でうまくいかなくなったら、無理に進めずすぐに立ち止まって再計画する
- 構築だけでなく、検証ステップにもPlanモードを使う
- 曖昧さを減らすため、実装前に詳細な仕様を書く

### 2. サブエージェント戦略
- メインのコンテキストウィンドウをクリーンに保つためにサブエージェントを積極的に活用する
- リサーチ・調査・並列分析はサブエージェントに任せる
- 複雑な問題には、サブエージェントを使ってより多くの計算リソースを投入する
- 集中して実行するために、サブエージェント1つにつき1タスクを割り当てる

### 3. 自己改善ループ
- ユーザーから修正を受けたら必ず `tasks/lessons.md` にそのパターンを記録する
- 同じミスを繰り返さないように、自分へのルールを書く
- ミス率が下がるまで、ルールを徹底的に改善し続ける
- セッション開始時に、そのプロジェクトに関連するlessonsをレビューする

### 4. 完了前に必ず検証する
- 動作を証明できるまで、タスクを完了とマークしない
- 必要に応じてmainブランチと自分の変更の差分を確認する
- 「スタッフエンジニアはこれを承認するか？」と自問する
- テストを実行し、ログを確認し、正しく動作することを示す

### 5. エレガントさを追求する（バランスよく）
- 重要な変更をする前に「もっとエレガントな方法はないか？」と一度立ち止まる
- ハック的な修正に感じたら「今知っていることをすべて踏まえて、エレガントな解決策を実装する」
- シンプルで明白な修正にはこのプロセスをスキップする（過剰設計しない）
- 提示する前に自分の作業に自問自答する

### 6. 自律的なバグ修正
- バグレポートを受けたら、手取り足取り教えてもらわずにそのまま修正する
- ログ・エラー・失敗しているテストを見て、自分で解決する
- ユーザーのコンテキスト切り替えをゼロにする
- 言われなくても、失敗しているCIテストを修正しに行く

---

## タスク管理

タスクは `docs/tasks/` 配下に **`YYYYMMDD_taskname.md`** の命名規則で管理する。

- **ファイル名:** `docs/tasks/YYYYMMDD_taskname.md`（例: `docs/tasks/20260302_add_edge_support.md`）
- **日付:** タスク作成日
- **taskname:** 内容を端的に表すスネークケースの英語名
- 各タスクファイルにはチェックリスト形式で計画・進捗・結果を記録する
- **完了したタスクファイルは `docs/done/` に移動する**（履歴として保持）
- 学び・修正パターンは `docs/tasks/lessons.md` に記録する

---

## コア原則

- **シンプル第一**：すべての変更をできる限りシンプルにする。影響するコードを最小限にする。
- **手を抜かない**：根本原因を見つける。一時的な修正は避ける。シニアエンジニアの水準を保つ。
- **影響を最小化する**：変更は必要な箇所のみにとどめる。バグを新たに引き込まない。

---

## セキュリティメモ（2026-03-02 監査済み・2026-03-02 CI強化後再確認済み）

**最終監査:** 2026-03-02 / govulncheck （最新） / 既知CVE: 0件

### 依存パッケージの状態

| パッケージ | バージョン | 状態 |
|---|---|---|
| `golang.org/x/sys` | v0.41.0 | 最新・問題なし |
| `github.com/go-ole/go-ole` | v1.3.0 | 最新・問題なし |
| `github.com/shirou/gopsutil/v3` | v3.24.5 | v3系最新。開発主軸はv4に移行中（緊急性なし） |
| `github.com/yusufpapurcu/wmi` | v1.2.4 | 最新・問題なし |

### 注意事項（未解消・継続監視）

- **gopsutil v3→v4 移行**: 将来的なセキュリティ修正がv4のみに提供されるリスクあり。移行時は `internal/monitor/process.go` のインポートパス変更が必要。今回のCI強化では未対応（go.mod に変更なし）。
- **間接依存の古いパッケージ** (`lufia/plan9stats`, `power-devops/perfstat` 等): WindowsビルドではOS固有のコードパスに到達しないため実害なし。gopsutil v4移行時に自動解消。govulncheck スキャン対象外（Windows ビルドで未到達パス）。
- **COM vtable 直接呼び出し** (`internal/monitor/chrome.go`): vtableオフセット（`vtblElementFromHandle = 6` 等）はWindowsメジャーアップデートで変わる可能性あり。コードレビューで対応すべき設計上のリスクであり、CI/CDでは自動検出不可。

### CI強化による改善（2026-03-02 マージ済み）

| 項目 | 変更前 | 変更後 | セキュリティへの効果 |
|---|---|---|---|
| `golangci/golangci-lint-action` | v6 | v9 | lintアクション自体の既知脆弱性リスク低減 |
| `actions/setup-go` | v5 | v6 | アクション自体のリスク低減 |
| `actions/checkout` | v4 | v6 | アクション自体のリスク低減 |
| `actions/upload-artifact` | v4 | v7 | アクション自体のリスク低減 |
| `.golangci.yml` | v1形式 | v2形式 | `govet shadow` 検出が有効（変数シャドウによるバグ予防） |
| `.gitattributes` | なし | LF強制 | CRLF改行によるスクリプトインジェクション誤検知の排除 |
| `govulncheck ./...` CI組み込み | 未対応 | **未対応のまま** | Dependabot（gomod）が週次で間接的に補完 |

### govulncheck CI組み込みの状況

`build.yml` には govulncheck ステップが存在しない。Dependabot（gomod）の週次スキャンが間接的な補完になっているが、CVE検出のタイムラグがある。govulncheck をCIに追加するには以下を `build.yml` に挿入する:

```yaml
- name: Vulnerability scan
  run: go install golang.org/x/vuln/cmd/govulncheck@latest && govulncheck ./...
```

## 使い方

### ビルド

```bash
go build -o logging.exe .
```

### 実行

```bash
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