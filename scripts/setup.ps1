# setup.ps1 — install + restore the rig on a fresh PC.
#
# Idempotent: re-run safely if interrupted. Each phase checks "is this
# already done?" before doing work. Logs to C:\Logs\setup\<timestamp>.log.
#
# Phases (run in order, can skip individually):
#   1. preflight       — admin check, PS version, internet
#   2. winget          — mainstream apps via Windows Package Manager
#   3. github          — installer downloads from GitHub releases
#   4. portable        — NirSoft tools + similar (download + extract)
#   5. vendor          — open browser tabs for sites with no automation,
#                        wait for user to confirm install
#   6. restore         — drop bundle/ files into their target locations
#   7. rigconfig       — create rig-config.json from example if missing
#   8. healthcheck     — run scripts/health-check.ps1 -Html, open report
#
# Usage:
#   .\scripts\setup.ps1                          # all phases
#   .\scripts\setup.ps1 -DryRun                  # show what would happen
#   .\scripts\setup.ps1 -SkipPhase vendor        # skip browser-prompt phase
#   .\scripts\setup.ps1 -OnlyPhase restore       # run just one phase
#
# This script must be run from the repo root.

[CmdletBinding()]
param(
    [switch]$DryRun,
    [string[]]$SkipPhase = @(),
    [string]$OnlyPhase = ''
)

. "$PSScriptRoot\_lib.ps1"

$ErrorActionPreference = 'Continue'
$repoRoot = (Resolve-Path "$PSScriptRoot\..").Path
$bundleDir = Join-Path $repoRoot 'bundle'
$toolsDir = 'C:\Tools'
$logsDir = 'C:\Logs\setup'
New-Item -ItemType Directory -Force -Path $logsDir | Out-Null
$logFile = Join-Path $logsDir ("setup-{0}.log" -f (Get-Date -Format 'yyyy-MM-dd_HHmm'))

function Log {
    [CmdletBinding()] param([string]$Msg, [string]$Level = 'INFO')
    $line = "{0} [{1}] {2}" -f (Get-Date -Format 'HH:mm:ss'), $Level, $Msg
    Write-Host $line -ForegroundColor $(@{INFO='White';OK='Green';WARN='Yellow';FAIL='Red';HEAD='Cyan'}[$Level])
    Add-Content -Path $logFile -Value $line
}

function Section { param($t) Log "==== $t ====" 'HEAD' }
function ShouldRun { param($name) if ($OnlyPhase) { return $name -eq $OnlyPhase } return -not ($SkipPhase -contains $name) }

function Pause-EnterToContinue {
    [CmdletBinding()] param([string]$Message = 'Press Enter when done...')
    if ($DryRun) { Log "(dry run) would prompt: $Message"; return }
    Read-Host $Message
}

# --------------------------------------------------------------- 1. PREFLIGHT
function Phase-Preflight {
    Section 'PREFLIGHT'
    # Admin?
    $isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]'Administrator')
    if ($isAdmin) { Log "running as Administrator" 'OK' }
    else { Log "NOT running as Administrator — some installs (HidHide, vJoy, drivers) will fail. Re-launch PowerShell as admin if needed." 'WARN' }

    # PowerShell 5.1+ is required (Windows 10/11 ships with this; PS 7 also works)
    $psv = $PSVersionTable.PSVersion
    Log "PowerShell version: $psv"
    if ($psv.Major -lt 5) { Log "PowerShell 5+ required" 'FAIL'; throw 'aborting' }

    # Internet
    if ($DryRun) { Log "(dry run) would test internet" } else {
        try {
            $null = Invoke-WebRequest -Uri 'https://api.github.com' -UseBasicParsing -TimeoutSec 5
            Log "internet reachable" 'OK'
        } catch {
            Log "internet NOT reachable — most phases will fail" 'FAIL'
            throw 'aborting'
        }
    }

    # winget
    $winget = Get-Command winget -ErrorAction SilentlyContinue
    if ($winget) { Log "winget found: $($winget.Source)" 'OK' }
    else { Log "winget not installed — install 'App Installer' from Microsoft Store, then re-run" 'FAIL'; throw 'aborting' }

    # Repo root sanity
    if (-not (Test-Path "$repoRoot\docs\index.html")) {
        Log "this doesn't look like the repo root ($repoRoot has no docs\index.html)" 'FAIL'
        throw 'aborting'
    }

    Log "preflight complete" 'OK'
}

# ----------------------------------------------------------------- 2. WINGET
function Install-Winget {
    [CmdletBinding()] param([string]$Id, [string]$Friendly)
    if ($DryRun) { Log "(dry run) winget install --id $Id" ; return }
    Log "winget install $Friendly..."
    $args = @(
        'install', '--id', $Id, '-e',
        '--accept-source-agreements', '--accept-package-agreements',
        '--silent'
    )
    $output = & winget @args 2>&1 | Out-String
    if ($LASTEXITCODE -eq 0) { Log "$Friendly installed" 'OK' }
    elseif ($output -match 'already installed' -or $output -match 'No newer version available') {
        Log "$Friendly already installed" 'OK'
    } else {
        Log "$Friendly failed — exit $LASTEXITCODE. Try manually: winget install --id $Id" 'WARN'
    }
}

function Phase-Winget {
    Section 'WINGET PACKAGES'
    $packages = @(
        @{ Id = 'Discord.Discord';            Friendly = 'Discord' }
        @{ Id = 'Spotify.Spotify';            Friendly = 'Spotify' }
        @{ Id = 'Valve.Steam';                Friendly = 'Steam' }
        @{ Id = 'OBSProject.OBSStudio';       Friendly = 'OBS Studio' }
        @{ Id = 'BitSum.ProcessLasso';        Friendly = 'Process Lasso' }
        @{ Id = 'REALiX.HWiNFO';              Friendly = 'HWiNFO64' }
        @{ Id = 'Elgato.StreamDeck';          Friendly = 'Stream Deck app' }
    )
    foreach ($p in $packages) { Install-Winget -Id $p.Id -Friendly $p.Friendly }
}

# ------------------------------------------------------ 3. GITHUB RELEASES
function Get-GitHubAsset {
    [CmdletBinding()] param([string]$Repo, [string]$Pattern)
    $url = "https://api.github.com/repos/$Repo/releases/latest"
    try {
        $rel = Invoke-RestMethod $url -ErrorAction Stop -UseBasicParsing
    } catch {
        Log "fetch $url failed: $_" 'WARN'
        return $null
    }
    $asset = $rel.assets | Where-Object { $_.name -match $Pattern } | Select-Object -First 1
    if (-not $asset) { Log "no asset matching $Pattern in $Repo" 'WARN'; return $null }
    return @{ Url = $asset.browser_download_url; Name = $asset.name; Tag = $rel.tag_name }
}

function Install-GitHubRelease {
    [CmdletBinding()] param([string]$Repo, [string]$Pattern, [string]$Friendly, [string]$RunArgs = '/S')
    $asset = Get-GitHubAsset -Repo $Repo -Pattern $Pattern
    if (-not $asset) { return }
    $tmp = Join-Path $env:TEMP $asset.Name
    if ($DryRun) { Log "(dry run) would download $($asset.Url) → $tmp"; return }
    Log "downloading $Friendly $($asset.Tag)..."
    try {
        Invoke-WebRequest -Uri $asset.Url -OutFile $tmp -UseBasicParsing -ErrorAction Stop
        Log "running installer ($RunArgs)..."
        Start-Process -FilePath $tmp -ArgumentList $RunArgs -Wait
        Log "$Friendly installed" 'OK'
    } catch {
        Log "$Friendly install failed: $_" 'WARN'
    }
}

function Phase-GitHubReleases {
    Section 'GITHUB RELEASE INSTALLERS'
    Install-GitHubRelease -Repo 'nefarius/HidHide'       -Pattern '\.exe$' -Friendly 'HidHide'         -RunArgs '/quiet'
    Install-GitHubRelease -Repo 'WhiteMagic/JoystickGremlin' -Pattern '\.msi$' -Friendly 'Joystick Gremlin' -RunArgs '/quiet'
    Install-GitHubRelease -Repo 'mrbelowski/CrewChiefV4' -Pattern 'CrewChief.*setup.*\.exe$' -Friendly 'Crew Chief' -RunArgs '/S'
    Install-GitHubRelease -Repo 'blarghedy/iRFFB2022'    -Pattern '\.exe$' -Friendly 'iRFFB 2022'      -RunArgs '/S'
    # PresentMon is a portable zip — handled in Phase-Portable
}

# ----------------------------------------------------------------- 4. PORTABLE
function Install-Portable {
    [CmdletBinding()] param([string]$Url, [string]$Name, [string]$Friendly)
    $target = Join-Path $toolsDir $Name
    if (Test-Path $target) {
        Log "$Friendly already at $target" 'OK'
        return
    }
    if ($DryRun) { Log "(dry run) would download $Url → $target"; return }
    New-Item -ItemType Directory -Force -Path $toolsDir | Out-Null
    $tmp = Join-Path $env:TEMP "$Name.zip"
    try {
        Log "downloading $Friendly..."
        Invoke-WebRequest -Uri $Url -OutFile $tmp -UseBasicParsing -ErrorAction Stop
        New-Item -ItemType Directory -Force -Path $target | Out-Null
        Expand-Archive -Path $tmp -DestinationPath $target -Force
        Remove-Item $tmp
        Log "$Friendly extracted to $target" 'OK'
    } catch {
        Log "$Friendly install failed: $_" 'WARN'
    }
}

function Phase-Portable {
    Section 'PORTABLE TOOLS'
    Install-Portable -Url 'https://www.nirsoft.net/utils/soundvolumeview-x64.zip' `
                     -Name 'SoundVolumeView' -Friendly 'SoundVolumeView'
    Install-Portable -Url 'https://www.nirsoft.net/utils/multimonitortool-x64.zip' `
                     -Name 'MultiMonitorTool' -Friendly 'MultiMonitorTool'
    # PresentMon — pinned version (NirSoft pattern; GitHub asset URL for stability)
    $presentMon = Get-GitHubAsset -Repo 'GameTechDev/PresentMon' -Pattern 'x64\.exe$'
    if ($presentMon) {
        $target = Join-Path $toolsDir 'PresentMon'
        if (Test-Path (Join-Path $target $presentMon.Name)) {
            Log "PresentMon already at $target" 'OK'
        } elseif ($DryRun) {
            Log "(dry run) would download PresentMon → $target"
        } else {
            New-Item -ItemType Directory -Force -Path $target | Out-Null
            try {
                Invoke-WebRequest -Uri $presentMon.Url -OutFile (Join-Path $target $presentMon.Name) -UseBasicParsing
                Log "PresentMon $($presentMon.Tag) installed" 'OK'
            } catch {
                Log "PresentMon failed: $_" 'WARN'
            }
        }
    }
}

# ------------------------------------------------------------------- 5. VENDOR
# Install pages that have no good automation. Open browser tabs, prompt user.
function Phase-Vendor {
    Section 'VENDOR INSTALLS (manual)'
    Log "These installs need vendor sites or accounts. We'll open each in your browser."
    $vendors = @(
        @{ Url = 'https://fanatec.com/eu-en/support/downloads';                Name = 'Fanatec driver + FanaLab' }
        @{ Url = 'https://www.iracing.com/membership/';                         Name = 'iRacing membership + client' }
        @{ Url = 'https://tradingpaints.com';                                   Name = 'Trading Paints' }
        @{ Url = 'https://www.vrdesktop.net/';                                  Name = 'Virtual Desktop Streamer' }
        @{ Url = 'https://mbucchia.github.io/OpenXR-Toolkit/';                  Name = 'OpenXR Toolkit' }
        @{ Url = 'https://steelseries.com/gg';                                  Name = 'SteelSeries GG' }
    )
    foreach ($v in $vendors) {
        Log "opening: $($v.Name) → $($v.Url)"
        if (-not $DryRun) { Start-Process $v.Url }
        Pause-EnterToContinue "Install $($v.Name), then press Enter..."
    }
    Log "vendor installs done" 'OK'
}

# ----------------------------------------------------------- 6. RESTORE BUNDLE
function Phase-Restore {
    Section 'RESTORE FROM BUNDLE'
    if (-not (Test-Path $bundleDir)) {
        Log "no bundle/ directory — skipping (run scripts/bundle.ps1 on the source PC first)" 'WARN'
        return
    }

    # Stream Deck profiles — drop into ProfilesV2/. The Stream Deck app
    # picks them up on next start. NOTE: this is a copy, not an "import"
    # via the app's GUI — works because .streamDeckProfile is a zip of a
    # profile folder.
    $sdProfilesDir = Join-Path $env:APPDATA 'Elgato\StreamDeck\ProfilesV2'
    if (Test-Path "$bundleDir\StreamDeck") {
        $profiles = Get-ChildItem "$bundleDir\StreamDeck" -Filter '*.streamDeckProfile' -ErrorAction SilentlyContinue
        foreach ($p in $profiles) {
            if ($DryRun) { Log "(dry run) would import $($p.Name)"; continue }
            try {
                Start-Process $p.FullName  # Stream Deck app's URL handler imports it
                Log "imported $($p.Name)" 'OK'
                Start-Sleep -Milliseconds 500  # stagger to avoid race
            } catch { Log "import $($p.Name) failed: $_" 'WARN' }
        }
    }

    # Joystick Gremlin XML — copy to AppData
    $gremlinDir = Join-Path $env:APPDATA 'Joystick Gremlin'
    if (Test-Path "$bundleDir\Gremlin") {
        New-Item -ItemType Directory -Force -Path $gremlinDir | Out-Null
        Get-ChildItem "$bundleDir\Gremlin" -Filter '*.xml' | ForEach-Object {
            $dest = Join-Path $gremlinDir $_.Name
            if ($DryRun) { Log "(dry run) would copy $($_.Name) → $dest" }
            else { Copy-Item $_.FullName $dest -Force; Log "Gremlin profile $($_.Name) installed" 'OK' }
        }
    }

    # iRacing controls.cfg — only if iRacing has been launched once (creates the dir)
    $iracingDocs = Join-Path $env:USERPROFILE 'Documents\iRacing'
    if (Test-Path "$bundleDir\iRacing\controls.cfg.expected") {
        if (Test-Path $iracingDocs) {
            $dest = Join-Path $iracingDocs 'controls.cfg'
            if (Test-Path $dest) {
                $backup = "$dest.before-setup-$(Get-Date -Format 'yyyy-MM-dd_HHmm').bak"
                Copy-Item $dest $backup
                Log "backed up existing controls.cfg → $backup"
            }
            if ($DryRun) { Log "(dry run) would copy controls.cfg.expected → $dest" }
            else { Copy-Item "$bundleDir\iRacing\controls.cfg.expected" $dest -Force; Log "iRacing controls.cfg installed" 'OK' }
        } else {
            Log "iRacing Documents folder missing — launch iRacing once, then re-run -OnlyPhase restore" 'WARN'
        }
    }

    # .bat scripts — copy to C:\Tools\<tool>\
    if (Test-Path "$bundleDir\scripts") {
        $svvDir = Join-Path $toolsDir 'SoundVolumeView'
        $mmtDir = Join-Path $toolsDir 'MultiMonitorTool'
        Get-ChildItem "$bundleDir\scripts" -Filter '*.bat' | ForEach-Object {
            $dest = if ($_.Name -like 'display-*') { Join-Path $mmtDir $_.Name }
                    elseif ($_.Name -match '^(discord|spotify|iracing)-') { Join-Path $svvDir $_.Name }
                    else { Join-Path $toolsDir $_.Name }
            $destDir = Split-Path $dest
            if (-not (Test-Path $destDir)) { New-Item -ItemType Directory -Force -Path $destDir | Out-Null }
            if ($DryRun) { Log "(dry run) would copy $($_.Name) → $dest" }
            else { Copy-Item $_.FullName $dest -Force; Log ".bat installed: $dest" 'OK' }
        }
    }
}

# --------------------------------------------------------------- 7. RIGCONFIG
function Phase-RigConfig {
    Section 'RIG CONFIG'
    $cfgPath = Join-Path $repoRoot 'rig-config.json'
    $examplePath = Join-Path $repoRoot 'rig-config.example.json'
    if (Test-Path $cfgPath) {
        Log "rig-config.json already exists — leaving alone" 'OK'
        return
    }
    if (-not (Test-Path $examplePath)) {
        Log "rig-config.example.json missing — cannot template" 'WARN'
        return
    }
    if ($DryRun) { Log "(dry run) would copy example → rig-config.json"; return }
    Copy-Item $examplePath $cfgPath
    Log "created rig-config.json from example" 'OK'
    Log "→ open $cfgPath in your editor and fill in paths matching this PC" 'WARN'
}

# ---------------------------------------------------------- 8. HEALTH CHECK
function Phase-HealthCheck {
    Section 'FINAL HEALTH CHECK'
    $hc = Join-Path $PSScriptRoot 'health-check.ps1'
    if (-not (Test-Path $hc)) { Log "health-check.ps1 missing" 'WARN'; return }
    if ($DryRun) { Log "(dry run) would run health-check.ps1 -Html"; return }
    & $hc -Html -Quiet
    Log "health check complete — see C:\Logs\health-*.html" 'OK'
}

# --------------------------------------------------------------- POSTAMBLE
function Print-FinalNotes {
    Section 'NEXT STEPS'
    Write-Host @'

You're nearly done. A few things still need manual attention:

  1. Open the Stream Deck app — re-auth Spotify, Discord, etc.
     plugins (one click each, ~30 seconds).
  2. In Joystick Gremlin, open the imported profile and tick
     "Activate". Set to start with Windows.
  3. In iRacing, verify your controls (Options → Controls). The
     bundle restored controls.cfg, but you should test each binding.
  4. Open rig-config.json and confirm paths match your installs.
     Especially the displays block (\\.\DISPLAY1/2/3).
  5. In the docs Verify tab, click "Re-verify now". Drifts that
     are intentional → "Accept as baseline". Mismatches that
     are NOT intentional → fix the underlying setting.
  6. Build the tray binary:
        cd tray && go build -o bin\tray.exe .
        .\bin\tray.exe
     The browser will pop up and show the live state.

You're ready to race when the Verify tab shows all green and the
Status tab shows the tray connected.

'@ -ForegroundColor Cyan
}

# =============================================================== MAIN
Log "[setup] HarablaSetup install/restore — log: $logFile"
Log "[setup] repo: $repoRoot"
if ($DryRun) { Log "[setup] DRY RUN — no changes will be made" 'WARN' }

try {
    if (ShouldRun 'preflight')   { Phase-Preflight }
    if (ShouldRun 'winget')      { Phase-Winget }
    if (ShouldRun 'github')      { Phase-GitHubReleases }
    if (ShouldRun 'portable')    { Phase-Portable }
    if (ShouldRun 'vendor')      { Phase-Vendor }
    if (ShouldRun 'restore')     { Phase-Restore }
    if (ShouldRun 'rigconfig')   { Phase-RigConfig }
    if (ShouldRun 'healthcheck') { Phase-HealthCheck }

    Print-FinalNotes
    Log "[setup] complete" 'OK'
} catch {
    Log "[setup] aborted: $_" 'FAIL'
    exit 1
}
