//go:build windows

package dmn

import (
	"os"
	"syscall"
)

const createnowindow = 0x08000000

func childattr(bool) *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: createnowindow, HideWindow: true}
}

func terminate(p *os.Process) error {
	return p.Kill()
}
