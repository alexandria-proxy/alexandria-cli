//go:build !linux && !darwin && !windows

package sysproxy

import "errors"

func enable(Opts) error { return errors.New("system proxy is not supported on this platform") }

func disable() error { return nil }
