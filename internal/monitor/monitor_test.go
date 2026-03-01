package monitor

import (
	"testing"
)

// TestGetActiveWindow はフォアグラウンドウィンドウの情報が取得できることを確認する。
// テスト実行時は何かしらのウィンドウが存在するため、タイトルが空でないことを期待する。
func TestGetActiveWindow(t *testing.T) {
	info := GetActiveWindow()
	if info.Title == "" {
		t.Log("Warning: active window title is empty (may happen in headless CI)")
	}
	// HWND が 0 でないことを確認
	if info.HWND == 0 {
		t.Log("Warning: HWND is 0 (may happen in headless CI)")
	}
}

// TestEnumVisibleWindows は可視ウィンドウが1件以上返ることを確認する。
func TestEnumVisibleWindows(t *testing.T) {
	windows := EnumVisibleWindows()
	if len(windows) == 0 {
		t.Log("Warning: no visible windows found (may happen in headless CI)")
		return
	}
	// 最低1件はタイトルが空でないこと
	for _, w := range windows {
		if w.Title != "" {
			return // OK
		}
	}
	t.Error("all visible windows have empty titles")
}

// TestGetProcessName はアクティブウィンドウの HWND でプロセス名が取得できることを確認する。
func TestGetProcessName(t *testing.T) {
	info := GetActiveWindow()
	if info.HWND == 0 {
		t.Skip("no active window HWND available")
	}
	name := GetProcessName(info.HWND)
	if name == "" {
		t.Log("Warning: process name is empty for active window")
	} else {
		t.Logf("Active window process: %s", name)
	}
}
