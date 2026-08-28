//go:build !linux && !darwin && !windows

package hwid

func machineid() string { return "" }

func release() string { return "unknown" }
