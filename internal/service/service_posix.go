//go:build !windows

package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func registerLaunchd(execPath string) error {
	home, err := homeDir()
	if err != nil {
		return err
	}

	plistDir := filepath.Join(home, "Library", "LaunchAgents")
	if err := os.MkdirAll(plistDir, 0755); err != nil {
		return err
	}

	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.dominhduc.brain-daemon</string>
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
    <string>/tmp/brain-daemon.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/brain-daemon.err</string>
</dict>
</plist>`, execPath)

	plistPath := filepath.Join(plistDir, "com.dominhduc.brain-daemon.plist")
	if err := os.WriteFile(plistPath, []byte(plist), 0644); err != nil {
		return err
	}
	return exec.Command("launchctl", "load", plistPath).Run()
}

func registerSystemd(execPath string) error {
	home, err := homeDir()
	if err != nil {
		return err
	}

	serviceDir := filepath.Join(home, ".config", "systemd", "user")
	if err := os.MkdirAll(serviceDir, 0755); err != nil {
		return err
	}

	service := fmt.Sprintf(`[Unit]
Description=agent-brain Daemon
After=network.target
StartLimitIntervalSec=0

[Service]
Type=simple
ExecStart=%s daemon run
Restart=always
RestartSec=5

[Install]
WantedBy=default.target`, execPath)

	servicePath := filepath.Join(serviceDir, "brain-daemon.service")
	if err := os.WriteFile(servicePath, []byte(service), 0644); err != nil {
		return err
	}

	exec.Command("systemctl", "--user", "daemon-reload").Run()
	exec.Command("systemctl", "--user", "enable", "brain-daemon.service").Run()
	return exec.Command("systemctl", "--user", "start", "brain-daemon.service").Run()
}

func startLaunchd() error {
	home, err := homeDir()
	if err != nil {
		return err
	}
	plistPath := filepath.Join(home, "Library", "LaunchAgents", "com.dominhduc.brain-daemon.plist")
	return exec.Command("launchctl", "load", plistPath).Run()
}

func startSystemd() error {
	return exec.Command("systemctl", "--user", "start", "brain-daemon.service").Run()
}

func stopLaunchd() error {
	home, err := homeDir()
	if err != nil {
		return err
	}
	plistPath := filepath.Join(home, "Library", "LaunchAgents", "com.dominhduc.brain-daemon.plist")
	return exec.Command("launchctl", "unload", plistPath).Run()
}

func stopSystemd() error {
	return exec.Command("systemctl", "--user", "stop", "brain-daemon.service").Run()
}
