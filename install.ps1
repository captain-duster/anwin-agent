$ErrorActionPreference = "Stop"

$GithubUser = "captain-duster"
$Repo = "anwin-agent"
$Version = "v1.0.0"
$BaseUrl = "https://github.com/$GithubUser/$Repo/releases/download/$Version"
$InstallDir = "$env:LOCALAPPDATA\anwin-agent"
$BinaryName = "anwin-agent.exe"

Write-Host ""
Write-Host "  +---------------------------------+"
Write-Host "  |     ANWIN Agent Installer       |"
Write-Host "  |     Version $Version            |"
Write-Host "  +---------------------------------+"
Write-Host ""

$OSArch = (Get-WmiObject Win32_OperatingSystem).OSArchitecture
$CPUArch = (Get-WmiObject Win32_Processor).Architecture

if ($OSArch -like "*64*") {
    if ($CPUArch -eq 12) {
        $File = "anwin-agent-windows-arm64.exe"
    } else {
        $File = "anwin-agent-windows-amd64.exe"
    }
} else {
    $File = "anwin-agent-windows-386.exe"
}

Write-Host "  Detected: Windows / $OSArch"
Write-Host "  Downloading: $File"
Write-Host ""

New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

$DownloadUrl = "$BaseUrl/$File"
$Destination = "$InstallDir\$BinaryName"

Invoke-WebRequest -Uri $DownloadUrl -OutFile $Destination

$CurrentPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($CurrentPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("PATH", "$CurrentPath;$InstallDir", "User")
    Write-Host "  Added to PATH"
}

Write-Host "  OK ANWIN Agent installed successfully"
Write-Host ""
Write-Host "  IMPORTANT: Close and reopen your terminal, then run:"
Write-Host "    anwin-agent setup"
Write-Host ""