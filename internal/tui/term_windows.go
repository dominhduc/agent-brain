//go:build windows

package tui

type termState struct{}

func CanUseRawMode() bool              { return true }
func EnableRawMode() (*termState, error) { return &termState{}, nil }
func (s *termState) Restore()           {}
func GetTerminalSize() (int, int, error) { return 80, 24, nil }
