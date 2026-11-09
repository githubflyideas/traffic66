//go:build !linux && !darwin && !windows

package tui

import "errors"

type rawState struct{}

func makeRaw() (*rawState, error) { return nil, errors.New("terminal UI is not supported on this OS") }
func (s *rawState) restore()      {}
func size() (int, int)            { return 100, 32 }
func isTerminal() bool            { return false }
