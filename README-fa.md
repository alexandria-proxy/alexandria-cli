<p align="center">
  <a href="https://github.com/alexandria-proxy/alexandria-cli" target="_blank" rel="noopener noreferrer">
    <img width="200" height="200" src="assets/logo.png" alt="Alexandria">
  </a>
</p>

<h1 align="center">Alexandria</h1>

<p align="center">
    <strong>کلاینتی سبک و مقاوم در برابر سانسور برای Xray-core در ترمینال شما</strong>
</p>

---

<p align="center">
    <a href="https://github.com/alexandria-proxy/alexandria-cli/releases"><img src="https://img.shields.io/github/v/release/alexandria-proxy/alexandria-cli?include_prereleases&style=flat-square&color=6C7BFF" alt="Release" /></a>
    <a href="LICENSE"><img src="https://img.shields.io/github/license/alexandria-proxy/alexandria-cli?style=flat-square&color=orange" alt="License" /></a>
    <a href="#نصب"><img src="https://img.shields.io/badge/platform-linux%20%7C%20macOS%20%7C%20windows-blue?style=flat-square" alt="Platforms" /></a>
    <a href="https://t.me/Alexandriavpn"><img src="https://img.shields.io/badge/telegram-join-26A5E4?style=flat-square&logo=telegram&logoColor=white" alt="Telegram" /></a>
    <a href="https://github.com/alexandria-proxy/alexandria-cli/stargazers"><img src="https://img.shields.io/github/stars/alexandria-proxy/alexandria-cli?style=flat-square&color=DAA520" alt="Stars" /></a>
</p>

<p align="center">
  <a href="./README.md">🇺🇸 English</a>
  /
  <a href="./README-ru.md">🇷🇺 Русский</a>
</p>

<p align="center">
  <img src="assets/screenshot.png" alt="Alexandria screenshot" width="760">
</p>

<div dir="rtl">

## فهرست مطالب

> **پیمایش سریع** — به هر بخش که خواستید بروید

-   [معرفی](#معرفی)
-   [تهیه سرور](#تهیه-سرور)
-   [نصب](#نصب)
-   [مستندات](#مستندات)

---

# معرفی

> **Alexandria چیست؟**

‏Alexandria یک کلاینت تک‌فایلی برای اتصال از طریق [Xray-core](https://github.com/XTLS/Xray-core) است. کافی است `alexandria-cli` را اجرا کنید تا وارد یک رابط متنی تعاملی (TUI) شوید. پراکسی در یک سرویس پس‌زمینه (daemon) فعال می‌ماند، حتی پس از بستن پنل، و هنگام باز کردن دوباره، اتصال را برقرار می‌کند.

---

# تهیه سرور

> **به اشتراک نیاز دارید؟**

ما خودمان هم **ارائه‌دهندهٔ VPN** مبتنی بر Xray هستیم. از ربات تلگرام ما یک اشتراک بگیرید و آمادهٔ اتصال شوید:

[![خرید اشتراک](https://img.shields.io/badge/خرید%20اشتراک-Telegram-26A5E4?style=for-the-badge&logo=telegram&logoColor=white)](https://t.me/alexandriavpnbot)

---

# نصب

> **شروع سریع** — Alexandria در چند ثانیه

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

### ساخت از سورس

به Go نسخهٔ ۱.۲۶ به بالا نیاز دارید. اسکریپت `scripts/fetch-core.sh` هستهٔ بررسی‌شدهٔ Xray را دریافت می‌کند تا برنامه هنگام اجرا آن را پیدا کند.

```bash
git clone --depth 1 https://github.com/alexandria-proxy/alexandria-cli
cd alexandria-cli
bash scripts/fetch-core.sh
go build -o alexandria-cli .
./alexandria-cli
```

فایل تازه‌ساخته‌شده هنوز در `PATH` نیست، پس آن را با `./` اجرا کنید. برای حالت TUN از `sudo ./alexandria-cli` استفاده کنید.

### پس از نصب

**نصب‌کننده** آرشیو از پیش ساخته‌شده را از [Releases](https://github.com/alexandria-proxy/alexandria-cli/releases) دریافت می‌کند، آن را با `checksums.txt` بررسی می‌کند و `alexandria-cli` به‌همراه `xray` همراه را در مسیری که به PATH افزوده شده قرار می‌دهد.

**اجرا:**

```bash
alexandria-cli
```

> **حالت TUN** به دسترسی بالا نیاز دارد. Alexandria را با `sudo` (در Linux / macOS) یا از ترمینال با دسترسی **Administrator** (در Windows) اجرا کنید. حالت پراکسی با کاربر عادی هم کار می‌کند.

فایل‌ها در `~/.config/alexandria/` قرار دارند.

---

# مستندات

**این راهنما را به زبان خود بخوانید:**

🇺🇸 **[English](README.md)**

🇮🇷 **[فارسی](README-fa.md)**

🇷🇺 **[Русский](README-ru.md)**

> **مشارکت:** ثبت issue و ارسال PR در [GitHub](https://github.com/alexandria-proxy/alexandria-cli) خوش‌آمد است.

</div>

---

<p align="center">
  <em>Resistance. Peace.</em>
</p>
