//go:build windows

package svc

import (
	"bytes"
	"encoding/binary"
	"encoding/xml"
	"errors"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"unicode/utf16"

	"golang.org/x/sys/windows/registry"
)

const (
	taskname  = "Alexandria"
	runkey    = `Software\Microsoft\Windows\CurrentVersion\Run`
	Starthint = "try: schtasks /run /tn " + taskname
)

func Supported() bool { return true }

func launcher(dir string) string { return filepath.Join(dir, "service.vbs") }

func taskexists() bool {
	return exec.Command("schtasks", "/query", "/tn", taskname).Run() == nil
}

func Installed() bool {
	dir, err := configdir()
	if err != nil {
		return false
	}
	b, err := os.ReadFile(launcher(dir))
	return err == nil && strings.Contains(string(b), "--service") && taskexists()
}

func escape(s string) string {
	var b bytes.Buffer
	if err := xml.EscapeText(&b, []byte(s)); err != nil {
		return s
	}
	return b.String()
}

func taskxml(who, vbs string) []byte {
	doc := `<?xml version="1.0" encoding="UTF-16"?>
<Task version="1.2" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">
  <Triggers>
    <LogonTrigger>
      <Enabled>true</Enabled>
      <UserId>` + escape(who) + `</UserId>
    </LogonTrigger>
  </Triggers>
  <Principals>
    <Principal id="Author">
      <UserId>` + escape(who) + `</UserId>
      <LogonType>InteractiveToken</LogonType>
      <RunLevel>HighestAvailable</RunLevel>
    </Principal>
  </Principals>
  <Settings>
    <MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy>
    <DisallowStartIfOnBatteries>false</DisallowStartIfOnBatteries>
    <StopIfGoingOnBatteries>false</StopIfGoingOnBatteries>
    <AllowStartOnDemand>true</AllowStartOnDemand>
    <ExecutionTimeLimit>PT0S</ExecutionTimeLimit>
    <Hidden>true</Hidden>
  </Settings>
  <Actions Context="Author">
    <Exec>
      <Command>wscript.exe</Command>
      <Arguments>"` + escape(vbs) + `"</Arguments>
    </Exec>
  </Actions>
</Task>
`
	var b bytes.Buffer
	_ = binary.Write(&b, binary.LittleEndian, uint16(0xFEFF))
	_ = binary.Write(&b, binary.LittleEndian, utf16.Encode([]rune(doc)))
	return b.Bytes()
}

func schtasks(args ...string) error {
	out, err := exec.Command("schtasks", args...).CombinedOutput()
	if err != nil {
		if msg := strings.TrimSpace(string(out)); msg != "" {
			return errors.New(msg)
		}
		return err
	}
	return nil
}

func install(exe, dir, home string) error {
	who, err := user.Current()
	if err != nil {
		return err
	}
	vbs := launcher(dir)
	script := "CreateObject(\"WScript.Shell\").Run \"\"\"" + exe + "\"\" --daemon --service --dir \"\"" + dir + "\"\"\", 0, False\r\n"
	if err := os.WriteFile(vbs, []byte(script), 0600); err != nil {
		return err
	}
	tmp, err := os.CreateTemp("", "alexandria-task-*.xml")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(taskxml(who.Username, vbs)); err != nil {
		tmp.Close()
		return err
	}
	tmp.Close()
	if err := schtasks("/create", "/tn", taskname, "/xml", tmp.Name(), "/f"); err != nil {
		return err
	}
	return Start()
}

func uninstall() error {
	err := schtasks("/delete", "/tn", taskname, "/f")
	if dir, derr := configdir(); derr == nil {
		_ = os.Remove(launcher(dir))
	}
	return err
}

func Start() error {
	return schtasks("/run", "/tn", taskname)
}

func legacy(dir, home string) {
	if k, err := registry.OpenKey(registry.CURRENT_USER, runkey, registry.SET_VALUE); err == nil {
		_ = k.DeleteValue(taskname)
		k.Close()
	}
	_ = os.Remove(filepath.Join(dir, "autostart.vbs"))
}
