//go:build !linux && !darwin && !windows

package capture

import "errors"

func open(string) (source, error) {
	return nil, errors.New("local capture is not supported on this OS")
}
func Interfaces() ([]string, error) {
	return nil, errors.New("local capture is not supported on this OS")
}
