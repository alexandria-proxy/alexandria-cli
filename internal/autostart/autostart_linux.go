package autostart

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	unitname   = label + ".service"
	systemunit = "/etc/systemd/system/" + unitname
)

func unitpath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "systemd", "user", unitname), nil
}

func desktoppath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "autostart", label+".desktop"), nil
}

func hassystemd() bool {
	if _, err := exec.LookPath("systemctl"); err != nil {
		return false
	}
	_, err := os.Stat("/run/systemd/system")
	return err == nil
}

func exists(path string, err error) bool {
	if err != nil {
		return false
	}
	_, serr := os.Stat(path)
	return serr == nil
}

func systeminstalled() bool {
	_, err := os.Stat(systemunit)
	return err == nil
}

func Enabled() bool {
	return systeminstalled() || exists(unitpath()) || exists(desktoppath())
}

func Enable() error {
	exe, err := exepath()
	if err != nil {
		return err
	}
	if elevated() {
		return enablesystem(exe)
	}
	if systeminstalled() {
		return nil
	}
	if !hassystemd() {
		return writedesktop(exe)
	}
	p, err := unitpath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(p, []byte(unitfile(exe, false)), 0644); err != nil {
		return err
	}
	_ = exec.Command("systemctl", "--user", "daemon-reload").Run()
	if out, err := exec.Command("systemctl", "--user", "enable", unitname).CombinedOutput(); err != nil {
		_ = os.Remove(p)
		return cmderr(out, err)
	}
	return nil
}

func enablesystem(exe string) error {
	if !hassystemd() {
		return errors.New("systemd is required for system-wide autostart")
	}
	if err := os.WriteFile(systemunit, []byte(unitfile(exe, true)), 0644); err != nil {
		return err
	}
	_ = exec.Command("systemctl", "daemon-reload").Run()
	if out, err := exec.Command("systemctl", "enable", unitname).CombinedOutput(); err != nil {
		_ = os.Remove(systemunit)
		return cmderr(out, err)
	}
	return nil
}

func unitfile(exe string, system bool) string {
	b := &strings.Builder{}
	b.WriteString("[Unit]\nDescription=Alexandria proxy\n")
	if system {
		b.WriteString("After=network.target\n")
	}
	b.WriteString("\n[Service]\nType=simple\n")
	if system {
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			b.WriteString("Environment=HOME=" + home + "\n")
		}
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			b.WriteString("Environment=XDG_CONFIG_HOME=" + xdg + "\n")
		}
	}
	b.WriteString("ExecStart=\"" + exe + "\" --daemon --autoconnect\n\n[Install]\nWantedBy=")
	if system {
		b.WriteString("multi-user.target\n")
	} else {
		b.WriteString("default.target\n")
	}
	return b.String()
}

func cmderr(out []byte, err error) error {
	if msg := strings.TrimSpace(string(out)); msg != "" {
		return errors.New(msg)
	}
	return err
}

func writedesktop(exe string) error {
	p, err := desktoppath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	entry := "[Desktop Entry]\n" +
		"Type=Application\n" +
		"Name=Alexandria\n" +
		"Exec=\"" + exe + "\" --daemon --autoconnect\n" +
		"Terminal=false\n" +
		"X-GNOME-Autostart-enabled=true\n"
	return os.WriteFile(p, []byte(entry), 0644)
}

func Disable() error {
	if systeminstalled() {
		if !elevated() {
			return ErrNeedsRoot
		}
		if hassystemd() {
			_ = exec.Command("systemctl", "disable", unitname).Run()
		}
		if err := os.Remove(systemunit); err != nil && !os.IsNotExist(err) {
			return err
		}
		if hassystemd() {
			_ = exec.Command("systemctl", "daemon-reload").Run()
		}
	}
	if p, err := desktoppath(); err == nil {
		_ = os.Remove(p)
	}
	p, err := unitpath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(p); err != nil {
		return nil
	}
	if hassystemd() {
		_ = exec.Command("systemctl", "--user", "disable", unitname).Run()
	}
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return err
	}
	if hassystemd() {
		_ = exec.Command("systemctl", "--user", "daemon-reload").Run()
	}
	return nil
}
