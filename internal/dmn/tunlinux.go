//go:build linux

package dmn

import (
	"os/exec"
	"strconv"

	"github.com/alexandria-proxy/alexandria-cli/internal/gen"
)

func tuncleanup(ifname string) {
	if ifname != "" {
		_ = exec.Command("ip", "link", "delete", ifname).Run()
	}

	table := strconv.Itoa(gen.Tableindex)
	for _, fam := range []string{"-4", "-6"} {
		_ = exec.Command("ip", fam, "route", "flush", "table", table).Run()
		for pref := gen.Ruleindex; pref < gen.Ruleindex+16; pref++ {
			p := strconv.Itoa(pref)
			for i := 0; i < 4; i++ {
				if exec.Command("ip", fam, "rule", "del", "priority", p).Run() != nil {
					break
				}
			}
		}
	}

	for _, fam := range []string{"inet", "ip", "ip6"} {
		_ = exec.Command("nft", "delete", "table", fam, "sing-box").Run()
	}
}
