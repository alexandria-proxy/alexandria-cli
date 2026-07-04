//go:build !unix && !windows

package daemon

import (
	"os"
	"syscall"
)

func detachattr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{}
}

func childattr() *syscall.SysProcAttr {
	return nil
}

func terminate(p *os.Process) error {
	return p.Kill()
}

func elevatehint() string {
	return "run alexandria with elevated privileges"
}

func iselevated() bool {
	return false
}
