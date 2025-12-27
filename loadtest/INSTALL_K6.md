# Installing k6 on Windows

## Option 1: Manual Installation (Recommended)

1. Download k6 from: https://github.com/grafana/k6/releases
2. Download the Windows installer (e.g., `k6-v0.48.0-amd64.exe`)
3. Rename it to `k6.exe`
4. Place it in a folder (e.g., `C:\k6\`)
5. Add that folder to your PATH environment variable

## Option 2: Using Chocolatey (Requires Admin)

Run PowerShell as Administrator:
```powershell
choco install k6 -y
```

## Option 3: Using Scoop

```powershell
scoop install k6
```

## Verify Installation

After installation, verify it works:
```powershell
k6 version
```

## Quick Download Link

Latest release: https://github.com/grafana/k6/releases/latest

Download the Windows 64-bit executable and add it to your PATH.

