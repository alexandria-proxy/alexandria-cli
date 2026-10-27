package core

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/alexandria-proxy/alexandria-cli/internal/conf"
)

var (
	Errnotfound        = errors.New("xray core not found alongside alexandria")
	Errsingboxnotfound = errors.New("sing-box not found alongside alexandria")
)

var libexecdir string

func exename(base string) string {
	if runtime.GOOS == "windows" {
		return base + ".exe"
	}
	return base
}

func bindir() (string, error) {
	return conf.File("bin")
}

func find(base string) string {
	name := exename(base)
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

func Xray() (string, error) {
	if p := find("xray"); p != "" {
		return p, nil
	}
	return "", Errnotfound
}

func Singbox() (string, error) {
	if p := find("sing-box"); p != "" {
		return p, nil
	}
	return "", Errsingboxnotfound
}
