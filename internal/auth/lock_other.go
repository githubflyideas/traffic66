//go:build windows

package auth

// lockDir: on Windows changes are serialised within the process only.
func lockDir(string) func() { return func() {} }
