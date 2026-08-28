//go:build linux

package sysproxy

import (
	"errors"
	"os/exec"
	"strconv"
)

const schema = "org.gnome.system.proxy"

func gsettings(args ...string) error {
	bin, err := exec.LookPath("gsettings")
	if err != nil {
		return errors.New("gsettings not found")
	}
	out, err := exec.Command(bin, args...).CombinedOutput()
	if err != nil {
		return errors.New(string(out))
	}
	return nil
}

func enable(o Opts) error {
	steps := [][]string{
		{"set", schema + ".http", "host", o.Host},
		{"set", schema + ".http", "port", strconv.Itoa(o.HTTP)},
		{"set", schema + ".https", "host", o.Host},
		{"set", schema + ".https", "port", strconv.Itoa(o.HTTP)},
		{"set", schema + ".socks", "host", o.Host},
		{"set", schema + ".socks", "port", strconv.Itoa(o.Socks)},
		{"set", schema, "ignore-hosts", "['localhost', '127.0.0.0/8', '::1']"},
		{"set", schema, "mode", "manual"},
	}
	for _, s := range steps {
		if err := gsettings(s...); err != nil {
			return err
		}
	}
	return nil
}

func disable() error {
	return gsettings("set", schema, "mode", "none")
}
