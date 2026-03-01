package main

import (
	"fmt"

	"github.com/octop162/logging-go/internal/monitor"
)

func main() {
	active := monitor.GetActiveWindow()
	fmt.Printf("Active Window: %q (%s)\n", active.Title, active.ProcessName)

	fmt.Println("\nVisible Windows:")
	for _, w := range monitor.EnumVisibleWindows() {
		fmt.Printf("  - %q (%s)\n", w.Title, w.ProcessName)
		if w.ProcessName == "chrome.exe" {
			url, err := monitor.GetChromeURL(w.HWND)
			if err != nil {
				fmt.Printf("    URL error: %v\n", err)
			} else {
				fmt.Printf("    URL: %s\n", url)
			}
			tabs, err := monitor.GetChromeTabTitles(w.HWND)
			if err != nil {
				fmt.Printf("    Tabs error: %v\n", err)
			} else if len(tabs) == 0 {
				fmt.Printf("    Tabs: (none found)\n")
			} else {
				for i, t := range tabs {
					fmt.Printf("    Tab[%d]: %s\n", i, t)
				}
			}
		}
	}
}
