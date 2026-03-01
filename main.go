package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/octop162/logging-go/internal/config"
	"github.com/octop162/logging-go/internal/monitor"
)

// LogRecord は1回のポーリングで収集したデータを表す。
type LogRecord struct {
	Timestamp         time.Time            `json:"timestamp"`
	ActiveWindow      monitor.WindowInfo   `json:"active_window"`
	BackgroundWindows []monitor.WindowInfo `json:"background_windows"`
}

func main() {
	// フラグ定義
	intervalFlag := flag.Int("interval", 0, "ポーリング間隔（秒）。設定ファイルより優先。デフォルト: 60")
	configFlag := flag.String("config", "", "設定ファイルのパス（デフォルト: 実行ファイルと同階層の config.toml）")
	logdirFlag := flag.String("logdir", "", "ログ出力ディレクトリ。設定ファイルより優先")
	flag.Parse()

	// 設定ファイル読み込み
	configPath := *configFlag
	if configPath == "" {
		exe, err := os.Executable()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		configPath = filepath.Join(filepath.Dir(exe), "config.toml")
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Config error: %v\n", err)
		os.Exit(1)
	}

	// --interval フラグが指定されていれば設定を上書き
	if *intervalFlag > 0 {
		cfg.Interval = *intervalFlag
	}

	// --logdir フラグが指定されていれば設定を上書き
	if *logdirFlag != "" {
		cfg.LogDir = *logdirFlag
	}

	fmt.Fprintf(os.Stderr, "Starting logging (interval: %ds, logdir: %s)\n", cfg.Interval, cfg.LogDir)

	// ログファイルを開く
	currentDate := time.Now().Format("2006-01-02")
	logFile, err := openLogFile(cfg.LogDir, currentDate)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
		os.Exit(1)
	}

	// シグナルハンドリング
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	// 初回は即座に実行
	collect(cfg, logFile)

	ticker := time.NewTicker(time.Duration(cfg.Interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 日付ローテーション
			newDate := time.Now().Format("2006-01-02")
			if newDate != currentDate {
				if closeErr := logFile.Close(); closeErr != nil {
					fmt.Fprintf(os.Stderr, "Failed to close log file: %v\n", closeErr)
				}
				currentDate = newDate
				logFile, err = openLogFile(cfg.LogDir, currentDate)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
					os.Exit(1)
				}
			}
			collect(cfg, logFile)
		case <-sigCh:
			fmt.Fprintln(os.Stderr, "\nShutting down...")
			if closeErr := logFile.Close(); closeErr != nil {
				fmt.Fprintf(os.Stderr, "Failed to close log file: %v\n", closeErr)
			}
			return
		}
	}
}

// openLogFile は指定ディレクトリに YYYY-MM-DD.jsonl ファイルを追記モードで開く。
func openLogFile(dir string, date string) (*os.File, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}
	path := filepath.Join(dir, date+".jsonl")
	return os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
}

// collect は1回分のデータ収集・出力を行う。
func collect(cfg *config.Config, w *os.File) {
	// アクティブウィンドウ取得
	active := monitor.GetActiveWindow()
	if active.ProcessName == "chrome.exe" {
		url, err := monitor.GetChromeURL(active.HWND)
		if err == nil && url != "" {
			active.URL = url
		}
		tabs, err := monitor.GetChromeTabTitles(active.HWND)
		if err == nil && len(tabs) > 0 {
			active.Tabs = tabs
		}
	}

	// バックグラウンドウィンドウ取得 + フィルタ
	allWindows := monitor.EnumVisibleWindows()
	var bg []monitor.WindowInfo
	for _, w := range allWindows {
		if cfg.IsExcluded(w.ProcessName) {
			continue
		}
		if w.ProcessName == "chrome.exe" {
			url, err := monitor.GetChromeURL(w.HWND)
			if err == nil && url != "" {
				w.URL = url
			}
			tabs, err := monitor.GetChromeTabTitles(w.HWND)
			if err == nil && len(tabs) > 0 {
				w.Tabs = tabs
			}
		}
		bg = append(bg, w)
	}

	record := LogRecord{
		Timestamp:         time.Now(),
		ActiveWindow:      active,
		BackgroundWindows: bg,
	}

	enc := json.NewEncoder(w)
	_ = enc.Encode(record)
}
