package daemon

import (
	"crypto/tls"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

const (
	probedial    = 4 * time.Second
	probewait    = 6 * time.Second
	probebudget  = 12 * time.Second
	uplinkbudget = 90 * time.Second
	uplinkpoll   = time.Second
	uplinksettle = 2 * time.Second
)

var probetargets = []string{"1.1.1.1:443", "8.8.8.8:443", "9.9.9.9:443"}

func healthy(mode string, socks int) bool {
	res := make(chan bool, len(probetargets))
	for _, t := range probetargets {
		go func(target string) {
			ok := probeproxy(socks, target)
			if ok && mode == "tun" {
				ok = probeplain(target)
			}
			res <- ok
		}(t)
	}

	deadline := time.After(probebudget)
	for range probetargets {
		select {
		case ok := <-res:
			if ok {
				return true
			}
		case <-deadline:
			return false
		}
	}
	return false
}

func probeproxy(socks int, target string) bool {
	if socks <= 0 {
		return false
	}
	c, err := net.DialTimeout("tcp", "127.0.0.1:"+strconv.Itoa(socks), probedial)
	if err != nil {
		return false
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(probewait))
	if !socksconnect(c, target) {
		return false
	}
	return shakes(c)
}

func probeplain(target string) bool {
	c, err := net.DialTimeout("tcp", target, probedial)
	if err != nil {
		return false
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(probewait))
	return shakes(c)
}

func shakes(c net.Conn) bool {
	return tls.Client(c, &tls.Config{InsecureSkipVerify: true}).Handshake() == nil
}

func socksconnect(c net.Conn, target string) bool {
	host, portstr, err := net.SplitHostPort(target)
	if err != nil {
		return false
	}
	port, err := strconv.Atoi(portstr)
	if err != nil || port <= 0 {
		return false
	}
	ip := net.ParseIP(host).To4()
	if ip == nil {
		return false
	}
	if _, err := c.Write([]byte{5, 1, 0}); err != nil {
		return false
	}
	greet := make([]byte, 2)
	if _, err := io.ReadFull(c, greet); err != nil || greet[0] != 5 || greet[1] != 0 {
		return false
	}
	req := append([]byte{5, 1, 0, 1}, ip...)
	req = append(req, byte(port>>8), byte(port))
	if _, err := c.Write(req); err != nil {
		return false
	}
	head := make([]byte, 4)
	if _, err := io.ReadFull(c, head); err != nil || head[0] != 5 || head[1] != 0 {
		return false
	}
	n := 0
	switch head[3] {
	case 1:
		n = 4
	case 4:
		n = 16
	case 3:
		size := make([]byte, 1)
		if _, err := io.ReadFull(c, size); err != nil {
			return false
		}
		n = int(size[0])
	default:
		return false
	}
	_, err = io.ReadFull(c, make([]byte, n+2))
	return err == nil
}

func waituplink(quit chan struct{}, budget time.Duration) {
	deadline := time.Now().Add(budget)
	for !uplinkready() {
		if time.Now().After(deadline) {
			return
		}
		select {
		case <-quit:
			return
		case <-time.After(uplinkpoll):
		}
	}
	select {
	case <-quit:
	case <-time.After(uplinksettle):
	}
}

func uplinkready() bool {
	ifs, err := net.Interfaces()
	if err != nil {
		return true
	}
	for _, i := range ifs {
		if i.Flags&net.FlagUp == 0 || i.Flags&net.FlagLoopback != 0 || istun(i.Name) {
			continue
		}
		addrs, err := i.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			n, ok := a.(*net.IPNet)
			if !ok || n.IP.IsLoopback() || n.IP.IsLinkLocalUnicast() || n.IP.IsUnspecified() {
				continue
			}
			return true
		}
	}
	return false
}

func istun(name string) bool {
	low := strings.ToLower(name)
	return strings.HasPrefix(low, "tun") || strings.HasPrefix(low, "utun") ||
		strings.Contains(low, "wintun") || strings.Contains(low, "sing-box") ||
		strings.Contains(low, "singbox")
}
