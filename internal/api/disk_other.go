//go:build !linux && !darwin && !freebsd && !windows

package api

func diskFree(string) uint64 { return 0 }
