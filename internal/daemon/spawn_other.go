//go:build !unix && !windows

package daemon

import (
	"os"
	"syscall"
)

func detachattr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{}
}

func terminate(p *os.Process) error {
	return p.Kill()
}

func iselevated() bool {
	return false
}
