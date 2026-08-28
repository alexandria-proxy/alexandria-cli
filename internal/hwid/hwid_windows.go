//go:build windows

package hwid

import (
	"strconv"

	"golang.org/x/sys/windows/registry"
)

func machineid() string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Cryptography`, registry.QUERY_VALUE|registry.WOW64_64KEY)
	if err != nil {
		return ""
	}
	defer k.Close()
	v, _, err := k.GetStringValue("MachineGuid")
	if err != nil {
		return ""
	}
	return v
}

func release() string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE)
	if err != nil {
		return "unknown"
	}
	defer k.Close()
	build, _, err := k.GetStringValue("CurrentBuild")
	if err != nil {
		return "unknown"
	}
	major := "10.0"
	if n, err := strconv.Atoi(build); err == nil && n >= 22000 {
		major = "11.0"
	}
	return major + "." + build
}
