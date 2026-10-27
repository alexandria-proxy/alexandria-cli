//go:build !unix && !windows

package ctl

import (
	"os"
	"syscall"
)

const hint = "rerun it with elevated privileges"

func detachattr() *syscall.SysProcAttr { return &syscall.SysProcAttr{} }

func terminate(p *os.Process) error { return p.Kill() }
