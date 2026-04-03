//go:build windows

package service

import "fmt"

func registerLaunchd(execPath string) error {
	return fmt.Errorf("launchd is not available on Windows")
}

func registerSystemd(execPath string) error {
	return fmt.Errorf("systemd is not available on Windows")
}

func startLaunchd() error {
	return fmt.Errorf("launchd is not available on Windows")
}

func startSystemd() error {
	return fmt.Errorf("systemd is not available on Windows")
}

func stopLaunchd() error {
	return fmt.Errorf("launchd is not available on Windows")
}

func stopSystemd() error {
	return fmt.Errorf("systemd is not available on Windows")
}
