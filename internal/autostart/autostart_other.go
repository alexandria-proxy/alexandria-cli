//go:build !linux && !darwin && !windows

package autostart

import "errors"

func Enabled() bool { return false }

func Enable() error { return errors.New("autostart is not supported on this platform") }

func Disable() error { return nil }
