$ErrorActionPreference = "Stop"

$GithubUser = "captain-duster"
$Repo = "anwin-agent"
$Version = "v2.0.0"
$BaseUrl = "https://github.com/$GithubUser/$Repo/releases/download/$Version"
$InstallDir = "$env:USERPROFILE\.local\bin"
$BinaryName = "anwin-agent.exe"

Write-Host ""
Write-Host "  +==========================================+" -ForegroundColor Cyan
Write-Host "  |                                          |" -ForegroundColor Cyan
Write-Host "  |        XXXXX  N   N W   W  I  N   N     |" -ForegroundColor Cyan
Write-Host "  |        A   A  NN  N W   W  I  NN  N     |" -ForegroundColor Cyan
Write-Host "  |        AAAAA  N N N W W W  I  N N N     |" -ForegroundColor Cyan
Write-Host "  |        A   A  N  NN  W W   I  N  NN     |" -ForegroundColor Cyan
Write-Host "  |        A   A  N   N   W W  I  N   N     |" -ForegroundColor Cyan
Write-Host "  |                                          |" -ForegroundColor Cyan
Write-Host "  |         Local Code Sync Agent            |" -ForegroundColor Cyan
Write-Host "  |             Version $Version               |" -ForegroundColor Cyan
Write-Host "  |                                          |" -ForegroundColor Cyan
Write-Host "  +==========================================+" -ForegroundColor Cyan
Write-Host ""

$OSArch = (Get-CimInstance Win32_OperatingSystem).OSArchitecture
$CPUArch = (Get-CimInstance Win32_Processor).Architecture

if ($OSArch -like "*ARM*") {
    $File = "anwin-agent-windows-arm64.exe"
} elseif ($OSArch -like "*64*") {
    $File = "anwin-agent-windows-amd64.exe"
} else {
    $File = "anwin-agent-windows-386.exe"
}

Write-Host "  [*]  Platform   ->  Windows / $OSArch" -ForegroundColor Yellow
Write-Host "  [*]  Binary     ->  $File" -ForegroundColor Yellow
Write-Host "  [*]  Installing ->  $InstallDir\$BinaryName" -ForegroundColor Yellow
Write-Host ""

New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

$DownloadUrl = "$BaseUrl/$File"
$Destination = "$InstallDir\$BinaryName"

Write-Host "  Downloading..." -ForegroundColor Yellow
Invoke-WebRequest -Uri $DownloadUrl -OutFile $Destination -UseBasicParsing

$CurrentPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($CurrentPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("PATH", "$CurrentPath;$InstallDir", "User")
    Write-Host "  [*]  Added to PATH" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "  +==========================================+" -ForegroundColor Green
Write-Host "  |   [OK]  Installation complete!           |" -ForegroundColor Green
Write-Host "  +==========================================+" -ForegroundColor Green
Write-Host "  |                                          |" -ForegroundColor Green
Write-Host "  |   Next steps:                            |" -ForegroundColor Green
Write-Host "  |     1.  Close and reopen terminal        |" -ForegroundColor Green
Write-Host "  |     2.  anwin-agent setup                |" -ForegroundColor Green
Write-Host "  |     3.  anwin-agent start                |" -ForegroundColor Green
Write-Host "  |                                          |" -ForegroundColor Green
Write-Host "  |   Docs:  https://anwin.ai/docs/agent     |" -ForegroundColor Green
Write-Host "  |                                          |" -ForegroundColor Green
Write-Host "  +==========================================+" -ForegroundColor Green
Write-Host ""
