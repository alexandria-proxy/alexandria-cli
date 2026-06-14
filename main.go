package main

import (
	_ "embed"
	"flag"
	"fmt"
	"os"

	"github.com/alexandria-proxy/alexandria-cli/internal/tui"
)

const version = "0.0.1-dev"

//go:embed assets/logo.txt
var logo string

func main() {
	daemonMode := flag.Bool("daemon", false, "run the background daemon (internal use)")
	showVersion := flag.Bool("version", false, "print version and bail")
	flag.Parse()

	if *showVersion {
		fmt.Println("alexandria", version)
		return
	}

	if *daemonMode {
		fmt.Println("daemon mode: not wired up yet")
		os.Exit(0)
	}

	if _, err := tui.RunLangPicker(logo); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
