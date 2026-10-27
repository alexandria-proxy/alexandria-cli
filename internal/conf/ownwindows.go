//go:build windows

package conf

import "golang.org/x/sys/windows"

func Elevated() bool {
	var token windows.Token
	if err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &token); err != nil {
		return false
	}
	defer token.Close()
	return token.IsElevated()
}

func Owner() (int, int, bool) { return 0, 0, false }

func Own(string) {}
