package main

import (
	"os"

	"github.com/alexandria-proxy/alexandria-cli/internal/cli"
	"github.com/alexandria-proxy/alexandria-cli/internal/core"
	"github.com/alexandria-proxy/alexandria-cli/internal/ipc"
	"github.com/alexandria-proxy/alexandria-cli/internal/sub"
)

var version = "0.20"

func main() {
	sub.Version = version
	ipc.Appversion = version
	os.Exit(cli.Execute(version, core.Versions()))
}
