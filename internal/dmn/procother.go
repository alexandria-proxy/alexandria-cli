//go:build !unix && !windows

package dmn

import (
	"os"
	"syscall"
)

func childattr(bool) *syscall.SysProcAttr { return nil }

func terminate(p *os.Process) error { return p.Kill() }
