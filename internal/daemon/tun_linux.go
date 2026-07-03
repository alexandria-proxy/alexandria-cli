//go:build linux

package daemon

import "os/exec"

func tuncleanup(ifname string) {
	if ifname == "" {
		return
	}
	_ = exec.Command("ip", "link", "delete", ifname).Run()
}
