# Start mihomo in the background with API controller enabled
$scriptDir = Split-Path -Parent -Path $MyInvocation.MyCommand.Definition
$configFile = Join-Path $scriptDir "config.yaml"

# Check if config file exists
if (-not (Test-Path $configFile)) {
    Write-Host "Error: Config file not found at $configFile" -ForegroundColor Red
    exit 1
}

Write-Host "Using config file: $configFile" -ForegroundColor Cyan

# Create process start info
$psi = New-Object System.Diagnostics.ProcessStartInfo
$psi.FileName = "E:\server\mihomo\mihomo-windows-amd64.exe"
$psi.Arguments = "-f `"$configFile`""
$psi.WorkingDirectory = $scriptDir
$psi.UseShellExecute = $true
$psi.WindowStyle = [System.Diagnostics.ProcessWindowStyle]::Hidden

# Start the process
$process = [System.Diagnostics.Process]::Start($psi)
Write-Host "Mihomo started in the background with PID: $($process.Id)" -ForegroundColor Green
Write-Host "API Controller: http://127.0.0.1:9090" -ForegroundColor Yellow
Write-Host "Secret: 123456" -ForegroundColor Yellow