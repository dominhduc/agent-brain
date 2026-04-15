package main

import (
	"fmt"
	"runtime"
)

func cmdVersion() {
	fmt.Printf("brain %s  %s/%s", version, runtime.GOOS, runtime.GOARCH)
	if commit != "" {
		short := commit
		if len(short) > 7 {
			short = short[:7]
		}
		fmt.Printf("  %s", short)
	}
	if date != "" {
		fmt.Printf("\n  built: %s", date)
	}
	fmt.Println()
}
