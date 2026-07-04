//go:build windows

package daemon

import (
	"os"
	"syscall"

	"golang.org/x/sys/windows"
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

func childattr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		CreationFlags: createnowindow,
		HideWindow:    true,
	}
}

func terminate(p *os.Process) error {
	return p.Kill()
}

func elevatehint() string {
	return "run your terminal as Administrator"
}

func iselevated() bool {
	var token windows.Token
	if err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &token); err != nil {
		return false
	}
	defer token.Close()
	return token.IsElevated()
}
