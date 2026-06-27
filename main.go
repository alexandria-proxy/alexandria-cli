package main

import (
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/alexandria-proxy/alexandria-cli/internal/config"
	"github.com/alexandria-proxy/alexandria-cli/internal/daemon"
	"github.com/alexandria-proxy/alexandria-cli/internal/tui"
)

var version = "rc-0.1"

//go:embed core/manifest.json
var coremanifest []byte

func coreversion() string {
	var m struct {
		Version string `json:"version"`
	}
	if json.Unmarshal(coremanifest, &m) == nil && m.Version != "" {
		return m.Version
	}
	return "unknown"
}

//go:embed assets/logo.txt
var logo string

//go:embed assets/clilogo_mono.txt
var menulogomono string

//go:embed assets/clilogo.txt
var menulogocolor string

func main() {
	daemonmode := flag.Bool("daemon", false, "run the background daemon (internal use)")
	showversion := flag.Bool("version", false, "print version and bail")
	flag.Parse()

	if *showversion {
		fmt.Println("alexandria", version)
		return
	}

	if *daemonmode || os.Getenv("ALEXANDRIA_DAEMON") == "1" {
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

	if err := tui.RunMenu(cfg.Lang, menulogomono, menulogocolor); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
