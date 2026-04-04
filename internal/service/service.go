package service

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

type ServiceInfo struct {
	Name    string
	Status  string
	Project string
}

func Register(execPath, workDir string) error {
	switch runtime.GOOS {
	case "darwin":
		return registerLaunchd(execPath, workDir)
	case "linux":
		return registerSystemd(execPath, workDir)
	default:
		return fmt.Errorf("unsupported OS for service registration: %s.\nWhat to do: run 'brain daemon run' manually.", runtime.GOOS)
	}
}

func Start(workDir string) error {
	switch runtime.GOOS {
	case "darwin":
		return startLaunchd(workDir)
	case "linux":
		return startSystemd(workDir)
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

func Stop(workDir string) error {
	switch runtime.GOOS {
	case "darwin":
		return stopLaunchd(workDir)
	case "linux":
		return stopSystemd(workDir)
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

func IsRunning(workDir string) bool {
	switch runtime.GOOS {
	case "darwin":
		return isRunningLaunchd(workDir)
	case "linux":
		return isRunningSystemd(workDir)
	default:
		return false
	}
}

func ListServices() ([]ServiceInfo, error) {
	if runtime.GOOS == "windows" {
		return nil, fmt.Errorf("not supported on Windows")
	}
	return listServices()
}

func StopCurrentProject() {
	brainDir, err := findCurrentProjectBrainDir()
	if err != nil {
		return
	}
	workDir := brainDir
	if workDir != "" {
		Stop(workDir)
	}
}

func findCurrentProjectBrainDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := cwd
	for {
		brainPath := filepath.Join(dir, ".brain")
		if _, err := os.Stat(brainPath); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no .brain directory found")
		}
		dir = parent
	}
}

func homeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return home, nil
}
