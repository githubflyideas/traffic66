//go:build !linux && !darwin && !windows

package store

func totalMemory() uint64 { return 4 << 30 }
