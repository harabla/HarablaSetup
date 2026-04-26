# bundle.ps1 — capture the current rig's portable config into bundle/
# Runs on Windows (the rig). Writes everything to <repo>/bundle/.
# Intended to run on the working PC after first-time manual setup, so the
# captured exports can be committed and re-deployed on any future PC.
#
# Usage:
#   .\scripts\bundle.ps1                  # capture everything
#   .\scripts\bundle.ps1 -Skip streamdeck # skip a specific component
#   .\scripts\bundle.ps1 -DryRun          # show what would be done

[CmdletBinding()]
param(
    [string[]]$Skip = @(),
    [switch]$DryRun
)

. "$PSScriptRoot\_lib.ps1"

$ErrorActionPreference = 'Continue'
$repoRoot = (Resolve-Path "$PSScriptRoot\..").Path
$bundleDir = Join-Path $repoRoot 'bundle'

function Write-Section { param($t) Write-Host "`n=== $t ===" -ForegroundColor Cyan }
function Write-Skip    { param($t) Write-Host "    skipped — $t" -ForegroundColor DarkGray }
function Write-OK      { param($t) Write-Host "    ✓ $t" -ForegroundColor Green }
function Write-Warn    { param($t) Write-Host "    ⚠ $t" -ForegroundColor Yellow }

function Should-Run { param($name) return -not ($Skip -contains $name) }

function Copy-Maybe {
    [CmdletBinding()] param([string]$From, [string]$To)
    if ($DryRun) { Write-OK "would copy $From → $To"; return }
    New-Item -ItemType Directory -Force -Path (Split-Path $To) | Out-Null
    Copy-Item -LiteralPath $From -Destination $To -Recurse -Force
}

# ---------------------------------------------------------------- Stream Deck
function Capture-StreamDeck {
    Write-Section 'Stream Deck profiles'
    $sdRoot = Join-Path $env:APPDATA 'Elgato\StreamDeck'
    $profilesDir = Join-Path $sdRoot 'ProfilesV2'
    if (-not (Test-Path $profilesDir)) {
        Write-Warn "no Stream Deck profiles found at $profilesDir — install Stream Deck app first"
        return
    }

    # Each profile is a folder named by UUID; manifest.json has the human name.
    $outDir = Join-Path $bundleDir 'StreamDeck'
    New-Item -ItemType Directory -Force -Path $outDir | Out-Null

    $profiles = Get-ChildItem $profilesDir -Directory
    foreach ($p in $profiles) {
        $manifest = Join-Path $p.FullName 'manifest.json'
        if (-not (Test-Path $manifest)) { continue }
        try {
            $m = Get-Content $manifest -Raw | ConvertFrom-Json
            $name = $m.Name
        } catch { $name = $p.Name }
        if (-not $name) { $name = $p.Name }
        $safe = $name -replace '[^\w\-]+', '_'
        $outFile = Join-Path $outDir "$safe.streamDeckProfile"

        # .streamDeckProfile is a zip of the profile folder.
        if ($DryRun) {
            Write-OK "would export profile '$name' → $safe.streamDeckProfile"
            continue
        }
        if (Test-Path $outFile) { Remove-Item $outFile -Force }
        # Compress-Archive doesn't accept .streamDeckProfile extension directly;
        # write to .zip, then rename.
        $tmpZip = "$outFile.zip"
        Compress-Archive -Path "$($p.FullName)\*" -DestinationPath $tmpZip -Force
        Move-Item $tmpZip $outFile -Force
        Write-OK "exported '$name' → $safe.streamDeckProfile"
    }

    # Plugin list
    $pluginsDir = Join-Path $sdRoot 'Plugins'
    if (Test-Path $pluginsDir) {
        $plugins = Get-ChildItem $pluginsDir -Directory | ForEach-Object { $_.Name }
        $outFile = Join-Path $outDir 'plugins.txt'
        if ($DryRun) {
            Write-OK "would write plugins.txt with $($plugins.Count) entries"
        } else {
            $plugins | Out-File $outFile -Encoding utf8
            Write-OK "captured $($plugins.Count) plugins → plugins.txt"
        }
    } else {
        Write-Warn 'plugins dir missing'
    }
}

# ---------------------------------------------------------------- Joystick Gremlin
function Capture-Gremlin {
    Write-Section 'Joystick Gremlin profile'
    # Gremlin stores profiles wherever the user saved them. Two common spots:
    $candidates = @(
        (Join-Path $env:APPDATA 'Joystick Gremlin'),
        (Join-Path $env:USERPROFILE 'Documents\Joystick Gremlin'),
        (Join-Path $env:USERPROFILE 'Documents')
    )
    $found = $null
    foreach ($d in $candidates) {
        if (Test-Path $d) {
            $xml = Get-ChildItem -Path $d -Recurse -Filter '*.xml' -ErrorAction SilentlyContinue |
                   Where-Object { $_.Length -lt 5MB } | Select-Object -First 1
            if ($xml) { $found = $xml; break }
        }
    }
    if (-not $found) {
        Write-Warn 'no Gremlin XML profile found — save yours to %APPDATA%\Joystick Gremlin\fanatec-iracing.xml'
        return
    }
    Copy-Maybe -From $found.FullName -To (Join-Path $bundleDir 'Gremlin\fanatec-iracing.xml')
    Write-OK "captured $($found.FullName)"
}

# ---------------------------------------------------------------- iRacing controls
function Capture-IRacing {
    Write-Section 'iRacing controls.cfg'
    try { $cfg = Get-RigConfig } catch { $cfg = $null }
    $iracingDocs = if ($cfg -and $cfg.games.iRacing.documents) {
        Expand-EnvPath $cfg.games.iRacing.documents
    } else {
        Join-Path $env:USERPROFILE 'Documents\iRacing'
    }
    $controls = Join-Path $iracingDocs 'controls.cfg'
    if (-not (Test-Path $controls)) {
        Write-Warn "controls.cfg not found at $controls — launch iRacing once to generate"
        return
    }
    Copy-Maybe -From $controls -To (Join-Path $bundleDir 'iRacing\controls.cfg.expected')
    Write-OK "captured $controls"
}

# ---------------------------------------------------------------- FanaLab profiles
function Capture-FanaLab {
    Write-Section 'FanaLab profiles'
    Write-Warn 'FanaLab exports manually — Settings → Export Profile, save 3 files into bundle\FanaLab\'
    Write-Warn '(this script does not automate FanaLab; it would require knowing the proprietary export format)'
}

# ---------------------------------------------------------------- Display + audio .bat
function Generate-Scripts {
    Write-Section 'Generate display + audio .bat files from rig-config.json'
    try { $cfg = Get-RigConfig } catch {
        Write-Warn "no rig-config.json — skipping (run from repo root)"
        return
    }
    $outDir = Join-Path $bundleDir 'scripts'
    New-Item -ItemType Directory -Force -Path $outDir | Out-Null

    # Display toggles per the rig-config.displays map
    $mmt = if ($cfg.tools.multiMonitorTool) { Expand-EnvPath $cfg.tools.multiMonitorTool } else { 'C:\Tools\MultiMonitorTool\MultiMonitorTool.exe' }
    foreach ($name in $cfg.displays.PSObject.Properties.Name) {
        $id = $cfg.displays.$name
        $bat = "@echo off`n`"$mmt`" /switch `"$id`""
        $f = Join-Path $outDir "display-toggle-$($name.ToLower()).bat"
        if ($DryRun) { Write-OK "would write $f" }
        else { $bat | Out-File $f -Encoding ascii; Write-OK "wrote $f" }
    }
    # Display presets
    $allIds = ($cfg.displays.PSObject.Properties | ForEach-Object { '"' + $_.Value + '"' }) -join ' '
    $presets = @{
        'all-on'   = "@echo off`n`"$mmt`" /enable $allIds"
        'all-off'  = "@echo off`n`"$mmt`" /disable $allIds"
        'work'     = "@echo off`n`"$mmt`" /enable $allIds"
        'vr-race'  = if ($cfg.displays.ultrawide -and $cfg.displays.topLeft -and $cfg.displays.topRight) {
            "@echo off`n`"$mmt`" /enable `"$($cfg.displays.ultrawide)`"`n`"$mmt`" /disable `"$($cfg.displays.topLeft)`" `"$($cfg.displays.topRight)`""
        } else { "@echo off`nrem vr-race needs ultrawide + topLeft + topRight in rig-config.displays" }
    }
    foreach ($preset in $presets.GetEnumerator()) {
        $f = Join-Path $outDir "display-$($preset.Key).bat"
        if ($DryRun) { Write-OK "would write $f" }
        else { $preset.Value | Out-File $f -Encoding ascii; Write-OK "wrote $f" }
    }

    # Audio scripts per cfg.audio (process name targets)
    $svv = if ($cfg.tools.soundVolumeView) { Expand-EnvPath $cfg.tools.soundVolumeView } else { 'C:\Tools\SoundVolumeView\SoundVolumeView.exe' }
    foreach ($appKey in $cfg.audio.PSObject.Properties.Name) {
        $appName = $cfg.audio.$appKey
        foreach ($dir in @('up','down')) {
            $delta = if ($dir -eq 'up') { '+5' } else { '-5' }
            $bat = "@echo off`n`"$svv`" /ChangeVolume `"$appName`" $delta"
            $f = Join-Path $outDir "$($appKey.ToLower())-$dir.bat"
            if ($DryRun) { Write-OK "would write $f" }
            else { $bat | Out-File $f -Encoding ascii; Write-OK "wrote $f" }
        }
    }
}

# ---------------------------------------------------------------- Main
Write-Host "[bundle] capturing rig configuration → $bundleDir" -ForegroundColor Cyan
if ($DryRun) { Write-Host "[bundle] DRY RUN — no files will be written" -ForegroundColor Yellow }

if (Should-Run 'streamdeck') { Capture-StreamDeck } else { Write-Skip 'streamdeck' }
if (Should-Run 'gremlin')    { Capture-Gremlin }    else { Write-Skip 'gremlin' }
if (Should-Run 'iracing')    { Capture-IRacing }    else { Write-Skip 'iracing' }
if (Should-Run 'fanalab')    { Capture-FanaLab }    else { Write-Skip 'fanalab' }
if (Should-Run 'scripts')    { Generate-Scripts }   else { Write-Skip 'scripts' }

Write-Host "`n[bundle] done. Review with 'git status', then commit:" -ForegroundColor Cyan
Write-Host "  git add bundle/ && git commit -m `"Update bundle from $env:COMPUTERNAME`"" -ForegroundColor Cyan
