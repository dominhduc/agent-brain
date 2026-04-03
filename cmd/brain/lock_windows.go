//go:build windows

package main

import "os"

func tryLockFile(f *os.File) error {
	return nil
}

func unlockFile(f *os.File) {
	os.Remove(f.Name())
}
