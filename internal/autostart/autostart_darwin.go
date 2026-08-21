package autostart

import (
	"bytes"
	"encoding/xml"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	agentlabel  = "org." + label + ".daemon"
	systemplist = "/Library/LaunchDaemons/" + agentlabel + ".plist"
)

func plistpath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents", agentlabel+".plist"), nil
}

func systeminstalled() bool {
	_, err := os.Stat(systemplist)
	return err == nil
}

func Enabled() bool {
	if systeminstalled() {
		return true
	}
	p, err := plistpath()
	if err != nil {
		return false
	}
	_, err = os.Stat(p)
	return err == nil
}

func Enable() error {
	exe, err := exepath()
	if err != nil {
		return err
	}
	if elevated() {
		if err := os.WriteFile(systemplist, []byte(plistdoc(exe, true)), 0644); err != nil {
			return err
		}
		_ = exec.Command("launchctl", "load", "-w", systemplist).Run()
		return nil
	}
	if systeminstalled() {
		return nil
	}
	p, err := plistpath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(p, []byte(plistdoc(exe, false)), 0644); err != nil {
		return err
	}
	_ = exec.Command("launchctl", "load", "-w", p).Run()
	return nil
}

func plistdoc(exe string, system bool) string {
	env := ""
	if system {
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			env = "\t<key>EnvironmentVariables</key>\n\t<dict>\n\t\t<key>HOME</key>\n\t\t<string>" + escape(home) + "</string>\n\t</dict>\n"
		}
	}
	return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>` + agentlabel + `</string>
	<key>ProgramArguments</key>
	<array>
		<string>` + escape(exe) + `</string>
		<string>--daemon</string>
		<string>--autoconnect</string>
	</array>
` + env + `	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<false/>
</dict>
</plist>
`
}

func escape(s string) string {
	var b bytes.Buffer
	if err := xml.EscapeText(&b, []byte(s)); err != nil {
		return s
	}
	return b.String()
}

func Disable() error {
	if systeminstalled() {
		if !elevated() {
			return ErrNeedsRoot
		}
		_ = exec.Command("launchctl", "unload", "-w", systemplist).Run()
		if err := os.Remove(systemplist); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	p, err := plistpath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(p); err != nil {
		return nil
	}
	_ = exec.Command("launchctl", "unload", "-w", p).Run()
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
