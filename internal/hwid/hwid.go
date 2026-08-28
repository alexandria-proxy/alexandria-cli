package hwid

import (
	"crypto/sha256"
	"encoding/hex"
	"net"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
)

var (
	once   sync.Once
	cached string
)

var virtualprefix = []string{
	"docker", "br-", "virbr", "veth", "vmnet", "vboxnet",
	"tun", "tap", "wg", "utun", "zt", "ham", "lo",
}

func ID() string {
	once.Do(func() {
		raw := machineid()
		if raw == "" {
			raw = hardwarefallback()
		}
		if raw == "" {
			raw = hostfallback()
		}
		sum := sha256.Sum256([]byte("alexandria:" + strings.ToLower(strings.TrimSpace(raw))))
		cached = hex.EncodeToString(sum[:])[:32]
	})
	return cached
}

func Version() string {
	return release()
}

func OS() string {
	switch runtime.GOOS {
	case "linux":
		return "Linux"
	case "windows":
		return "Windows"
	case "darwin":
		return "MacOS"
	case "freebsd":
		return "FreeBSD"
	case "openbsd":
		return "OpenBSD"
	case "netbsd":
		return "NetBSD"
	}
	return strings.ToUpper(runtime.GOOS[:1]) + runtime.GOOS[1:]
}

func virtual(name string) bool {
	low := strings.ToLower(name)
	for _, p := range virtualprefix {
		if strings.HasPrefix(low, p) {
			return true
		}
	}
	return false
}

func hardwarefallback() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	macs := []string{}
	for _, i := range ifaces {
		if i.Flags&net.FlagLoopback != 0 || len(i.HardwareAddr) == 0 || virtual(i.Name) {
			continue
		}
		macs = append(macs, i.HardwareAddr.String())
	}
	sort.Strings(macs)
	return strings.Join(macs, ",")
}

func hostfallback() string {
	host, _ := os.Hostname()
	return host
}
