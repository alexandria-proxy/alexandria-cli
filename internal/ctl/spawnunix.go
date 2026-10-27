//go:build unix

package ctl

import (
	"os"
	"syscall"
)

const hint = "rerun it with sudo"

func detachattr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}

func terminate(p *os.Process) error {
	return p.Signal(syscall.SIGTERM)
}
