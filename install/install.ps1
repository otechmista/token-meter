$ErrorActionPreference = "Stop"

$installDir = Join-Path $env:LOCALAPPDATA "TokenMeter\bin"
$exePath = Join-Path $installDir "tkm.exe"
$tmpPath = Join-Path $installDir "tkm.exe.tmp"
$url = "https://github.com/otechmista/token-meter/releases/latest/download/tkm-windows-amd64.exe"
$isUpdate = Test-Path $exePath

Write-Host "TokenMeter install/update"
Write-Host "Target: $exePath"

New-Item -ItemType Directory -Force $installDir | Out-Null
Invoke-WebRequest -Uri $url -OutFile $tmpPath
Move-Item -Force $tmpPath $exePath
Unblock-File $exePath

$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$installDir*") {
  [Environment]::SetEnvironmentVariable("Path", "$userPath;$installDir", "User")
  Write-Host "Added TokenMeter to user Path."
  Write-Host "Close and reopen PowerShell before using tkm."
}

if ($isUpdate) {
  Write-Host "Updated existing TokenMeter."
} else {
  Write-Host "Installed TokenMeter."
}

& $exePath --top 0 (Get-Location).Path
Write-Host ""
Write-Host "Run: tkm ."
