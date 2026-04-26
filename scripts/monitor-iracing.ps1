# monitor-iracing.ps1 — wrap an iRacing session with full telemetry capture.
#
# Snapshots iRacing graphics .ini files + OpenXR Toolkit config + recent
# setups. Starts HWiNFO64 (CSV log), PresentMon (frame data), and a process
# poller. Launches iRacing UI. Waits for the iRacing process tree to exit.
# Stops loggers. Generates an HTML report and opens it.
#
# Usage:
#   powershell -NoProfile -File scripts\monitor-iracing.ps1
#
# Wired to Stream Deck:
#   System: Open → powershell.exe -NoProfile -File <repo>\scripts\monitor-iracing.ps1

[CmdletBinding()]
param()

. "$PSScriptRoot\_lib.ps1"

$ErrorActionPreference = 'Continue'
$cfg = Get-RigConfig
$timestamp = Get-Date -Format 'yyyy-MM-dd_HHmm'
$logRoot = Expand-EnvPath $cfg.logs
$logDir = Join-Path $logRoot "iracing\$timestamp"
New-Item -ItemType Directory -Force -Path $logDir | Out-Null

Write-Host "[monitor] session: $timestamp" -ForegroundColor Cyan
Write-Host "[monitor] log dir: $logDir"

# ---------- 1. SNAPSHOT graphics + VR settings ----------
$iracingDocs = Expand-EnvPath $cfg.games.iRacing.documents
foreach ($f in $cfg.games.iRacing.settings) {
    $src = Join-Path $iracingDocs $f
    if (Test-Path $src) {
        Copy-Item $src (Join-Path $logDir "$f.snapshot")
    }
}
$openxr = Expand-EnvPath $cfg.vr.openXRToolkit
if ($openxr -and (Test-Path "$openxr\settings.cfg")) {
    Copy-Item "$openxr\settings.cfg" (Join-Path $logDir 'openxr-toolkit.cfg.snapshot')
}
# 10 most-recently-modified setups (any car)
$setupsDir = Join-Path $iracingDocs 'setups'
if (Test-Path $setupsDir) {
    Get-ChildItem $setupsDir -Recurse -Filter '*.sto' |
      Sort-Object LastWriteTime -Descending | Select-Object -First 10 |
      ForEach-Object { Copy-Item $_.FullName (Join-Path $logDir "setup-$($_.Name)") }
}

# ---------- 2. Start HWiNFO with CSV logging ----------
$hwinfo = Expand-EnvPath $cfg.tools.hwinfo
if ($hwinfo -and (Test-Path $hwinfo)) {
    Start-Process $hwinfo -ArgumentList "/log=$logDir\hwinfo.csv" -WindowStyle Minimized
} else {
    Write-Warning "HWiNFO64 not found at $hwinfo — skipping sensor log"
}

# ---------- 3. Start PresentMon ----------
$presentMon = Expand-EnvPath $cfg.tools.presentMon
if ($presentMon -and (Test-Path $presentMon)) {
    Start-Process $presentMon `
      -ArgumentList "-process_name iRacingSim64DX11.exe -output_file `"$logDir\presentmon.csv`" -terminate_on_proc_exit -no_top" `
      -WindowStyle Hidden
} else {
    Write-Warning "PresentMon not found at $presentMon — skipping frame log"
}

# ---------- 4. Background process poller (5s interval) ----------
$pollerJob = Start-Job -ScriptBlock {
    param($logFile)
    'timestamp,process,cpu_percent,ram_bytes' | Out-File $logFile
    while ($true) {
        $now = Get-Date -Format 'HH:mm:ss'
        Get-Process | Where-Object { $_.WorkingSet64 -gt 50MB } |
          Sort-Object WorkingSet64 -Descending | Select-Object -First 15 |
          ForEach-Object {
            try {
                $cs = Get-Counter "\Process($($_.ProcessName))\% Processor Time" -ErrorAction Stop
                $cpu = [math]::Round($cs.CounterSamples[0].CookedValue, 2)
            } catch { $cpu = 0 }
            "$now,$($_.ProcessName),$cpu,$($_.WorkingSet64)" | Out-File $logFile -Append
          }
        Start-Sleep -Seconds 5
    }
} -ArgumentList (Join-Path $logDir 'processes.csv')

# ---------- 5. Launch iRacing UI (or attach if already running) ----------
$alreadyRunning = (Get-Process iRacingSim64DX11 -ErrorAction SilentlyContinue) -or
                  (Get-Process iRacingUI       -ErrorAction SilentlyContinue)
if ($alreadyRunning) {
    Write-Host "[monitor] iRacing already running — attaching" -ForegroundColor Cyan
} else {
    $iracingUi = Expand-EnvPath $cfg.games.iRacing.ui
    if (-not (Test-Path $iracingUi)) {
        Write-Error "iRacing UI not found at $iracingUi (check rig-config.json games.iRacing.ui)"
        Stop-Job $pollerJob; Remove-Job $pollerJob
        exit 1
    }
    $ui = Start-Process $iracingUi -PassThru
    Write-Host "[monitor] iRacing launched (PID $($ui.Id))" -ForegroundColor Green
}

# ---------- 6. Wait for iRacing process tree to exit ----------
Start-Sleep -Seconds 30  # let iRacing actually start
while ((Get-Process iRacingSim64DX11 -ErrorAction SilentlyContinue) -or
       (Get-Process iRacingUI -ErrorAction SilentlyContinue)) {
    Start-Sleep -Seconds 5
}
Write-Host "[monitor] iRacing exited; cleaning up" -ForegroundColor Yellow

# ---------- 7. Cleanup ----------
Stop-Job $pollerJob; Remove-Job $pollerJob
Stop-Process -Name HWiNFO64 -Force -ErrorAction SilentlyContinue
Stop-Process -Name PresentMon* -Force -ErrorAction SilentlyContinue

# ---------- 8. Generate report ----------
& "$PSScriptRoot\generate-report.ps1" -LogDir $logDir
Start-Process (Join-Path $logDir 'report.html')
Write-Host "[monitor] report opened: $logDir\report.html" -ForegroundColor Cyan
