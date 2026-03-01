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

	fmt.Fprintf(os.Stderr, "Starting logging (interval: %ds)\n", cfg.Interval)

	// シグナルハンドリング
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	// 初回は即座に実行
	collect(cfg)

	ticker := time.NewTicker(time.Duration(cfg.Interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			collect(cfg)
		case <-sigCh:
			fmt.Fprintln(os.Stderr, "\nShutting down...")
			return
		}
	}
}

// collect は1回分のデータ収集・出力を行う。
func collect(cfg *config.Config) {
	// アクティブウィンドウ取得
	active := monitor.GetActiveWindow()
	if active.ProcessName == "chrome.exe" {
		url, err := monitor.GetChromeURL(active.HWND)
		if err == nil && url != "" {
			active.URL = url
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
		}
		bg = append(bg, w)
	}

	record := LogRecord{
		Timestamp:         time.Now(),
		ActiveWindow:      active,
		BackgroundWindows: bg,
	}

	enc := json.NewEncoder(os.Stdout)
	_ = enc.Encode(record)
}
