//go:build unix

package daemon

import (
	"os"
	"syscall"
)

func detachattr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}

func terminate(p *os.Process) error {
	return p.Signal(syscall.SIGTERM)
}
