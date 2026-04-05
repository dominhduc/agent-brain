//go:build linux

package tui

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

type termState struct {
	termios syscall.Termios
}

func CanUseRawMode() bool {
	var termios syscall.Termios
	_, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, os.Stdin.Fd(), uintptr(syscall.TCGETS), uintptr(unsafe.Pointer(&termios)), 0, 0, 0)
	return errno == 0
}

func EnableRawMode() (*termState, error) {
	fd := os.Stdin.Fd()
	var oldTermios syscall.Termios
	_, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, fd, uintptr(syscall.TCGETS), uintptr(unsafe.Pointer(&oldTermios)), 0, 0, 0)
	if errno != 0 {
		return nil, fmt.Errorf("TCGETS failed: %v", errno)
	}
	newTermios := oldTermios
	newTermios.Lflag &^= syscall.ECHO | syscall.ICANON
	newTermios.Cc[syscall.VMIN] = 0
	newTermios.Cc[syscall.VTIME] = 2
	_, _, errno = syscall.Syscall6(syscall.SYS_IOCTL, fd, uintptr(syscall.TCSETS), uintptr(unsafe.Pointer(&newTermios)), 0, 0, 0)
	if errno != 0 {
		return nil, fmt.Errorf("TCSETS failed: %v", errno)
	}
	return &termState{termios: oldTermios}, nil
}

func (s *termState) Restore() {
	fd := os.Stdin.Fd()
	syscall.Syscall6(syscall.SYS_IOCTL, fd, uintptr(syscall.TCSETS), uintptr(unsafe.Pointer(&s.termios)), 0, 0, 0)
}

func GetTerminalSize() (int, int, error) {
	type winsize struct {
		Row, Col, Xpixel, Ypixel uint16
	}
	var ws winsize
	_, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, os.Stdout.Fd(), uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(&ws)), 0, 0, 0)
	if errno != 0 {
		return 80, 24, nil
	}
	return int(ws.Col), int(ws.Row), nil
}
