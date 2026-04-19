//go:build windows

package service

import "fmt"

func registerLaunchd(execPath, workDir string) error {
	return fmt.Errorf("launchd is not available on Windows")
}

func registerSystemd(execPath, workDir string) error {
	return fmt.Errorf("systemd is not available on Windows")
}

func startLaunchd(workDir string) error {
	return fmt.Errorf("launchd is not available on Windows")
}

func startSystemd(workDir string) error {
	return fmt.Errorf("systemd is not available on Windows")
}

func stopLaunchd(workDir string) error {
	return fmt.Errorf("launchd is not available on Windows")
}

func stopSystemd(workDir string) error {
	return fmt.Errorf("systemd is not available on Windows")
}

func isRunningLaunchd(workDir string) bool {
	return false
}

func isRunningSystemd(workDir string) bool {
	return false
}

func isRunningNohup(workDir string) bool {
	return false
}

func registerNohup(execPath, workDir string) error {
	return fmt.Errorf("nohup is not available on Windows")
}

func startNohup(workDir string) error {
	return fmt.Errorf("nohup is not available on Windows")
}

func stopNohup(workDir string) error {
	return fmt.Errorf("nohup is not available on Windows")
}

func nohupAvailable() bool {
	return false
}

func systemdAvailable() bool {
	return false
}

func listServices() ([]ServiceInfo, error) {
	return nil, fmt.Errorf("service listing not supported on Windows")
}
