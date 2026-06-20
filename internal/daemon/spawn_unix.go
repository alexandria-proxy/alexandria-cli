//go:build unix

package daemon

import "syscall"

func detachattr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}
