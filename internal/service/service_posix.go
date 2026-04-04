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

	exec.Command("systemctl", "--user", "daemon-reload").Run()
	exec.Command("systemctl", "--user", "enable", fmt.Sprintf("%s.service", name)).Run()
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

	return services, nil
}
