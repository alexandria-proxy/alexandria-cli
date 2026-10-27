//go:build !unix && !windows

package conf

func Elevated() bool { return false }

func Owner() (int, int, bool) { return 0, 0, false }

func Own(string) {}
