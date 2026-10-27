//go:build unix

package dmn

import (
	"os"
	"syscall"

	"github.com/alexandria-proxy/alexandria-cli/internal/conf"
)

func childattr(drop bool) *syscall.SysProcAttr {
	if !drop || !conf.Elevated() {
		return nil
	}
	uid, gid, ok := conf.Owner()
	if !ok {
		return nil
	}
	return &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid)}}
}

func terminate(p *os.Process) error {
	return p.Signal(syscall.SIGTERM)
}
