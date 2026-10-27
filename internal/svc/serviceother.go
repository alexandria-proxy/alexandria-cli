//go:build !linux && !darwin && !windows

package svc

const Starthint = "start it with: sudo alexandria-cli conn"

func Supported() bool { return false }

func Installed() bool { return false }

func install(exe, dir, home string) error { return Errnotsupported }

func uninstall() error { return Errnotsupported }

func Start() error { return nil }

func legacy(dir, home string) {}
