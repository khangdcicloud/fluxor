# Quick Start Guide for k6 Load Testing

## Step 1: Install k6

### Windows Installation

1. **Download k6:**
   - Go to: https://github.com/grafana/k6/releases/latest
   - Download: `k6-v0.48.0-windows-amd64.zip` (or latest version)
   - Extract the zip file
   - You'll find `k6.exe` inside

2. **Add to PATH (Option A - User PATH):**
   ```powershell
   # Create a folder for k6 (e.g., C:\k6\)
   # Copy k6.exe to that folder
   # Then add to PATH:
   [Environment]::SetEnvironmentVariable("Path", $env:Path + ";C:\k6", "User")
   ```

3. **Or use directly (Option B):**
   - Place `k6.exe` in the `loadtest` folder
   - Run tests with: `.\k6.exe run loadtest.js`

4. **Verify installation:**
   ```powershell
   k6 version
   ```

## Step 2: Start the Server

Make sure your server is running:
```powershell
cd loadtest
.\simple_server.exe
```

Or in another terminal:
```powershell
cd loadtest
go run simple_server.go
```

## Step 3: Run Load Tests

### Quick Smoke Test (30 seconds, 10 users)
```powershell
k6 run loadtest/smoke_test.js
```

### Full Load Test (10k users, ~5 minutes)
```powershell
k6 run loadtest/load_test.js
```

### Spike Test (sudden traffic burst)
```powershell
k6 run loadtest/spike-test.js
```

### Stress Test (find breaking point)
```powershell
k6 run loadtest/stress-test.js
```

## Step 4: Customize Tests

Set custom base URL:
```powershell
$env:BASE_URL="http://localhost:8080"
k6 run loadtest/smoke_test.js
```

Or with inline environment variable:
```powershell
k6 run -e BASE_URL=http://localhost:8080 loadtest/smoke_test.js
```

## Troubleshooting

- **"k6 not found"**: Make sure k6.exe is in your PATH or in the current directory
- **"Connection refused"**: Make sure the server is running on port 8080
- **Permission errors**: Try running PowerShell as Administrator

