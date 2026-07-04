//go:build unix

package daemon

import (
	"os"
	"syscall"
)

func detachattr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}

func childattr() *syscall.SysProcAttr {
	return nil
}

func terminate(p *os.Process) error {
	return p.Signal(syscall.SIGTERM)
}

func elevatehint() string {
	return "run alexandria with sudo/doas"
}

func iselevated() bool {
	return os.Geteuid() == 0
}
