package xray

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

var ErrNotFound = errors.New("xray core not found alongside alexandria")

// set via ldflags for distro packages that drop xray in a libexec dir (e.g. /usr/lib/alexandria)
var libexecdir string

func binname() string {
	if runtime.GOOS == "windows" {
		return "xray.exe"
	}
	return "xray"
}

func bindir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "alexandria", "bin"), nil
}

func Locate() string {
	name := binname()
	if exe, err := os.Executable(); err == nil {
		if p := filepath.Join(filepath.Dir(exe), name); isexec(p) {
			return p
		}
	}
	if libexecdir != "" {
		if p := filepath.Join(libexecdir, name); isexec(p) {
			return p
		}
	}
	if d, err := bindir(); err == nil {
		if p := filepath.Join(d, name); isexec(p) {
			return p
		}
	}
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	return ""
}

func isexec(p string) bool {
	fi, err := os.Stat(p)
	if err != nil || fi.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return fi.Mode()&0111 != 0
}

func Ensure() (string, error) {
	if p := Locate(); p != "" {
		return p, nil
	}
	return "", ErrNotFound
}
