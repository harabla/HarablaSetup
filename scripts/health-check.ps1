# health-check.ps1 -- runnable rig health check.
# Outputs JSON to stdout (consumable by tray /api/health, or pipe to ConvertFrom-Json).
# Pass -Html to also write an HTML report to <logs>/health-<timestamp>.html.
#
# Usage:
#   powershell -NoProfile -File scripts\health-check.ps1
#   powershell -NoProfile -File scripts\health-check.ps1 -Html

[CmdletBinding()]
param(
    [switch]$Html,
    [switch]$Quiet
)

. "$PSScriptRoot\_lib.ps1"

$ErrorActionPreference = 'Continue'
$cfg = Get-RigConfig

$checks = @{}

# Tools -- each path in cfg.tools must exist
foreach ($key in $cfg.tools.PSObject.Properties.Name) {
    $checks["tool_$key"] = Test-CheckPath -Name "Tool: $key" -Path $cfg.tools.$key
}

# Games -- UI / sim / exe paths
foreach ($gname in $cfg.games.PSObject.Properties.Name) {
    $g = $cfg.games.$gname
    if ($g.ui)  { $checks["game_${gname}_ui"]  = Test-CheckPath -Name "$gname (UI)"  -Path $g.ui }
    if ($g.sim) { $checks["game_${gname}_sim"] = Test-CheckPath -Name "$gname (sim)" -Path $g.sim }
    # exe field can be a string path or an array of process names (for the watcher); only check if it's a single path
    if ($g.exe -and $g.exe -is [string]) { $checks["game_${gname}_exe"] = Test-CheckPath -Name "$gname (exe)" -Path $g.exe }
}

# VR
if ($cfg.vr.openXRToolkit)  { $checks['vr_openxr_toolkit']  = Test-CheckPath -Name 'OpenXR Toolkit cfg' -Path $cfg.vr.openXRToolkit }
if ($cfg.vr.virtualDesktop) { $checks['vr_virtual_desktop'] = Test-CheckPath -Name 'Virtual Desktop'    -Path $cfg.vr.virtualDesktop }

# vJoy devices (Device 1 = buttons no-FFB, Device 2 = axis FFB-on)
$checks['vjoy_device_1'] = Test-VJoyDevice -DeviceId 1 -ExpectFFB:$false
$checks['vjoy_device_2'] = Test-VJoyDevice -DeviceId 2 -ExpectFFB:$true

# Critical processes
$checks['gremlin_running']   = Test-Process -Name 'Joystick Gremlin running' -ProcessNames @('JoystickGremlin','joystick_gremlin')
$checks['fanalab_running']   = Test-Process -Name 'FanaLab running'           -ProcessNames @('FanaLab')
$checks['hidhide_installed'] = Test-Process -Name 'HidHide service'           -ProcessNames @('HidHideClient','HidHide')

# iRacing controls.cfg present
$controlsCfg = Join-Path (Expand-EnvPath $cfg.games.iRacing.documents) 'controls.cfg'
$checks['iracing_controls'] = if (Test-Path $controlsCfg) {
    @{ name = 'iRacing controls.cfg'; status = 'ok'; detail = $controlsCfg }
} else {
    @{ name = 'iRacing controls.cfg'; status = 'warn'; detail = 'not found'; fix_hint = "launch iRacing once to generate, then bind from PC Setup table" }
}

# Logs dir writable
$logsDir = Expand-EnvPath $cfg.logs
if (-not (Test-Path $logsDir)) {
    try { New-Item -ItemType Directory -Force -Path $logsDir | Out-Null } catch {}
}
$checks['logs_dir'] = if (Test-Path $logsDir) {
    @{ name = 'Logs directory'; status = 'ok'; detail = $logsDir }
} else {
    @{ name = 'Logs directory'; status = 'fail'; detail = "cannot create $logsDir" }
}

# Summary
$ok   = ($checks.Values | Where-Object { $_.status -eq 'ok' }).Count
$warn = ($checks.Values | Where-Object { $_.status -eq 'warn' }).Count
$fail = ($checks.Values | Where-Object { $_.status -eq 'fail' }).Count

if (-not $Quiet) {
    Write-Host "[health] $ok ok, $warn warn, $fail fail" -ForegroundColor $(if ($fail -gt 0) { 'Red' } elseif ($warn -gt 0) { 'Yellow' } else { 'Green' })
}

# Always emit JSON to stdout -- tray reads this
$checks | ConvertTo-Json -Depth 5

# Optional HTML report
if ($Html) {
    if (-not (Test-Path $logsDir)) { $logsDir = $env:TEMP }
    $ts = Get-Date -Format 'yyyy-MM-dd_HHmm'
    $out = Join-Path $logsDir "health-$ts.html"
    $rows = ($checks.GetEnumerator() | Sort-Object { $_.Value.status } | ForEach-Object {
        $c = $_.Value
        $color = @{ ok = '#5a8'; warn = '#cb8'; fail = '#c55' }[$c.status]
        "<tr style='border-left:3px solid $color'><td>$($c.name)</td><td style='color:$color'>$($c.status)</td><td>$($c.detail)</td><td style='color:#cb8'>$($c.fix_hint)</td></tr>"
    }) -join "`n"
    @"
<!doctype html><html><head><meta charset='utf-8'><title>Rig health $ts</title>
<style>body{font-family:system-ui;background:#0a0d12;color:#e0e0e0;padding:24px;max-width:1200px;margin:auto}
table{border-collapse:collapse;width:100%}th,td{padding:8px 12px;border-bottom:1px solid #222;text-align:left}
th{color:#888;font-size:12px;text-transform:uppercase}</style></head><body>
<h1>Rig health -- $ts</h1>
<p>$ok ok / $warn warn / $fail fail</p>
<table><tr><th>Check</th><th>Status</th><th>Detail</th><th>Fix</th></tr>$rows</table>
</body></html>
"@ | Out-File -FilePath $out -Encoding utf8
    Write-Host "[health] html report: $out"
}
