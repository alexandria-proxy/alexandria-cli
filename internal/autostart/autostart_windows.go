package autostart

import (
	"errors"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const (
	runkey    = `Software\Microsoft\Windows\CurrentVersion\Run`
	valuename = "Alexandria"
	taskname  = "Alexandria"
)

func launcherpath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, label, "autostart.vbs"), nil
}

func writelauncher(exe string) (string, error) {
	vbs, err := launcherpath()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(vbs), 0700); err != nil {
		return "", err
	}
	script := "CreateObject(\"WScript.Shell\").Run \"\"\"" + exe + "\"\" --daemon --autoconnect\", 0, False\r\n"
	if err := os.WriteFile(vbs, []byte(script), 0600); err != nil {
		return "", err
	}
	return vbs, nil
}

func taskinstalled() bool {
	return exec.Command("schtasks", "/query", "/tn", taskname).Run() == nil
}

func runkeyset() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, runkey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	v, _, err := k.GetStringValue(valuename)
	return err == nil && v != ""
}

func Enabled() bool {
	return taskinstalled() || runkeyset()
}

func Enable() error {
	exe, err := exepath()
	if err != nil {
		return err
	}
	vbs, err := writelauncher(exe)
	if err != nil {
		return err
	}
	if elevated() {
		who, err := user.Current()
		if err != nil {
			return err
		}
		out, err := exec.Command("schtasks", "/create", "/tn", taskname,
			"/tr", "wscript.exe \""+vbs+"\"",
			"/sc", "onlogon", "/ru", who.Username, "/rl", "highest", "/f").CombinedOutput()
		if err != nil {
			if msg := strings.TrimSpace(string(out)); msg != "" {
				return errors.New(msg)
			}
			return err
		}
		return nil
	}
	if taskinstalled() {
		return nil
	}
	k, _, err := registry.CreateKey(registry.CURRENT_USER, runkey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.SetStringValue(valuename, "wscript.exe \""+vbs+"\"")
}

func Disable() error {
	if taskinstalled() {
		if !elevated() {
			return ErrNeedsRoot
		}
		_ = exec.Command("schtasks", "/delete", "/tn", taskname, "/f").Run()
	}
	if k, err := registry.OpenKey(registry.CURRENT_USER, runkey, registry.SET_VALUE); err == nil {
		defer k.Close()
		_ = k.DeleteValue(valuename)
	}
	if vbs, err := launcherpath(); err == nil {
		_ = os.Remove(vbs)
	}
	return nil
}
