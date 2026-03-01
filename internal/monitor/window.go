package monitor

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32                   = windows.NewLazySystemDLL("user32.dll")
	procGetForegroundWindow  = user32.NewProc("GetForegroundWindow")
	procGetWindowTextW       = user32.NewProc("GetWindowTextW")
	procGetWindowTextLengthW = user32.NewProc("GetWindowTextLengthW")
	procEnumWindows          = user32.NewProc("EnumWindows")
	procIsWindowVisible      = user32.NewProc("IsWindowVisible")
)

// WindowInfo はウィンドウの情報を保持する。
type WindowInfo struct {
	HWND        windows.HWND `json:"-"`
	Title       string       `json:"title"`
	ProcessName string       `json:"process_name"`
	URL         string       `json:"url,omitempty"`
}

// getWindowText は指定した HWND のウィンドウタイトルを返す。
func getWindowText(hwnd windows.HWND) string {
	length, _, _ := procGetWindowTextLengthW.Call(uintptr(hwnd))
	if length == 0 {
		return ""
	}
	buf := make([]uint16, length+1)
	// r1 はコピーした文字数。0 の場合は取得失敗だが空文字を返すだけでよい。
	r1, _, _ := procGetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), uintptr(length+1))
	if r1 == 0 {
		return ""
	}
	return windows.UTF16ToString(buf)
}

// GetActiveWindow はフォアグラウンドウィンドウの情報を返す。
func GetActiveWindow() WindowInfo {
	hwnd, _, _ := procGetForegroundWindow.Call()
	h := windows.HWND(hwnd)
	return WindowInfo{
		HWND:        h,
		Title:       getWindowText(h),
		ProcessName: GetProcessName(h),
	}
}

// EnumVisibleWindows はタイトルを持つ可視ウィンドウの一覧を返す。
func EnumVisibleWindows() []WindowInfo {
	var result []WindowInfo

	cb := func(hwnd uintptr, _ uintptr) uintptr {
		visible, _, _ := procIsWindowVisible.Call(hwnd)
		if visible == 0 {
			return 1
		}
		title := getWindowText(windows.HWND(hwnd))
		if title == "" {
			return 1
		}
		h := windows.HWND(hwnd)
		result = append(result, WindowInfo{
			HWND:        h,
			Title:       title,
			ProcessName: GetProcessName(h),
		})
		return 1
	}

	// r1 が 0 の場合は列挙失敗。失敗しても収集済みの result をそのまま返す。
	_, _, _ = procEnumWindows.Call(
		windows.NewCallback(cb),
		0,
	)
	return result
}
