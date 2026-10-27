# Alexandria

Command-line client for [Xray-core](https://github.com/XTLS/Xray-core) with a background daemon, TUN mode and subscription management.

## Get a Server

[Alexandria VPN Bot](https://t.me/alexandriavpnbot)

## License

[Mozilla Public License Version 2.0](https://github.com/alexandria-proxy/alexandria-cli/blob/master/LICENSE)

## Telegram

[Alexandria](https://t.me/Alexandriavpn)

## Installation

- Linux / macOS
  - `curl -fsSL https://raw.githubusercontent.com/alexandria-proxy/alexandria-cli/master/sh/install.sh | sh`
- Windows
  - `irm https://raw.githubusercontent.com/alexandria-proxy/alexandria-cli/master/sh/install.ps1 | iex`
- Arch Linux
  - `yay -S alexandria-cli`

## Usage

- Connection
  - `conn [-t|-p] [-b] [SERVER]`
  - `dconn`
  - `stat [-j] [-q] [-w]`
- Servers
  - `ls [-s ping|name] [-r] [-u] [-j] [SUB]...`
  - `ping [-j] [SUB]...`
- Subscriptions
  - `sub add|ls|up|rm|info`
- Logs
  - `logs [-f] [-n NUM] [FILTER]...`
- Settings
  - `conf ls|get|set|unset|edit|path`
  - `route ls|add|rm|edit|path`
  - `autostart on|off|stat`
  - `svc on|off|stat`
  - `reset [-f] user|conf|tun`

Reference: `alexandria-cli --help`, `alexandria-cli COMMAND --help`.

## TUN Mode

The first `conn` installs a system service (systemd, launchd or Task Scheduler) and requests elevation once. Subsequent commands run unprivileged. xray runs as the invoking user; only sing-box retains root.

## Files

- Linux: `~/.config/alexandria/`
- macOS: `~/Library/Application Support/alexandria/`
- Windows: `%AppData%\alexandria\`

## Credits

- [Xray-core](https://github.com/XTLS/Xray-core)
- [sing-box](https://github.com/SagerNet/sing-box)
- Go modules: [go.mod](https://github.com/alexandria-proxy/alexandria-cli/blob/master/go.mod)

### Bundled Third-Party Components Redistribution

**Release archives include unmodified third-party binaries distributed under their own licenses. Versions and checksums are pinned in [internal/core/manifest.json](https://github.com/alexandria-proxy/alexandria-cli/blob/master/internal/core/manifest.json).**

#### Xray-core

- Project: Xray-core
- License: Mozilla Public License Version 2.0
- Component(s): xray, geoip.dat, geosite.dat
- Source: https://github.com/XTLS/Xray-core

#### sing-box

- Project: sing-box
- License: GNU General Public License v3.0 or later
- Component(s): sing-box
- Source: https://github.com/SagerNet/sing-box

## One-line Compilation

### Linux / macOS

```bash
CGO_ENABLED=0 go build -o alexandria-cli -trimpath -ldflags="-s -w" .
```

### Windows (PowerShell)

```powershell
$env:CGO_ENABLED=0
go build -o alexandria-cli.exe -trimpath -ldflags="-s -w" .
```

### Cores

```bash
bash sh/fetchcore.sh
```
