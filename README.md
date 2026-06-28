<p align="center">
  <a href="https://github.com/alexandria-proxy/alexandria-cli" target="_blank" rel="noopener noreferrer">
    <img width="200" height="200" src="assets/logo.png" alt="Alexandria">
  </a>
</p>

<h1 align="center">Alexandria</h1>

<p align="center">
    <strong>A lightweight, censorship-resistant Xray-core client for your terminal</strong>
</p>

---

<p align="center">
    <a href="https://github.com/alexandria-proxy/alexandria-cli/releases"><img src="https://img.shields.io/github/v/release/alexandria-proxy/alexandria-cli?include_prereleases&style=flat-square&color=6C7BFF" alt="Release" /></a>
    <a href="LICENSE"><img src="https://img.shields.io/github/license/alexandria-proxy/alexandria-cli?style=flat-square&color=orange" alt="License" /></a>
    <a href="#installation-guide"><img src="https://img.shields.io/badge/platform-linux%20%7C%20macOS%20%7C%20windows-blue?style=flat-square" alt="Platforms" /></a>
    <a href="https://t.me/Alexandriavpn"><img src="https://img.shields.io/badge/telegram-join-26A5E4?style=flat-square&logo=telegram&logoColor=white" alt="Telegram" /></a>
    <a href="https://github.com/alexandria-proxy/alexandria-cli/stargazers"><img src="https://img.shields.io/github/stars/alexandria-proxy/alexandria-cli?style=flat-square&color=DAA520" alt="Stars" /></a>
</p>

<p align="center">
  <a href="./README-fa.md">🇮🇷 فارسی</a>
  /
  <a href="./README-ru.md">🇷🇺 Русский</a>
</p>

<p align="center">
  <img src="assets/screenshot.png" alt="Alexandria screenshot" width="760">
</p>

## Table of Contents

> **Quick Navigation** - Jump to any section below

-   [Overview](#overview)
-   [Get a server](#get-a-server)
-   [Installation guide](#installation-guide)
-   [Documentation](#documentation)

---

# Overview

> **What is Alexandria?**

Alexandria is a single-binary client for connecting through [Xray-core](https://github.com/XTLS/Xray-core). Run `alexandria-cli` and you drop into an interactive TUI. The proxy keeps running in a background daemon even after you close the panel, and reconnects when you open it again.

---

# Get a server

> **Need a subscription?**

We're an Xray **VPN provider** too. Grab a subscription from our Telegram bot and you're ready to connect:

[![Buy a subscription](https://img.shields.io/badge/Buy%20a%20subscription-Telegram-26A5E4?style=for-the-badge&logo=telegram&logoColor=white)](https://t.me/alexandriavpnbot)

---

# Installation guide

> **Quick Start** - Get Alexandria running in seconds

### Linux / macOS

```bash
curl -fsSL https://raw.githubusercontent.com/alexandria-proxy/alexandria-cli/main/scripts/install.sh | sh
```

### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/alexandria-proxy/alexandria-cli/main/scripts/install.ps1 | iex
```

### Arch Linux

```bash
yay -S alexandria-cli
```

### Build from source

Needs Go 1.26+. `scripts/fetch-core.sh` fetches the vetted Xray core so the binary finds it at runtime.

```bash
git clone --depth 1 https://github.com/alexandria-proxy/alexandria-cli
cd alexandria-cli
bash scripts/fetch-core.sh
go build -o alexandria-cli .
./alexandria-cli
```

The fresh binary isn't on your `PATH` yet, so run it with `./`. For TUN mode use `sudo ./alexandria-cli`.

### After installation

<div align="left">

**The installer** pulls the prebuilt archive from [Releases](https://github.com/alexandria-proxy/alexandria-cli/releases), verifies it against `checksums.txt`, and drops `alexandria-cli` + the bundled `xray` into a PATH-wired prefix.

**Run it:**

```bash
alexandria-cli
```

> **TUN mode** needs elevated privileges. Start Alexandria with `sudo` (Linux / macOS) or from an **Administrator** terminal (Windows). Proxy mode runs fine as a normal user.

**Files are located at** `~/.config/alexandria/`

</div>

---

# Documentation

<div align="left">

**Read this guide in your language:**

🇺🇸 **[English](README.md)**

🇮🇷 **[فارسی](README-fa.md)**

🇷🇺 **[Русский](README-ru.md)**

</div>

> **Contributing:** issues and PRs are welcome on [GitHub](https://github.com/alexandria-proxy/alexandria-cli).

---

<p align="center">
  <em>Resistance. Peace.</em>
</p>
