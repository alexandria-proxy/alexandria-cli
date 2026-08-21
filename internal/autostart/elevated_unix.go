//go:build unix

package autostart

import "os"

func elevated() bool {
	return os.Geteuid() == 0
}
