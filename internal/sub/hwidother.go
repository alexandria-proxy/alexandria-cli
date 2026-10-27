//go:build !linux && !darwin && !windows

package sub

func machineid() string { return "" }
