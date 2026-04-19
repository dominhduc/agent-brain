//go:build !windows

package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
)

var (
	systemdCheckOnce sync.Once
	systemdCheckResult bool
)

func projectHash(projectPath string) string {
	hash := sha256.Sum256([]byte(projectPath))
	return hex.EncodeToString(hash[:4])
}

func serviceName(projectPath string) string {
	if projectPath == "" {
		return "brain-daemon"
	}
	return fmt.Sprintf("brain-daemon.%s", projectHash(projectPath))
}

func systemdAvailable() bool {
	systemdCheckOnce.Do(func() {
		path, err := exec.LookPath("systemctl")
		if err != nil {
			systemdCheckResult = false
			return
		}
		if path == "" {
			systemdCheckResult = false
			return
		}
		uid := os.Getuid()
		if _, err := os.Stat(fmt.Sprintf("/run/user/%d/systemd", uid)); err != nil {
			systemdCheckResult = false
			return
		}
		systemdCheckResult = true
	})
	return systemdCheckResult
}

func nohupLogPath(workDir string) string {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = filepath.Join(os.TempDir(), "brain")
	}
	return filepath.Join(cacheDir, fmt.Sprintf("brain-daemon-%s.log", projectHash(workDir)))
}

func nohupScriptPath(workDir string) string {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = filepath.Join(os.TempDir(), "brain")
	}
	return filepath.Join(cacheDir, fmt.Sprintf("brain-daemon-%s.sh", projectHash(workDir)))
}

func registerLaunchd(execPath, workDir string) error {
	home, err := homeDir()
	if err != nil {
		return err
	}

	name := serviceName(workDir)
	plistDir := filepath.Join(home, "Library", "LaunchAgents")
	if err := os.MkdirAll(plistDir, 0755); err != nil {
		return err
	}

	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.dominhduc.%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>daemon</string>
        <string>run</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/brain-daemon-%s.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/brain-daemon-%s.err</string>
</dict>
</plist>`, name, execPath, projectHash(workDir), projectHash(workDir))

	plistPath := filepath.Join(plistDir, fmt.Sprintf("com.dominhduc.%s.plist", name))
	if err := os.WriteFile(plistPath, []byte(plist), 0644); err != nil {
		return err
	}
	return exec.Command("launchctl", "load", plistPath).Run()
}

func registerSystemd(execPath, workDir string) error {
	home, err := homeDir()
	if err != nil {
		return err
	}

	name := serviceName(workDir)
	serviceDir := filepath.Join(home, ".config", "systemd", "user")
	if err := os.MkdirAll(serviceDir, 0755); err != nil {
		return err
	}

	workDirName := filepath.Base(workDir)
	service := fmt.Sprintf(`[Unit]
Description=agent-brain Daemon for %s
After=network.target
StartLimitIntervalSec=0

[Service]
Type=simple
ExecStart=%s daemon run
WorkingDirectory=%s
Restart=always
RestartSec=5

[Install]
WantedBy=default.target`, workDirName, execPath, workDir)

	servicePath := filepath.Join(serviceDir, fmt.Sprintf("%s.service", name))
	if err := os.WriteFile(servicePath, []byte(service), 0644); err != nil {
		return err
	}

	if err := exec.Command("systemctl", "--user", "daemon-reload").Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: systemd daemon-reload failed: %v\n", err)
	}
	if err := exec.Command("systemctl", "--user", "enable", fmt.Sprintf("%s.service", name)).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: systemd enable failed: %v\n", err)
	}
	return exec.Command("systemctl", "--user", "start", fmt.Sprintf("%s.service", name)).Run()
}

func startLaunchd(workDir string) error {
	home, err := homeDir()
	if err != nil {
		return err
	}
	name := serviceName(workDir)
	plistPath := filepath.Join(home, "Library", "LaunchAgents", fmt.Sprintf("com.dominhduc.%s.plist", name))
	return exec.Command("launchctl", "load", plistPath).Run()
}

func startSystemd(workDir string) error {
	name := serviceName(workDir)
	return exec.Command("systemctl", "--user", "start", fmt.Sprintf("%s.service", name)).Run()
}

func stopLaunchd(workDir string) error {
	home, err := homeDir()
	if err != nil {
		return err
	}
	name := serviceName(workDir)
	plistPath := filepath.Join(home, "Library", "LaunchAgents", fmt.Sprintf("com.dominhduc.%s.plist", name))
	return exec.Command("launchctl", "unload", plistPath).Run()
}

func stopSystemd(workDir string) error {
	name := serviceName(workDir)
	return exec.Command("systemctl", "--user", "stop", fmt.Sprintf("%s.service", name)).Run()
}

func isRunningSystemd(workDir string) bool {
	name := serviceName(workDir)
	cmd := exec.Command("systemctl", "--user", "is-active", fmt.Sprintf("%s.service", name))
	return cmd.Run() == nil
}

func isRunningLaunchd(workDir string) bool {
	home, err := homeDir()
	if err != nil {
		return false
	}
	name := serviceName(workDir)
	plistPath := filepath.Join(home, "Library", "LaunchAgents", fmt.Sprintf("com.dominhduc.%s.plist", name))
	cmd := exec.Command("launchctl", "list", fmt.Sprintf("com.dominhduc.%s", name))
	return cmd.Run() == nil && plistPath != ""
}

func listServices() ([]ServiceInfo, error) {
	var services []ServiceInfo

	home, err := homeDir()
	if err != nil {
		return nil, err
	}

	if systemdAvailable() {
		serviceDir := filepath.Join(home, ".config", "systemd", "user")
		entries, err := os.ReadDir(serviceDir)
		if err == nil {
			for _, e := range entries {
				if strings.HasPrefix(e.Name(), "brain-daemon.") && strings.HasSuffix(e.Name(), ".service") {
					name := strings.TrimSuffix(e.Name(), ".service")
					projPath := ""
					if strings.Contains(name, ".") {
						parts := strings.SplitN(name, ".", 2)
						if len(parts) == 2 {
							hash := parts[1]
							projPath = fmt.Sprintf("(hash: %s)", hash)
						}
					}
					services = append(services, ServiceInfo{
						Name:    name,
						Status:  "unknown",
						Project: projPath,
					})
				}
			}
		}
	}

	return services, nil
}

func nohupAvailable() bool {
	_, err := exec.LookPath("nohup")
	return err == nil
}

func startBackground(execPath, workDir string) (int, string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return 0, "", fmt.Errorf("cannot determine cache directory: %w", err)
	}
	runDir := filepath.Join(cacheDir, "brain")
	if err := os.MkdirAll(runDir, 0700); err != nil {
		return 0, "", fmt.Errorf("cannot create cache directory: %w", err)
	}

	logPath := nohupLogPath(workDir)

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		logFile = os.Stdout
		logPath = "(stdout)"
	}

	if nohupAvailable() {
		scriptPath := nohupScriptPath(workDir)
		script := fmt.Sprintf("#!/bin/sh\nexec %q daemon run\n", execPath)
		if err := os.WriteFile(scriptPath, []byte(script), 0700); err != nil {
			return 0, "", fmt.Errorf("cannot write daemon script: %w", err)
		}

		cmd := exec.Command("nohup", scriptPath)
		cmd.Stdout = logFile
		cmd.Stderr = logFile
		cmd.Dir = workDir
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

		if err := cmd.Start(); err != nil {
			return 0, "", fmt.Errorf("cannot start daemon: %w", err)
		}
		return cmd.Process.Pid, logPath, nil
	}

	cmd := exec.Command(execPath, "daemon", "run")
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Dir = workDir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		return 0, "", fmt.Errorf("cannot start daemon: %w", err)
	}
	return cmd.Process.Pid, logPath, nil
}

func registerNohup(execPath, workDir string) error {
	pid, logPath, err := startBackground(execPath, workDir)
	if err != nil {
		return err
	}

	fmt.Printf("Daemon started (pid: %d).\n", pid)
	fmt.Printf("Log: %s\n", logPath)
	return nil
}

func startNohup(workDir string) error {
	execPath, err := os.Executable()
	if err != nil {
		execPath = "brain"
	}
	return registerNohup(execPath, workDir)
}

func stopNohup(workDir string) error {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return fmt.Errorf("cannot determine cache directory: %w", err)
	}
	lockDir := filepath.Join(cacheDir, "brain")
	pidFile := filepath.Join(lockDir, fmt.Sprintf("brain-daemon-%s.pid", projectHash(workDir)))

	data, err := os.ReadFile(pidFile)
	if err != nil {
		return fmt.Errorf("cannot read PID file: %w\nWhat to do: the daemon may not be running", err)
	}

	var pid int
	fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &pid)
	if pid <= 0 {
		return fmt.Errorf("invalid PID in lock file")
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("cannot find process %d: %w", pid, err)
	}

	if err := proc.Signal(syscall.Signal(0)); err != nil {
		os.Remove(pidFile)
		os.Remove(nohupScriptPath(workDir))
		return fmt.Errorf("daemon process %d is not running", pid)
	}

	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("cannot stop process %d: %w", pid, err)
	}

	os.Remove(pidFile)
	os.Remove(nohupScriptPath(workDir))
	return nil
}

func isProcessBrainDaemon(pid int) bool {
	cmdline, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return true
	}
	parts := strings.Split(string(cmdline), "\x00")
	for _, p := range parts {
		if strings.Contains(p, "brain") {
			return true
		}
	}
	return false
}

func isRunningNohup(workDir string) bool {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return false
	}
	lockDir := filepath.Join(cacheDir, "brain")
	pidFile := filepath.Join(lockDir, fmt.Sprintf("brain-daemon-%s.pid", projectHash(workDir)))

	data, err := os.ReadFile(pidFile)
	if err != nil {
		return false
	}

	var pid int
	fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &pid)
	if pid <= 0 {
		return false
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	if err := proc.Signal(syscall.Signal(0)); err != nil {
		return false
	}

	if !isProcessBrainDaemon(pid) {
		return false
	}

	return true
}
