// Gaming reference — render bindings from inline JSON, plus search / detail / car-class toggle.

const $ = (sel, root = document) => root.querySelector(sel);
const $$ = (sel, root = document) => Array.from(root.querySelectorAll(sel));

// ---------- Inline SVG icons (placeholders until real Stream Deck PNGs arrive) ----------
// Why inline rather than <img src="icons/foo.svg">: the page opens via file://,
// where SVGs loaded as <img> can't inherit currentColor from the parent cell.
// Inlining via innerHTML lets the icon's stroke pick up the cell's text colour.
const svg = (body, vb = '0 0 24 24') =>
  `<svg viewBox="${vb}" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${body}</svg>`;
const ICONS = {
  crosshair: svg('<circle cx="12" cy="12" r="6"/><line x1="12" y1="2" x2="12" y2="6"/><line x1="12" y1="18" x2="12" y2="22"/><line x1="2" y1="12" x2="6" y2="12"/><line x1="18" y1="12" x2="22" y2="12"/><circle cx="12" cy="12" r="1.4" fill="currentColor"/>'),
  eq:        svg('<rect x="4"  y="9"  width="3" height="11" rx="0.5" fill="currentColor" stroke="none"/><rect x="10" y="4"  width="3" height="16" rx="0.5" fill="currentColor" stroke="none"/><rect x="16" y="13" width="3" height="7"  rx="0.5" fill="currentColor" stroke="none"/>'),
  micMute:   svg('<rect x="9" y="3" width="6" height="11" rx="3"/><path d="M5 11a7 7 0 0 0 14 0"/><line x1="12" y1="18" x2="12" y2="22"/><line x1="3" y1="3" x2="21" y2="21" stroke-width="2.2"/>'),
  scissors:  svg('<circle cx="6" cy="7" r="2.8"/><circle cx="6" cy="17" r="2.8"/><line x1="20" y1="4" x2="8.4" y2="15.6"/><line x1="14.6" y1="14.4" x2="20" y2="20"/><line x1="8.4" y1="8.4" x2="11.6" y2="11.6"/>'),
  live:      svg('<circle cx="12" cy="12" r="3" fill="currentColor"/><path d="M6.5 6.5a8 8 0 0 0 0 11"/><path d="M17.5 6.5a8 8 0 0 1 0 11"/><path d="M3 3a13 13 0 0 0 0 18"/><path d="M21 3a13 13 0 0 1 0 18"/>'),
  volDown:   svg('<path d="M3 9v6h4l5 4V5L7 9H3z" fill="currentColor"/><line x1="16" y1="12" x2="22" y2="12" stroke-width="2.2"/>'),
  volUp:     svg('<path d="M3 9v6h4l5 4V5L7 9H3z" fill="currentColor"/><line x1="19" y1="9" x2="19" y2="15" stroke-width="2.2"/><line x1="16" y1="12" x2="22" y2="12" stroke-width="2.2"/>'),
  discDown:  svg('<path d="M21 11.5a8 8 0 0 1-12.6 6.5L3 19.5l1.6-5A8 8 0 1 1 21 11.5z"/><line x1="9" y1="12" x2="15" y2="12" stroke-width="2.2"/>'),
  discUp:    svg('<path d="M21 11.5a8 8 0 0 1-12.6 6.5L3 19.5l1.6-5A8 8 0 1 1 21 11.5z"/><line x1="9" y1="12" x2="15" y2="12" stroke-width="2.2"/><line x1="12" y1="9" x2="12" y2="15" stroke-width="2.2"/>'),
  home:      svg('<path d="M3 11l9-8 9 8"/><path d="M5 10v10h14V10"/><path d="M10 20v-6h4v6"/>'),
  swap:      svg('<path d="M3 9h13l-3-3M3 9l3 3"/><path d="M21 15H8l3-3M21 15l-3 3"/>'),
  play:      svg('<polygon points="7,4 7,20 20,12" fill="currentColor" stroke="none"/>'),
  prev:      svg('<line x1="6" y1="5" x2="6" y2="19" stroke-width="2.4"/><polygon points="20,5 8,12 20,19" fill="currentColor" stroke="none"/>'),
  next:      svg('<polygon points="4,5 16,12 4,19" fill="currentColor" stroke="none"/><line x1="18" y1="5" x2="18" y2="19" stroke-width="2.4"/>'),
  heart:     svg('<path d="M12 20s-7.5-4.7-9.5-9.5C1 7 4 4 7 4c2 0 3.4 1.1 5 3 1.6-1.9 3-3 5-3 3 0 6 3 4.5 6.5C19.5 15.3 12 20 12 20z"/>'),
};

const data = {
  home:      readJSON('data-streamdeck-home'),
  pubg:      readJSON('data-streamdeck-pubg'),
  iracing:   readJSON('data-streamdeck-iracing'),
  streaming: readJSON('data-streamdeck-streaming'),
  audio:     readJSON('data-streamdeck-audio'),
  display:   readJSON('data-streamdeck-display'),
  keyboard:  readJSON('data-pubg-keyboard'),
  scimitar:  readJSON('data-pubg-scimitar'),
  wheel:     readJSON('data-iracing-wheel'),
  preflight: readJSON('data-preflight'),
  meta:      readJSON('data-meta'),
};

// ---------- App running-state ----------
// Two sources, in priority order:
//   1. Live data from the tray's /api/state endpoint (when the Go tray is
//      running on the same host). Real OS process names from Get-Process.
//   2. localStorage mock — clicking a launcher cell or pre-flight pill
//      toggles a per-app boolean, persisted across reloads. Used for design
//      preview when the tray isn't running.
const RUNNING_KEY = 'gaming-ref:running';
function loadRunning() {
  try { return JSON.parse(localStorage.getItem(RUNNING_KEY)) || {}; } catch (_) { return {}; }
}
function saveRunning(state) {
  try { localStorage.setItem(RUNNING_KEY, JSON.stringify(state)); } catch (_) {}
}
const running = loadRunning();

// Live state populated by probeTray() — null until first successful fetch.
const live = { processes: null, displays: null, vjoy: null, health: null, connected: false };

// Map a friendly preflight/launcher name (e.g. "iRacing", "Crew Chief") to
// process-name substrings the live API might return. Lowercased + spaces
// stripped match against any process name reported by the tray.
function processMatches(appName) {
  if (!live.processes) return false;
  const norm = s => (s || '').toLowerCase().replace(/[\s_-]/g, '');
  const target = norm(appName);
  // Hand-tuned aliases for apps whose process name doesn't contain the friendly name
  const aliases = {
    'iracing':         ['iracingsim64dx11', 'iracingui'],
    'crewchief':       ['crewchief', 'crewchiefv4'],
    'simhub':          ['simhub', 'simhubwpf'],
    'tradingpaints':   ['tradingpaints'],
    'fanalab':         ['fanalab'],
    'joystickgremlin': ['joystickgremlin', 'joystick_gremlin'],
    'vjoy':            ['vjoyconf'],          // vJoy itself is a driver, no process; vJoyConf is the GUI
    'hidhide':         ['hidhideclient', 'hidhide'],
    'soundvolumeview': ['soundvolumeview'],
    'virtualdesktop':  ['virtualdesktop', 'vdstreamerservice'],
    'irffb2022':       ['irffb', 'irffb2022'],
    'hwinfo64':        ['hwinfo64'],
    'spotify':         ['spotify'],
    'obsstudio':       ['obs64', 'obs32'],
    'discord':         ['discord'],
    'icue':            ['icue'],
    'steelseriesgg':   ['steelseries', 'sssgame'],
  };
  const candidates = aliases[target] || [target];
  return live.processes.some(p => {
    const np = norm(p.name);
    return candidates.some(c => np.includes(c));
  });
}

function isRunning(app) {
  if (live.connected) return processMatches(app);
  return !!running[app];
}
function toggleRunning(app) {
  if (live.connected) return; // live data — no manual toggle
  running[app] = !running[app];
  saveRunning(running);
  rerenderRunningSurfaces();
}
function rerenderRunningSurfaces() {
  $$('.streamdeck').forEach(el => {
    const key = el.dataset.grid.replace('streamdeck-', '');
    if (data[key]) renderStreamdeck(el, data[key]);
  });
  renderAllPreflight();
}

// Probe the tray API every 10s. Single-shot first, then start the loop.
async function probeTray() {
  try {
    const r = await fetch('/api/state', { cache: 'no-store' });
    if (!r.ok) throw new Error('tray returned ' + r.status);
    const s = await r.json();
    live.processes = s.processes;
    live.displays  = s.displays;
    live.vjoy      = s.vjoy;
    live.health    = s.health;
    live.configFrom = s.config_from;
    live.os         = s.os;
    live.lastRefresh = new Date();
    if (!live.connected) {
      live.connected = true;
      updateLiveBadge(true);
      // Fetch full config once on connect (it's static for a session)
      try {
        const cr = await fetch('/api/config', { cache: 'no-store' });
        if (cr.ok) live.config = await cr.json();
      } catch (_) {}
    }
    // Game-launch watcher state (live games + last verify per game)
    try {
      const gr = await fetch('/api/games', { cache: 'no-store' });
      if (gr.ok) live.games = await gr.json();
    } catch (_) {}

    rerenderRunningSurfaces();
    renderHealthCards();
    renderStatusPage();
    renderOverviewTile();
    refreshVerifyDrifts();
  } catch (e) {
    if (live.connected) {
      live.connected = false;
      updateLiveBadge(false);
      rerenderRunningSurfaces();
      renderHealthCards();
      renderStatusPage();
      renderOverviewTile();
      refreshVerifyDrifts();
    }
  }
}

// Settings drift — fetched on every probe; renders into #verify-results.
async function refreshVerifyDrifts() {
  const root = $('#verify-results');
  if (!root) return;
  if (!live.connected) {
    root.innerHTML = '<div class="verify-empty">Tray not connected — drift detection unavailable.</div>';
    return;
  }
  try {
    const r = await fetch('/api/verify', { cache: 'no-store' });
    if (!r.ok) throw new Error('verify ' + r.status);
    const results = await r.json();
    renderVerifyResults(results);
  } catch (e) {
    root.innerHTML = `<div class="verify-empty">Verify endpoint failed: ${escapeHTML(String(e))}</div>`;
  }
}

function renderVerifyResults(results) {
  const root = $('#verify-results');
  if (!root) return;
  if (!results || !results.length) {
    root.innerHTML = '<div class="verify-empty">No games configured with settings_files. Add some to rig-config.json.</div>';
    return;
  }
  root.innerHTML = results.map(r => {
    const totalDrifts = r.drift_count || 0;
    const status = totalDrifts > 0 ? 'drift' : (r.missing_files > 0 ? 'warn' : 'ok');
    return `
    <div class="verify-target verify-target--${status}">
      <div class="verify-target-head">
        <h3>${escapeHTML(r.target)}</h3>
        <span class="verify-target-summary">
          ${totalDrifts > 0 ? `<span class="verify-count drift">${totalDrifts} drift${totalDrifts !== 1 ? 's' : ''}</span>` : ''}
          ${r.ok_count ? `<span class="verify-count ok">${r.ok_count} ok</span>` : ''}
          ${r.missing_files ? `<span class="verify-count warn">${r.missing_files} missing</span>` : ''}
        </span>
      </div>
      ${(r.files || []).map(f => renderVerifyFile(r.target, f)).join('')}
    </div>`;
  }).join('');

  // Wire accept-baseline buttons
  root.querySelectorAll('.verify-baseline-btn').forEach(btn => {
    btn.addEventListener('click', async () => {
      const game = btn.dataset.game;
      const file = btn.dataset.file;
      const key = btn.dataset.key;
      const actual = btn.dataset.actual;
      btn.disabled = true; btn.textContent = 'saving…';
      const res = await fetch(`/api/verify/${encodeURIComponent(game)}/baseline`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ file, key, actual }),
      }).then(r => r.json()).catch(e => ({ status: 'error', detail: String(e) }));
      if (res.status === 'ok') {
        btn.textContent = '✓ accepted';
        setTimeout(refreshVerifyDrifts, 400);
      } else {
        btn.textContent = '✗ ' + (res.detail || 'failed');
        btn.disabled = false;
      }
    });
  });
}

function renderVerifyFile(target, f) {
  const fname = (f.path || '').split(/[\\\/]/).pop();
  if (f.status === 'ok') {
    return `<div class="verify-file ok">
      <div class="verify-file-head"><span class="vf-cat">${escapeHTML(f.category)}</span><code>${escapeHTML(fname)}</code><span class="vf-ok">✓ all expected values match</span></div>
    </div>`;
  }
  if (f.status === 'missing') {
    return `<div class="verify-file warn">
      <div class="verify-file-head"><span class="vf-cat">${escapeHTML(f.category)}</span><code>${escapeHTML(fname)}</code><span class="vf-warn">⚠ file not found at <code>${escapeHTML(f.path)}</code></span></div>
    </div>`;
  }
  if (f.status === 'error') {
    return `<div class="verify-file warn">
      <div class="verify-file-head"><span class="vf-cat">${escapeHTML(f.category)}</span><code>${escapeHTML(fname)}</code><span class="vf-warn">⚠ ${escapeHTML(f.error || 'parse error')}</span></div>
    </div>`;
  }
  // drift
  const drifts = f.drifts || [];
  return `<div class="verify-file drift">
    <div class="verify-file-head"><span class="vf-cat">${escapeHTML(f.category)}</span><code>${escapeHTML(fname)}</code><span class="vf-drift">${drifts.length} drift${drifts.length !== 1 ? 's' : ''}</span></div>
    <table class="verify-drift-table">
      <thead><tr><th>Key</th><th>Expected</th><th>Actual</th><th></th></tr></thead>
      <tbody>
      ${drifts.map(d => `<tr>
        <td><code>${escapeHTML(d.key)}</code></td>
        <td><code class="d-expected">${escapeHTML(d.expected)}</code></td>
        <td><code class="d-actual ${d.reason === 'missing_key' ? 'missing' : ''}">${d.reason === 'missing_key' ? '(missing)' : escapeHTML(d.actual)}</code></td>
        <td>${d.reason === 'value_mismatch'
          ? `<button class="wheel-calib-btn verify-baseline-btn"
                data-game="${escapeAttr(target)}"
                data-file="${escapeAttr(f.path)}"
                data-key="${escapeAttr(d.key)}"
                data-actual="${escapeAttr(d.actual)}">Accept as baseline</button>`
          : ''}</td>
      </tr>`).join('')}
      </tbody>
    </table>
  </div>`;
}

function renderOverviewTile() {
  const trayEl   = $('#overview-live-tray');
  const healthEl = $('#overview-live-health');
  const procsEl  = $('#overview-live-procs');
  const sessionEl = $('#overview-live-session');
  const gameEl   = $('#overview-live-game');
  if (!trayEl) return;

  if (!live.connected) {
    trayEl.textContent = '○ not connected';
    trayEl.className = 'off';
    healthEl.textContent = '—';
    procsEl.textContent = '—';
    sessionEl.textContent = '—';
    if (gameEl) { gameEl.textContent = '—'; gameEl.className = ''; }
    return;
  }
  trayEl.textContent = '● connected';
  trayEl.className = 'on';

  if (live.health) {
    const counts = { ok: 0, warn: 0, fail: 0 };
    Object.values(live.health).forEach(c => counts[c.status] = (counts[c.status] || 0) + 1);
    const summary = `${counts.ok} ok · ${counts.warn || 0} warn · ${counts.fail || 0} fail`;
    healthEl.textContent = summary;
    healthEl.className = counts.fail ? 'off' : counts.warn ? '' : 'on';
  }

  if (live.processes) {
    procsEl.textContent = live.processes.length + ' processes';
  }

  // Active game from /api/games — find one running, show name + verify state
  if (gameEl && live.games) {
    const running = Object.values(live.games).find(g => g.running);
    if (running) {
      const v = running.last_verify;
      let suffix = '';
      let cls = 'on';
      if (v) {
        if (v.drift_count > 0)      { suffix = ` · ${v.drift_count} drift${v.drift_count !== 1 ? 's' : ''}`; cls = ''; }
        else if (v.missing_files)   { suffix = ` · ${v.missing_files} file${v.missing_files !== 1 ? 's' : ''} missing`; cls = ''; }
        else if (v.ok_count)        { suffix = ' · verified ✓'; cls = 'on'; }
      }
      gameEl.textContent = `🎮 ${running.name}${suffix}`;
      gameEl.className = cls;
    } else {
      gameEl.textContent = 'none running';
      gameEl.className = '';
    }
  }

  sessionEl.textContent = '— (planned)';
}
function renderHealthCards() {
  const wrap = $('#health-live');
  if (!wrap) return;
  if (!live.connected || !live.health) {
    wrap.hidden = true;
    return;
  }
  wrap.hidden = false;
  const meta = $('#health-config-from');
  if (meta) meta.textContent = live.configFrom ? '· config: ' + live.configFrom : '';
  const cards = $('#health-cards');
  const entries = Object.entries(live.health);
  cards.innerHTML = entries.map(([k, c]) => `
    <div class="health-card health-card--${escapeHTML(c.status)}">
      <div class="health-card-head">
        <span class="health-status">${c.status === 'ok' ? '✓' : c.status === 'warn' ? '!' : '✗'}</span>
        <strong>${escapeHTML(c.name)}</strong>
      </div>
      ${c.detail ? `<div class="health-detail">${escapeHTML(c.detail)}</div>` : ''}
      ${c.fix_hint ? `<div class="health-fix">${escapeHTML(c.fix_hint)}</div>` : ''}
    </div>
  `).join('');
}
function renderStatusPage() {
  const offline = $('#status-offline');
  const online  = $('#status-online');
  if (!offline || !online) return;
  if (!live.connected) {
    offline.hidden = false;
    online.hidden = true;
    return;
  }
  offline.hidden = true;
  online.hidden = false;

  $('#status-os').textContent = live.os || '?';
  $('#status-config').textContent = live.configFrom || '?';
  $('#status-updated').textContent = live.lastRefresh
    ? live.lastRefresh.toLocaleTimeString()
    : '—';

  // Games (watcher state)
  const gamesEl = $('#status-games');
  if (gamesEl && live.games) {
    const entries = Object.values(live.games);
    if (!entries.length) {
      gamesEl.innerHTML = '<div class="status-games-empty">No games configured. Add to rig-config.json.</div>';
    } else {
      gamesEl.innerHTML = entries.map(g => {
        const v = g.last_verify;
        let verifyChip = '<span class="status-game-chip dim">no verify yet</span>';
        if (v) {
          if (v.drift_count > 0) {
            verifyChip = `<span class="status-game-chip warn">${v.drift_count} drift${v.drift_count !== 1 ? 's' : ''}</span>`;
          } else if (v.missing_files) {
            verifyChip = `<span class="status-game-chip warn">${v.missing_files} file${v.missing_files !== 1 ? 's' : ''} missing</span>`;
          } else {
            verifyChip = `<span class="status-game-chip ok">✓ ${v.ok_count} ok</span>`;
          }
        }
        const since = g.last_launch
          ? ` · since ${new Date(g.last_launch).toLocaleTimeString()}`
          : '';
        return `<div class="status-game ${g.running ? 'on' : 'off'}">
          <div class="status-game-name">${g.running ? '● ' : '○ '}<strong>${escapeHTML(g.name)}</strong></div>
          <div class="status-game-meta">${g.running ? 'running' : 'not running'}${since}</div>
          ${verifyChip}
        </div>`;
      }).join('');
    }
  }

  // Displays
  const dispEl = $('#status-displays');
  if (dispEl && live.displays) {
    dispEl.innerHTML = live.displays.map(d => {
      const idAttr = d.id.replace(/\\/g, '\\\\');
      return `<div class="status-disp ${d.active ? 'on' : 'off'}">
        <div class="status-disp-name">${escapeHTML(d.name)}${d.primary ? ' <span class="status-tag">primary</span>' : ''}</div>
        <div class="status-disp-meta">${d.width || '?'}×${d.height || '?'} <span class="status-disp-id">${escapeHTML(d.id)}</span></div>
        <button class="wheel-calib-btn status-disp-toggle" data-display-id="${escapeHTML(d.id)}">${d.active ? 'Disable' : 'Enable'}</button>
      </div>`;
    }).join('');
  }

  // vJoy
  const vjEl = $('#status-vjoy');
  if (vjEl && live.vjoy) {
    if (!live.vjoy.installed) {
      vjEl.innerHTML = '<div class="status-vjoy-empty">vJoy driver not installed.</div>';
    } else {
      vjEl.innerHTML = (live.vjoy.devices || []).map(d => `
        <div class="status-vjoy-row">
          <strong>Device ${d.id}</strong>
          <span class="${d.enabled ? 'on' : 'off'}">${d.enabled ? '● enabled' : '○ disabled'}</span>
          <span>${d.buttons} btns · ${d.axes} axes ${d.ffb ? '· FFB on' : ''}</span>
        </div>
      `).join('');
    }
  }

  // Top processes
  const procsEl = $('#status-procs');
  if (procsEl && live.processes) {
    procsEl.innerHTML = `<table class="status-proc-table">
      <thead><tr><th>Process</th><th>CPU %</th><th>RAM</th></tr></thead>
      <tbody>
      ${live.processes.slice(0, 15).map(p => `<tr>
        <td>${escapeHTML(p.name)}${p.description ? ` <span class="status-proc-desc">${escapeHTML(p.description)}</span>` : ''}</td>
        <td>${p.cpu_percent.toFixed(1)}</td>
        <td>${formatBytes(p.ram_bytes)}</td>
      </tr>`).join('')}
      </tbody></table>`;
  }

  // Health (reuse the same renderer as PC Setup)
  const healthEl = $('#status-health');
  if (healthEl && live.health) {
    healthEl.innerHTML = Object.entries(live.health).map(([k, c]) => `
      <div class="health-card health-card--${escapeHTML(c.status)}">
        <div class="health-card-head">
          <span class="health-status">${c.status === 'ok' ? '✓' : c.status === 'warn' ? '!' : '✗'}</span>
          <strong>${escapeHTML(c.name)}</strong>
        </div>
        ${c.detail ? `<div class="health-detail">${escapeHTML(c.detail)}</div>` : ''}
        ${c.fix_hint ? `<div class="health-fix">${escapeHTML(c.fix_hint)}</div>` : ''}
      </div>`).join('');
  }
}

function formatBytes(n) {
  if (!n) return '0';
  if (n >= 1e9) return (n / 1e9).toFixed(1) + ' GB';
  if (n >= 1e6) return (n / 1e6).toFixed(0) + ' MB';
  if (n >= 1e3) return (n / 1e3).toFixed(0) + ' KB';
  return n + ' B';
}

// Wire status-page action buttons (delegated; runs once at DOM ready)
window.addEventListener('DOMContentLoaded', () => {
  document.addEventListener('click', async (e) => {
    const btn = e.target.closest('button');
    if (!btn) return;
    if (btn.id === 'action-refresh') {
      btn.disabled = true;
      await probeTray();
      btn.disabled = false;
    } else if (btn.id === 'verify-refresh') {
      btn.disabled = true;
      await refreshVerifyDrifts();
      btn.disabled = false;
    } else if (btn.id === 'action-health') {
      btn.disabled = true;
      btn.textContent = 'Running…';
      const r = await trayAction('scripts.health');
      btn.textContent = r.status === 'ok' ? 'Done · refreshing' : 'Error: ' + (r.detail || '');
      await probeTray();
      setTimeout(() => { btn.textContent = 'Run health check'; btn.disabled = false; }, 1500);
    } else if (btn.dataset.preset) {
      btn.disabled = true;
      await trayAction('displays.preset', { preset: btn.dataset.preset });
      await probeTray();
      btn.disabled = false;
    } else if (btn.dataset.displayId) {
      btn.disabled = true;
      await trayAction('displays.toggle', { id: btn.dataset.displayId });
      await probeTray();
      btn.disabled = false;
    }
  });
});

function updateLiveBadge(on) {
  const el = $('#tray-status');
  if (!el) return;
  el.classList.toggle('live', on);
  el.classList.toggle('static', !on);
  el.title = on
    ? 'Tray connected — running state from live processes'
    : 'Tray not running — running state from localStorage (design mode)';
  el.textContent = on ? '● Live' : '○ Static';
}
// Boot: try once immediately; poll every 10s thereafter.
window.addEventListener('DOMContentLoaded', () => {
  probeTray();
  setInterval(probeTray, 10_000);
});

// Map a Display-profile cell to a tray action. Recognizes labels we ship.
// Display IDs come from the live config so the mapping stays correct.
function cellToDisplayAction(cell) {
  const cfg = (live.config && live.config.displays) || {};
  const label = (cell.label || '').toLowerCase();
  if (label.startsWith('top-l'))   return { action: 'displays.toggle', params: { id: cfg.topLeft } };
  if (label.startsWith('top-r'))   return { action: 'displays.toggle', params: { id: cfg.topRight } };
  if (label.startsWith('ultra'))   return { action: 'displays.toggle', params: { id: cfg.ultrawide } };
  if (label === 'all on')          return { action: 'displays.preset', params: { preset: 'all-on' } };
  if (label === 'all off')         return { action: 'displays.preset', params: { preset: 'all-off' } };
  if (label === 'vr race')         return { action: 'displays.preset', params: { preset: 'vr-race' } };
  if (label === 'work')            return { action: 'displays.preset', params: { preset: 'work' } };
  return null;
}

// POST an action to the tray. Resolves to { status, detail, output }.
async function trayAction(action, params = {}) {
  if (!live.connected) return { status: 'error', detail: 'tray not connected' };
  try {
    const r = await fetch('/api/action', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ action, params }),
    });
    return await r.json();
  } catch (e) {
    return { status: 'error', detail: String(e) };
  }
}

function readJSON(id) {
  const el = document.getElementById(id);
  if (!el) return null;
  try { return JSON.parse(el.textContent); }
  catch (e) { console.error('Bad JSON in #' + id, e); return null; }
}

// ---------- Stream Deck grids ----------
function renderStreamdeck(container, grid) {
  container.innerHTML = '';
  grid.cells.forEach((cell, idx) => {
    const div = document.createElement('div');
    const hasIcon = cell.icon && ICONS[cell.icon];
    const noLabel = !cell.label;
    const isProfileSwitch = cell.type === 'folder' && cell.target;
    const isLauncher = cell.type === 'launcher' && cell.app;
    const launcherRunning = isLauncher && isRunning(cell.app);
    div.className = 'sd-cell'
      + (cell.type === 'empty' ? ' empty' : '')
      + (hasIcon ? ' has-icon' : '')
      + (noLabel && cell.glyph ? ' glyph-hero' : '')
      + (isProfileSwitch ? ' profile-switch' : '')
      + (isLauncher ? ' launcher' : '')
      + (isLauncher && !launcherRunning ? ' launcher-off' : '');
    div.dataset.cat = cell.cat;
    div.dataset.searchable = [cell.label, cell.notes, cell.type, cell.macro, cell.app, cell.plugin].filter(Boolean).join(' ').toLowerCase();
    if (cell.plugin) div.title = cell.plugin;
    const badge = isProfileSwitch
      ? '<span class="sd-jump" aria-hidden="true">↗</span>'
      : (isLauncher ? `<span class="sd-status ${launcherRunning ? 'on' : 'off'}" aria-label="${launcherRunning ? 'Running' : 'Not running'}"></span>` : '');
    const visual = hasIcon
      ? `<div class="sd-icon">${ICONS[cell.icon]}</div>`
      : `<div class="sd-glyph">${cell.glyph || ''}</div>`;
    div.innerHTML = `
      ${badge}
      ${visual}
      <div class="sd-label">${cell.label || ''}</div>
    `;
    if (cell.type !== 'empty') {
      div.addEventListener('click', async () => {
        if (isProfileSwitch) {
          setTab(cell.target);
          return;
        }
        if (isLauncher) {
          toggleRunning(cell.app);
          return;
        }
        // Display cells: when tray is connected, actually fire the action
        if (live.connected && cell.type === 'system' && container.dataset.grid === 'streamdeck-display') {
          const action = cellToDisplayAction(cell);
          if (action) {
            div.classList.add('sd-firing');
            const r = await trayAction(action.action, action.params);
            div.classList.remove('sd-firing');
            div.classList.add(r.status === 'ok' ? 'sd-fired-ok' : 'sd-fired-err');
            setTimeout(() => div.classList.remove('sd-fired-ok', 'sd-fired-err'), 800);
            return;
          }
        }
        // Default: show detail panel
        showDetail({
          title: cell.label,
          cat: cell.cat,
          rows: [
            ['Type', cell.type],
            cell.plugin ? ['Plugin', cell.plugin] : null,
            ['Notes', cell.notes],
          ].filter(Boolean),
          macro: cell.macro,
        });
      });
    }
    container.appendChild(div);
  });
}

$$('.streamdeck').forEach(el => {
  const key = el.dataset.grid.replace('streamdeck-', '');
  if (data[key]) renderStreamdeck(el, data[key]);
});

// ---------- Pre-flight strips ----------
function renderPreflight(container, apps) {
  const allOn = apps.every(a => isRunning(a.name));
  container.classList.toggle('preflight--ready', allOn);
  container.innerHTML = `
    <span class="preflight-label">Pre-flight</span>
    ${apps.map(a => {
      const on = isRunning(a.name);
      return `<button class="pf-pill ${on ? 'on' : 'off'}" data-app="${escapeAttr(a.name)}" title="${escapeAttr(a.notes || '')}">
        <span class="pf-dot" aria-hidden="true"></span>${escapeHTML(a.name)}
      </button>`;
    }).join('')}
    <span class="preflight-status">${allOn ? 'All systems go' : `${apps.filter(a => !isRunning(a.name)).length} not running`}</span>
  `;
  container.querySelectorAll('.pf-pill').forEach(btn => {
    btn.addEventListener('click', () => toggleRunning(btn.dataset.app));
  });
}
function renderAllPreflight() {
  $$('.preflight').forEach(el => {
    const key = el.dataset.preflight;
    if (data.preflight && data.preflight[key]) renderPreflight(el, data.preflight[key]);
  });
}
function escapeAttr(s) { return String(s).replace(/&/g, '&amp;').replace(/"/g, '&quot;'); }
renderAllPreflight();

// ---------- Keyboard ----------
function renderKeyboard() {
  const root = $('#keyboard');
  const kb = data.keyboard;
  if (!kb) return;

  const wrap = document.createElement('div');
  wrap.className = 'kb-and-nav';

  const main = document.createElement('div');
  main.className = 'keyboard';

  kb.rows.forEach(row => {
    const r = document.createElement('div');
    r.className = 'kb-row';
    row.forEach(key => r.appendChild(makeKey(key)));
    main.appendChild(r);
  });

  wrap.appendChild(main);

  if (kb.navCluster) {
    const nav = document.createElement('div');
    nav.className = 'nav-cluster';
    kb.navCluster.forEach(row => {
      const r = document.createElement('div');
      r.className = 'kb-row';
      row.forEach(key => r.appendChild(makeKey(key)));
      nav.appendChild(r);
    });
    wrap.appendChild(nav);
  }

  root.innerHTML = '';
  root.appendChild(wrap);
}

function makeKey(key) {
  const el = document.createElement('div');
  el.className = 'kb-key';
  el.dataset.cat = key.cat || 'ui';
  const u = 44, gap = 4;
  const w = key.w || 1;
  el.style.flex = `0 0 ${w * u + (w - 1) * gap}px`;
  if (w > 1) el.style.minWidth = `${w * u + (w - 1) * gap}px`;
  el.dataset.searchable = [key.k, key.label, key.moved].filter(Boolean).join(' ').toLowerCase();
  el.innerHTML = `
    <div class="k-name">${key.k}</div>
    ${key.label ? `<div class="k-label">${key.label}</div>` : ''}
    ${key.moved ? `<div class="k-moved">→ ${key.moved}</div>` : ''}
  `;
  el.addEventListener('click', () => showDetail({
    title: key.k + (key.label ? ' — ' + key.label : ''),
    cat: key.cat,
    rows: [
      ['Category', data.meta.categories[key.cat]?.name || key.cat],
      key.moved ? ['Moved to Scimitar', key.moved] : null,
    ].filter(Boolean),
  }));
  return el;
}

renderKeyboard();

// Keyboard legend — keyboard cat names don't match the --cat-* variable
// names (cat="move" but the colour is --cat-blue), so map explicitly.
const KB_CAT_COLOUR = {
  move:    '--cat-blue',
  combat:  '--cat-red',
  heal:    '--cat-green',
  comms:   '--cat-purple',
  vehicle: '--cat-teal',
  moved:   '--cat-gold',
};
function renderLegend() {
  const root = $('#keyboard-legend');
  if (!root) return;
  root.innerHTML = Object.entries(KB_CAT_COLOUR).map(([c, colorVar]) => {
    const name = data.meta.categories[c]?.name || c;
    return `<span class="swatch"><span class="dot" style="background: var(${colorVar})"></span>${name}</span>`;
  }).join('');
}
renderLegend();

// ---------- Scimitar ----------
function renderScimitar() {
  const root = $('#scimitar');
  if (!root || !data.scimitar) return;
  root.innerHTML = '';
  data.scimitar.buttons.forEach(b => {
    const div = document.createElement('div');
    div.className = 'sc-btn';
    div.dataset.cat = b.cat;
    div.dataset.row = b.row;
    div.dataset.searchable = [b.id, b.label, b.key, b.cat].join(' ').toLowerCase();
    div.innerHTML = `
      <div class="sc-id">${b.id}</div>
      <div class="sc-label">${b.label}</div>
      <div class="sc-key">${b.key}</div>
    `;
    div.addEventListener('click', () => showDetail({
      title: `${b.id} — ${b.label}`,
      cat: b.cat,
      rows: [
        ['Bound key', b.key],
        ['Category', data.meta.categories[b.cat]?.name || b.cat],
        ['Difficulty', b.row === 'easy' ? '★ Easy (thumb rest)' : b.row === 'hard' ? 'Hard (top row)' : 'Medium'],
      ],
    }));
    root.appendChild(div);
  });
}
renderScimitar();

// ---------- Wheel (schematic SVG + class-aware legend) ----------
let currentClass = 'gt3';

// Component positions overlaid on docs/img/wheel.webp (Fanatec ClubSport SW
// Formula V2.5). Coords are percentages of the image — easy to nudge if any
// marker sits off its component.
const WHEEL_COMPONENTS = [
  { id: 'B1', x: 29.3, y: 29.2, group: 'Buttons' },
  { id: 'B2', x: 33.3, y: 28.0, group: 'Buttons' },
  { id: 'B3', x: 37.4, y: 27.2, group: 'Buttons' },
  { id: 'B4', x: 65.4, y: 27.0, group: 'Buttons' },
  { id: 'B5', x: 69.2, y: 27.7, group: 'Buttons' },
  { id: 'B6', x: 73.5, y: 29.0, group: 'Buttons' },
  { id: 'TOG-L', x: 41.7, y: 42.3, group: 'Buttons' },
  { id: 'TOG-R', x: 61.5, y: 42.2, group: 'Buttons' },
  { id: 'ENC-L', x: 37.6, y: 35.2, group: 'Thumb encoders' },
  { id: 'ENC-R', x: 64.6, y: 34.2, group: 'Thumb encoders' },
  { id: 'UPDN-L', x: 47.9, y: 40.7, group: 'Up/Down switches' },
  { id: 'UPDN-R', x: 59.2, y: 40.4, group: 'Up/Down switches' },
  { id: 'MPS-L', x: 46.1, y: 51.0, group: 'Multi-position switches' },
  { id: 'FUNKY', x: 51.7, y: 48.3, group: 'FunkySwitch' },
  { id: 'MPS-R', x: 58.0, y: 50.9, group: 'Multi-position switches' },
  { id: 'B7', x: 37.3, y: 48.3, group: 'Joysticks' },
  { id: 'B8', x: 65.3, y: 47.9, group: 'Joysticks' },
  { id: 'B9', x: 47.8, y: 63.0, group: 'Buttons' },
  { id: 'B10', x: 51.7, y: 58.6, group: 'Buttons' },
  { id: 'B11', x: 55.4, y: 62.6, group: 'Buttons' },
];

function joystickRows(j) {
  const rows = [];
  if (j.up || j.down) rows.push(['↑ ↓', `${j.up || '—'} / ${j.down || '—'}`]);
  if (j.left || j.right) rows.push(['← →', `${j.left || '—'} / ${j.right || '—'}`]);
  if (j.push) rows.push(['Push', j.push]);
  return rows;
}

// Persist current MPS-R "context" position (1-12) so the encoder display can
// reflect the active mode. This is purely UI — the real position lives in
// hardware.
const MPS_POS_KEY = 'gaming-ref:mps-pos';
function loadMpsPos() {
  try { return JSON.parse(localStorage.getItem(MPS_POS_KEY) || '{}'); }
  catch { return {}; }
}
function saveMpsPos(p) { localStorage.setItem(MPS_POS_KEY, JSON.stringify(p)); }

function bindingsByClass(cls) {
  const w = data.wheel;
  const out = {};
  const mpsPos = loadMpsPos();

  // Encoders — each may be direct (cw/ccw) or context-driven (modes + context)
  w.encoders.forEach(e => {
    if (e.modes && e.context) {
      const activePos = mpsPos[e.context] || 1;
      const mode = e.modes.find(m => m.pos === activePos) || e.modes[0];
      out[e.id] = {
        contextDriven: true,
        contextId: e.context,
        activePos,
        activeLabel: mode.label,
        text: `${mode.ccw}  /  ${mode.cw}`,
        modes: e.modes,
      };
    } else {
      out[e.id] = { text: `${e.ccw}  /  ${e.cw}` };
    }
  });

  // MPS — each may be direct (all / byClass) or a context selector (selects)
  ['left', 'right'].forEach(side => {
    const m = w.mps[side];
    if (m.selects) {
      const target = w.encoders.find(e => e.id === m.selects);
      out[m.id] = {
        contextSelector: true,
        selects: m.selects,
        text: m.all || `Context selector for ${m.selects}`,
        activePos: mpsPos[m.id] || 1,
        modes: (target && target.modes) || [],
      };
    } else if (m.byClass) {
      out[m.id] = { text: m.byClass[cls], classDep: true };
    } else {
      out[m.id] = m.all;
    }
  });

  out['TOG-L'] = { text: w.toggles[0].action };
  out['TOG-R'] = { text: w.toggles[1].action };
  out['FUNKY'] = {
    funky: true,
    rows: [
      ['↑ ↓', `${w.funky.up} / ${w.funky.down}`],
      ['← →', `${w.funky.left} / ${w.funky.right}`],
      ['Push', w.funky.push || '—'],
      ['Rot',  w.funky.rotary],
    ],
  };
  (w.updown || []).forEach(t => {
    out[t.id] = {
      funky: true,
      rows: [
        ['↑', t.up || '—'],
        ['↓', t.down || '—'],
      ],
      label: t.label,
    };
  });
  (w.joysticks || []).forEach(j => {
    const rows = joystickRows(j).length ? joystickRows(j) : [['—', '—']];
    if (j.via) rows.push(['via', j.via]);
    out[j.id] = { funky: true, rows };
  });
  w.buttons.forEach(b => {
    out[b.id] = {
      text: b.all ? b.all : b.byClass[cls],
      classDep: !b.all,
    };
  });
  return out;
}

// Load any user-calibrated overrides from localStorage
const WHEEL_CALIB_KEY = 'gaming-ref:wheel-calib';
function loadCalib() {
  try { return JSON.parse(localStorage.getItem(WHEEL_CALIB_KEY) || '{}'); }
  catch { return {}; }
}
function saveCalib(c) {
  localStorage.setItem(WHEEL_CALIB_KEY, JSON.stringify(c));
}
function effectiveComponents() {
  const calib = loadCalib();
  return WHEEL_COMPONENTS.map(c => calib[c.id] ? { ...c, x: calib[c.id].x, y: calib[c.id].y } : c);
}

function renderWheel() {
  const root = $('#wheel');
  if (!root || !data.wheel) return;
  const cls = currentClass;
  const bindings = bindingsByClass(cls);
  const components = effectiveComponents();

  // Photo-overlay markers (absolute-positioned dots on top of the wheel image)
  const markerHTML = components.map(c => {
    const label = c.id.replace(/^B(\d+)$/, '$1').replace(/^FUNKY$/, 'F').replace(/^ENC-([LR])$/, 'E$1').replace(/^MPS-([LR])$/, 'M$1').replace(/^TOG-([LR])$/, 'T$1');
    const cls = `wheel-marker wm-${c.group.replace(/[^a-z0-9]+/gi, '-').toLowerCase()}`;
    return `<button type="button" class="${cls}" data-wid="${c.id}" style="left:${c.x}%;top:${c.y}%" aria-label="${c.id}"><span class="wm-dot"></span><span class="wm-id">${label}</span></button>`;
  }).join('');

  // Legend grouped
  const order = ['Multi-position switches', 'Thumb encoders', 'FunkySwitch', 'Up/Down switches', 'Joysticks', 'Buttons'];
  const grouped = {};
  components.forEach(c => { (grouped[c.group] = grouped[c.group] || []).push(c); });

  const renderRows = (rows) => rows
    .map(([dir, act]) => `<div class="wleg-funky-row"><span class="wleg-funky-dir">${escapeHTML(dir)}</span> ${escapeHTML(act || '—')}</div>`)
    .join('');

  const renderModesList = (modes, activePos, contextId) => {
    return `<div class="wleg-modes" data-context="${contextId || ''}">
      ${modes.map(m => `
        <button type="button" class="wleg-mode-row ${m.pos === activePos ? 'active' : ''}" data-pos="${m.pos}">
          <span class="wleg-mode-pos">${m.pos}</span>
          <span class="wleg-mode-label">${escapeHTML(m.label)}${m.via ? ` <span class="wleg-mode-via">${escapeHTML(m.via)}</span>` : ''}</span>
          <span class="wleg-mode-act">${escapeHTML(m.ccw)} / ${escapeHTML(m.cw)}</span>
        </button>
      `).join('')}
    </div>`;
  };

  const legHTML = order.filter(g => grouped[g]).map(group => {
    const items = grouped[group].map(c => {
      const b = bindings[c.id];
      let text, classDep = false, funky = false, extraHtml = '', extraClass = '';
      if (typeof b === 'string') {
        text = escapeHTML(b);
      } else if (b && b.funky) {
        funky = true;
        text = renderRows(b.rows);
      } else if (b && b.contextDriven) {
        extraClass = 'context-driven';
        text = `<div class="wleg-ctx-current">
          <span class="wleg-ctx-tag">${escapeHTML(b.contextId)} pos ${b.activePos}</span>
          <span class="wleg-ctx-label">${escapeHTML(b.activeLabel)}</span>
        </div>
        <div class="wleg-ctx-binding">${escapeHTML(b.text)}</div>`;
        extraHtml = renderModesList(b.modes, b.activePos, b.contextId);
      } else if (b && b.contextSelector) {
        extraClass = 'context-selector';
        text = `<div class="wleg-ctx-selector-head">${escapeHTML(b.text)}</div>`;
        extraHtml = renderModesList(b.modes, b.activePos, c.id);
      } else if (b && b.classDep) {
        text = escapeHTML(b.text);
        classDep = true;
      } else if (b && typeof b === 'object') {
        text = escapeHTML(b.text || '—');
      } else {
        text = '—';
      }
      return `<div class="wheel-leg ${classDep ? 'class-dep' : ''} ${funky ? 'funky' : ''} ${extraClass}" data-wid="${c.id}">
        <span class="wleg-id">${c.id}</span>
        <span class="wleg-act">${text}${classDep ? `<span class="wleg-class">${cls.toUpperCase()}</span>` : ''}${extraHtml}</span>
      </div>`;
    }).join('');
    return `<div class="wheel-leg-group"><h5>${escapeHTML(group)}</h5>${items}</div>`;
  }).join('');

  const wheelData = data.wheel;
  const selector = wheelData.mps && (wheelData.mps.left.selects ? wheelData.mps.left : wheelData.mps.right.selects ? wheelData.mps.right : null);
  const ctxBanner = selector ? `<div class="wheel-modifier-banner">
    <span class="wleg-l2-tag">CTX</span>
    <strong>${escapeHTML(selector.id)} rotary picks what ${escapeHTML(selector.selects)} adjusts.</strong>
    Click any row in the ${escapeHTML(selector.id)} or ${escapeHTML(selector.selects)} legend to preview a different position. ENC-R = Brake Bias is direct (always). Wired in hardware via <a href="#vjoy">vJoy + Gremlin</a>.
  </div>` : '';
  root.innerHTML = `
    ${ctxBanner}
    <div class="wheel-diagram-wrap">
      <div class="wheel-photo-col">
        <div class="wheel-calib-bar">
          <button type="button" id="wheel-calib-toggle" class="wheel-calib-btn">Calibrate markers</button>
          <div class="wheel-calib-actions" hidden>
            <span class="wheel-calib-hint">Drag any marker into place. Changes save automatically.</span>
            <button type="button" id="wheel-calib-export" class="wheel-calib-btn">Copy code</button>
            <button type="button" id="wheel-calib-reset" class="wheel-calib-btn wheel-calib-btn--danger">Reset</button>
            <button type="button" id="wheel-calib-done" class="wheel-calib-btn wheel-calib-btn--primary">Done</button>
          </div>
        </div>
        <div class="wheel-photo">
          <img src="img/wheel.webp" alt="Fanatec ClubSport SW Formula V2.5" draggable="false">
          ${markerHTML}
        </div>
      </div>
      <div class="wheel-legend">${legHTML}</div>
    </div>
  `;

  // Hover sync between SVG markers and legend rows.
  root.querySelectorAll('[data-wid]').forEach(el => {
    const id = el.dataset.wid;
    el.addEventListener('mouseenter', () => {
      root.querySelectorAll(`[data-wid="${CSS.escape(id)}"]`).forEach(x => x.classList.add('hl'));
    });
    el.addEventListener('mouseleave', () => {
      root.querySelectorAll(`[data-wid="${CSS.escape(id)}"]`).forEach(x => x.classList.remove('hl'));
    });
  });

  // Click any mode row to "rotate" the context — re-renders the dependent encoder.
  root.querySelectorAll('.wleg-mode-row').forEach(btn => {
    btn.addEventListener('click', (e) => {
      e.stopPropagation();
      const container = btn.closest('.wleg-modes');
      const ctx = container.dataset.context;
      if (!ctx) return;
      const pos = parseInt(btn.dataset.pos, 10);
      const cur = loadMpsPos();
      cur[ctx] = pos;
      saveMpsPos(cur);
      renderWheel();
    });
  });

  // Click a marker on the photo OR a legend row → open detail panel on the right.
  const showWheelDetail = (id) => {
    const c = components.find(x => x.id === id);
    if (!c) return;
    const wd = data.wheel;
    const b = bindings[id];
    const rows = [];
    rows.push(['Group', c.group]);
    // Find label from the source data structure
    let label = id;
    [...(wd.encoders || []), wd.mps?.left, wd.mps?.right, ...(wd.toggles || []),
     ...(wd.updown || []), ...(wd.joysticks || []), ...(wd.buttons || [])]
      .filter(Boolean).forEach(x => { if (x.id === id) label = x.label || id; });
    if (label !== id) rows.push(['Label', label]);

    if (b && b.contextDriven) {
      rows.push(['Type', `Context-driven encoder (selector: ${b.contextId})`]);
      rows.push(['Active position', `${b.activePos} — ${b.activeLabel}`]);
      rows.push(['Active binding', b.text]);
      rows.push(['All positions', '']);
      b.modes.forEach(m => rows.push([`  pos ${m.pos}`, `${m.label}: ${m.ccw} / ${m.cw}${m.via ? `  [${m.via}]` : ''}`]));
    } else if (b && b.contextSelector) {
      rows.push(['Type', `Context selector → ${b.selects}`]);
      rows.push(['Current position', b.activePos]);
      rows.push(['Positions', '']);
      b.modes.forEach(m => rows.push([`  pos ${m.pos}`, m.label]));
    } else if (b && b.funky) {
      rows.push(['Type', c.group === 'Joysticks' ? '4-way joystick + click' : c.group === 'Up/Down switches' ? '2-way momentary switch' : 'Multi-direction switch']);
      b.rows.forEach(([dir, act]) => rows.push([dir, act]));
    } else if (b && b.classDep) {
      rows.push(['Binding (current class)', b.text]);
      // pull all classes from source
      const btn = (wd.buttons || []).find(x => x.id === id);
      if (btn && btn.byClass) {
        rows.push(['By class', '']);
        Object.entries(btn.byClass).forEach(([k, v]) => rows.push([`  ${k.toUpperCase()}`, v]));
      }
    } else if (b && b.text) {
      rows.push(['Binding', b.text]);
    } else if (typeof b === 'string') {
      rows.push(['Binding', b]);
    }
    showDetail({ title: id, rows });
  };

  root.querySelectorAll('.wheel-marker').forEach(m => {
    m.addEventListener('click', (e) => {
      // Don't fire detail when calibrating (drag mode owns clicks)
      if (root.querySelector('.wheel-photo-col')?.classList.contains('calibrating')) return;
      e.preventDefault();
      showWheelDetail(m.dataset.wid);
    });
  });
  root.querySelectorAll('.wheel-leg').forEach(row => {
    row.addEventListener('click', (e) => {
      // Mode-row clicks have their own handler; don't trigger detail
      if (e.target.closest('.wleg-mode-row')) return;
      showWheelDetail(row.dataset.wid);
    });
  });

  // Calibration mode: drag markers to reposition; persists to localStorage.
  setupWheelCalibration(root);
}

function setupWheelCalibration(root) {
  const photo = root.querySelector('.wheel-photo');
  const photoCol = root.querySelector('.wheel-photo-col');
  const toggleBtn = root.querySelector('#wheel-calib-toggle');
  const actions = root.querySelector('.wheel-calib-actions');
  const exportBtn = root.querySelector('#wheel-calib-export');
  const resetBtn = root.querySelector('#wheel-calib-reset');
  const doneBtn = root.querySelector('#wheel-calib-done');
  if (!photo || !toggleBtn) return;

  const enter = () => {
    photoCol.classList.add('calibrating');
    toggleBtn.hidden = true;
    actions.hidden = false;
  };
  const exit = () => {
    photoCol.classList.remove('calibrating');
    toggleBtn.hidden = false;
    actions.hidden = true;
  };
  toggleBtn.addEventListener('click', enter);
  doneBtn.addEventListener('click', exit);

  resetBtn.addEventListener('click', () => {
    if (!confirm('Reset all calibrations to defaults?')) return;
    saveCalib({});
    renderWheel();
  });

  exportBtn.addEventListener('click', () => {
    const calib = loadCalib();
    const lines = WHEEL_COMPONENTS.map(c => {
      const o = calib[c.id] || c;
      return `  { id: '${c.id}', x: ${o.x.toFixed(1)}, y: ${o.y.toFixed(1)}, group: '${c.group}' },`;
    }).join('\n');
    const out = `const WHEEL_COMPONENTS = [\n${lines}\n];`;
    navigator.clipboard.writeText(out).then(() => {
      exportBtn.textContent = 'Copied!';
      setTimeout(() => { exportBtn.textContent = 'Copy code'; }, 1500);
    });
  });

  // Drag handling
  let dragging = null;
  let dragOffset = { x: 0, y: 0 };

  photo.addEventListener('pointerdown', (e) => {
    if (!photoCol.classList.contains('calibrating')) return;
    const marker = e.target.closest('.wheel-marker');
    if (!marker) return;
    e.preventDefault();
    dragging = marker;
    marker.setPointerCapture(e.pointerId);
    const photoRect = photo.getBoundingClientRect();
    const markerRect = marker.getBoundingClientRect();
    // offset from marker center to pointer
    dragOffset.x = e.clientX - (markerRect.left + markerRect.width / 2);
    dragOffset.y = e.clientY - (markerRect.top + markerRect.height / 2);
    marker.classList.add('dragging');
  });

  photo.addEventListener('pointermove', (e) => {
    if (!dragging) return;
    const r = photo.getBoundingClientRect();
    const cx = e.clientX - dragOffset.x;
    const cy = e.clientY - dragOffset.y;
    const xPct = Math.max(0, Math.min(100, ((cx - r.left) / r.width) * 100));
    const yPct = Math.max(0, Math.min(100, ((cy - r.top) / r.height) * 100));
    dragging.style.left = xPct.toFixed(2) + '%';
    dragging.style.top = yPct.toFixed(2) + '%';
  });

  const endDrag = (e) => {
    if (!dragging) return;
    const id = dragging.dataset.wid;
    const x = parseFloat(dragging.style.left);
    const y = parseFloat(dragging.style.top);
    const calib = loadCalib();
    calib[id] = { x: Number(x.toFixed(2)), y: Number(y.toFixed(2)) };
    saveCalib(calib);
    dragging.classList.remove('dragging');
    dragging = null;
  };
  photo.addEventListener('pointerup', endDrag);
  photo.addEventListener('pointercancel', endDrag);
}

function renderFanalab() {
  const root = $('#fanalab-table');
  if (!root) return;
  root.innerHTML = `
    <table>
      <thead><tr><th>Profile</th><th>Right MPS</th><th>B1</th><th>B2</th></tr></thead>
      <tbody>
        <tr><td>GT3 / GTE</td><td>ABS Level</td><td>Spare</td><td>Spare</td></tr>
        <tr><td>LMP / GTP</td><td>ABS Level</td><td>DRS</td><td>ERS Deploy</td></tr>
        <tr><td>Formula</td><td>Engine Map</td><td>DRS</td><td>Overtake</td></tr>
      </tbody>
    </table>
  `;
}

renderWheel();
renderFanalab();

$$('.cc-btn').forEach(btn => {
  btn.addEventListener('click', () => {
    $$('.cc-btn').forEach(b => b.classList.remove('active'));
    btn.classList.add('active');
    currentClass = btn.dataset.cc;
    renderWheel();
  });
});

// ---------- Macro table ----------
function renderMacroTable() {
  const tbody = $('#macro-table tbody');
  if (!tbody) return;
  const macros = data.iracing.cells.filter(c => c.macro);
  tbody.innerHTML = macros.map(m => `
    <tr>
      <td>${m.label}</td>
      <td class="macro-string">${escapeHTML(m.macro)}</td>
      <td><button class="copy-btn" data-macro="${escapeAttr(m.macro)}">Copy</button></td>
    </tr>
  `).join('');
  tbody.addEventListener('click', e => {
    const btn = e.target.closest('.copy-btn');
    if (!btn) return;
    navigator.clipboard.writeText(btn.dataset.macro).then(() => {
      btn.textContent = 'Copied';
      btn.classList.add('copied');
      setTimeout(() => { btn.textContent = 'Copy'; btn.classList.remove('copied'); }, 1200);
    }).catch(() => { btn.textContent = 'Failed'; });
  });
}
renderMacroTable();

// ---------- Checklist ----------
function renderChecklist() {
  const root = $('#checklist');
  if (!root || !data.meta) return;
  root.innerHTML = data.meta.checklist.map(col => `
    <div class="checklist-col">
      <h3>${col.title}</h3>
      <ul>${col.items.map(i => `<li>${escapeHTML(i)}</li>`).join('')}</ul>
    </div>
  `).join('');
}
renderChecklist();

// ---------- Detail panel ----------
function showDetail({ title, cat, rows, macro }) {
  const panel = $('#detail');
  const body = $('.detail-body', panel);
  const catName = data.meta.categories[cat]?.name || cat;
  body.innerHTML = `
    <h4>${escapeHTML(title)}</h4>
    ${cat ? `<div class="d-cat">${catName}</div>` : ''}
    ${(rows || []).map(([k, v]) => `<div class="d-row"><strong>${k}:</strong> ${escapeHTML(String(v))}</div>`).join('')}
    ${macro ? `<div class="d-row"><strong>Macro:</strong><code class="d-macro">${escapeHTML(macro)}</code></div>` : ''}
  `;
  panel.classList.remove('hidden');
}
$('.detail-close').addEventListener('click', () => $('#detail').classList.add('hidden'));
$('#shortcut-toggle')?.addEventListener('click', () => $('#shortcut-help')?.classList.toggle('hidden'));
$('#shortcut-help')?.addEventListener('click', e => { if (e.target.id === 'shortcut-help') e.currentTarget.classList.add('hidden'); });

// ---------- Global keyboard shortcuts ----------
// /     focus search
// Esc   close detail panel; if already closed, blur+clear search
// 1..5  switch tabs by position
// ?     toggle shortcut help overlay
document.addEventListener('keydown', e => {
  const inField = e.target.matches('input, textarea, select');
  if (e.key === 'Escape') {
    const panel = $('#detail');
    if (!panel.classList.contains('hidden')) {
      panel.classList.add('hidden');
    } else if (searchEl.value || document.activeElement === searchEl) {
      searchEl.value = '';
      searchEl.blur();
      applySearch();
    } else {
      $('#shortcut-help')?.classList.add('hidden');
    }
    return;
  }
  if (inField) return;
  if (e.key === '/') { e.preventDefault(); searchEl.focus(); searchEl.select(); return; }
  if (e.key === '?') { e.preventDefault(); $('#shortcut-help')?.classList.toggle('hidden'); return; }
  if (/^[1-9]$/.test(e.key)) {
    const idx = Number(e.key) - 1;
    if (sections[idx]) { e.preventDefault(); setTab(sections[idx].id); }
  }
});

// ---------- Search ----------
const searchEl = $('#search');
function applySearch() {
  const q = searchEl.value.trim().toLowerCase();
  const all = $$('[data-searchable]');
  if (!q) {
    all.forEach(el => el.classList.remove('match-hit'));
    document.body.classList.remove('search-dim');
    return;
  }
  document.body.classList.add('search-dim');
  let firstVisibleHit = null;
  all.forEach(el => {
    const hit = el.dataset.searchable.includes(q);
    el.classList.toggle('match-hit', hit);
    // Only auto-scroll to a hit that's actually visible (in the active tab).
    if (hit && !firstVisibleHit && el.offsetParent !== null) firstVisibleHit = el;
  });
  if (firstVisibleHit) firstVisibleHit.scrollIntoView({ behavior: 'smooth', block: 'center' });
}
searchEl.addEventListener('input', applySearch);

// ---------- Tabs ----------
const sections = $$('section');
const navLinks = $$('#nav a');
const TAB_STORAGE_KEY = 'gaming-ref:active-tab';
const validTabs = new Set(sections.map(s => s.id));

function setTab(id, { push = true } = {}) {
  if (!validTabs.has(id)) id = sections[0].id;
  sections.forEach(s => s.classList.toggle('active', s.id === id));
  navLinks.forEach(l => l.classList.toggle('active', l.dataset.section === id));
  $('#detail').classList.add('hidden');
  if (push) history.replaceState(null, '', '#' + id);
  try { localStorage.setItem(TAB_STORAGE_KEY, id); } catch (_) {}
  window.scrollTo({ top: 0, behavior: 'instant' in window ? 'instant' : 'auto' });
  // Search persists across tabs — re-run highlight against the now-visible
  // section so a query like "M7" lights up on every tab that has it.
  if (searchEl.value.trim()) applySearch();
}

navLinks.forEach(l => l.addEventListener('click', e => {
  e.preventDefault();
  setTab(l.dataset.section);
}));

// Initial tab: hash → localStorage → first.
// Suppress the browser's native anchor scroll-into-view so the section heading
// isn't left under the sticky topbar. We read the hash, then strip it before
// the browser commits its scroll, and let setTab handle the tab switch.
// __initialHash is stashed by the inline head script (which strips the hash
// before the browser anchor-scrolls).
const hashTab = window.__initialHash || location.hash.replace('#', '');
let initial = null;
try { initial = localStorage.getItem(TAB_STORAGE_KEY); } catch (_) {}
setTab(hashTab || initial || sections[0].id, { push: false });

window.addEventListener('hashchange', () => {
  setTab(location.hash.replace('#', ''), { push: false });
});

// ---------- Last edited stamp ----------
// Uses the document's Last-Modified header if available (works on file:// in
// most browsers via document.lastModified). Falls back to today's date.
(function stampLastEdited() {
  const el = document.getElementById('last-edited');
  if (!el) return;
  try {
    const d = new Date(document.lastModified);
    if (!isNaN(d)) {
      el.textContent = d.toISOString().slice(0, 10);
      return;
    }
  } catch (_) {}
  el.textContent = new Date().toISOString().slice(0, 10);
})();

// ---------- Validation ----------
// Warn in the console if any binding references an unknown category — catches
// typos like "ambr" before they cause a silent uncoloured render.
(function validate() {
  if (!data.meta) return;
  const known = new Set(Object.keys(data.meta.categories));
  const issues = [];
  const check = (cat, where) => {
    if (cat && cat !== 'empty' && !known.has(cat)) issues.push(`${where}: unknown category "${cat}"`);
  };
  ['home', 'pubg', 'iracing', 'streaming'].forEach(g => {
    (data[g]?.cells || []).forEach((c, i) => check(c.cat, `streamdeck-${g}[${i}] ${c.label}`));
  });
  (data.keyboard?.rows || []).flat().forEach(k => check(k.cat, `keyboard "${k.k}"`));
  (data.keyboard?.navCluster || []).flat().forEach(k => check(k.cat, `nav "${k.k}"`));
  (data.scimitar?.buttons || []).forEach(b => check(b.cat, `scimitar ${b.id}`));
  if (issues.length) {
    console.warn('[gaming-reference] data validation found %d issue(s):', issues.length);
    issues.forEach(i => console.warn('  • ' + i));
  } else {
    console.log('[gaming-reference] data validated: %d categories, all bindings OK',
      known.size);
  }
})();

// ---------- Helpers ----------
function escapeHTML(s) {
  return String(s).replace(/[&<>"']/g, m => ({ '&':'&amp;', '<':'&lt;', '>':'&gt;', '"':'&quot;', "'":'&#39;' }[m]));
}
function escapeAttr(s) { return escapeHTML(s).replace(/"/g, '&quot;'); }

// ---------- Verify: mouse hardware tests --------------------------------
// All in-browser. No data leaves the page.

(function () {
  // -- 1. Polling rate: count pointermove events per second on the test pad
  const pollPad = $('#mtest-poll-pad');
  if (!pollPad) return; // verify section not present

  let pollEvents = 0;
  let pollEventsByMs = []; // sliding 10s window of {ts, count}
  let pollPeak = 0;
  pollPad.addEventListener('pointermove', () => { pollEvents++; });
  setInterval(() => {
    const now = Date.now();
    const hz = pollEvents;
    pollEvents = 0;
    pollEventsByMs.push({ ts: now, hz });
    pollEventsByMs = pollEventsByMs.filter(e => now - e.ts < 10_000);
    if (hz > pollPeak) pollPeak = hz;
    const avg = pollEventsByMs.reduce((a, e) => a + e.hz, 0) / Math.max(pollEventsByMs.length, 1);
    $('#mtest-poll-now').textContent = hz;
    $('#mtest-poll-peak').textContent = pollPeak;
    $('#mtest-poll-avg').textContent = Math.round(avg);
    const verdict = $('#mtest-poll-verdict');
    if (avg === 0) {
      verdict.textContent = '— move to begin';
      verdict.className = 'mtest-verdict';
    } else if (avg >= 800) {
      verdict.textContent = '✓ ~1000 Hz gaming mouse — good';
      verdict.className = 'mtest-verdict ok';
    } else if (avg >= 400) {
      verdict.textContent = '⚠ ~500 Hz — could go higher (driver / Windows polling rate setting)';
      verdict.className = 'mtest-verdict warn';
    } else if (avg >= 90 && avg <= 150) {
      verdict.textContent = '✗ ~125 Hz — USB default, not gaming-grade. Check driver / Windows registry MouseHID polling';
      verdict.className = 'mtest-verdict fail';
    } else {
      verdict.textContent = `~${Math.round(avg)} Hz — verify with MouseTester for hardware-level reading (browser may throttle)`;
      verdict.className = 'mtest-verdict warn';
    }
  }, 1000);

  // -- 2. Acceleration: track total px moved during slow vs fast drag
  const accelPad = $('#mtest-accel-pad');
  let accelDragging = false;
  let accelStart = null;
  let accelTotal = 0;
  let slowResult = null;
  let fastResult = null;
  accelPad.addEventListener('pointerdown', e => {
    accelDragging = true;
    accelStart = { x: e.clientX, y: e.clientY, t: performance.now() };
    accelTotal = 0;
    accelPad.setPointerCapture(e.pointerId);
  });
  accelPad.addEventListener('pointermove', e => {
    if (!accelDragging) return;
    accelTotal += Math.abs(e.movementX); // count px in raw counts
  });
  accelPad.addEventListener('pointerup', e => {
    if (!accelDragging) return;
    accelDragging = false;
    accelPad.releasePointerCapture(e.pointerId);
    const dur = performance.now() - accelStart.t;
    const speed = accelTotal / dur; // px/ms
    if (speed < 1.5) {
      // SLOW
      slowResult = accelTotal;
      $('#mtest-accel-slow').textContent = accelTotal.toFixed(0);
    } else {
      // FAST
      fastResult = accelTotal;
      $('#mtest-accel-fast').textContent = accelTotal.toFixed(0);
    }
    if (slowResult && fastResult) {
      const ratio = (fastResult / slowResult).toFixed(2);
      $('#mtest-accel-ratio').textContent = ratio + '×';
      const verdict = $('#mtest-accel-verdict');
      if (Math.abs(ratio - 1.0) < 0.15) {
        verdict.textContent = '✓ Counts match — Enhance Pointer Precision is OFF (correct for gaming).';
        verdict.className = 'mtest-verdict ok';
      } else if (ratio > 1.0) {
        verdict.textContent = `✗ Fast drag covered ${ratio}× more — Enhance Pointer Precision is likely ON. Turn it off in Windows Settings → Mouse.`;
        verdict.className = 'mtest-verdict fail';
      } else {
        verdict.textContent = `Slow drag covered more than fast (${ratio}×) — unusual; re-test with consistent same-distance drags.`;
        verdict.className = 'mtest-verdict warn';
      }
    } else if (slowResult) {
      $('#mtest-accel-verdict').textContent = 'Now drag the same distance FAST.';
    } else {
      $('#mtest-accel-verdict').textContent = 'Slow drag recorded. Now drag the same distance fast.';
    }
  });
  $('#mtest-accel-reset').addEventListener('click', () => {
    slowResult = fastResult = null;
    $('#mtest-accel-slow').textContent = $('#mtest-accel-fast').textContent = $('#mtest-accel-ratio').textContent = '—';
    $('#mtest-accel-verdict').textContent = 'drag slowly first…';
    $('#mtest-accel-verdict').className = 'mtest-verdict';
  });

  // -- 3. Drawing canvas (smoothing / jitter visual)
  const canvas = $('#mtest-draw');
  const ctx = canvas.getContext('2d');
  ctx.strokeStyle = '#5a8';
  ctx.lineWidth = 1;
  ctx.fillStyle = '#0a0d12';
  ctx.fillRect(0, 0, canvas.width, canvas.height);
  let drawing = false;
  let last = null;
  canvas.addEventListener('pointerdown', e => {
    drawing = true;
    const r = canvas.getBoundingClientRect();
    last = { x: e.clientX - r.left, y: e.clientY - r.top };
  });
  canvas.addEventListener('pointermove', e => {
    if (!drawing) return;
    const r = canvas.getBoundingClientRect();
    const x = e.clientX - r.left, y = e.clientY - r.top;
    ctx.beginPath();
    ctx.moveTo(last.x, last.y);
    ctx.lineTo(x, y);
    ctx.stroke();
    last = { x, y };
  });
  canvas.addEventListener('pointerup', () => { drawing = false; });
  canvas.addEventListener('pointerleave', () => { drawing = false; });
  $('#mtest-draw-clear').addEventListener('click', () => {
    ctx.fillRect(0, 0, canvas.width, canvas.height);
  });
})();

// ---------- Verify: Scimitar mapping check ------------------------------
(function () {
  const grid = $('#scim-test-grid');
  if (!grid) return;
  const scim = data.scimitar?.buttons || [];
  if (!scim.length) {
    grid.innerHTML = '<p class="planned-meta">No Scimitar mapping found in PUBG data.</p>';
    return;
  }

  // Render the grid: M1-M12 with expected key, plus a "match" toggle
  let matched = JSON.parse(localStorage.getItem('gaming-ref:scim-test') || '{}');
  function render() {
    grid.innerHTML = scim.map(b => `
      <div class="scim-test-row ${matched[b.id] ? 'matched' : ''}" data-id="${b.id}">
        <span class="scim-test-id">${b.id}</span>
        <span class="scim-test-label">${escapeHTML(b.label)}</span>
        <span class="scim-test-key">expected: <code>${escapeHTML(b.key)}</code></span>
        <button type="button" class="wheel-calib-btn scim-test-toggle">${matched[b.id] ? '✓ matched' : 'mark match'}</button>
      </div>
    `).join('');
    grid.querySelectorAll('.scim-test-toggle').forEach(btn => {
      btn.addEventListener('click', () => {
        const id = btn.closest('.scim-test-row').dataset.id;
        matched[id] = !matched[id];
        localStorage.setItem('gaming-ref:scim-test', JSON.stringify(matched));
        render();
      });
    });
  }
  render();

  $('#scim-log-clear').addEventListener('click', () => {
    matched = {};
    localStorage.removeItem('gaming-ref:scim-test');
    $('#scim-log').innerHTML = '';
    render();
  });

  // Capture inputs anywhere in the verify section
  const verifySection = $('#verify');
  let recent = [];
  function logEvent(label) {
    const t = new Date().toLocaleTimeString();
    $('#scim-event-key').textContent = label;
    $('#scim-event-time').textContent = 'at ' + t;
    recent.unshift({ label, t });
    recent = recent.slice(0, 12);
    $('#scim-log').innerHTML = recent.map(r =>
      `<li><code>${escapeHTML(r.label)}</code> <span class="scim-log-t">${r.t}</span></li>`).join('');
  }

  verifySection.addEventListener('keydown', e => {
    if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') return;
    let label = e.key;
    if (e.ctrlKey)  label = 'Ctrl+' + label;
    if (e.altKey)   label = 'Alt+' + label;
    if (e.shiftKey && e.key.length > 1) label = 'Shift+' + label;
    logEvent(label);
  });
  verifySection.addEventListener('mousedown', e => {
    if (e.button === 0) return; // skip primary clicks (used for buttons)
    const names = { 1: 'Mouse middle', 2: 'Mouse right', 3: 'Mouse back', 4: 'Mouse forward' };
    logEvent(names[e.button] || ('Mouse btn ' + e.button));
  });
})();
