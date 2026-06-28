param([switch]$Force)

$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

$Repo = if ($env:ALEXANDRIA_REPO) { $env:ALEXANDRIA_REPO } else { "alexandria-proxy/alexandria-cli" }
$AppDir = if ($env:ALEXANDRIA_HOME) { $env:ALEXANDRIA_HOME } else { "$env:LOCALAPPDATA\Alexandria" }
$ConfigDir = Join-Path $env:APPDATA "alexandria"

if ($env:ALEXANDRIA_FORCE -eq "1") { $Force = $true }

$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
	"AMD64" { "amd64" }
	"ARM64" { "arm64" }
	default { throw "unsupported arch: $env:PROCESSOR_ARCHITECTURE" }
}

function Stop-Alexandria {
	$root = $AppDir.ToLower()
	Get-Process -Name "alexandria-cli", "xray", "sing-box" -ErrorAction SilentlyContinue | Where-Object {
		$_.Path -and $_.Path.ToLower().StartsWith($root)
	} | ForEach-Object {
		Write-Host "stopping $($_.ProcessName) ..."
		Stop-Process -Id $_.Id -Force -ErrorAction SilentlyContinue
	}
	$pidfile = Join-Path $ConfigDir "control.pid"
	if (Test-Path $pidfile) {
		try {
			$procId = [int]((Get-Content $pidfile -ErrorAction Stop | Select-Object -First 1).Trim())
			Stop-Process -Id $procId -Force -ErrorAction SilentlyContinue
		} catch {}
		Remove-Item $pidfile -Force -ErrorAction SilentlyContinue
	}
	$sock = Join-Path $ConfigDir "control.sock"
	if (Test-Path $sock) { Remove-Item $sock -Force -ErrorAction SilentlyContinue }
	Start-Sleep -Milliseconds 500
}

function Sync-Tree($Src, $Dst) {
	New-Item -ItemType Directory -Force -Path $Dst | Out-Null
	$srcRoot = (Resolve-Path $Src).Path
	$dstRoot = (Resolve-Path $Dst).Path
	Get-ChildItem -Path $Src -Recurse -File | ForEach-Object {
		$rel = $_.FullName.Substring($srcRoot.Length).TrimStart('\')
		$target = Join-Path $Dst $rel
		$copy = $true
		if (Test-Path $target) {
			$a = (Get-FileHash $_.FullName -Algorithm SHA256).Hash
			$b = (Get-FileHash $target -Algorithm SHA256).Hash
			if ($a -eq $b) { $copy = $false }
		}
		if ($copy) {
			New-Item -ItemType Directory -Force -Path (Split-Path $target -Parent) | Out-Null
			Copy-Item $_.FullName $target -Force
			Write-Host "  updated $rel"
		}
	}
	Get-ChildItem -Path $Dst -Recurse -File | ForEach-Object {
		$rel = $_.FullName.Substring($dstRoot.Length).TrimStart('\')
		if (-not (Test-Path (Join-Path $Src $rel))) {
			Remove-Item $_.FullName -Force
			Write-Host "  removed $rel"
		}
	}
}

$version = $env:ALEXANDRIA_VERSION
if (-not $version) {
	$rel = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases?per_page=1"
	$version = $rel[0].tag_name
}
if (-not $version) { throw "could not find a release for $Repo" }
$version = $version.TrimStart('v')

$cli = Join-Path $AppDir "alexandria-cli.exe"
if (-not $Force -and (Test-Path $cli)) {
	$current = ""
	try { $current = (& $cli --version 2>$null) } catch {}
	$current = ("$current" -replace '^alexandria\s+', '').Trim()
	if ($current -and $current -eq $version) {
		Write-Host "alexandria-cli $version is already installed (use -Force to reinstall)"
		return
	}
}

$base = "https://github.com/$Repo/releases/download/v$version"
$archive = "alexandria-windows-$arch.zip"
$tmp = Join-Path ([System.IO.Path]::GetTempPath()) ([System.Guid]::NewGuid())
New-Item -ItemType Directory -Path $tmp | Out-Null

try {
	Write-Host "downloading $archive ..."
	Invoke-WebRequest -Uri "$base/$archive" -OutFile "$tmp\$archive"
	Invoke-WebRequest -Uri "$base/checksums.txt" -OutFile "$tmp\checksums.txt"

	$line = Select-String -Path "$tmp\checksums.txt" -SimpleMatch $archive | Select-Object -First 1
	$want = ($line.Line -split '\s+')[0].ToLower()
	$got = (Get-FileHash "$tmp\$archive" -Algorithm SHA256).Hash.ToLower()
	if (-not $want -or $want -ne $got) { throw "checksum verification failed for $archive" }

	$stage = Join-Path $tmp "stage"
	New-Item -ItemType Directory -Path $stage | Out-Null
	Expand-Archive -Path "$tmp\$archive" -DestinationPath $stage -Force

	Stop-Alexandria

	Write-Host "syncing into $AppDir ..."
	Sync-Tree $stage $AppDir

	$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
	if ($userPath -notlike "*$AppDir*") {
		[Environment]::SetEnvironmentVariable("Path", "$userPath;$AppDir", "User")
		Write-Host "added $AppDir to your PATH (restart the terminal)"
	}
	Write-Host "installed alexandria-cli $version to $AppDir (run: alexandria-cli)"
} finally {
	Remove-Item -Recurse -Force $tmp
}
