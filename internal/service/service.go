package service

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

func Register(execPath, workDir string) error {
	switch runtime.GOOS {
	case "darwin":
		return registerLaunchd(execPath)
	case "linux":
		return registerSystemd(execPath, workDir)
	default:
		return fmt.Errorf("unsupported OS for service registration: %s.\nWhat to do: run 'brain daemon run' manually.", runtime.GOOS)
	}
}

func Start() error {
	switch runtime.GOOS {
	case "darwin":
		return startLaunchd()
	case "linux":
		return startSystemd()
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

func Stop() error {
	switch runtime.GOOS {
	case "darwin":
		return stopLaunchd()
	case "linux":
		return stopSystemd()
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

func IsRunning() bool {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("launchctl", "list", "com.dominhduc.brain-daemon").Run() == nil
	case "linux":
		return exec.Command("systemctl", "--user", "is-active", "brain-daemon.service").Run() == nil
	default:
		return false
	}
}

func homeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return home, nil
}
