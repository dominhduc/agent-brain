package service

import (
	"testing"
)

func TestHomeDir(t *testing.T) {
	home, err := homeDir()
	if err != nil {
		t.Fatalf("homeDir() returned error: %v", err)
	}
	if home == "" {
		t.Fatal("homeDir() returned empty string")
	}
}

func TestRegister_UnsupportedOS(t *testing.T) {
	err := Register("/fake/path", "/fake/workdir")
	_ = err
}

func TestStop_WithWorkDir(t *testing.T) {
	_ = Stop("/fake/workdir")
}

func TestIsRunning(t *testing.T) {
	_ = IsRunning("/fake/workdir")
}

func TestServiceName(t *testing.T) {
	name := serviceName("/some/path")
	if name == "" {
		t.Error("serviceName should return non-empty string")
	}
}
