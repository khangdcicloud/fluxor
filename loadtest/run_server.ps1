# Run the simple loadtest server
Write-Host "Starting simple FastHTTP server..." -ForegroundColor Green
Write-Host "Server will run on http://localhost:8080" -ForegroundColor Yellow
Write-Host "Press Ctrl+C to stop the server" -ForegroundColor Yellow
Write-Host ""

# Check if executable exists, if not build it
if (-not (Test-Path "simple_server.exe")) {
    Write-Host "Building server..." -ForegroundColor Cyan
    go build -o simple_server.exe simple_server.go
    if ($LASTEXITCODE -ne 0) {
        Write-Host "Build failed!" -ForegroundColor Red
        exit 1
    }
}

# Run the server
.\simple_server.exe

