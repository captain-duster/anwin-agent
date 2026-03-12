[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$OutputEncoding = [System.Text.Encoding]::UTF8
$ErrorActionPreference = "Stop"

$GithubUser = "captain-duster"
$Repo = "anwin-agent"
$Version = "v1.0.0"
$BaseUrl = "https://github.com/$GithubUser/$Repo/releases/download/$Version"
$InstallDir = "$env:USERPROFILE\.local\bin"
$BinaryName = "anwin-agent.exe"

Write-Host ""
Write-Host "  ╔═════════════════════════════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "  ║     █████╗ ███╗   ██╗██╗    ██╗██╗███╗   ██╗    █████╗ ██╗      ║" -ForegroundColor Cyan
Write-Host "  ║     ██╔══██╗████╗  ██║██║    ██║██║████╗  ██║   ██╔══██╗██║     ║" -ForegroundColor Cyan
Write-Host "  ║     ███████║██╔██╗ ██║██║ █╗ ██║██║██╔██╗ ██║   ███████║██║     ║" -ForegroundColor Cyan
Write-Host "  ║     ██╔══██║██║╚██╗██║██║███╗██║██║██║╚██╗██║   ██╔══██║██║     ║" -ForegroundColor Cyan
Write-Host "  ║     ██║  ██║██║ ╚████║╚███╔███╔╝██║██║ ╚████║   ██║  ██║██║     ║" -ForegroundColor Cyan
Write-Host "  ║     ╚═╝  ╚═╝╚═╝  ╚═══╝ ╚══╝╚══╝ ╚═╝╚═╝  ╚═══╝  ╚═╝  ╚═╝╚═╝      ║" -ForegroundColor Cyan
Write-Host "  ║                                                                 ║" -ForegroundColor Cyan
Write-Host "  ║                            anwin.ai                             ║" -ForegroundColor Cyan
Write-Host "  ║                                                                 ║" -ForegroundColor Cyan
Write-Host "  ║                local code sync agent  *  $Version                 ║" -ForegroundColor Cyan
Write-Host "  ╚═════════════════════════════════════════════════════════════════╝" -ForegroundColor Cyan
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

Write-Host "  ◆  platform   →  windows / $OSArch" -ForegroundColor Yellow
Write-Host "  ◆  binary     →  $File" -ForegroundColor Yellow
Write-Host "  ◆  installing →  $InstallDir\$BinaryName" -ForegroundColor Yellow
Write-Host ""

New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

$DownloadUrl = "$BaseUrl/$File"
$Destination = "$InstallDir\$BinaryName"

Write-Host "  downloading..." -ForegroundColor Yellow

$webClient = New-Object System.Net.WebClient
$done = $false

$webClient.DownloadProgressChanged = [System.Net.DownloadProgressChangedEventHandler]{
    param($s, $e)
    $bar   = "#" * [math]::Floor($e.ProgressPercentage / 2)
    $empty = " " * (50 - [math]::Floor($e.ProgressPercentage / 2))
    Write-Host -NoNewline "`r  [$bar$empty] $($e.ProgressPercentage)%"
}

$webClient.DownloadFileCompleted = [System.ComponentModel.AsyncCompletedEventHandler]{
    param($s, $e)
    $script:done = $true
}

$webClient.DownloadFileAsync([uri]$DownloadUrl, $Destination)
while (-not $done) { Start-Sleep -Milliseconds 100 }
Write-Host ""

$CurrentPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($CurrentPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("PATH", "$CurrentPath;$InstallDir", "User")
}

Write-Host ""
Write-Host "  ╔═════════════════════════════════════════════════════════════════╗" -ForegroundColor Green
Write-Host "  ║   ✓  installation complete!                                    ║" -ForegroundColor Green
Write-Host "  ╠═════════════════════════════════════════════════════════════════╣" -ForegroundColor Green
Write-Host "  ║                                                                 ║" -ForegroundColor Green
Write-Host "  ║   next steps:                                                   ║" -ForegroundColor Green
Write-Host "  ║     1.  close and reopen terminal                               ║" -ForegroundColor Green
Write-Host "  ║     2.  anwin-agent setup                                       ║" -ForegroundColor Green
Write-Host "  ║     3.  anwin-agent start                                       ║" -ForegroundColor Green
Write-Host "  ║                                                                 ║" -ForegroundColor Green
Write-Host "  ║   docs:  https://anwin.ai/docs/agent                           ║" -ForegroundColor Green
Write-Host "  ╚═════════════════════════════════════════════════════════════════╝" -ForegroundColor Green
Write-Host ""