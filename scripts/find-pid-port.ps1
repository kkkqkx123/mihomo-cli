# Define the keyword for the process name to be searched
$ProcessKeyword = "mihomo"

Write-Host "Searching for processes that contain [$ProcessKeyword]..." -ForegroundColor Cyan

# 1. Use tasklist to find the process
$tasklist = tasklist | findstr $ProcessKeyword

if ($tasklist) {
    # Parse the process information, properly handling spaces
    $parts = $tasklist -split '\s+', 5  # Limit the split to 5 parts
    $processName = $parts[0]
    $processId = $parts[1]  # Use the process ID instead of the PID

    Write-Host "Process found: $processName" -ForegroundColor Green
    Write-Host "Process ID (PID): $processId" -ForegroundColor Yellow

    # 2. Use netstat to find the ports used by the process
    Write-Host "`nQuerying the ports occupied by PID [$processId]..." -ForegroundColor Cyan

    $netstat = netstat -ano | findstr "$processId"

    if ($netstat) {
        Write-Host "`n--- Network connection information ---" -ForegroundColor Magenta
        $netstat

        Write-Host "`n--- Listening ports ---" -ForegroundColor Green
        $netstat | findstr "LISTENING" | ForEach-Object {
            Write-Host $_  # Display the listening ports directly
        }
    } else {
        Write-Host "No network connections for the process were found." -ForegroundColor DarkGray
    }
} else {
    Write-Host "Error: No process containing '$ProcessKeyword' was found." -ForegroundColor Red
    exit 1
} 