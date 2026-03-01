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
	}
}
