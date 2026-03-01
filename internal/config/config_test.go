package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_FileExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
interval = 30
exclude_processes = ["explorer.exe", "Taskmgr.exe"]
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Interval != 30 {
		t.Errorf("Interval = %d, want 30", cfg.Interval)
	}
	if len(cfg.ExcludeProcesses) != 2 {
		t.Fatalf("ExcludeProcesses len = %d, want 2", len(cfg.ExcludeProcesses))
	}
	if cfg.ExcludeProcesses[0] != "explorer.exe" {
		t.Errorf("ExcludeProcesses[0] = %q, want %q", cfg.ExcludeProcesses[0], "explorer.exe")
	}
}

func TestLoad_FileNotExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.toml")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Interval != 60 {
		t.Errorf("Interval = %d, want default 60", cfg.Interval)
	}
	if len(cfg.ExcludeProcesses) != 0 {
		t.Errorf("ExcludeProcesses should be empty, got %v", cfg.ExcludeProcesses)
	}
}

func TestLoad_InvalidTOML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.toml")
	if err := os.WriteFile(path, []byte("{{invalid toml"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load should return error for invalid TOML")
	}
}

func TestLoad_ZeroInterval(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `interval = 0`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Interval != 60 {
		t.Errorf("Interval = %d, want default 60 when set to 0", cfg.Interval)
	}
}

func TestIsExcluded_CaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `exclude_processes = ["Explorer.exe"]`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	tests := []struct {
		name string
		want bool
	}{
		{"Explorer.exe", true},
		{"explorer.exe", true},
		{"EXPLORER.EXE", true},
		{"explorer.EXE", true},
	}
	for _, tt := range tests {
		if got := cfg.IsExcluded(tt.name); got != tt.want {
			t.Errorf("IsExcluded(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestIsExcluded_NotInList(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `exclude_processes = ["Explorer.exe"]`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.IsExcluded("chrome.exe") {
		t.Error("IsExcluded(chrome.exe) should be false")
	}
	if cfg.IsExcluded("notepad.exe") {
		t.Error("IsExcluded(notepad.exe) should be false")
	}
}
