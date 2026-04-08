package main

import (
	"fmt"
	"runtime"
)

func cmdVersion() {
	fmt.Printf("agent-brain %s", version)
	if commit != "" {
		fmt.Printf("  commit: %s", commit)
	}
	if date != "" {
		fmt.Printf("  built: %s", date)
	}
	fmt.Println()
	fmt.Printf("  os/arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
}
