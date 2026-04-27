# Shared helpers for the rig scripts. Dot-source from each script:
#   . "$PSScriptRoot\_lib.ps1"

$script:RepoRoot   = (Split-Path -Parent $PSScriptRoot)
$script:ConfigPath = Join-Path $RepoRoot 'rig-config.json'

function Get-RigConfig {
    [CmdletBinding()]
    param([string]$Path = $script:ConfigPath)
    if (-not (Test-Path $Path)) {
        throw "rig-config.json not found at $Path. Copy rig-config.example.json and fill in your paths."
    }
    $raw = Get-Content $Path -Raw
    # Strip JSON comment fields ("_comment": ...) which our example has but
    # JSON parsers don't accept -- quick regex strip before parsing.
    $cleaned = $raw -replace '(?m)^\s*"_comment".*$', ''
    $cleaned = $cleaned -replace ',(\s*})', '$1'
    return $cleaned | ConvertFrom-Json
}

function Expand-EnvPath {
    [CmdletBinding()]
    param([string]$Path)
    if (-not $Path) { return $null }
    return [Environment]::ExpandEnvironmentVariables($Path)
}

function Test-CheckPath {
    [CmdletBinding()]
    param([string]$Name, [string]$Path)
    $expanded = Expand-EnvPath $Path
    if ($expanded -and (Test-Path $expanded)) {
        return @{ name = $Name; status = 'ok';   detail = $expanded }
    } else {
        return @{ name = $Name; status = 'fail'; detail = "missing: $expanded"; fix_hint = "verify install + rig-config.json path" }
    }
}

function Test-Process {
    [CmdletBinding()]
    param([string]$Name, [string[]]$ProcessNames)
    foreach ($pn in $ProcessNames) {
        if (Get-Process -Name $pn -ErrorAction SilentlyContinue) {
            return @{ name = $Name; status = 'ok';   detail = "$pn running" }
        }
    }
    return @{ name = $Name; status = 'warn'; detail = "not running ($($ProcessNames -join ' or '))"; fix_hint = "launch the app via pre-flight" }
}

function Test-VJoyDevice {
    [CmdletBinding()]
    param([int]$DeviceId, [bool]$ExpectFFB = $false, [int]$ExpectButtons = 0)
    # vJoy enabled state: HKLM\SYSTEM\CurrentControlSet\services\vjoy\Parameters\DeviceNN
    $key = "HKLM:\SYSTEM\CurrentControlSet\services\vjoy\Parameters\Device{0:D2}" -f $DeviceId
    if (-not (Test-Path $key)) {
        return @{ name = "vJoy Device $DeviceId"; status = 'fail'; detail = 'not configured'; fix_hint = "open vJoyConf and enable Device $DeviceId" }
    }
    try {
        $enabled = (Get-ItemProperty -Path $key -Name 'EnableEFFECT' -ErrorAction SilentlyContinue).EnableEFFECT
        $ffb = ($enabled -ne $null -and $enabled -ne 0)
    } catch { $ffb = $null }
    $detail = "registered" + ($(if ($ffb) { ', FFB on' } else { ', FFB off' }))
    if ($ExpectFFB -ne $null -and ($ffb -ne $ExpectFFB)) {
        return @{ name = "vJoy Device $DeviceId"; status = 'warn'; detail = "$detail (expected FFB=$ExpectFFB)"; fix_hint = 'reconfigure in vJoyConf' }
    }
    return @{ name = "vJoy Device $DeviceId"; status = 'ok'; detail = $detail }
}
