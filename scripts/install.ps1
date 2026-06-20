$ErrorActionPreference = "Stop"

$Repo = if ($env:ALEXANDRIA_REPO) { $env:ALEXANDRIA_REPO } else { "alexandria-proxy/alexandria-cli" }
$AppDir = if ($env:ALEXANDRIA_HOME) { $env:ALEXANDRIA_HOME } else { "$env:LOCALAPPDATA\Alexandria" }

$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
	"AMD64" { "amd64" }
	"ARM64" { "arm64" }
	default { throw "unsupported arch: $env:PROCESSOR_ARCHITECTURE" }
}

$version = $env:ALEXANDRIA_VERSION
if ($version) {
	$base = "https://github.com/$Repo/releases/download/v$($version.TrimStart('v'))"
} else {
	$base = "https://github.com/$Repo/releases/latest/download"
}

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

	New-Item -ItemType Directory -Force -Path $AppDir | Out-Null
	Expand-Archive -Path "$tmp\$archive" -DestinationPath $AppDir -Force

	$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
	if ($userPath -notlike "*$AppDir*") {
		[Environment]::SetEnvironmentVariable("Path", "$userPath;$AppDir", "User")
		Write-Host "added $AppDir to your PATH (restart the terminal)"
	}
	Write-Host "installed alexandria-cli to $AppDir (run: alexandria-cli)"
} finally {
	Remove-Item -Recurse -Force $tmp
}
