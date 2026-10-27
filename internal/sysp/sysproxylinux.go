//go:build linux

package sysp

import (
	"errors"
	"os"
	"os/exec"
	"strconv"
	"syscall"

	"github.com/alexandria-proxy/alexandria-cli/internal/conf"
)

const schema = "org.gnome.system.proxy"

func gsettings(args ...string) error {
	bin, err := exec.LookPath("gsettings")
	if err != nil {
		return errors.New("gsettings not found")
	}
	cmd := exec.Command(bin, args...)
	asowner(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.New(string(out))
	}
	return nil
}

func asowner(cmd *exec.Cmd) {
	if os.Geteuid() != 0 {
		return
	}
	uid, gid, ok := conf.Owner()
	if !ok {
		return
	}
	run := "/run/user/" + strconv.Itoa(uid)
	cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid)}}
	cmd.Env = append(os.Environ(), "XDG_RUNTIME_DIR="+run, "DBUS_SESSION_BUS_ADDRESS=unix:path="+run+"/bus")
	if home, err := conf.Home(); err == nil {
		cmd.Env = append(cmd.Env, "HOME="+home)
	}
}

func enable(o Opts) error {
	steps := [][]string{
		{"set", schema + ".http", "host", o.Host},
		{"set", schema + ".http", "port", strconv.Itoa(o.Http)},
		{"set", schema + ".https", "host", o.Host},
		{"set", schema + ".https", "port", strconv.Itoa(o.Http)},
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
