# 実装計画 — タスクチェックリスト

## ディレクトリ構成（最終形）

```
logging-go/
├── CLAUDE.md
├── docs/
│   ├── requirements.md
│   └── implementation-plan.md
├── go.mod
├── go.sum
├── main.go
└── internal/
    └── monitor/
        ├── window.go     # アクティブウィンドウ・EnumWindows
        ├── process.go    # プロセス名取得
        └── chrome.go     # UI Automation / URL取得
```

---

## Phase 1: プロジェクト初期化

> **目標:** ビルドが通る骨格を作る。依存パッケージと品質ツールをすべて準備する。

- [x] `go mod init github.com/octop162/logging-go` を実行
- [x] 依存パッケージを追加:
  - [x] `go get golang.org/x/sys/windows`
  - [x] `go get github.com/go-ole/go-ole`
  - [x] `go get github.com/shirou/gopsutil/v3`
- [x] golangci-lint をインストール（未導入の場合）:
  ```bash
  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
  ```
- [x] `.golangci.yml` を作成（使用するlinterを明示）
- [x] `internal/monitor/` ディレクトリを作成
- [x] `internal/monitor/window.go` — スタブファイル作成（パッケージ宣言のみ）
- [x] `internal/monitor/process.go` — スタブファイル作成
- [x] `internal/monitor/chrome.go` — スタブファイル作成
- [x] `main.go` — エントリーポイントのスタブ作成
- [x] `go build -o logging.exe .` でビルドが通ることを確認
- [x] `gofmt -l .` を実行してフォーマットエラーがないことを確認
- [x] `golangci-lint run` を実行してlintエラーがないことを確認

**完了条件:** `go build` がエラーなく通り、`golangci-lint run` がクリーン。

---

## Phase 2: アクティブウィンドウ + EnumWindows

> **目標:** アクティブウィンドウとすべての可視ウィンドウを取得して標準出力に表示できる。

### window.go の実装
- [x] `GetForegroundWindow` でフォアグラウンドウィンドウのHWNDを取得
- [x] `GetWindowText` でウィンドウタイトルを取得する関数 `GetActiveWindow()` を実装
- [x] `EnumWindows` + `syscall.NewCallback` でコールバックを実装
- [x] `IsWindowVisible` で可視ウィンドウのみフィルタリング
- [x] `GetWindowTextLength` が0のウィンドウ（タイトルなし）を除外
- [x] `EnumVisibleWindows()` として列挙結果を `[]WindowInfo` で返す

### 動作確認
- [x] `main.go` から `GetActiveWindow()` を呼び出して出力
- [x] `main.go` から `EnumVisibleWindows()` を呼び出して全ウィンドウ一覧を出力
- [x] `go build && ./logging.exe` で期待通りの出力が得られることを確認

- [x] `gofmt -w .` を実行してコードをフォーマット
- [x] `golangci-lint run` を実行してlintエラーがないことを確認

**完了条件:** アクティブウィンドウのタイトルと、可視ウィンドウの一覧が標準出力に表示され、`golangci-lint run` がクリーン。

---

## Phase 3: プロセス名取得

> **目標:** 各ウィンドウのHWNDからプロセス名（例: `chrome.exe`）を取得できる。

### process.go の実装
- [x] `GetWindowThreadProcessId` でHWNDからPIDを取得
- [x] gopsutil の `process.NewProcess(pid)` でプロセスオブジェクトを取得
- [x] `proc.Name()` でプロセス名を取得する関数 `GetProcessName(hwnd)` を実装
- [x] エラー時は空文字を返す（プロセスが終了済みの場合など）

### Phase 2との統合
- [x] `WindowInfo` 構造体に `ProcessName string` フィールドを追加
- [x] `EnumVisibleWindows()` 内で各HWNDに対して `GetProcessName` を呼び出す
- [x] `GetActiveWindow()` の戻り値にもプロセス名を含める
- [x] 出力にプロセス名が表示されることを確認

- [x] `gofmt -w .` を実行してコードをフォーマット
- [x] `golangci-lint run` を実行してlintエラーがないことを確認

**完了条件:** ウィンドウ一覧にプロセス名が付与されて表示され、`golangci-lint run` がクリーン。

---

## Phase 4: Chrome URL取得（UI Automation）

> **目標:** Chromeがフォアグラウンドにあるとき、現在のタブのURLを取得できる。

### vtable定義
- [x] `IUIAutomation` インターフェースのvtableを手動定義
  - CLSID: `{FF48DBA4-60EF-4201-AA87-54103EEF594E}`
  - IID: `{30CBE57D-D9D0-452A-AB13-7AC5AC4825EE}`
- [x] `IUIAutomationElement` インターフェースのvtable定義
- [x] `IUIAutomationCondition` インターフェースのvtable定義
- [x] `IUIAutomationValuePattern` インターフェースのvtable定義

### chrome.go の実装
- [x] `ole.CoInitializeEx` でCOM初期化（`COINIT_APARTMENTTHREADED`）
- [x] `ole.CoCreateInstance` で `CUIAutomation` インスタンス生成
- [x] `IUIAutomation.ElementFromHandle` でChromeウィンドウのルート要素取得
- [x] `IUIAutomation.CreatePropertyCondition` でEdit要素の検索条件作成（ControlType = Edit）
- [x] `IUIAutomationElement.FindFirst` でアドレスバー要素を検索
- [x] `IUIAutomationElement.GetCurrentPattern` で `ValuePattern` を取得
- [x] `IUIAutomationValuePattern.CurrentValue` でURLを取得
- [x] 関数 `GetChromeURL(hwnd windows.HWND) (string, error)` としてエクスポート

### 動作確認
- [x] Chromeを開いた状態で `./logging.exe` を実行し、URLが取得できることを確認
- [x] Chrome以外がアクティブのとき、URLが空文字になることを確認
- [x] Chromeが開いていないとき、エラーが適切にハンドリングされることを確認

### 追加機能: Chrome 全タブタイトル取得
- [x] `IUIAutomationElementArray` のvtable定義（`get_Length`, `GetElement`）
- [x] `FindAll` で TabItem(50019) を列挙する実装
- [x] ウェブコンテンツ内の Tab UI（YouTube フィルタ、X の「おすすめ/フォロー中」等）を除外
  - Document(50030) の子孫 TabItem を除外セットとして減算するアルゴリズム
- [x] 関数 `GetChromeTabTitles(hwnd windows.HWND) ([]string, error)` としてエクスポート
- [x] 複数タブ・複数ウィンドウでの動作確認

- [x] `gofmt -w .` を実行してコードをフォーマット
- [x] `golangci-lint run` を実行してlintエラーがないことを確認

**完了条件:** ChromeのアドレスバーのURLと全タブタイトルが標準出力に表示され、`golangci-lint run` がクリーン。

---

## Phase 5: 統合 + ポーリングループ

> **目標:** 全コンポーネントを統合し、1分間隔で構造化ログを出力する常駐プロセスを完成させる。

### データ構造定義
- [ ] `LogRecord` 構造体を定義（JSON出力用）:
  ```go
  type LogRecord struct {
      Timestamp       time.Time    `json:"timestamp"`
      ActiveWindow    WindowInfo   `json:"active_window"`
      BackgroundWindows []WindowInfo `json:"background_windows"`
  }
  ```
- [ ] `WindowInfo` 構造体に `URL string` フィールドを追加（Chrome時のみ使用）

### main.go の実装
- [ ] `flag` パッケージで `--interval` オプション（デフォルト: 60秒）を追加
- [ ] `time.NewTicker` で指定間隔のポーリングループを実装
- [ ] 各ティックで以下を収集:
  - [ ] `GetActiveWindow()` でアクティブウィンドウ取得
  - [ ] プロセス名が `chrome.exe` の場合 `GetChromeURL()` を呼び出し
  - [ ] `EnumVisibleWindows()` でバックグラウンドウィンドウ一覧取得
- [ ] `encoding/json` で `LogRecord` をJSON化して標準出力に書き出す
- [ ] シグナル（Ctrl+C）でクリーンに終了する

### 動作確認
- [ ] `go build -o logging.exe .` でビルドが通る
- [ ] `./logging.exe` を実行し、1分ごとにJSONログが出力されることを確認
- [ ] `./logging.exe --interval 10` で10秒間隔に変更できることを確認
- [ ] Task Manager でCPU使用率 <1%、メモリ <50MB であることを確認

- [ ] `gofmt -w .` を実行してコードをフォーマット
- [ ] `golangci-lint run` を実行してlintエラーがないことを確認

**完了条件:** 全コンポーネントが統合され、構造化JSONログが定期出力され、`golangci-lint run` がクリーン。

---

## Phase 6: 品質・最終確認

> **目標:** 本番運用に耐えうる品質にする。

- [ ] `go test ./...` でテストが通る（window/processの単体テスト）
- [ ] `golangci-lint run` でlintエラーがない
- [ ] 長時間実行（1時間以上）でメモリリークがないことを確認
- [ ] COM オブジェクトの `Release()` が適切に呼ばれていることを確認
- [ ] README または CLAUDE.md に最終的な使い方を追記

**完了条件:** lintクリーン、テスト通過、長時間安定動作。

---

## フェーズ完了サマリー

| Phase | 内容                        | 状態 |
|-------|-----------------------------|------|
| 1     | プロジェクト初期化           | [x]  |
| 2     | アクティブウィンドウ + EnumWindows | [x]  |
| 3     | プロセス名取得               | [x]  |
| 4     | Chrome URL取得（UI Automation）| [x]  |
| 5     | 統合 + ポーリングループ       | [ ]  |
| 6     | 品質・最終確認               | [ ]  |
