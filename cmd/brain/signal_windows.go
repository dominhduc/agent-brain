//go:build windows

package main

import (
	"context"
	"os"
	"os/signal"
)

func setupSignalContext() (context.Context, func()) {
	return signal.NotifyContext(context.Background(), os.Interrupt)
}
