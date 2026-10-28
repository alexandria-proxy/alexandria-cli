package svc

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/alexandria-proxy/alexandria-cli/internal/conf"
)

const label = "alexandria"

var (
	Errnotsupported = errors.New("no systemd, launchd or Task Scheduler")
	Errneedsroot    = errors.New("installing the service needs root")
)

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

func Install() error {
	if !Supported() {
		return Errnotsupported
	}
	if !conf.Elevated() {
		return Errneedsroot
	}
	exe, err := exepath()
	if err != nil {
		return err
	}
	dir, err := conf.Mkdir()
	if err != nil {
		return err
	}
	home, err := conf.Home()
	if err != nil {
		return err
	}
	legacy(dir, home)
	return install(exe, dir, home)
}

func Uninstall() error {
	if !Installed() {
		return nil
	}
	if !conf.Elevated() {
		return Errneedsroot
	}
	return uninstall()
}
