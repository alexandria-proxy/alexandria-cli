//go:build !unix && !windows

package daemon

import "syscall"

func detachAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{}
}
