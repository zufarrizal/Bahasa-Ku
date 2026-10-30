<#
.SYNOPSIS
    Skrip Instalasi Otomatis Satu Perintah untuk Bahasa Ku (Windows)
.DESCRIPTION
    Mengunduh ku.exe, mendaftarkannya ke PATH Windows, dan memasang ekstensi VS Code secara otomatis.
#>

$ErrorActionPreference = "Stop"

Write-Host "==================================================" -ForegroundColor Cyan
Write-Host "       Pemasang Otomatis Bahasa Ku (Windows)       " -ForegroundColor Cyan
Write-Host "==================================================" -ForegroundColor Cyan

$InstallDir = "$env:LOCALAPPDATA\Bahasa-Ku"
$ExeUrl = "https://github.com/zufarrizal/Bahasa-Ku/releases/download/v0.3.5/ku-windows-amd64.exe"
$VsixUrl = "https://github.com/zufarrizal/Bahasa-Ku/releases/download/v0.3.5/bahasa-ku-vscode-0.2.0.vsix"
$ExeTarget = "$InstallDir\ku.exe"
$VsixTarget = "$InstallDir\bahasa-ku.vsix"

# 1. Buat direktori instalasi
if (!(Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

# 2. Unduh executable ku.exe
Write-Host "[1/3] Mengunduh alat Bahasa Ku (ku.exe)..." -ForegroundColor Yellow
Invoke-WebRequest -Uri $ExeUrl -OutFile $ExeTarget

# 3. Daftarkan ke PATH Windows Pengguna (agar bisa dipanggil di mana saja)
Write-Host "[2/3] Mendaftarkan 'ku' ke PATH Windows Anda..." -ForegroundColor Yellow
$UserPath = [Environment]::GetEnvironmentVariable("Path", [EnvironmentVariableTarget]::User)
if ($UserPath -notlike "*$InstallDir*") {
    $NewPath = "$UserPath;$InstallDir"
    [Environment]::SetEnvironmentVariable("Path", $NewPath, [EnvironmentVariableTarget]::User)
    $env:Path = "$env:Path;$InstallDir"
    Write-Host "      -> Berhasil ditambahkan ke PATH!" -ForegroundColor Green
} else {
    Write-Host "      -> Sudah terdaftar di PATH." -ForegroundColor Gray
}

# 4. Otomatis Pasang Ekstensi VS Code jika ada VS Code terpasang
Write-Host "[3/3] Memeriksa instalasi Visual Studio Code..." -ForegroundColor Yellow
$CodeCmd = Get-Command "code" -ErrorAction SilentlyContinue
if ($CodeCmd) {
    Write-Host "      -> Mengunduh dan memasang ekstensi VS Code..." -ForegroundColor Yellow
    Invoke-WebRequest -Uri $VsixUrl -OutFile $VsixTarget
    & code --install-extension $VsixTarget --force | Out-Null
    Remove-Item -Force $VsixTarget -ErrorAction SilentlyContinue
    Write-Host "      -> Ekstensi VS Code Bahasa Ku berhasil terpasang otomatis!" -ForegroundColor Green
} else {
    # Fallback: Ekstrak folder ekstensi langsung ke direktori default VS Code
    $VscodeExtDir = "$HOME\.vscode\extensions\bahasa-ku"
    Write-Host "      -> Memasang folder ekstensi ke $VscodeExtDir..." -ForegroundColor Yellow
    Invoke-WebRequest -Uri $VsixUrl -OutFile $VsixTarget
    # Ekstraksi manual
    Add-Type -AssemblyName System.IO.Compression.FileSystem
    $TempZip = "$InstallDir\vsix_temp"
    if (Test-Path $TempZip) { Remove-Item -Recurse -Force $TempZip }
    [System.IO.Compression.ZipFile]::ExtractToDirectory($VsixTarget, $TempZip)
    if (Test-Path "$TempZip\extension") {
        if (Test-Path $VscodeExtDir) { Remove-Item -Recurse -Force $VscodeExtDir }
        Copy-Item -Recurse -Force "$TempZip\extension" $VscodeExtDir
    }
    Remove-Item -Recurse -Force $TempZip -ErrorAction SilentlyContinue
    Remove-Item -Force $VsixTarget -ErrorAction SilentlyContinue
    Write-Host "      -> Ekstensi VS Code berhasil dipasang ke profil Anda!" -ForegroundColor Green
}

Write-Host "`n==================================================" -ForegroundColor Green
Write-Host "  Pemasangan Selesai 100%! Selamat Menggunakan Ku! 🎉" -ForegroundColor Green
Write-Host "==================================================" -ForegroundColor Green
Write-Host "Buka terminal baru dan coba perintah berikut:" -ForegroundColor Cyan
Write-Host "  ku              (Buka terminal REPL)" -ForegroundColor White
Write-Host "  ku inisiasi web my-app (Buat proyek web)" -ForegroundColor White
Write-Host "  ku versi        (Cek versi)`n" -ForegroundColor White
