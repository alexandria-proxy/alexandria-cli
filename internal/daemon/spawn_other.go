//go:build !unix && !windows

package daemon

import "syscall"

func detachattr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{}
}
