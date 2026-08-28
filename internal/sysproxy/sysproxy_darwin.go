//go:build darwin

package sysproxy

import (
	"errors"
	"os/exec"
	"strconv"
	"strings"
)

func services() []string {
	out, err := exec.Command("networksetup", "-listallnetworkservices").Output()
	if err != nil {
		return nil
	}
	var list []string
	for i, line := range strings.Split(string(out), "\n") {
		s := strings.TrimSpace(line)
		if i == 0 || s == "" || strings.HasPrefix(s, "*") {
			continue
		}
		list = append(list, s)
	}
	return list
}

func run(args ...string) error {
	out, err := exec.Command("networksetup", args...).CombinedOutput()
	if err != nil {
		return errors.New(strings.TrimSpace(string(out)))
	}
	return nil
}

func enable(o Opts) error {
	svc := services()
	if len(svc) == 0 {
		return errors.New("no network services found")
	}
	port := strconv.Itoa(o.HTTP)
	socks := strconv.Itoa(o.Socks)
	for _, s := range svc {
		_ = run("-setwebproxy", s, o.Host, port)
		_ = run("-setsecurewebproxy", s, o.Host, port)
		_ = run("-setsocksfirewallproxy", s, o.Host, socks)
		_ = run("-setwebproxystate", s, "on")
		_ = run("-setsecurewebproxystate", s, "on")
		_ = run("-setsocksfirewallproxystate", s, "on")
	}
	return nil
}

func disable() error {
	for _, s := range services() {
		_ = run("-setwebproxystate", s, "off")
		_ = run("-setsecurewebproxystate", s, "off")
		_ = run("-setsocksfirewallproxystate", s, "off")
	}
	return nil
}
