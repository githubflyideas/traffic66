(() => {
'use strict';

// ------------------------------------------------------------ i18n
// Ordered by total number of speakers (Ethnologue); Japanese and Korean follow the top 11.
const LANGS = [['en','English'],['zh','中文'],['hi','हिन्दी'],['es','Español'],['ar','العربية'],['fr','Français'],
  ['bn','বাংলা'],['pt','Português'],['ru','Русский'],['id','Bahasa Indonesia'],['ur','اردو'],['ja','日本語'],['ko','한국어']];
const RTL = new Set(['ar', 'ur']);
let LANG = 'en', DICT = {}, EN = {};
const t = (k, p) => {
  let s = DICT[k] ?? EN[k] ?? k;
  if (p) for (const [a, b] of Object.entries(p)) s = s.split('{' + a + '}').join(b);
  return s;
};
const esc = s => String(s ?? '').replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
async function loadLang(code) {
  const get = async c => (await fetch('i18n/' + c + '.json')).json();
  if (!Object.keys(EN).length) EN = await get('en');
  DICT = code === 'en' ? EN : await get(code).catch(() => EN);
  LANG = code;
  document.documentElement.lang = code;
  document.documentElement.dir = RTL.has(code) ? 'rtl' : 'ltr';
  try { localStorage.setItem('t66.lang', code); } catch (e) {}
  regionNames = null;
  document.querySelectorAll('[data-i]').forEach(el => { el.textContent = t(el.dataset.i); });
  $('#q').placeholder = t('top.search');
  $('#refresh').textContent = t('top.refresh');
}
function pickLang() {
  let saved = null;
  try { saved = localStorage.getItem('t66.lang'); } catch (e) {}
  if (saved && LANGS.some(l => l[0] === saved)) return saved;
  for (const l of navigator.languages || [navigator.language || 'en']) {
    const c = l.toLowerCase().split('-')[0];
    if (LANGS.some(x => x[0] === c)) return c;
  }
  return 'en';
}

// ------------------------------------------------------------ helpers
const $ = s => document.querySelector(s);
const nf = (v, d = 0) => new Intl.NumberFormat(LANG, {maximumFractionDigits: d, minimumFractionDigits: d}).format(v);
function fmtBps(v) {
  if (v == null || !isFinite(v)) return '—';
  const u = [['Tbps', 1e12], ['Gbps', 1e9], ['Mbps', 1e6], ['Kbps', 1e3]];
  for (const [n, m] of u) if (Math.abs(v) >= m) return nf(v / m, Math.abs(v / m) < 10 ? 2 : Math.abs(v / m) < 100 ? 1 : 0) + ' ' + n;
  return nf(v) + ' bps';
}
function fmtBytes(v) {
  if (v == null) return '—';
  const u = [['TB', 1e12], ['GB', 1e9], ['MB', 1e6], ['KB', 1e3]];
  for (const [n, m] of u) if (v >= m) return nf(v / m, v / m < 10 ? 2 : v / m < 100 ? 1 : 0) + ' ' + n;
  return nf(v) + ' B';
}
const fmtPct = (v, d = 1) => (v > 0 ? '+' : '') + nf(v * 100, d) + '%';
function fmtAxisBps(v) {
  if (v === 0) return '0';
  for (const [n, m] of [['T', 1e12], ['G', 1e9], ['M', 1e6], ['K', 1e3]]) if (v >= m) return nf(v / m, v / m < 10 && v % m ? 1 : 0) + n;
  return nf(v);
}
let regionNames = null;
function country(cc) {
  if (cc === '__internal__') return t('internal');
  if (cc === '__other__') return t('other');
  if (!cc || cc === '__unknown__') return t('unknown');
  try { regionNames = regionNames || new Intl.DisplayNames([LANG], {type: 'region'}); return regionNames.of(cc) || cc; } catch (e) { return cc; }
}
const PROTO = {1: 'ICMP', 2: 'IGMP', 4: 'IPIP', 6: 'TCP', 17: 'UDP', 41: 'IPv6', 47: 'GRE', 50: 'ESP', 51: 'AH', 58: 'ICMPv6', 89: 'OSPF', 132: 'SCTP'};
const proto = p => PROTO[p] || String(p);
const ENCAP = ['', 'GRE', 'IPIP', '6in4', 'IPv6-in-IPv6', 'VXLAN', 'GENEVE', 'MPLS'];
const DIRS = {1: 'outbound', 2: 'inbound', 3: 'internal', 4: 'transit'};
const COLORS = ['var(--c1)', 'var(--c2)', 'var(--c3)', 'var(--c4)', 'var(--c5)', 'var(--c6)'];
const OTHER = 'var(--other)';
const appLabel = a => a === '__other__' ? t('other') : a;
function fmtTime(ms, span) {
  const d = new Date(ms);
  if (span > 36 * 3600e3) return new Intl.DateTimeFormat(LANG, {month: 'numeric', day: 'numeric', hour: '2-digit', minute: '2-digit'}).format(d);
  return new Intl.DateTimeFormat(LANG, {hour: '2-digit', minute: '2-digit'}).format(d);
}
function fmtAxisTime(ms, span) {
  const d = new Date(ms);
  if (span > 36 * 3600e3) return new Intl.DateTimeFormat(LANG, {month: 'numeric', day: 'numeric'}).format(d);
  return new Intl.DateTimeFormat(LANG, {hour: '2-digit', minute: '2-digit'}).format(d);
}
function ago(ms) {
  const s = Math.max(0, (Date.now() - ms) / 1000);
  const rtf = new Intl.RelativeTimeFormat(LANG, {numeric: 'auto'});
  if (s < 60) return rtf.format(-Math.round(s), 'second');
  if (s < 3600) return rtf.format(-Math.round(s / 60), 'minute');
  if (s < 86400) return rtf.format(-Math.round(s / 3600), 'hour');
  return rtf.format(-Math.round(s / 86400), 'day');
}
function toast(msg) {
  const el = $('#toast'); el.textContent = msg; el.style.display = 'block';
  clearTimeout(toast.h); toast.h = setTimeout(() => el.style.display = 'none', 1800);
}
const ICON = {
  ok: '<svg viewBox="0 0 16 16" aria-hidden="true"><circle cx="8" cy="8" r="7" fill="currentColor"/><path d="M4.5 8.2l2.2 2.2 4.8-4.8" stroke="#fff" stroke-width="1.8" fill="none"/></svg>',
  warn: '<svg viewBox="0 0 16 16" aria-hidden="true"><path d="M8 1.5l7 12.5H1z" fill="var(--warn)"/><path d="M8 6v4M8 11.6v.4" stroke="#1b2432" stroke-width="1.6"/></svg>',
  bad: '<svg viewBox="0 0 16 16" aria-hidden="true"><circle cx="8" cy="8" r="7" fill="currentColor"/><path d="M5.5 5.5l5 5M10.5 5.5l-5 5" stroke="#fff" stroke-width="1.8"/></svg>'
};
const status = (kind, text) => `<span class="st ${kind}">${ICON[kind]}${esc(text)}</span>`;
const bar = (x, max, col) => `<div class="sb"><span style="width:${Math.max(1.5, max > 0 ? x / max * 100 : 0)}%${col ? ';background:' + col : ''}"></span></div>`;

// ------------------------------------------------------------ names
const names = new Map();      // ip -> name
const asked = new Set();
function ipCell(ip, field = 'ip') {
  if (!ip) return '—';
  const n = names.get(ip);
  return `<button class="v" data-k="${field}" data-val="${esc(ip)}" data-ip="${esc(ip)}" aria-haspopup="menu">${n ? esc(n) + `<span class="ip">${esc(ip)}</span>` : esc(ip)}</button>`;
}
function V(field, val, label) {
  return `<button class="v" data-k="${field}" data-val="${esc(val)}" data-label="${esc(label)}" aria-haspopup="menu">${esc(label)}</button>`;
}
async function resolveNames() {
  const ips = [...new Set([...document.querySelectorAll('.view.on [data-ip]')].map(e => e.dataset.ip))].filter(ip => !names.has(ip) && !asked.has(ip));
  if (!ips.length) return;
  ips.forEach(ip => asked.add(ip));
  try {
    const res = await fetch('/api/resolve', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({ips: ips.slice(0, 300)})});
    if (!res.ok) return;
    const m = await res.json();
    for (const [ip, n] of Object.entries(m)) names.set(ip, n);
    // lookups skipped by the rate limit may be retried later
    setTimeout(() => ips.forEach(ip => { if (!names.has(ip)) asked.delete(ip); }), 60000);
    document.querySelectorAll('[data-ip]').forEach(el => {
      const n = names.get(el.dataset.ip);
      if (n && !el.querySelector('.ip')) el.innerHTML = esc(n) + `<span class="ip">${esc(el.dataset.ip)}</span>`;
    });
  } catch (e) {}
}

// ------------------------------------------------------------ state & API
const VIEWS = ['overview', 'topn', 'conv', 'sankey', 'geo', 'threats', 'records', 'ifaces', 'sources'];
const RANGES = ['15m', '1h', '6h', '24h', '7d', '30d'];
const state = {v: 'overview', r: '24h', f: [], dim: 'client', ifc: null, ifdir: 'in'};
function readHash() {
  const p = new URLSearchParams(location.hash.slice(1));
  if (VIEWS.includes(p.get('v'))) state.v = p.get('v');
  if (RANGES.includes(p.get('r'))) state.r = p.get('r');
  if (p.get('dim')) state.dim = p.get('dim');
  state.f = (p.get('f') || '').split(',').filter(Boolean).map(s => {
    const neg = s[0] === '!'; if (neg) s = s.slice(1);
    const i = s.indexOf(':');
    return {f: decodeURIComponent(s.slice(0, i)), v: decodeURIComponent(s.slice(i + 1)), neg};
  }).filter(x => x.f && x.v);
}
function writeHash() {
  const p = new URLSearchParams({v: state.v, r: state.r});
  if (state.v === 'topn') p.set('dim', state.dim);
  if (state.f.length) p.set('f', state.f.map(x => (x.neg ? '!' : '') + encodeURIComponent(x.f) + ':' + encodeURIComponent(x.v)).join(','));
  history.replaceState(null, '', '#' + p.toString());
}
async function api(path, extra = {}) {
  const p = new URLSearchParams({range: state.r, ...extra});
  if (state.f.length) p.set('f', JSON.stringify(state.f.map(x => ({f: x.f, v: x.v, neg: x.neg}))));
  const res = await fetch('/api/' + path + '?' + p);
  if (res.status === 401) { showLogin(); throw new Error('login'); }
  const j = await res.json().catch(() => ({}));
  if (!res.ok) { const e = new Error(j.error || res.statusText); e.kind = j.kind; throw e; }
  return j;
}
const spanMs = () => ({'15m': 9e5, '1h': 36e5, '6h': 216e5, '24h': 864e5, '7d': 6048e5, '30d': 2592e6})[state.r];

// ------------------------------------------------------------ tooltip & charts
const tip = $('#tip');
function showTip(e, html) {
  tip.innerHTML = html; tip.style.display = 'block';
  const r = tip.getBoundingClientRect();
  let l = e.clientX + 14; if (l + r.width > innerWidth - 8) l = e.clientX - r.width - 14;
  let tp = e.clientY - r.height - 12; if (tp < 8) tp = e.clientY + 16;
  tip.style.left = l + 'px'; tip.style.top = tp + 'px';
}
const hideTip = () => tip.style.display = 'none';
function niceMax(m) {
  if (m <= 0) return 1;
  const p = Math.pow(10, Math.floor(Math.log10(m / 4)));
  for (const s of [1, 2, 2.5, 5, 10]) if (s * p * 4 >= m) return s * p * 4;
  return m;
}
function tsChart(el, {times, areas = [], lines = [], fmtY = fmtAxisBps, fmtV = fmtBps}) {
  const draw = () => {
    const W = el.clientWidth, H = el.clientHeight; if (W < 120 || H < 60 || !times || !times.length) { el.innerHTML = ''; return; }
    const rtl = document.documentElement.dir === 'rtl';
    const L = rtl ? 8 : 50, R = rtl ? 50 : 8, T = 8, B = 22, n = times.length;
    const x = i => rtl ? W - R - (W - L - R) * i / Math.max(1, n - 1) : L + (W - L - R) * i / Math.max(1, n - 1);
    const cum = areas.map(() => new Array(n).fill(null));
    for (let i = 0; i < n; i++) { let s = 0; areas.forEach((a, k) => { const v = a.data[i]; if (v == null || v < 0) return; s += v; cum[k][i] = s; }); }
    let top = 0;
    cum.forEach(c => c.forEach(v => { if (v != null && v > top) top = v; }));
    lines.forEach(l => l.data.forEach(v => { if (v != null && v > top) top = v; }));
    const Y = niceMax(top * 1.05), y = v => T + (H - T - B) * (1 - v / Y);
    const span = times[n - 1] - times[0];
    let s = `<svg viewBox="0 0 ${W} ${H}" height="${H}" role="img" aria-label="${esc(el.getAttribute('aria-label') || '')}">`;
    for (let k = 0; k <= 4; k++) {
      const yy = y(Y * k / 4);
      s += `<line class="gl" x1="${L}" x2="${W - R}" y1="${yy}" y2="${yy}"/><text class="ax" x="${rtl ? W - R + 8 : L - 8}" y="${yy + 4}" text-anchor="${rtl ? 'start' : 'end'}">${fmtY(Y * k / 4)}</text>`;
    }
    const ticks = W < 600 ? 4 : 7, step = Math.max(1, Math.round(n / ticks));
    for (let i = 0; i < n; i += step) s += `<text class="ax" x="${x(i)}" y="${H - 5}" text-anchor="middle">${fmtAxisTime(times[i], span)}</text>`;
    areas.forEach((a, k) => {
      let seg = [], paths = '';
      const flush = () => {
        if (seg.length < 1) { seg = []; return; }
        let up = '', lo = '';
        seg.forEach(i => { up += (up ? 'L' : 'M') + x(i).toFixed(1) + ' ' + y(cum[k][i]).toFixed(1); });
        for (let j = seg.length - 1; j >= 0; j--) { const i = seg[j]; const b = k ? (cum[k - 1][i] || 0) : 0; lo += 'L' + x(i).toFixed(1) + ' ' + y(b).toFixed(1); }
        paths += up + lo + 'Z'; seg = [];
      };
      for (let i = 0; i < n; i++) { if (cum[k][i] == null) flush(); else seg.push(i); }
      flush();
      s += `<path d="${paths}" fill="${a.color}" fill-opacity=".88" stroke="var(--surface)" stroke-width="1"/>`;
    });
    lines.forEach(l => {
      let d = '', pen = false;
      l.data.forEach((v, i) => { if (v == null || v < 0) { pen = false; return; } d += (pen ? 'L' : 'M') + x(i).toFixed(1) + ' ' + y(v).toFixed(1); pen = true; });
      s += `<path d="${d}" fill="none" stroke="${l.color}" stroke-width="2" ${l.dash ? 'stroke-dasharray="4 3"' : ''}/>`;
    });
    s += `<line class="xh" x1="0" x2="0" y1="${T}" y2="${H - B}" stroke="var(--ink-2)" stroke-width="1" style="display:none"/><rect class="hit" x="${Math.min(L, R)}" y="${T}" width="${W - L - R}" height="${H - T - B}" fill="transparent"/></svg>`;
    el.innerHTML = s;
    const svg = el.querySelector('svg'), xh = svg.querySelector('.xh'), hit = svg.querySelector('.hit');
    hit.onmousemove = e => {
      const r = svg.getBoundingClientRect();
      let fx = (e.clientX - r.left - L) / (W - L - R); if (rtl) fx = (W - R - (e.clientX - r.left)) / (W - L - R);
      const i = Math.max(0, Math.min(n - 1, Math.round(fx * (n - 1))));
      xh.setAttribute('x1', x(i)); xh.setAttribute('x2', x(i)); xh.style.display = '';
      let h = `<div class="t">${fmtTime(times[i], span)}</div>`, tot = 0, any = false;
      [...areas].reverse().forEach(a => { const v = a.data[i]; if (v == null || v < 0) return; any = true; tot += v; h += `<div class="r"><span><i style="background:${a.color}"></i>${esc(a.name)}</span><b>${fmtV(v)}</b></div>`; });
      if (any && areas.length > 1) h += `<div class="r tot"><span>${t('total')}</span><span>${fmtV(tot)}</span></div>`;
      lines.forEach(l => { const v = l.data[i]; if (v == null || v < 0) return; h += `<div class="r"><span><i style="background:${l.color}"></i>${esc(l.name)}</span><b>${fmtV(v)}</b></div>`; });
      showTip(e, h);
    };
    hit.onmouseleave = () => { xh.style.display = 'none'; hideTip(); };
  };
  el._draw = draw; draw();
  if (!el._ro) { el._ro = new ResizeObserver(() => el._draw && el._draw()); el._ro.observe(el); }
}
function donut(el, parts, field) {
  const tot = parts.reduce((s, p) => s + p.v, 0);
  if (!tot) { el.innerHTML = `<div class="empty">${t('empty.nodata')}</div>`; return; }
  const R = 52, r = 34, C = 56; let a0 = -Math.PI / 2;
  let s = `<svg viewBox="0 0 112 112" role="img" aria-label="${esc(parts.map(p => p.n + ' ' + nf(p.v / tot * 100) + '%').join(', '))}">`;
  parts.forEach(p => {
    const frac = p.v / tot;
    if (frac >= 0.9999) { s += `<circle cx="${C}" cy="${C}" r="${(R + r) / 2}" fill="none" stroke="${p.c}" stroke-width="${R - r}"/>`; return; }
    const a1 = a0 + 2 * Math.PI * frac, big = a1 - a0 > Math.PI ? 1 : 0, P = (a, rad) => [C + rad * Math.cos(a), C + rad * Math.sin(a)];
    const [x1, y1] = P(a0, R), [x2, y2] = P(a1, R), [x3, y3] = P(a1, r), [x4, y4] = P(a0, r);
    s += `<path d="M${x1} ${y1}A${R} ${R} 0 ${big} 1 ${x2} ${y2}L${x3} ${y3}A${r} ${r} 0 ${big} 0 ${x4} ${y4}Z" fill="${p.c}" stroke="var(--surface)" stroke-width="2"><title>${esc(p.n)}</title></path>`;
    a0 = a1;
  });
  el.innerHTML = s + '</svg><ul>' + parts.map(p => `<li><i style="background:${p.c}"></i>${p.val != null ? V(field, p.val, p.n) : esc(p.n)}<b>${nf(p.v / tot * 100)}%</b></li>`).join('') + '</ul>';
}

// ------------------------------------------------------------ popover & filters
const pop = $('#pop'); let ctx = null;
const LOOKUP = {
  ip: v => 'https://bgp.he.net/ip/' + encodeURIComponent(v),
  asn: v => 'https://bgp.he.net/AS' + encodeURIComponent(v.replace(/^AS/i, '')),
  port: v => 'https://www.speedguide.net/port.php?port=' + encodeURIComponent(v.split('/')[0]),
};
function openPop(c, rect) {
  ctx = c;
  pop.innerHTML = `<div class="who">${esc(t('field.' + c.k))}<strong>${esc(c.label)}</strong></div>
    <button role="menuitem" data-a="only">${t('pop.only')}<kbd>F</kbd></button>
    <button role="menuitem" data-a="not">${t('pop.not')}<kbd>X</kbd></button>
    <button role="menuitem" data-a="records">${t('pop.records')}<kbd>↵</kbd></button>
    ${LOOKUP[c.k] ? `<button role="menuitem" data-a="lookup">${t('pop.lookup')}</button>` : ''}`;
  pop.style.display = 'block';
  const pr = pop.getBoundingClientRect();
  pop.style.left = Math.max(8, Math.min(rect.left, innerWidth - pr.width - 8)) + 'px';
  pop.style.top = (rect.bottom + pr.height + 8 > innerHeight ? rect.top - pr.height - 6 : rect.bottom + 6) + 'px';
  pop.querySelector('button').focus();
}
function act(a) {
  if (!ctx) return;
  pop.style.display = 'none';
  if (a === 'lookup') { window.open(LOOKUP[ctx.k](ctx.v), '_blank', 'noopener'); return; }
  state.f = state.f.filter(x => !(x.f === ctx.k && x.v === ctx.v));
  if (a === 'records') { state.f = state.f.filter(x => x.f !== ctx.k); state.f.push({f: ctx.k, v: ctx.v, neg: false}); go('records'); return; }
  state.f.push({f: ctx.k, v: ctx.v, neg: a === 'not'});
  render();
}
pop.addEventListener('click', e => { const b = e.target.closest('button'); if (b) act(b.dataset.a); });
document.addEventListener('keydown', e => {
  if (pop.style.display !== 'block') return;
  const k = e.key.toLowerCase();
  if (k === 'escape') pop.style.display = 'none';
  else if (k === 'f') act('only');
  else if (k === 'x') act('not');
  else if (k === 'enter' && !document.activeElement?.dataset?.a) { e.preventDefault(); act('records'); }
});
document.addEventListener('click', e => {
  const rm = e.target.closest('[data-rm]');
  if (rm) { state.f.splice(+rm.dataset.rm, 1); render(); return; }
  if (e.target.closest('[data-clear]')) { state.f = []; render(); return; }
  const v = e.target.closest('.v');
  if (v) { e.stopPropagation(); openPop({k: v.dataset.k, v: v.dataset.val, label: v.dataset.label || v.textContent}, v.getBoundingClientRect()); return; }
  if (!e.target.closest('#pop')) pop.style.display = 'none';
});
function filterLabel(x) {
  if (x.f === 'country') return country(x.v);
  if (x.f === 'dir') return t('dir.' + Object.keys(DIRS).find(k => DIRS[k] === x.v));
  if (x.f === 'proto') return proto(+x.v) || x.v;
  if (x.f === 'encap') return ENCAP[+x.v] || x.v;
  if (x.f === 'ip' && names.has(x.v)) return names.get(x.v) + ' ' + x.v;
  if (x.f === 'segment' || x.f === 'app') return appLabel(x.v);
  return x.v;
}
function renderFilters() {
  const fb = $('#filters');
  fb.innerHTML = state.f.length
    ? state.f.map((x, i) => `<span class="chip ${x.neg ? 'neg' : ''}">${esc(x.neg ? t('filters.not') : t('filters.only'))} ${esc(t('field.' + x.f))}: ${esc(filterLabel(x))}<button data-rm="${i}" aria-label="${esc(t('filters.remove'))}">✕</button></span>`).join('')
      + (state.f.length > 1 ? `<button class="btn" data-clear style="padding:2px 10px">${t('filters.clear')}</button>` : '')
    : `<span class="hint">${t('filters.hint')}</span>`;
}
// search box: guess the field from what was typed
$('#q').addEventListener('keydown', e => {
  if (e.key !== 'Enter') return;
  const v = e.target.value.trim(); if (!v) return;
  let f = 'app', val = v;
  if (/^[0-9a-f:.]+(\/\d+)?$/i.test(v) && (v.includes('.') || v.includes(':'))) f = 'ip';
  else if (/^as\d+$/i.test(v)) { f = 'asn'; val = v.slice(2); }
  else if (/^\d+(\/(tcp|udp))?$/i.test(v)) f = 'port';
  else if (/^[a-z]{2}$/i.test(v)) { f = 'country'; val = v.toUpperCase(); }
  state.f.push({f, v: val, neg: false});
  e.target.value = '';
  render();
});

// ------------------------------------------------------------ views
const views = {};
let loadSeq = 0;
function panel(cls, title, sub, body) {
  return `<div class="panel ${cls}"><div class="ph"><h2>${title}</h2>${sub ? `<span class="sub">${sub}</span>` : ''}</div>${body}</div>`;
}
// Database failures get a plain explanation; the details are in the log.
function errMsg(err) {
  if (err.kind === 'memory') return t('err.db_memory');
  if (err.kind === 'storage') return t('err.db_busy');
  return t('err.load', {e: err.message});
}
function errorBox(err) { return `<div class="panel"><div class="empty">${esc(errMsg(err))}</div></div>`; }

views.overview = async (el) => {
  const [d, ifs] = await Promise.all([api('overview'), api('ifaces').catch(() => null)]);
  const tot = d.totals, base = d.base_totals, basis = d.basis;
  const change = base.wire > 0 ? (tot.wire - base.wire) / base.wire : null;
  // accuracy: worst interface deviation relative to its statistical error
  let acc = {n: '—', d: t('kpi.no_counters'), kind: null};
  const withCtr = (ifs?.ifaces || []).filter(f => f.has_counters);
  if (withCtr.length) {
    let worst = 0, bad = false;
    withCtr.forEach(f => { const dv = Math.max(Math.abs(f.in_dev), Math.abs(f.out_dev)); worst = Math.max(worst, dv); if (dv > Math.max(0.02, 2 * f.stat_err)) bad = true; });
    acc = {n: nf(worst * 100, 1) + '<small>%</small>', d: bad ? t('kpi.check') : t('kpi.trust'), kind: bad ? 'warn' : 'ok'};
  }
  const peakTxt = d.peak_at ? t('kpi.peak', {v: fmtBps(d.peak_bps), t: fmtTime(d.peak_at, spanMs())}) : '';
  const series = d.series || {times: [], names: [], values: []};
  const areas = series.names.map((n, i) => ({name: appLabel(n), color: n === '__other__' ? OTHER : COLORS[i % 6], data: series.values[i]}));
  const lines = d.baseline ? [{name: t(basis === 'week' ? 'last_week' : 'prev_period'), color: 'var(--base)', dash: true, data: d.baseline}] : [];
  const movers = d.movers || [];
  const mmax = Math.max(1, ...movers.map(m => m.delta_bps));
  const moverHTML = movers.length ? movers.map(m => `<li><div>${ipCell(m.ip)}</div><div class="amt up">+${fmtBps(m.delta_bps)}</div>${bar(m.delta_bps, mmax)}
      <div class="why">${m.before_bps < m.now_bps * 0.02 ? `<span class="tag">${t('ov.mover_new')}</span> ` : ''}${m.peer ? esc(t('ov.mover_reason', {peer: names.get(m.peer) || m.peer, port: m.port, cc: country(m.cc)})) : ''}${m.threat ? ` <span class="tag" style="background:#fdecec;color:var(--crit)">${esc(t('thr.list'))}: ${esc(m.threat)}</span>` : ''}</div></li>`).join('')
    : `<li class="muted">${t('ov.no_movers')}</li>`;
  const dirParts = (d.dir || []).map((p, i) => ({n: t('dir.' + p.key), v: p.wire, c: COLORS[i % 6], val: DIRS[p.key]}));
  const protoParts = (d.proto || []).map((p, i) => ({n: p.key === '__other__' ? t('other') : proto(+p.key), v: p.wire, c: p.key === '__other__' ? OTHER : COLORS[i % 6], val: p.key === '__other__' ? null : p.key}));
  const ccParts = (d.country || []).map((p, i) => ({n: country(p.key), v: p.wire, c: p.key === '__other__' ? OTHER : COLORS[i % 6], val: p.key === '__other__' ? null : p.key}));
  const cl = d.top_clients || [], cmax = Math.max(1, ...cl.map(r => r.wire));
  const sv = d.top_services || [], smax = Math.max(1, ...sv.map(r => r.wire));
  el.innerHTML = `<div class="grid">
    <div class="panel c12"><div class="kpis">
      <div class="kpi"><div class="lab">${t('kpi.now')}</div><div class="n">${fmtBps(d.now_bps)}</div><div class="d ${change > 0.1 ? 'up' : 'muted'}">${change == null ? '&nbsp;' : esc(t(basis === 'week' ? 'kpi.vs_week' : 'kpi.vs_prev', {p: fmtPct(change, 0)}))}</div></div>
      <div class="kpi"><div class="lab">${t('kpi.total', {r: t('range.' + state.r)})}</div><div class="n">${fmtBytes(tot.wire)}</div><div class="d muted">${esc(peakTxt)}</div></div>
      <div class="kpi"><div class="lab">${t('kpi.hosts')}</div><div class="n">${nf(tot.hosts)}</div><div class="d muted">&nbsp;</div></div>
      <div class="kpi"><div class="lab">${t('kpi.peers')}</div><div class="n">${nf(tot.peers)}</div><div class="d muted">${esc(t('kpi.countries', {n: nf(tot.countries)}))}</div></div>
      <div class="kpi"><div class="lab">${t('kpi.accuracy')}</div><div class="n">${acc.n}</div><div class="d">${acc.kind ? status(acc.kind, acc.d) : `<span class="muted">${esc(acc.d)}</span>`}</div></div>
    </div></div>
    <div class="panel c8"><div class="ph"><h2>${t('ov.bw_title')}</h2><span class="sub">${lines.length ? t(basis === 'week' ? 'ov.bw_sub_week' : 'ov.bw_sub_prev') : ''}</span></div>
      <div class="chart" id="chStack" style="height:250px" aria-label="${esc(t('ov.bw_title'))}"></div>
      <div class="legend">${areas.map((a, i) => `<span><i style="background:${a.color}"></i>${series.names[i] === '__other__' ? esc(a.name) : V('app', series.names[i], a.name)}</span>`).join('')}${lines.length ? `<span><i class="dash"></i>${esc(lines[0].name)}</span>` : ''}</div></div>
    ${panel('c4', t('ov.movers'), t(basis === 'week' ? 'ov.movers_sub_week' : 'ov.movers_sub_prev'), `<ul class="movers">${moverHTML}</ul>`)}
    ${panel('c4', t('ov.dir'), '', '<div class="donut" id="dDir"></div>')}
    ${panel('c4', t('ov.proto'), '', '<div class="donut" id="dProto"></div>')}
    ${panel('c4', t('ov.country'), '', '<div class="donut" id="dCC"></div>')}
    ${panel('c6', t('ov.top_clients'), t('ov.top_clients_sub'), `<table><tr><th></th><th>${t('col.client')}</th><th class="num">${t('col.traffic')}</th><th style="width:26%">${t('col.share')}</th><th class="num">${t(basis === 'week' ? 'col.vs_week' : 'col.vs_prev')}</th></tr>
      ${cl.map((r, i) => { const b = +r.extra; const ch = b > 0 ? (r.wire - b) / b : null; return `<tr><td class="rank">${i + 1}</td><td>${ipCell(r.key)}</td><td class="num">${fmtBytes(r.wire)}</td><td>${bar(r.wire, cmax)}</td><td class="num ${ch > 0.3 ? 'up' : ''}">${ch == null ? `<span class="tag">${t('ov.mover_new')}</span>` : fmtPct(ch, 0)}</td></tr>`; }).join('') || `<tr><td colspan="5" class="empty">${t('empty.nodata')}</td></tr>`}</table>`)}
    ${panel('c6', t('ov.top_services'), t('ov.top_services_sub'), `<table><tr><th></th><th>${t('col.server')}</th><th>${t('col.service')}</th><th class="num">${t('col.traffic')}</th><th style="width:24%">${t('col.share')}</th></tr>
      ${sv.map((r, i) => `<tr><td class="rank">${i + 1}</td><td>${ipCell(r.key)}</td><td class="nw">${V('port', r.key2, r.key2)} <span class="muted">${esc(r.extra)}</span></td><td class="num">${fmtBytes(r.wire)}</td><td>${bar(r.wire, smax)}</td></tr>`).join('') || `<tr><td colspan="5" class="empty">${t('empty.nodata')}</td></tr>`}</table>`)}
  </div>`;
  tsChart($('#chStack'), {times: series.times, areas, lines});
  donut($('#dDir'), dirParts, 'dir');
  donut($('#dProto'), protoParts, 'proto');
  donut($('#dCC'), ccParts, 'country');
};

const DIMS = ['client', 'server', 'conv', 'app', 'port', 'country', 'asn', 'segment', 'exporter', 'encap', 'vlan'];
// Conversations: who talks to whom, as its own page (the Top-N table for
// client, server and service, without the dimension tabs).
views.conv = el => topTable(el, 'conv', true);
views.topn = el => {
  if (!DIMS.includes(state.dim)) state.dim = 'client';
  return topTable(el, state.dim, false);
};
async function topTable(el, dim, standalone) {
  const d = await api('topn', {dim, limit: 66});
  const rows = d.rows || [], max = Math.max(1, ...rows.map(r => r.wire)), total = rows.reduce((s, r) => s + r.wire, 0);
  const cell = r => {
    switch (dim) {
      case 'client': return [ipCell(r.key)];
      case 'server': return [ipCell(r.key), r.extra ? esc(country(r.extra)) : ''];
      case 'conv': return [ipCell(r.key), ipCell(r.key2), V('port', r.key3, r.key3), esc(country(r.extra))];
      case 'app': return [V('app', r.key, r.key)];
      case 'port': return [V('port', r.key, r.key), esc(r.extra)];
      case 'country': return [V('country', r.key, country(r.key)), `<span class="muted">${esc(r.key.startsWith('__') ? '' : r.key)}</span>`];
      case 'asn': return [V('asn', r.key, 'AS' + r.key), esc(r.extra)];
      case 'segment': return [V('segment', r.key, r.key)];
      case 'exporter': return [ipCell(r.key, 'exporter')];
      case 'encap': return [r.key === '0' ? `<span class="muted">${t('encap.none')}</span>` : V('encap', r.key, ENCAP[+r.key] || r.key)];
      case 'vlan': return [r.key === '0' ? `<span class="muted">${t('vlan.none')}</span>` : V('vlan', r.key, r.key)];
    }
    return [esc(r.key)];
  };
  const heads = {client: ['col.client'], server: ['col.server', 'col.country'], conv: ['col.client', 'col.server', 'col.service', 'col.country'],
    app: ['col.app'], port: ['col.port', 'col.app'], country: ['col.country', ''], asn: ['col.asn', 'col.org'], segment: ['col.segment'], exporter: ['col.device'], encap: ['col.encap'], vlan: ['col.vlan']}[dim];
  const showPeers = ['client', 'server'].includes(dim) && rows.some(r => r.peers);
  el.innerHTML = `<div class="panel">
    <div class="ph">${standalone
      ? `<h2>${t('topn.title', {n: 66})}</h2><span class="sub">${esc(t('conv.sub'))}${spanMs() > 216e5 ? ' · ' + esc(t('conv.slow')) : ''}</span>`
      : `<h2>${t('topn.title', {n: 66})}</h2><div class="seg" role="group">${DIMS.map(k => `<button data-dim="${k}" aria-pressed="${k === dim}">${t('dim.' + k)}</button>`).join('')}</div>`}</div>
    <table><tr><th></th>${heads.map(h => `<th>${h ? t(h) : ''}</th>`).join('')}<th class="num">${t('col.traffic')}</th><th class="num">%</th><th style="width:18%">${t('col.share')}</th>${showPeers ? `<th class="num">${t('col.peers')}</th>` : ''}<th class="num">${t('col.flows')}</th></tr>
    ${rows.map((r, i) => `<tr><td class="rank">${i + 1}</td>${cell(r).map(c => `<td>${c}</td>`).join('')}<td class="num">${fmtBytes(r.wire)}</td><td class="num muted">${nf(r.wire / Math.max(1, total) * 100, 1)}</td><td>${bar(r.wire, max)}</td>${showPeers ? `<td class="num">${nf(r.peers)}</td>` : ''}<td class="num">${nf(r.flows)}</td></tr>`).join('') || `<tr><td colspan="9" class="empty">${t('empty.nodata')}</td></tr>`}
    </table></div>`;
  el.querySelectorAll('[data-dim]').forEach(b => b.onclick = () => { state.dim = b.dataset.dim; render(); });
}

views.sankey = async (el) => {
  const d = await api('sankey');
  el.innerHTML = `<div class="panel"><div class="ph"><h2>${t('sankey.title')}</h2><span class="sub">${t('sankey.hint')}</span></div>
    <div class="chart sankey" id="chSankey" style="height:440px" aria-label="${esc(t('sankey.title'))}"></div></div>`;
  const sEl = $('#chSankey');
  if (!(d.seg_app || []).length) { sEl.innerHTML = `<div class="empty">${t('empty.nodata')}</div>`; return; }
  const cols = [[], [], []], ids = [{}, {}, {}];
  const node = (c, k) => { if (!(k in ids[c])) { ids[c][k] = cols[c].length; cols[c].push({k, in: 0, out: 0}); } return cols[c][ids[c][k]]; };
  d.seg_app.forEach(l => { node(0, l.S).out += l.V; node(1, l.T).in += l.V; });
  d.app_cc.forEach(l => { node(1, l.S).out += l.V; node(2, l.T).in += l.V; });
  cols.forEach(c => c.sort((a, b) => (a.k === '__other__') - (b.k === '__other__') || Math.max(b.in, b.out) - Math.max(a.in, a.out)));
  const label = (c, k) => c === 2 ? country(k) : appLabel(k);
  const field = ['segment', 'app', 'country'];
  const draw = () => {
    const W = sEl.clientWidth, H = sEl.clientHeight; if (W < 300) return;
    const rtl = document.documentElement.dir === 'rtl';
    const nodeW = 12, pad = 14, colX = rtl ? [W - 130, Math.round(W / 2 - 6), 118] : [118, Math.round(W / 2 - 6), W - 130];
    const nv = n => Math.max(n.in, n.out);
    const k = Math.min(...cols.map(c => (H - pad * (c.length - 1)) / Math.max(1, c.reduce((s, n) => s + nv(n), 0))));
    cols.forEach((c, ci) => { let yy = 0; c.forEach((n, i) => { n.x = colX[ci]; n.y = yy; n.h = nv(n) * k; n.o = 0; n.i = 0; n.color = ci === 0 ? (n.k === '__other__' ? OTHER : COLORS[i % 6]) : 'var(--ink-2)'; yy += n.h + pad; }); });
    let s = `<svg viewBox="0 0 ${W} ${H}" height="${H}" role="img" aria-label="${esc(t('sankey.title'))}">`;
    const link = (A, B, v, col, lab) => {
      const w = v * k, y0 = A.y + A.o + w / 2, y1 = B.y + B.i + w / 2; A.o += w; B.i += w;
      const x0 = rtl ? A.x : A.x + nodeW, x1 = rtl ? B.x + nodeW : B.x, mx = (x0 + x1) / 2;
      return `<path d="M${x0} ${y0}C${mx} ${y0} ${mx} ${y1} ${x1} ${y1}" fill="none" stroke="${col}" stroke-opacity=".3" stroke-width="${Math.max(1, w - 1)}" data-l="${esc(lab)}" data-v="${v}"/>`;
    };
    const segColor = {}; cols[0].forEach(n => segColor[n.k] = n.color);
    [...d.seg_app].sort((a, b) => ids[0][a.S] - ids[0][b.S] || ids[1][a.T] - ids[1][b.T]).forEach(l => s += link(cols[0][ids[0][l.S]], cols[1][ids[1][l.T]], l.V, segColor[l.S], label(0, l.S) + ' → ' + label(1, l.T)));
    [...d.app_cc].sort((a, b) => ids[1][a.S] - ids[1][b.S] || ids[2][a.T] - ids[2][b.T]).forEach(l => s += link(cols[1][ids[1][l.S]], cols[2][ids[2][l.T]], l.V, 'var(--ink-3)', label(1, l.S) + ' → ' + label(2, l.T)));
    cols.forEach((c, ci) => c.forEach(n => {
      s += `<rect x="${n.x}" y="${n.y}" width="${nodeW}" height="${Math.max(2, n.h)}" rx="2" fill="${n.color}" data-c="${ci}" data-k="${esc(n.k)}"><title>${esc(label(ci, n.k))} ${fmtBytes(nv(n))}</title></rect>`;
      const right = (ci === 2) !== rtl, tx = right ? n.x + nodeW + 8 : n.x - 8, anc = right ? 'start' : 'end';
      if (n.h >= 26) {
        s += `<text x="${tx}" y="${n.y + n.h / 2 - 6}" text-anchor="${anc}" dominant-baseline="middle">${esc(label(ci, n.k))}</text>`;
        s += `<text class="nv" x="${tx}" y="${n.y + n.h / 2 + 9}" text-anchor="${anc}" dominant-baseline="middle">${fmtBytes(nv(n))}</text>`;
      } else if (n.h >= 9 || ci !== 1) {
        s += `<text x="${tx}" y="${n.y + n.h / 2}" text-anchor="${anc}" dominant-baseline="middle">${esc(label(ci, n.k))} <tspan class="nv">${fmtBytes(nv(n))}</tspan></text>`;
      }
    }));
    sEl.innerHTML = s + '</svg>';
    sEl.querySelectorAll('path[data-l]').forEach(p => { p.onmousemove = e => showTip(e, `<div class="t">${esc(p.dataset.l)}</div><b>${fmtBytes(+p.dataset.v)}</b>`); p.onmouseleave = hideTip; });
    sEl.querySelectorAll('rect[data-k]').forEach(r => r.onclick = e => {
      e.stopPropagation(); const ci = +r.dataset.c, kk = r.dataset.k; if (kk === '__other__') return;
      openPop({k: field[ci], v: kk, label: label(ci, kk)}, r.getBoundingClientRect());
    });
  };
  draw();
  if (!sEl._ro) { sEl._ro = new ResizeObserver(draw); sEl._ro.observe(sEl); }
};

views.geo = async (el) => {
  const [c, a] = await Promise.all([api('topn', {dim: 'country', limit: 66}), api('topn', {dim: 'asn', limit: 66})]);
  const cr = (c.rows || []).filter(r => r.key !== '__internal__'), ar = a.rows || [];
  const cm = Math.max(1, ...cr.map(r => r.wire)), am = Math.max(1, ...ar.map(r => r.wire));
  el.innerHTML = `<div class="grid">
    ${panel('c6', t('geo.countries'), t('geo.countries_sub'), `<table><tr><th></th><th>${t('col.country')}</th><th class="num">${t('col.traffic')}</th><th style="width:40%">${t('col.share')}</th></tr>
      ${cr.map((r, i) => `<tr><td class="rank">${i + 1}</td><td>${V('country', r.key, country(r.key))} <span class="muted">${esc(r.key)}</span></td><td class="num">${fmtBytes(r.wire)}</td><td>${bar(r.wire, cm)}</td></tr>`).join('') || `<tr><td colspan="4" class="empty">${t('geo.no_table')}</td></tr>`}</table>`)}
    ${panel('c6', t('geo.as'), '', `<table><tr><th></th><th>${t('col.asn')}</th><th>${t('col.org')}</th><th class="num">${t('col.traffic')}</th><th style="width:32%">${t('col.share')}</th></tr>
      ${ar.map((r, i) => `<tr><td class="rank">${i + 1}</td><td class="nw">${V('asn', r.key, 'AS' + r.key)}</td><td>${esc(r.extra)}</td><td class="num">${fmtBytes(r.wire)}</td><td>${bar(r.wire, am)}</td></tr>`).join('') || `<tr><td colspan="5" class="empty">${t('geo.no_table')}</td></tr>`}</table>`)}
  </div>`;
};

views.threats = async (el) => {
  const d = await api('threats', {limit: 66});
  const rows = d.rows || [], lists = d.lists || {};
  const outRows = rows.filter(r => r.extra.split('|')[1] === '1');
  const hosts = new Set(rows.map(r => r.extra.split('|')[1] === '2' ? r.key3 : r.key2));
  const sent = outRows.reduce((s, r) => s + r.wire, 0);
  const nLists = Object.keys(lists).length, nEntries = Object.values(lists).reduce((a, b) => a + b, 0);
  el.innerHTML = `<div class="grid">
    <div class="panel c12"><div class="kpis" style="grid-template-columns:repeat(4,1fr)">
      <div class="kpi"><div class="lab">${t('thr.hosts')}</div><div class="n">${nf(hosts.size)}</div></div>
      <div class="kpi"><div class="lab">${t('thr.upload')}</div><div class="n ${sent ? 'up' : ''}">${fmtBytes(sent)}</div></div>
      <div class="kpi"><div class="lab">${t('thr.flows')}</div><div class="n">${nf(rows.reduce((s, r) => s + r.flows, 0))}</div></div>
      <div class="kpi"><div class="lab">${t('thr.lists')}</div><div class="n">${nf(nLists)}</div><div class="d muted">${esc(t('thr.entries', {n: nf(nEntries)}))}</div></div>
    </div></div>
    ${panel('c12', t('thr.table'), t('thr.sub'), nLists === 0 ? `<div class="empty">${t('thr.no_lists')}</div>` : `<table><tr><th>${t('col.list')}</th><th>${t('col.client')}</th><th>${t('col.server')}</th><th>${t('col.country')}</th><th>${t('col.direction')}</th><th class="num">${t('col.traffic')}</th><th class="num">${t('col.flows')}</th></tr>
      ${rows.map(r => { const [cc, dir] = r.extra.split('|'); const out = dir === '1'; return `<tr><td>${status(out ? 'bad' : 'warn', r.key)}</td><td>${ipCell(r.key2)}</td><td>${ipCell(r.key3)}</td><td>${cc ? V('country', cc, country(cc)) : ''}</td><td class="${out ? 'up' : ''}">${t('dir.' + dir)}</td><td class="num ${out ? 'up' : ''}">${fmtBytes(r.wire)}</td><td class="num">${nf(r.flows)}</td></tr>`; }).join('') || `<tr><td colspan="7" class="empty">${t('thr.none')}</td></tr>`}</table>`)}
  </div>`;
};

const REC_COLS = [['time', 1], ['client', 1], ['cport', 1], ['server', 1], ['port', 1], ['proto', 1], ['app', 1], ['country', 1], ['encap', 1], ['traffic', 1], ['packets', 1], ['device', 1],
  ['in_if', 0], ['out_if', 0], ['vlan', 0], ['flags', 0], ['sampling', 0], ['asn', 0], ['direction', 0], ['list', 0]];
function recCols() {
  try { const s = JSON.parse(localStorage.getItem('t66.cols') || 'null'); if (Array.isArray(s)) return new Set(s); } catch (e) {}
  return new Set(REC_COLS.filter(c => c[1]).map(c => c[0]));
}
const tcpFlags = f => ['FIN', 'SYN', 'RST', 'PSH', 'ACK', 'URG'].filter((_, i) => f & (1 << i)).join(' ');
views.records = async (el) => {
  const d = await api('records', {limit: 200});
  const rows = d.rows || [], on = recCols();
  const cols = REC_COLS.filter(c => on.has(c[0])).map(c => c[0]);
  const cell = (c, r) => ({
    time: `<span class="muted nw">${new Intl.DateTimeFormat(LANG, {hour: '2-digit', minute: '2-digit', second: '2-digit'}).format(new Date(r.ts))}</span>`,
    client: ipCell(r.client), cport: `<span class="muted">${r.cport || ''}</span>`, server: ipCell(r.server),
    port: r.port ? V('port', r.port + '/' + (r.proto === 17 ? 'udp' : 'tcp'), r.port) : '—', proto: V('proto', r.proto, proto(r.proto)), app: V('app', r.app, r.app),
    country: r.dir === 3 ? `<span class="muted">${t('internal')}</span>` : (r.cc ? V('country', r.cc, country(r.cc)) : '—'),
    encap: r.encap ? V('encap', r.encap, ENCAP[r.encap]) : '<span class="muted">—</span>', traffic: fmtBytes(r.wire), packets: nf(r.pkts),
    device: ipCell(r.exporter, 'exporter'), in_if: r.in_if || '', out_if: r.out_if || '', vlan: r.vlan ? V('vlan', r.vlan, r.vlan) : '',
    flags: r.proto === 6 ? `<span class="muted">${tcpFlags(r.tcp_flags)}</span>` : '', sampling: r.mult > 1 ? '1:' + nf(r.mult) : '1:1',
    asn: r.asn ? V('asn', r.asn, 'AS' + r.asn) : '', direction: t('dir.' + r.dir), list: r.threat ? status('bad', r.threat) : ''
  })[c];
  const num = new Set(['cport', 'port', 'traffic', 'packets', 'in_if', 'out_if', 'sampling']);
  el.innerHTML = `<div class="panel"><div class="ph"><h2>${t('rec.title')}</h2><span class="sub">${esc(t('rec.latest', {n: rows.length}))}</span>
      <div class="cols"><button class="btn" id="colBtn">${t('rec.columns')}</button><div class="menu" id="colMenu" hidden>${REC_COLS.map(c => `<label><input type="checkbox" value="${c[0]}" ${on.has(c[0]) ? 'checked' : ''}>${t('rc.' + c[0])}</label>`).join('')}</div></div></div>
    <div style="overflow-x:auto"><table><tr>${cols.map(c => `<th class="${num.has(c) ? 'num' : ''}">${t('rc.' + c)}</th>`).join('')}</tr>
    ${rows.map(r => `<tr>${cols.map(c => `<td class="${num.has(c) ? 'num' : ''}">${cell(c, r)}</td>`).join('')}</tr>`).join('') || `<tr><td colspan="${cols.length}" class="empty">${t('empty.nodata')}</td></tr>`}</table></div></div>`;
  $('#colBtn').onclick = e => { e.stopPropagation(); $('#colMenu').hidden = !$('#colMenu').hidden; };
  $('#colMenu').onchange = () => {
    const sel = [...$('#colMenu').querySelectorAll('input:checked')].map(i => i.value);
    try { localStorage.setItem('t66.cols', JSON.stringify(sel)); } catch (e) {}
    render();
  };
};

function ifName(f) { return f.name || ((f.device || f.exporter) + ' #' + f.ifindex); }
function devKind(f) {
  if (!f.has_counters) return null;
  const dv = Math.max(Math.abs(f.in_dev), Math.abs(f.out_dev));
  return dv > Math.max(0.02, 2 * (f.stat_err || 0)) ? 'warn' : 'ok';
}
views.ifaces = async (el) => {
  const d = await api('ifaces');
  const list = (d.ifaces || []).slice().sort((a, b) => (!!b.has_counters - !!a.has_counters) || ((devKind(b) === 'warn') - (devKind(a) === 'warn')));
  if (!list.length) { el.innerHTML = `<div class="panel"><div class="empty">${t('empty.nodata')}</div></div>`; return; }
  if (!state.ifc || !list.some(f => f.exporter === state.ifc.exporter && f.ifindex === state.ifc.ifindex)) state.ifc = {exporter: list[0].exporter, ifindex: list[0].ifindex};
  el.innerHTML = `<div class="grid">
    ${panel('c4', t('if.title'), t('if.sub'), `<div class="iflist">${list.map(f => {
      const k = devKind(f), cur = f.exporter === state.ifc.exporter && f.ifindex === state.ifc.ifindex;
      return `<button data-e="${esc(f.exporter)}" data-i="${f.ifindex}" aria-current="${cur}"><span>${esc(ifName(f))}<br><span class="muted" style="font-size:12.5px">${esc(f.device || f.exporter)} · ${f.ifindex}</span></span>${k ? status(k, nf(Math.max(Math.abs(f.in_dev), Math.abs(f.out_dev)) * 100, 1) + '%') : `<span class="muted" style="font-size:12.5px">${t('if.no_ctr_short')}</span>`}</button>`;
    }).join('')}</div>`)}
    <div class="panel c8" id="recon"><div class="empty">…</div></div></div>`;
  el.querySelectorAll('.iflist button').forEach(b => b.onclick = () => { state.ifc = {exporter: b.dataset.e, ifindex: +b.dataset.i}; render(); });
  const rc = await api('recon', {exporter: state.ifc.exporter, ifindex: state.ifc.ifindex});
  const r = rc.recon, dirIn = state.ifdir === 'in';
  const dev = dirIn ? r.in_dev : r.out_dev, err = dirIn ? r.in_stat_err : r.out_stat_err;
  let verdict, vk = 'ok';
  if (!r.has_counters) { verdict = t('if.no_counters'); vk = 'none'; }
  else if (Math.abs(dev) <= Math.max(0.02, 2 * err)) {
    verdict = t('if.ok', {dev: fmtPct(dev), err: '±' + nf(err * 100, 1) + '%'});
  } else {
    vk = 'warn';
    const causes = [];
    const src = rc.source;
    if (src && src.lost_pct >= 0.5) causes.push(t('cause.loss', {p: nf(src.lost_pct, 1) + '%'}));
    if (src && src.sampling_state === 'waiting') causes.push(t('cause.sampling'));
    if (dev < 0) causes.push(t('cause.unsampled_iface'));
    if (dev > 0) causes.push(t('cause.dup'));
    verdict = t(dev < 0 ? 'if.low' : 'if.high', {dev: nf(Math.abs(dev) * 100, 1) + '%', causes: causes.join('; ')});
  }
  const ctr = dirIn ? r.in_counter : r.out_counter, flw = dirIn ? r.in_flow : r.out_flow;
  $('#recon').innerHTML = `<div class="ph"><div><div class="sub">${esc(ifName({...rc, exporter: r.exporter, ifindex: r.ifindex}))} · ${esc(rc.device || r.exporter)}</div>
      <div style="font-size:22px;font-weight:650">${r.has_counters ? t('if.dev', {v: fmtPct(dev)}) : '—'} ${r.has_counters && err ? `<span class="muted" style="font-size:13px;font-weight:400">${t('if.stat', {v: nf(err * 100, 1) + '%'})}</span>` : ''}</div></div>
      <div class="seg" role="group"><button data-d="in" aria-pressed="${dirIn}">${t('if.in')}</button><button data-d="out" aria-pressed="${!dirIn}">${t('if.out')}</button></div></div>
    <div class="chart" id="chRecon" style="height:240px" aria-label="${esc(t('if.title'))}"></div>
    <div class="legend"><span><i class="line" style="background:var(--c1)"></i>${t('if.flow')}</span>${r.has_counters ? `<span><i class="line" style="background:var(--c2)"></i>${t('if.counter')}</span>` : ''}</div>
    <div class="verdict ${vk === 'ok' ? '' : vk}">${verdict}</div>`;
  $('#recon').querySelectorAll('[data-d]').forEach(b => b.onclick = () => { state.ifdir = b.dataset.d; render(); });
  tsChart($('#chRecon'), {times: r.times, lines: [...(r.has_counters ? [{name: t('if.counter'), color: 'var(--c2)', data: ctr}] : []), {name: t('if.flow'), color: 'var(--c1)', data: flw}]});
};

views.sources = async (el) => {
  const [d, inv] = await Promise.all([api('sources'), api('inventory').catch(() => ({text: ''}))]);
  const srcs = d.sources || [];
  const issueText = (code, s) => t('issue.' + code, {n: nf(s.pending), p: nf(s.lost_pct, 1) + '%', s: Math.round(Math.abs(s.clock_skew_ns) / 1e9) + ' s'});
  const rows = srcs.map(s => {
    const kind = s.status;
    return `<tr class="${s.issues?.length ? 'has-fix' : ''}"><td>${s.name ? esc(s.name) + ' ' : ''}<span class="muted" style="font-size:12.5px">${esc(s.exporter)}${s.domain ? ' / ' + s.domain : ''}</span></td>
      <td class="nw">${esc(s.proto)}</td><td class="num">${nf(s.rec_per_sec, 1)}</td>
      <td class="nw">${s.sampling ? esc(s.sampling) : (s.sampling_state === 'waiting' ? '<span class="muted">?</span>' : '1:1')}${s.effective ? ` <span class="muted">(${esc(t('src.effective', {v: '1:' + nf(s.effective)}))})</span>` : ''}</td>
      <td class="num">${nf(s.lost_pct, 2)}%</td><td class="nw muted">${s.last_seen ? ago(Date.parse(s.last_seen)) : ''}</td>
      <td>${status(kind, t('st.' + kind))}</td></tr>
      ${s.issues?.length ? `<tr><td colspan="7" style="padding-top:0">${s.issues.map(c => `<div class="issue">${esc(issueText(c, s))}</div>`).join('')}</td></tr>` : ''}`;
  }).join('');
  const lis = (d.listeners || []).map(l => `<span class="tag">UDP ${esc(l.addr)} · ${esc(l.proto)} · ${nf(l.packets)}</span>`).join(' ');
  const caps = (d.captures || []).map(c => `<span class="tag">${esc(c.iface)} · ${esc(c.method)} · ${nf(c.packets)}${c.error ? ' · ' + esc(c.error) : ''}</span>`).join(' ');
  const snmp = (d.snmp || []).map(x => `<span class="tag" title="${esc(x.error || '')}">${esc(x.name ? x.name + " " + x.exporter : x.exporter)} · ${x.ok
    ? '<span style="color:var(--good)">✔</span> ' + esc(t('src.snmp_ok', {n: nf(x.interfaces)}))
    : '<span style="color:var(--crit)">✖</span> ' + esc(t('src.snmp_fail'))}</span>`).join(' ');
  el.innerHTML = `<div class="grid">
    ${panel('c12', t('src.title'), t('src.sub'), `<table><tr><th>${t('col.device')}</th><th>${t('col.proto')}</th><th class="num">${t('col.rate')}</th><th>${t('col.sampling')}</th><th class="num">${t('col.lost')}</th><th>${t('col.last')}</th><th>${t('col.status')}</th></tr>
      ${rows || `<tr><td colspan="7" class="empty">${t('empty.first')}</td></tr>`}</table>
      <p style="margin:12px 0 0;font-size:12.5px"><span class="muted">${t('src.listeners')}</span> ${lis || '—'} ${caps}</p>
      ${snmp ? `<p style="margin:6px 0 0;font-size:12.5px"><span class="muted">${t('src.snmp')}</span> ${snmp}</p>` : ''}`)}
    ${panel('c12', t('src.names'), t('src.names_sub'), `<textarea class="inv" id="inv" spellcheck="false">${esc(inv.text || '')}</textarea>
      <div style="display:flex;gap:10px;align-items:center;margin-top:8px"><button class="primary" id="invSave">${t('src.save')}</button><span id="invMsg" class="muted" style="font-size:13px"></span></div>`)}
  </div>`;
  $('#invSave').onclick = async () => {
    const res = await fetch('/api/inventory', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({text: $('#inv').value})});
    const j = await res.json().catch(() => ({}));
    $('#invMsg').textContent = res.ok ? t('src.saved') : (j.error || res.statusText);
    $('#invMsg').style.color = res.ok ? 'var(--good)' : 'var(--crit)';
    if (res.ok) loadStatus();
  };
};

// ------------------------------------------------------------ shell
async function render() {
  renderFilters();
  writeHash();
  $('#title').textContent = t('nav.' + state.v);
  document.title = t('nav.' + state.v) + ' · traffic66';
  document.querySelectorAll('#nav button').forEach(b => b.dataset.v === state.v ? b.setAttribute('aria-current', 'page') : b.removeAttribute('aria-current'));
  document.querySelectorAll('#range button').forEach(b => b.setAttribute('aria-pressed', b.dataset.r === state.r));
  document.querySelectorAll('.view').forEach(s => s.classList.toggle('on', s.id === 'v-' + state.v));
  const el = $('#v-' + state.v), seq = ++loadSeq;
  try {
    await views[state.v](el);
  } catch (e) {
    if (e.message !== 'login' && seq === loadSeq) el.innerHTML = errorBox(e);
  }
  if (seq === loadSeq) resolveNames();
}
function go(v) { state.v = v; render(); window.scrollTo(0, 0); }

async function loadStatus() {
  try {
    const res = await fetch('/api/status');
    if (res.status === 401) { showLogin(); return; }
    const s = await res.json();
    for (const [ip, n] of Object.entries(s.hosts || {})) names.set(ip, n);
    $('#demo').hidden = !s.demo; $('#demo').textContent = t('demo.badge');
    const live = s.records_per_sec > 0.2;
    $('#self').innerHTML = `<div class="live ${live ? 'ok' : 'idle'}"><i></i>${t(live ? 'self.receiving' : 'self.idle')}</div>
      <span class="k">${t('self.ingest')}</span><span class="val">${esc(t('self.per_sec', {n: nf(s.records_per_sec, s.records_per_sec < 10 ? 1 : 0)}))}</span>
      <span class="k">${t('self.dropped')}</span><span class="val" style="${s.dropped ? 'color:var(--crit)' : ''}">${nf(s.dropped)}</span>
      <span class="k">${t('self.disk')}</span><span class="val">${s.days_left >= 0 ? esc(t('self.days', {n: nf(Math.min(s.days_left, 9999))})) : fmtBytes(s.disk_bytes)}</span>
      ${s.write_errors ? `<span class="k" style="color:var(--crit)">!</span><span class="val" style="color:var(--crit)" title="${esc(s.last_error)}">${nf(s.write_errors)}</span>` : ''}`;
    $('#srcBadge').hidden = !s.source_warnings; $('#srcBadge').textContent = s.source_warnings;
  } catch (e) {}
}

function showLogin() {
  $('#app').hidden = true; $('#login').hidden = false;
  $('#loginForm [name=user]').focus();
}
$('#loginForm').addEventListener('submit', async e => {
  e.preventDefault();
  const f = new FormData(e.target);
  const res = await fetch('/api/login', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({user: f.get('user'), password: f.get('password')})});
  if (!res.ok) { $('#loginErr').hidden = false; $('#loginErr').textContent = t('login.bad'); return; }
  $('#login').hidden = true; $('#app').hidden = false;
  loadStatus(); render();
});
$('#logout').onclick = async () => { await fetch('/api/logout', {method: 'POST'}); showLogin(); };

async function init() {
  $('#lang').innerHTML = LANGS.map(([c, n]) => `<option value="${c}">${n}</option>`).join('');
  const lang = pickLang();
  await loadLang(lang);
  $('#lang').value = lang;
  $('#lang').onchange = async () => {
    await loadLang($('#lang').value);
    $('#range').querySelectorAll('button').forEach(b => b.textContent = t('range.' + b.dataset.r));
    loadStatus(); render();
  };
  $('#range').innerHTML = RANGES.map(r => `<button data-r="${r}">${t('range.' + r)}</button>`).join('');
  $('#range').onclick = e => { const b = e.target.closest('[data-r]'); if (b) { state.r = b.dataset.r; render(); } };
  $('#nav').onclick = e => { const b = e.target.closest('[data-v]'); if (b) go(b.dataset.v); };
  $('#share').onclick = () => { navigator.clipboard?.writeText(location.href).then(() => toast(t('top.copied')), () => toast(location.href)); };
  document.addEventListener('click', e => { if (!e.target.closest('.cols')) { const m = $('#colMenu'); if (m) m.hidden = true; } });
  readHash();
  window.addEventListener('hashchange', () => { const before = location.hash; readHash(); if (before) render(); });
  const res = await fetch('/api/status');
  if (res.status === 401) { showLogin(); return; }
  $('#app').hidden = false;
  await loadStatus();
  render();
  setInterval(loadStatus, 10000);
  setInterval(() => {
    if (document.hidden || pop.style.display === 'block' || document.activeElement?.tagName === 'TEXTAREA' || document.activeElement?.id === 'q') return;
    if (state.v === 'sources' || state.v === 'records' || state.v === 'overview' || state.r === '15m' || state.r === '1h') render();
  }, 30000);
}
init();
})();
