package cli

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/alexandria-proxy/alexandria-cli/internal/conf"
	"github.com/alexandria-proxy/alexandria-cli/internal/ctl"
	"github.com/alexandria-proxy/alexandria-cli/internal/ipc"
	"github.com/alexandria-proxy/alexandria-cli/internal/svc"
)

func autostartcmd() *cobra.Command {
	set := func(on bool) func(c *cobra.Command, args []string) error {
		return func(c *cobra.Command, args []string) error {
			if on && !svc.Supported() {
				return svc.Errnotsupported
			}
			if on && !svc.Installed() {
				if err := installsvc(); err != nil {
					return err
				}
			}
			cfg, err := loadcfg()
			if err != nil {
				return err
			}
			cfg.Autostart = on
			if err := conf.Save(cfg); err != nil {
				return err
			}
			if verbose(c) {
				say("autostart %s", onoff(on))
			}
			return nil
		}
	}
	on := &cobra.Command{
		Use:     "on [OPTION]...",
		Aliases: []string{"enable"},
		Short:   "reconnect to the last server at boot",
		Args:    nargs,
		RunE:    set(true),
	}
	off := &cobra.Command{
		Use:     "off [OPTION]...",
		Aliases: []string{"disable"},
		Short:   "stay disconnected at boot",
		Args:    nargs,
		RunE:    set(false),
	}
	stat := &cobra.Command{
		Use:     "stat",
		Aliases: []string{"status"},
		Short:   "print on or off",
		Args:    nargs,
		RunE: func(c *cobra.Command, args []string) error {
			cfg, err := loadcfg()
			if err != nil {
				return err
			}
			say("%s", onoff(cfg.Autostart && svc.Installed()))
			return nil
		},
	}
	addverbose(on)
	addverbose(off)
	return parent("autostart COMMAND", "reconnect at boot: on, off, stat",
		"The service starts at boot anyway; autostart decides whether it reconnects to\n"+
			"the last used server right away.", "settings", on, off, stat)
}

func svccmd() *cobra.Command {
	on := &cobra.Command{
		Use:     "on [OPTION]...",
		Aliases: []string{"install"},
		Short:   "install and start the system service",
		Long: "Install the alexandria system service (systemd, launchd or Task Scheduler) and\n" +
			"start it. Asks for your password once; 'conn' does this by itself.",
		Args: nargs,
		RunE: func(c *cobra.Command, args []string) error {
			if !svc.Supported() {
				return svc.Errnotsupported
			}
			if !svc.Installed() || conf.Elevated() {
				if err := installsvc(); err != nil {
					return err
				}
			}
			if err := ctl.Ready(); err != nil {
				return err
			}
			if verbose(c) {
				say("svc on")
			}
			return nil
		},
	}
	off := &cobra.Command{
		Use:     "off [OPTION]...",
		Aliases: []string{"uninstall"},
		Short:   "stop and remove the system service",
		Args:    nargs,
		RunE: func(c *cobra.Command, args []string) error {
			if !svc.Installed() {
				return nil
			}
			if err := elevated(ctl.Uninstall()); err != nil {
				return err
			}
			if verbose(c) {
				say("svc off")
			}
			return nil
		},
	}
	stat := &cobra.Command{
		Use:     "stat",
		Aliases: []string{"status"},
		Short:   "print on or off",
		Args:    nargs,
		RunE: func(c *cobra.Command, args []string) error {
			say("%s", onoff(svc.Installed()))
			return nil
		},
	}
	addverbose(on)
	addverbose(off)
	c := parent("svc COMMAND", "the root system service: on, off, stat",
		"The alexandria service runs the daemon as root (Administrator on Windows), so TUN\n"+
			"works without sudo. The first 'conn' installs it and asks for your password once.",
		"settings", on, off, stat)
	c.Aliases = []string{"service"}
	return c
}

func resetcmd() *cobra.Command {
	kinds := []struct{ use, kind, short, ask string }{
		{"user", "user", "wipe subscriptions, settings and routes", "wipe all subscriptions, settings and routes"},
		{"conf", "settings", "reset settings to their defaults", "reset every setting to its default"},
		{"tun", "tun", "drop the connection and clean TUN leftovers", "disconnect and clean up the TUN device and routes"},
	}
	var kids []*cobra.Command
	for _, k := range kinds {
		kids = append(kids, &cobra.Command{
			Use:   k.use + " [OPTION]...",
			Short: k.short,
			Args:  nargs,
			RunE: func(c *cobra.Command, args []string) error {
				force, _ := c.Flags().GetBool("force")
				if !force {
					ok, err := confirm(k.ask + "?")
					if err != nil {
						return err
					}
					if !ok {
						return nil
					}
				}
				if err := ctl.Ensure(); err != nil {
					return err
				}
				if _, err := ctl.Call(ipc.Request{Cmd: ipc.Cmdreset, Kind: k.kind}); err != nil {
					return err
				}
				if verbose(c) {
					say("reset %s", k.use)
				}
				return nil
			},
		})
	}
	c := parent("reset COMMAND", "reset things: user, conf, tun",
		"Reset user data, settings or the TUN state. Asks first unless --force is given.", "settings", kids...)
	c.PersistentFlags().BoolP("force", "f", false, "don't ask for confirmation")
	c.PersistentFlags().BoolP("verbose", "v", false, "explain what is being done")
	return c
}

func confirm(question string) (bool, error) {
	if !isterm(os.Stdin) {
		return false, errors.New("refusing to reset without a terminal to confirm, use --force")
	}
	fmt.Fprintf(stderr, "%s [y/N] ", question)
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true, nil
	}
	return false, nil
}

func editfile(path string) error {
	ed := os.Getenv("VISUAL")
	if ed == "" {
		ed = os.Getenv("EDITOR")
	}
	if ed == "" {
		ed = "vi"
		if runtime.GOOS == "windows" {
			ed = "notepad"
		}
	}
	parts := strings.Fields(ed)
	cmd := exec.Command(parts[0], append(parts[1:], path)...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("editor %s: %v", parts[0], err)
	}
	return nil
}
