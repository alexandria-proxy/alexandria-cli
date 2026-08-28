//go:build linux

package hwid

import (
	"os"
	"strings"
)

func machineid() string {
	sources := []string{
		"/etc/machine-id",
		"/var/lib/dbus/machine-id",
		"/sys/class/dmi/id/product_uuid",
		"/sys/class/dmi/id/board_serial",
	}
	for _, p := range sources {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		v := strings.TrimSpace(string(b))
		if v != "" && v != "None" && v != "0" {
			return v
		}
	}
	return ""
}
