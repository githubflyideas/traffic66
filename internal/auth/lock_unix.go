//go:build !windows

package auth

import (
	"os"
	"path/filepath"
	"syscall"
)

// lockDir takes an exclusive lock that other processes respect too (the
// passwd command next to a running service), for one change of the file.
func lockDir(dir string) func() {
	f, err := os.OpenFile(filepath.Join(dir, FileName+".lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return func() {}
	}
	if syscall.Flock(int(f.Fd()), syscall.LOCK_EX) != nil {
		f.Close()
		return func() {}
	}
	return func() { syscall.Flock(int(f.Fd()), syscall.LOCK_UN); f.Close() }
}
