//go:build unix

package conf

import (
	"os"
	"path/filepath"
	"syscall"
)

func Elevated() bool {
	return os.Geteuid() == 0
}

func Owner() (int, int, bool) {
	d, err := Dir()
	if err != nil {
		return 0, 0, false
	}
	fi, err := os.Stat(filepath.Dir(d))
	if err != nil {
		return 0, 0, false
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok || st.Uid == 0 {
		return 0, 0, false
	}
	return int(st.Uid), int(st.Gid), true
}

func Own(path string) {
	if os.Geteuid() != 0 {
		return
	}
	if uid, gid, ok := Owner(); ok {
		_ = os.Lchown(path, uid, gid)
	}
}
