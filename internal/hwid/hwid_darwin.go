//go:build darwin

package hwid

import (
	"os/exec"
	"strings"
)

func machineid() string {
	out, err := exec.Command("ioreg", "-rd1", "-c", "IOPlatformExpertDevice").Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, "IOPlatformUUID") {
			continue
		}
		if _, v, ok := strings.Cut(line, "="); ok {
			return strings.Trim(strings.TrimSpace(v), `"`)
		}
	}
	return ""
}
