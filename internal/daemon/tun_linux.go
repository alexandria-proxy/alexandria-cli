//go:build linux

package daemon

import (
	"os/exec"
	"strconv"

	"github.com/alexandria-proxy/alexandria-cli/internal/singbox"
)

func tuncleanup(ifname string) {
	if ifname != "" {
		_ = exec.Command("ip", "link", "delete", ifname).Run()
	}

	table := strconv.Itoa(singbox.TableIndex)
	for _, fam := range []string{"-4", "-6"} {
		_ = exec.Command("ip", fam, "route", "flush", "table", table).Run()
		for pref := singbox.RuleIndex; pref < singbox.RuleIndex+16; pref++ {
			p := strconv.Itoa(pref)
			for i := 0; i < 4; i++ {
				if exec.Command("ip", fam, "rule", "del", "priority", p).Run() != nil {
					break
				}
			}
		}
	}
}
