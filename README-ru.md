<p align="center">
  <a href="https://github.com/alexandria-proxy/alexandria-cli" target="_blank" rel="noopener noreferrer">
    <img width="200" height="200" src="assets/logo.png" alt="Alexandria">
  </a>
</p>

<h1 align="center">Alexandria</h1>

<p align="center">
    <strong>Лёгкий, устойчивый к цензуре клиент Xray-core для твоего терминала</strong>
</p>

---

<p align="center">
    <a href="https://github.com/alexandria-proxy/alexandria-cli/releases"><img src="https://img.shields.io/github/v/release/alexandria-proxy/alexandria-cli?include_prereleases&style=flat-square&color=6C7BFF" alt="Release" /></a>
    <a href="LICENSE"><img src="https://img.shields.io/github/license/alexandria-proxy/alexandria-cli?style=flat-square&color=orange" alt="License" /></a>
    <a href="#установка"><img src="https://img.shields.io/badge/platform-linux%20%7C%20macOS%20%7C%20windows-blue?style=flat-square" alt="Platforms" /></a>
    <a href="https://t.me/Alexandriavpn"><img src="https://img.shields.io/badge/telegram-join-26A5E4?style=flat-square&logo=telegram&logoColor=white" alt="Telegram" /></a>
    <a href="https://github.com/alexandria-proxy/alexandria-cli/stargazers"><img src="https://img.shields.io/github/stars/alexandria-proxy/alexandria-cli?style=flat-square&color=DAA520" alt="Stars" /></a>
</p>

<p align="center">
  <a href="./README.md">🇺🇸 English</a>
  /
  <a href="./README-fa.md">🇮🇷 فارسی</a>
</p>

<p align="center">
  <img src="assets/screenshot.png" alt="Alexandria screenshot" width="760">
</p>

## Содержание

> **Быстрая навигация** — переходи к нужному разделу

-   [Обзор](#обзор)
-   [Получить сервер](#получить-сервер)
-   [Установка](#установка)
-   [Документация](#документация)

---

# Обзор

> **Что такое Alexandria?**

Alexandria — это клиент в одном бинарнике для подключения через [Xray-core](https://github.com/XTLS/Xray-core). Запускаешь `alexandria-cli` и попадаешь в интерактивный TUI. Прокси продолжает работать в фоновом демоне даже после того, как ты закрыл панель, и переподключается, когда ты снова её открываешь.

---

# Получить сервер

> **Нужна подписка?**

Мы ещё и **VPN-провайдер** на Xray. Вы можете оформить подписку в нашем Telegram-боте:

[![Купить подписку](https://img.shields.io/badge/Купить%20подписку-Telegram-26A5E4?style=for-the-badge&logo=telegram&logoColor=white)](https://t.me/alexandriavpnbot)

---

# Установка

> **Быстрый старт** — Alexandria за пару секунд

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

### Сборка из исходников

Нужен Go 1.26+. `scripts/fetch-core.sh` подтягивает проверенное ядро Xray, чтобы бинарник нашёл его при запуске.

```bash
git clone --depth 1 https://github.com/alexandria-proxy/alexandria-cli
cd alexandria-cli
bash scripts/fetch-core.sh
go build -o alexandria-cli .
./alexandria-cli
```

Свежий бинарник ещё не в `PATH`, поэтому запускай через `./`. Для TUN-режима — `sudo ./alexandria-cli`.

### После установки

<div align="left">

**Установщик** скачивает готовый архив из [Releases](https://github.com/alexandria-proxy/alexandria-cli/releases), проверяет его по `checksums.txt` и кладёт `alexandria-cli` + встроенный `xray` в каталог, добавленный в PATH.

**Запуск:**

```bash
alexandria-cli
```

> **TUN-режим** требует повышенных прав. Запускай Alexandria через `sudo` (Linux / macOS) или из терминала с правами **администратора** (Windows). Proxy-режим работает от обычного пользователя.

**Файлы лежат в** `~/.config/alexandria/`

</div>

---

# Документация

<div align="left">

**Читай это руководство на своём языке:**

🇺🇸 **[English](README.md)**

🇮🇷 **[فارسی](README-fa.md)**

🇷🇺 **[Русский](README-ru.md)**

</div>

> **Контрибьютинг:** issues и PR приветствуются на [GitHub](https://github.com/alexandria-proxy/alexandria-cli).

---

<p align="center">
  <em>Resistance. Peace.</em>
</p>
