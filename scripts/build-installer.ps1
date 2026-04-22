$ErrorActionPreference = "Stop"

Set-Location (Join-Path $PSScriptRoot "..")

if (!(Test-Path "dist")) { New-Item -ItemType Directory -Path "dist" | Out-Null }
if (!(Test-Path "releases")) { New-Item -ItemType Directory -Path "releases" | Out-Null }

go build -o "dist\power-scheduler.exe" ".\cmd\power-scheduler"
go build -o "dist\power-desktop.exe" ".\cmd\power-desktop"

if (Test-Path "dist\web") { Remove-Item -Recurse -Force "dist\web" }
Copy-Item -Recurse "web" "dist\web"

$work = Join-Path ([IO.Path]::GetTempPath()) ("power-installer-" + [Guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $work | Out-Null
New-Item -ItemType Directory -Path (Join-Path $work "src") | Out-Null

$payload = Join-Path $work "src\payload.zip"
Compress-Archive -Path "dist\power-desktop.exe", "dist\web" -DestinationPath $payload -Force

$source = @'
package main

import (
	"archive/zip"
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

//go:embed payload.zip
var payload []byte

func main() {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		home, err := os.UserHomeDir()
		check(err)
		localAppData = filepath.Join(home, "AppData", "Local")
	}
	installDir := filepath.Join(localAppData, "PowerScheduler")
	check(os.MkdirAll(installDir, 0o755))
	check(extractPayload(installDir))

	target := filepath.Join(installDir, "power-desktop.exe")
	createShortcuts(target, installDir)
	_ = exec.Command(target).Start()
}

func extractPayload(dest string) error {
	reader, err := zip.NewReader(bytes.NewReader(payload), int64(len(payload)))
	if err != nil {
		return err
	}
	for _, item := range reader.File {
		target := filepath.Join(dest, item.Name)
		cleanDest, err := filepath.Abs(dest)
		if err != nil {
			return err
		}
		cleanTarget, err := filepath.Abs(target)
		if err != nil {
			return err
		}
		if cleanTarget != cleanDest && !strings.HasPrefix(cleanTarget, cleanDest+string(os.PathSeparator)) {
			return fmt.Errorf("invalid payload path: %s", item.Name)
		}
		if item.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		src, err := item.Open()
		if err != nil {
			return err
		}
		dst, err := os.Create(target)
		if err != nil {
			_ = src.Close()
			return err
		}
		_, copyErr := io.Copy(dst, src)
		closeErr := dst.Close()
		_ = src.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

func createShortcuts(target, workDir string) {
	home, err := os.UserHomeDir()
	if err == nil {
		createShortcut(filepath.Join(home, "Desktop", "PowerScheduler.lnk"), target, workDir)
	}
	appData := os.Getenv("APPDATA")
	if appData != "" {
		startMenu := filepath.Join(appData, "Microsoft", "Windows", "Start Menu", "Programs", "PowerScheduler")
		if os.MkdirAll(startMenu, 0o755) == nil {
			createShortcut(filepath.Join(startMenu, "PowerScheduler.lnk"), target, workDir)
		}
	}
}

func createShortcut(link, target, workDir string) {
	script := fmt.Sprintf(
		"$s=(New-Object -ComObject WScript.Shell).CreateShortcut(%s);$s.TargetPath=%s;$s.WorkingDirectory=%s;$s.Save()",
		psQuote(link), psQuote(target), psQuote(workDir),
	)
	_ = exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", script).Run()
}

func psQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func check(err error) {
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
'@

Set-Content -Encoding UTF8 -Path (Join-Path $work "src\main.go") -Value $source
$previousGo111Module = $env:GO111MODULE
$env:GO111MODULE = "off"
$installerOut = Join-Path (Resolve-Path "releases").Path "PowerSchedulerInstaller.exe"
try {
    Push-Location (Join-Path $work "src")
    try {
        go build -o $installerOut .
    } finally {
        Pop-Location
    }
} finally {
    $env:GO111MODULE = $previousGo111Module
}

$hash = Get-FileHash "releases\PowerSchedulerInstaller.exe" -Algorithm SHA256
"PowerSchedulerInstaller.exe  SHA256  $($hash.Hash)" | Set-Content -Encoding ASCII "releases\PowerSchedulerInstaller.sha256.txt"

Remove-Item -Recurse -Force $work
Write-Host "Installer generated: releases\PowerSchedulerInstaller.exe"
