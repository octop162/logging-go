package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// defaultLogDir は実行ファイルと同階層の "logs" を返す。
func defaultLogDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "logs"
	}
	return filepath.Join(filepath.Dir(exe), "logs")
}

// Config はアプリケーション設定を保持する。
type Config struct {
	Interval         int      `toml:"interval"`          // ポーリング間隔（秒）
	LogDir           string   `toml:"log_dir"`           // ログ出力ディレクトリ
	ExcludeProcesses []string `toml:"exclude_processes"` // 除外プロセス名リスト
	excludeSet       map[string]struct{}
}

// Load は TOML 設定ファイルを読み込む。
// ファイルが存在しない場合はデフォルト設定を返す。
func Load(path string) (*Config, error) {
	cfg := &Config{
		Interval: 60,
		LogDir:   defaultLogDir(),
	}
	_, err := toml.DecodeFile(path, cfg)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			cfg.buildExcludeSet()
			return cfg, nil
		}
		return nil, err
	}
	if cfg.Interval <= 0 {
		cfg.Interval = 60
	}
	if cfg.LogDir == "" {
		cfg.LogDir = defaultLogDir()
	}
	cfg.buildExcludeSet()
	return cfg, nil
}

// buildExcludeSet は除外リストを小文字化した set に変換する。
func (c *Config) buildExcludeSet() {
	c.excludeSet = make(map[string]struct{}, len(c.ExcludeProcesses))
	for _, p := range c.ExcludeProcesses {
		c.excludeSet[strings.ToLower(p)] = struct{}{}
	}
}

// IsExcluded はプロセス名が除外リストに含まれるか判定する（大文字小文字無視）。
func (c *Config) IsExcluded(processName string) bool {
	_, ok := c.excludeSet[strings.ToLower(processName)]
	return ok
}
