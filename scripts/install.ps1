# traffic66 one-line install for Windows (PowerShell 5.1 or later).
#
#   irm https://raw.githubusercontent.com/githubflyideas/traffic66/main/scripts/install.ps1 | iex
#
# Downloads the latest release into .\traffic66 (or $env:TRAFFIC66_DIR) and
# starts the collector, which prints the address and UDP ports to point your
# devices at. $env:TRAFFIC66_RUN replaces "serve" (for example
# "serve -listen netflow=:9995"); $env:TRAFFIC66_NO_START = 1 only downloads.
# $env:TRAFFIC66_VERSION = "v2.3.3" picks a release.
$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
$repo = 'githubflyideas/traffic66'
$dir = if ($env:TRAFFIC66_DIR) { $env:TRAFFIC66_DIR } else { Join-Path (Get-Location) 'traffic66' }
$url = if ($env:TRAFFIC66_VERSION) { "https://github.com/$repo/releases/download/$($env:TRAFFIC66_VERSION)/traffic66-windows-amd64.zip" }
       else { "https://github.com/$repo/releases/latest/download/traffic66-windows-amd64.zip" }
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
$zip = Join-Path ([IO.Path]::GetTempPath()) 'traffic66-windows-amd64.zip'
Write-Host "traffic66: downloading $url"
Invoke-WebRequest -UseBasicParsing $url -OutFile $zip
$tmp = Join-Path ([IO.Path]::GetTempPath()) ('traffic66-' + [guid]::NewGuid())
Expand-Archive $zip $tmp -Force
$inner = Get-ChildItem $tmp -Directory | Select-Object -First 1
$src = if ($inner -and (Test-Path (Join-Path $inner.FullName 'traffic66.exe'))) { $inner.FullName } else { $tmp }
New-Item -ItemType Directory -Force $dir | Out-Null
Copy-Item (Join-Path $src '*') $dir -Recurse -Force
Remove-Item $zip, $tmp -Recurse -Force
Get-ChildItem $dir -Recurse | Unblock-File
& (Join-Path $dir 'traffic66.exe') version
if ($env:TRAFFIC66_NO_START) {
    Write-Host "traffic66: installed in $dir; start it with: cd $dir; .\traffic66.exe"
} else {
    $run = if ($env:TRAFFIC66_RUN) { $env:TRAFFIC66_RUN } else { 'serve' }
    Set-Location $dir
    & .\traffic66.exe ($run -split ' ')
}
