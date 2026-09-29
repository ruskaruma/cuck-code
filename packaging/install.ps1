# Cuck Code installer for Windows.
#
#   irm https://raw.githubusercontent.com/ruskaruma/cuck-code/main/packaging/install.ps1 | iex
#
# Environment:
#   CUCK_VERSION      release tag to install (default: latest)
#   CUCK_INSTALL_DIR  where to put cuck.exe (default: %LOCALAPPDATA%\Programs\cuck)
#   CUCK_SKIP_SETUP   set to 1 to skip the "get cucked?" question
$ErrorActionPreference = 'Stop'

$repo = 'ruskaruma/cuck-code'
$dir = if ($env:CUCK_INSTALL_DIR) { $env:CUCK_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA 'Programs\cuck' }
$arch = if ($env:PROCESSOR_ARCHITECTURE -eq 'ARM64') { 'arm64' } else { 'amd64' }
$asset = "cuck-windows-$arch.exe"
$base = if ($env:CUCK_VERSION) { "https://github.com/$repo/releases/download/$($env:CUCK_VERSION)" } else { "https://github.com/$repo/releases/latest/download" }

$tmp = Join-Path ([IO.Path]::GetTempPath()) ([Guid]::NewGuid())
New-Item -ItemType Directory -Path $tmp | Out-Null
try {
    Write-Host "Downloading $asset..."
    Invoke-WebRequest "$base/$asset" -OutFile (Join-Path $tmp 'cuck.exe') -UseBasicParsing
    Invoke-WebRequest "$base/checksums.txt" -OutFile (Join-Path $tmp 'checksums.txt') -UseBasicParsing

    $line = Get-Content (Join-Path $tmp 'checksums.txt') | Where-Object { $_ -match " $([regex]::Escape($asset))$" }
    $want = ($line -split ' ')[0]
    $got = (Get-FileHash (Join-Path $tmp 'cuck.exe') -Algorithm SHA256).Hash.ToLower()
    if (-not $want -or $want -ne $got) { throw "checksum mismatch for $asset" }

    New-Item -ItemType Directory -Force -Path $dir | Out-Null
    Move-Item -Force (Join-Path $tmp 'cuck.exe') (Join-Path $dir 'cuck.exe')
} finally {
    Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
}

$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if (($userPath -split ';') -notcontains $dir) {
    [Environment]::SetEnvironmentVariable('Path', "$dir;$userPath", 'User')
    Write-Host "Added $dir to your PATH (new terminals will see it)."
}
$env:Path = "$dir;$env:Path"
Write-Host "Installed $(& (Join-Path $dir 'cuck.exe') --version) to $dir"

if (-not $env:CUCK_SKIP_SETUP) {
    & (Join-Path $dir 'cuck.exe') setup
}
