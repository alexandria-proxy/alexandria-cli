//go:build windows

package sysproxy

import (
	"strconv"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const inetpath = `Software\Microsoft\Windows\CurrentVersion\Internet Settings`

func refresh() {
	dll, err := windows.LoadDLL("wininet.dll")
	if err != nil {
		return
	}
	defer dll.Release()
	proc, err := dll.FindProc("InternetSetOptionW")
	if err != nil {
		return
	}
	proc.Call(0, 39, 0, 0)
	proc.Call(0, 37, 0, 0)
}

func openkey() (registry.Key, error) {
	return registry.OpenKey(registry.CURRENT_USER, inetpath, registry.SET_VALUE)
}

func enable(o Opts) error {
	k, err := openkey()
	if err != nil {
		return err
	}
	defer k.Close()

	server := o.Host + ":" + strconv.Itoa(o.HTTP)
	if err := k.SetStringValue("ProxyServer", server); err != nil {
		return err
	}
	if err := k.SetStringValue("ProxyOverride", "localhost;127.*;10.*;172.16.*;192.168.*;<local>"); err != nil {
		return err
	}
	if err := k.SetDWordValue("ProxyEnable", 1); err != nil {
		return err
	}
	refresh()
	return nil
}

func disable() error {
	k, err := openkey()
	if err != nil {
		return err
	}
	defer k.Close()
	if err := k.SetDWordValue("ProxyEnable", 0); err != nil {
		return err
	}
	refresh()
	return nil
}
