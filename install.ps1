# install.ps1 - install clibo, the online_clipboard CLI, on Windows.
#
# Usage (in PowerShell):
#   irm https://clipboard.lab.rm-info.fr/install.ps1 | iex
#
# Overrides (env vars):
#   $env:CLIBO_BASE  Source server (default: https://clipboard.lab.rm-info.fr).
#                    Point at a self-hosted instance to install from a fork.
#   $env:CLIBO_BIN   Install directory (default: $env:LOCALAPPDATA\Programs\clibo).
#                    Binary placed inside as clibo.exe; the directory is added
#                    to your user PATH on first install.

[Diagnostics.CodeAnalysis.SuppressMessageAttribute(
    'PSAvoidUsingWriteHost', '',
    Justification = 'Status messages for an interactive install script, like rustup/deno/bun. Write-Output would pollute the pipeline when invoked via "irm ... | iex".'
)]
Param()

$ErrorActionPreference = 'Stop'

if (-not [Environment]::Is64BitOperatingSystem) {
    Write-Error 'clibo: only 64-bit Windows is supported'
    exit 1
}

$base = if ($env:CLIBO_BASE) { $env:CLIBO_BASE } else { 'https://clipboard.lab.rm-info.fr' }
$installDir = if ($env:CLIBO_BIN) { $env:CLIBO_BIN } else { Join-Path $env:LOCALAPPDATA 'Programs\clibo' }
$exePath = Join-Path $installDir 'clibo.exe'
$url = "$base/cli/windows-amd64.exe"

New-Item -ItemType Directory -Force -Path $installDir | Out-Null

Write-Host "clibo: downloading $url"
try {
    Invoke-WebRequest -Uri $url -OutFile $exePath -UseBasicParsing
} catch {
    Write-Error "clibo: download failed: $_"
    exit 1
}

# Add install dir to the user PATH if not already present. User scope avoids
# needing admin and persists across reboots; the current process won't see
# the new PATH until a new terminal is opened.
$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
$pathEntries = if ($userPath) { $userPath.Split(';') } else { @() }
if ($pathEntries -notcontains $installDir) {
    $newPath = if ($userPath) { "$userPath;$installDir" } else { $installDir }
    [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
    Write-Host "clibo: added $installDir to user PATH (open a new terminal to pick it up)"
}

Write-Host "clibo: installed $exePath"
try {
    & $exePath --version
} catch {
    Write-Host 'clibo: installed binary did not respond to --version'
}
