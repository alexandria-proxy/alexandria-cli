//go:build linux

package svc

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	unitname  = label + ".service"
	unitpath  = "/etc/systemd/system/" + unitname
	Starthint = "try: sudo systemctl restart " + label
)

func Supported() bool {
	if _, err := exec.LookPath("systemctl"); err != nil {
		return false
	}
	_, err := os.Stat("/run/systemd/system")
	return err == nil
}

func Installed() bool {
	b, err := os.ReadFile(unitpath)
	return err == nil && strings.Contains(string(b), "--service")
}

func unitfile(exe, dir, home string) string {
	return "[Unit]\n" +
		"Description=Alexandria proxy\n" +
		"After=network-online.target\n" +
		"Wants=network-online.target\n\n" +
		"[Service]\n" +
		"Type=simple\n" +
		"Environment=HOME=" + home + "\n" +
		"ExecStart=\"" + exe + "\" --daemon --service --dir \"" + dir + "\"\n" +
		"Restart=always\n" +
		"RestartSec=1\n\n" +
		"[Install]\n" +
		"WantedBy=multi-user.target\n"
}

func systemctl(args ...string) error {
	out, err := exec.Command("systemctl", args...).CombinedOutput()
	if err != nil {
		if msg := strings.TrimSpace(string(out)); msg != "" {
			return errors.New(msg)
		}
		return err
	}
	return nil
}

func install(exe, dir, home string) error {
	if err := os.WriteFile(unitpath, []byte(unitfile(exe, dir, home)), 0644); err != nil {
		return err
	}
	if err := systemctl("daemon-reload"); err != nil {
		return err
	}
	if err := systemctl("enable", unitname); err != nil {
		return err
	}
	return systemctl("restart", unitname)
}

func uninstall() error {
	_ = systemctl("disable", "--now", unitname)
	if err := os.Remove(unitpath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return systemctl("daemon-reload")
}

func Start() error { return nil }

func legacy(dir, home string) {
	base := filepath.Dir(dir)
	for _, p := range []string{
		filepath.Join(base, "systemd", "user", unitname),
		filepath.Join(base, "systemd", "user", "default.target.wants", unitname),
		filepath.Join(base, "autostart", label+".desktop"),
	} {
		_ = os.Remove(p)
	}
}
