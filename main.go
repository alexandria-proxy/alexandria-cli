package main

import (
	_ "embed"
	"flag"
	"fmt"
	"os"

	"github.com/alexandria-proxy/alexandria-cli/internal/config"
	"github.com/alexandria-proxy/alexandria-cli/internal/daemon"
	"github.com/alexandria-proxy/alexandria-cli/internal/tui"
)

const version = "0.0.7-dev"

//go:embed assets/logo.txt
var logo string

//go:embed assets/clilogo_mono.txt
var menuLogoMono string

//go:embed assets/clilogo.txt
var menuLogoColor string

func main() {
	daemonMode := flag.Bool("daemon", false, "run the background daemon (internal use)")
	showVersion := flag.Bool("version", false, "print version and bail")
	flag.Parse()

	if *showVersion {
		fmt.Println("alexandria", version)
		return
	}

	if *daemonMode {
		if err := daemon.Run(); err != nil {
			fmt.Fprintln(os.Stderr, "daemon:", err)
			os.Exit(1)
		}
		return
	}

	cfg, _ := config.Load()

	if cfg.Lang == "" {
		lang, err := tui.RunLangPicker(logo)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		if lang == "" {
			return // bailed without picking
		}
		cfg.Lang = lang
		if err := config.Save(cfg); err != nil {
			fmt.Fprintln(os.Stderr, "warning: couldn't save config:", err)
		}
	}

	if err := tui.RunMenu(cfg.Lang, menuLogoMono, menuLogoColor); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
