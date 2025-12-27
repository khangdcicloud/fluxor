# PowerShell script to download and install k6 for Windows

Write-Host "Installing k6 for Windows..." -ForegroundColor Green

# Create k6 directory in user's local folder
$k6Dir = "$env:LOCALAPPDATA\k6"
$k6Exe = "$k6Dir\k6.exe"

# Check if already installed
if (Test-Path $k6Exe) {
    Write-Host "k6 is already installed at: $k6Exe" -ForegroundColor Yellow
    Write-Host "Version: $(& $k6Exe version)" -ForegroundColor Cyan
    exit 0
}

# Create directory if it doesn't exist
if (-not (Test-Path $k6Dir)) {
    New-Item -ItemType Directory -Path $k6Dir -Force | Out-Null
}

# Get latest k6 version
Write-Host "Fetching latest k6 version..." -ForegroundColor Cyan
try {
    $latestRelease = Invoke-RestMethod -Uri "https://api.github.com/repos/grafana/k6/releases/latest"
    $version = $latestRelease.tag_name -replace 'v', ''
    
    # Try different naming patterns for Windows releases
    $downloadUrl = $null
    $patterns = @("*windows-amd64.exe", "*windows-amd64.zip", "*windows_amd64.exe", "*windows_amd64.zip", "*win64.exe", "*win64.zip")
    
    foreach ($pattern in $patterns) {
        $asset = $latestRelease.assets | Where-Object { $_.name -like $pattern } | Select-Object -First 1
        if ($asset) {
            $downloadUrl = $asset.browser_download_url
            break
        }
    }
    
    if (-not $downloadUrl) {
        Write-Host "Error: Could not find Windows download URL" -ForegroundColor Red
        Write-Host "Available assets:" -ForegroundColor Yellow
        $latestRelease.assets | ForEach-Object { Write-Host "  - $($_.name)" -ForegroundColor Gray }
        Write-Host "`nPlease download manually from: https://github.com/grafana/k6/releases/latest" -ForegroundColor Yellow
        exit 1
    }
    
    Write-Host "Downloading k6 v$version..." -ForegroundColor Cyan
    Write-Host "URL: $downloadUrl" -ForegroundColor Gray
    
    # Download k6
    $zipPath = "$k6Dir\k6.zip"
    Invoke-WebRequest -Uri $downloadUrl -OutFile $zipPath -UseBasicParsing
    
    # Extract (k6 releases are usually .exe files, not .zip)
    if ($downloadUrl -like "*.zip") {
        Expand-Archive -Path $zipPath -DestinationPath $k6Dir -Force
        Remove-Item $zipPath
    } else {
        # If it's a direct .exe download, rename it
        Move-Item -Path $zipPath -Destination $k6Exe -Force
    }
    
    # Verify installation
    if (Test-Path $k6Exe) {
        Write-Host "`nk6 installed successfully!" -ForegroundColor Green
        Write-Host "Location: $k6Exe" -ForegroundColor Cyan
        Write-Host "`nVersion:" -ForegroundColor Yellow
        & $k6Exe version
        
        # Add to PATH (user level)
        $currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
        if ($currentPath -notlike "*$k6Dir*") {
            Write-Host "`nAdding k6 to PATH..." -ForegroundColor Cyan
            [Environment]::SetEnvironmentVariable("Path", "$currentPath;$k6Dir", "User")
            Write-Host "k6 added to PATH. Please restart your terminal for PATH changes to take effect." -ForegroundColor Yellow
        }
        
        Write-Host "`nYou can now run k6 tests!" -ForegroundColor Green
        Write-Host "Example: k6 run loadtest/smoke_test.js" -ForegroundColor Cyan
    } else {
        Write-Host "Error: k6.exe not found after installation" -ForegroundColor Red
        exit 1
    }
    
} catch {
    Write-Host "Error installing k6: $_" -ForegroundColor Red
    Write-Host "`nManual installation:" -ForegroundColor Yellow
    Write-Host "1. Download from: https://github.com/grafana/k6/releases/latest" -ForegroundColor White
    Write-Host "2. Extract k6.exe to: $k6Dir" -ForegroundColor White
    Write-Host "3. Add $k6Dir to your PATH" -ForegroundColor White
    exit 1
}

