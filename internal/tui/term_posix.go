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
	fd := os.Stdin.Fd()
	if !isTerminal(fd) {
		return false
	}
	return isStdinReadable(fd)
}

func isTerminal(fd uintptr) bool {
	var termios syscall.Termios
	_, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, fd, uintptr(syscall.TCGETS), uintptr(unsafe.Pointer(&termios)), 0, 0, 0)
	return errno == 0
}

func isStdinReadable(fd uintptr) bool {
	oldFlags, _, errno := syscall.Syscall(syscall.SYS_FCNTL, fd, syscall.F_GETFL, 0)
	if errno != 0 {
		return false
	}
	syscall.Syscall(syscall.SYS_FCNTL, fd, syscall.F_SETFL, oldFlags|syscall.O_NONBLOCK)
	defer syscall.Syscall(syscall.SYS_FCNTL, fd, syscall.F_SETFL, oldFlags)

	buf := make([]byte, 1)
	n, _, errno := syscall.Syscall(syscall.SYS_READ, fd, uintptr(unsafe.Pointer(&buf[0])), 1)
	if errno != 0 {
		return false
	}
	if n > 0 {
		syscall.Syscall(syscall.SYS_WRITE, fd, uintptr(unsafe.Pointer(&buf[0])), n)
	}
	return true
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
