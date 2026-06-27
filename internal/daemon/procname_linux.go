//go:build linux

package daemon

import (
	"syscall"
	"unsafe"
)

const prsetname = 15

func setprocname(name string) {
	b := append([]byte(name), 0)
	_, _, _ = syscall.Syscall(syscall.SYS_PRCTL, prsetname, uintptr(unsafe.Pointer(&b[0])), 0)
}
