//go:build windows

package daemon

import (
	"os"
	"syscall"
)

const (
	detachedprocess       = 0x00000008
	createnewprocessgroup = 0x00000200
	createnowindow        = 0x08000000
)

func detachattr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		CreationFlags: detachedprocess | createnewprocessgroup | createnowindow,
		HideWindow:    true,
	}
}

func terminate(p *os.Process) error {
	return p.Kill()
}

func iselevated() bool {
	return false
}
