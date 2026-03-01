package monitor

import (
	"unsafe"

	"github.com/shirou/gopsutil/v3/process"
	"golang.org/x/sys/windows"
)

var (
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
)

// GetProcessName は HWND からプロセス名（例: chrome.exe）を返す。
// 取得できない場合は空文字を返す。
func GetProcessName(hwnd windows.HWND) string {
	var pid uint32
	_, _, _ = procGetWindowThreadProcessId.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&pid)))
	if pid == 0 {
		return ""
	}
	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		return ""
	}
	name, err := proc.Name()
	if err != nil {
		return ""
	}
	return name
}
