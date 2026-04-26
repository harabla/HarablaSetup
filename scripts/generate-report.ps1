# generate-report.ps1 — turn a session log dir into a single HTML report.
#
# Reads:  hwinfo.csv, presentmon.csv, processes.csv, *.snapshot
# Writes: report.html (self-contained, Chart.js from CDN)

[CmdletBinding()]
param(
    [Parameter(Mandatory)][string]$LogDir
)

$ErrorActionPreference = 'Continue'

# ---------- Parse processes.csv ----------
$procsCsvPath = Join-Path $LogDir 'processes.csv'
$topByCpu = @()
if (Test-Path $procsCsvPath) {
    $procs = Import-Csv $procsCsvPath
    $topByCpu = $procs | Group-Object process | ForEach-Object {
        [PSCustomObject]@{
            Name    = $_.Name
            AvgCpu  = ($_.Group.cpu_percent | Measure-Object -Average).Average
            PeakCpu = ($_.Group.cpu_percent | Measure-Object -Maximum).Maximum
            AvgRam  = ($_.Group.ram_bytes   | Measure-Object -Average).Average / 1MB
        }
    } | Sort-Object PeakCpu -Descending | Select-Object -First 15
}

# ---------- Parse PresentMon for FPS / frametime ----------
$avgFps = 0; $p99 = 0; $stutters = 0; $msPerFrame = @()
$presentMonPath = Join-Path $LogDir 'presentmon.csv'
if (Test-Path $presentMonPath) {
    $frames = Import-Csv $presentMonPath
    $msPerFrame = $frames.MsBetweenPresents | ForEach-Object { [double]$_ }
    if ($msPerFrame.Count -gt 0) {
        $avgFps = [math]::Round(1000 / ($msPerFrame | Measure-Object -Average).Average, 1)
        $sortedMs = $msPerFrame | Sort-Object
        $p99 = $sortedMs[[int]($sortedMs.Count * 0.99)]
        $stutters = ($msPerFrame | Where-Object { $_ -gt 33 }).Count
    }
}

# ---------- Parse iRacing app.ini for graphics summary ----------
$iniSettings = @{}
$iniSnapshot = Join-Path $LogDir 'app.ini.snapshot'
if (Test-Path $iniSnapshot) {
    Get-Content $iniSnapshot | ForEach-Object {
        if ($_ -match '^\s*([^=;\s]+)\s*=\s*(.+)$') {
            $iniSettings[$matches[1]] = $matches[2].Trim()
        }
    }
}
$keysOfInterest = @('Resolution','MultiSamples','MaxQuality','MirrorHigh','ShadowMaps','Particles','EventDistance','TextureQuality')
$gfxRows = ($keysOfInterest | Where-Object { $iniSettings.ContainsKey($_) } |
    ForEach-Object { "<tr><td>$_</td><td>$($iniSettings[$_])</td></tr>" }) -join "`n"
if (-not $gfxRows) { $gfxRows = "<tr><td colspan=2 style='color:#888'>no app.ini snapshot</td></tr>" }

# ---------- Top processes table ----------
$procRows = ($topByCpu | ForEach-Object {
    "<tr><td>$($_.Name)</td><td>$([math]::Round($_.PeakCpu,1))</td><td>$([math]::Round($_.AvgCpu,1))</td><td>$([math]::Round($_.AvgRam,0))</td></tr>"
}) -join "`n"
if (-not $procRows) { $procRows = "<tr><td colspan=4 style='color:#888'>no processes.csv</td></tr>" }

# ---------- Snapshot file list ----------
$snapshots = (Get-ChildItem $LogDir | ForEach-Object {
    "<li><code>$($_.Name)</code> ($([math]::Round($_.Length/1KB,1)) KB)</li>"
}) -join "`n"

# ---------- Frametime chart data (cap at 5000 points to keep HTML small) ----------
$msSlice = if ($msPerFrame.Count -gt 5000) { $msPerFrame | Select-Object -First 5000 } else { $msPerFrame }
$msJs = if ($msSlice.Count -gt 0) { ($msSlice | ForEach-Object { [string]$_ }) -join ',' } else { '' }

# ---------- Build HTML ----------
$sessionName = Split-Path $LogDir -Leaf
$html = @"
<!doctype html><html><head><meta charset='utf-8'>
<title>iRacing session report — $sessionName</title>
<script src='https://cdn.jsdelivr.net/npm/chart.js'></script>
<style>
body{font-family:system-ui;background:#0a0d12;color:#e0e0e0;padding:24px;max-width:1200px;margin:auto}
h1,h2{color:#5a8;border-bottom:1px solid #333;padding-bottom:6px}
table{border-collapse:collapse;width:100%;margin:12px 0}
th,td{text-align:left;padding:6px 12px;border-bottom:1px solid #222}
th{color:#888;font-size:12px;text-transform:uppercase}
.metric{display:inline-block;background:#1a2030;padding:12px 18px;border-radius:8px;margin:6px}
.metric .v{font-size:24px;font-weight:700;color:#5a8}
.metric .l{font-size:11px;color:#888;text-transform:uppercase}
canvas{background:#1a1d22;border-radius:8px;padding:12px;margin:12px 0}
code{background:#1a2030;padding:2px 6px;border-radius:3px;font-size:12px}
ul{line-height:1.7}
</style></head><body>
<h1>iRacing session — $sessionName</h1>

<div>
  <div class='metric'><div class='v'>$avgFps</div><div class='l'>Avg FPS</div></div>
  <div class='metric'><div class='v'>$([math]::Round($p99,1)) ms</div><div class='l'>P99 frametime</div></div>
  <div class='metric'><div class='v'>$stutters</div><div class='l'>Stutters (>33ms)</div></div>
</div>

<h2>Top processes by peak CPU</h2>
<table><tr><th>Process</th><th>Peak CPU %</th><th>Avg CPU %</th><th>Avg RAM (MB)</th></tr>
$procRows
</table>

<h2>iRacing graphics settings</h2>
<table><tr><th>Setting</th><th>Value</th></tr>
$gfxRows
</table>

<h2>Frametime over session</h2>
<canvas id='ft' height='80'></canvas>
<script>
const ms = [$msJs];
if (ms.length > 0) {
  new Chart(document.getElementById('ft'), {
    type: 'line',
    data: { labels: ms.map((_,i)=>i), datasets: [{ label:'ms', data: ms, borderColor:'#5a8', borderWidth:1, pointRadius:0 }] },
    options: { animation:false, scales:{ y:{ suggestedMin:0, suggestedMax:50 } }, plugins:{ legend:{display:false} } }
  });
} else {
  document.getElementById('ft').replaceWith(Object.assign(document.createElement('div'), { textContent:'no frame data', style:'color:#888;padding:20px' }));
}
</script>

<h2>Files in this session</h2>
<ul>
$snapshots
</ul>

</body></html>
"@

$html | Out-File -FilePath (Join-Path $LogDir 'report.html') -Encoding utf8
Write-Host "[report] generated: $(Join-Path $LogDir 'report.html')" -ForegroundColor Cyan
