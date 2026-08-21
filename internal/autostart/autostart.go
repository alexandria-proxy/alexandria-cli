package autostart

import (
	"errors"
	"os"
	"path/filepath"
)

const label = "alexandria"

var ErrNeedsRoot = errors.New("autostart is installed system-wide and needs root to change")

func exepath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		return real, nil
	}
	return exe, nil
}
