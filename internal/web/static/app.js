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
function fmtPps(v) {
  if (v == null || !isFinite(v)) return '—';
  for (const [n, m] of [['M', 1e6], ['k', 1e3]]) if (Math.abs(v) >= m) return nf(v / m, Math.abs(v / m) < 10 ? 2 : Math.abs(v / m) < 100 ? 1 : 0) + ' ' + n + 'p/s';
  return nf(v, v < 10 ? 1 : 0) + ' p/s';
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
// categorical colours in fixed order (validated for colour-blind readers); a
// ninth series is folded into "other", never given a recycled colour
const COLORS = ['var(--c1)', 'var(--c2)', 'var(--c3)', 'var(--c4)', 'var(--c5)', 'var(--c6)', 'var(--c7)', 'var(--c8)'];
const color = (i, key) => key === '__other__' || i >= COLORS.length ? OTHER : COLORS[i];
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
  return `<button class="v" data-k="${field}" data-val="${esc(ip)}" data-ip="${esc(ip)}" aria-haspopup="menu">${n ? esc(n) + ` <span class="ip">${esc(ip)}</span>` : esc(ip)}</button>`;
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
      if (n && !el.querySelector('.ip')) el.innerHTML = esc(n) + ` <span class="ip">${esc(el.dataset.ip)}</span>`;
    });
  } catch (e) {}
}

// ------------------------------------------------------------ state & API
const VIEWS = ['overview', 'findings', 'topn', 'traffic', 'sankey', 'geo', 'threats', 'records', 'cleanup', 'sandbox', 'ifaces', 'sources', 'detail'];
const RANGES = ['15m', '1h', '6h', '24h', '7d', '30d'];
const state = {v: 'overview', r: '24h', f: [], dim: 'conv', ifc: null, ifdir: 'both', sort: {k: 'wire', asc: false}, det: null};
function readHash() {
  const p = new URLSearchParams(location.hash.slice(1));
  if (VIEWS.includes(p.get('v'))) state.v = p.get('v');
  if (RANGES.includes(p.get('r'))) state.r = p.get('r');
  if (p.get('r') === 'custom' && +p.get('from') > 0 && +p.get('to') > +p.get('from')) { state.r = 'custom'; state.cf = +p.get('from'); state.ct = +p.get('to'); }
  if (p.get('dim')) state.dim = p.get('dim');
  state.tm = p.get('tm') === 'table' || (p.get('v') === 'topn' && p.get('dim')) ? 'table' : 'talkers';
  state.sk = ['segment', 'conv'].includes(p.get('by')) ? p.get('by') : 'host';
  state.ra = p.get('ra') === 'client' ? 'client' : 'server';
  state.rb = p.get('rb') === 'server' ? 'server' : 'service';
  state.ds = p.get('ds') === 'sb' ? 'sb' : '';
  const dd = p.get('d'); if (dd && dd.includes(':')) state.det = {f: dd.slice(0, dd.indexOf(':')), v: dd.slice(dd.indexOf(':') + 1)};
  state.f = (p.get('f') || '').split(',').filter(Boolean).map(s => {
    const neg = s[0] === '!'; if (neg) s = s.slice(1);
    const i = s.indexOf(':');
    return {f: decodeURIComponent(s.slice(0, i)), v: decodeURIComponent(s.slice(i + 1)), neg};
  }).filter(x => x.f && x.v);
}
function writeHash(push) {
  const p = new URLSearchParams({v: state.v, r: state.r});
  if (state.r === 'custom') { p.set('from', state.cf); p.set('to', state.ct); }
  if (state.ds) p.set('ds', state.ds);
  if (state.v === 'topn') { if (state.tm === 'table') { p.set('tm', 'table'); p.set('dim', state.dim); } }
  if (state.v === 'sankey' && state.sk !== 'host') p.set('by', state.sk);
  if (state.v === 'traffic') { if (state.ra === 'client') p.set('ra', 'client'); if (state.rb === 'server') p.set('rb', 'server'); }
  if (state.v === 'detail' && state.det) p.set('d', state.det.f + ':' + state.det.v);
  if (state.f.length) p.set('f', state.f.map(x => (x.neg ? '!' : '') + encodeURIComponent(x.f) + ':' + encodeURIComponent(x.v)).join(','));
  const h = '#' + p.toString();
  if (push && h !== location.hash) history.pushState(null, '', h); else history.replaceState(null, '', h);
}
// pages whose data can come from the sandbox (uploaded capture files)
const SB_DATA = new Set(['overview', 'topn', 'sankey', 'series', 'records', 'threats', 'findings', 'rings', 'geolines']);
const inSB = () => state.ds === 'sb' && sbInfo.ready;
let sbInfo = {files: [], ready: false};
// started as "traffic66 file.pcap": there is no live data, only the files
let offlineMode = false;
// the server's clock, shown under the logo
let clockSkew = 0;
function tickClock() {
  try { $('#clock').textContent = new Intl.DateTimeFormat(LANG, {month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit'}).format(new Date(Date.now() + clockSkew)); } catch (e) {}
}
setInterval(tickClock, 1000);
async function api(path, extra = {}, filters = state.f) {
  if (inSB() && (path === 'ifaces' || path === 'recon')) return {ifaces: []};
  const p = new URLSearchParams(inSB() && SB_DATA.has(path) ? {ds: 'sb', ...sbRange(), ...extra} : state.r === 'custom' ? {from: state.cf, to: state.ct, ...extra} : {range: state.r, ...extra});
  if (filters.length) p.set('f', JSON.stringify(filters.map(x => ({f: x.f, v: x.v, neg: x.neg}))));
  const res = await fetch('/api/' + path + '?' + p);
  if (res.status === 401) { showLogin(); throw new Error('login'); }
  const j = await res.json().catch(() => ({}));
  if (!res.ok) { const e = new Error(j.error || res.statusText); e.kind = j.kind; throw e; }
  return j;
}
// the sandbox's time range: the whole capture, on whole minutes
function sbRange() {
  const a = Math.floor(Date.parse(sbInfo.first) / 6e4) * 6e4, b = Math.ceil((Date.parse(sbInfo.last) + 1) / 6e4) * 6e4;
  return {from: a, to: Math.max(b, a + 6e4)};
}
const spanMs = () => { if (inSB()) { const r = sbRange(); return r.to - r.from; } if (state.r === 'custom') return state.ct - state.cf; return ({'15m': 9e5, '1h': 36e5, '6h': 216e5, '24h': 864e5, '7d': 6048e5, '30d': 2592e6})[state.r]; };
const rangeLabel = () => inSB() ? t('sb.range') : state.r === 'custom' ? customLabel() : t('range.' + state.r);
// "10/05 08:00 – 10/06 08:00": a custom range in the reader's time
function customLabel() {
  const a = new Date(state.cf), b = new Date(state.ct), same = a.toDateString() === b.toDateString();
  const df = new Intl.DateTimeFormat(LANG, {month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit'});
  return df.format(a) + ' – ' + (same ? new Intl.DateTimeFormat(LANG, {hour: '2-digit', minute: '2-digit'}).format(b) : df.format(b));
}
// the custom range picker: start and end as date and time
const localInput = ms => { const d = new Date(ms - new Date(ms).getTimezoneOffset() * 6e4); return d.toISOString().slice(0, 16); };
function openRangePop(btn) {
  let pop = $('#rangePop');
  if (!pop) { pop = document.createElement('div'); pop.id = 'rangePop'; pop.className = 'rangepop'; document.body.appendChild(pop); }
  const now = Date.now(), to = state.r === 'custom' ? state.ct : now, from = state.r === 'custom' ? state.cf : now - (spanMs() || 864e5);
  pop.innerHTML = `<label>${t('range.from')}<input type="datetime-local" id="rpFrom" value="${localInput(from)}"></label>
    <label>${t('range.to')}<input type="datetime-local" id="rpTo" value="${localInput(to)}"></label>
    <div class="rpmsg" id="rpMsg"></div>
    <div class="rpbtns"><button class="btn" id="rpCancel">${t('range.cancel')}</button><button class="primary" id="rpApply">${t('range.apply')}</button></div>`;
  pop.hidden = false;
  const r = btn.getBoundingClientRect();
  pop.style.top = (r.bottom + 6) + 'px';
  pop.style.left = Math.max(8, Math.min(r.right - pop.offsetWidth, innerWidth - pop.offsetWidth - 8)) + 'px';
  $('#rpCancel').onclick = () => { pop.hidden = true; };
  $('#rpApply').onclick = () => {
    const f = new Date($('#rpFrom').value).getTime(), tt = new Date($('#rpTo').value).getTime();
    if (!(f > 0) || !(tt > f)) { $('#rpMsg').textContent = t('range.bad'); return; }
    state.r = 'custom'; state.cf = f; state.ct = Math.min(tt, Date.now() + 6e4);
    pop.hidden = true; render(true);
  };
}
function renderRange() {
  $('#range').innerHTML = RANGES.map(r => `<button data-r="${r}">${t('range.' + r)}</button>`).join('') +
    `<button data-r="custom" class="custom">${state.r === 'custom' ? esc(customLabel()) : t('range.custom')}</button>`;
  $('#range').querySelectorAll('button').forEach(b => b.setAttribute('aria-pressed', b.dataset.r === state.r));
}

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
      let d = '', pen = false, area = '', seg = [];
      const close = () => { if (seg.length > 1) area += 'M' + seg.map(i => x(i).toFixed(1) + ' ' + y(l.data[i]).toFixed(1)).join('L') + `L${x(seg[seg.length - 1]).toFixed(1)} ${y(0)}L${x(seg[0]).toFixed(1)} ${y(0)}Z`; seg = []; };
      l.data.forEach((v, i) => { if (v == null || v < 0) { pen = false; close(); return; } d += (pen ? 'L' : 'M') + x(i).toFixed(1) + ' ' + y(v).toFixed(1); pen = true; seg.push(i); });
      close();
      if (l.fill && area) s += `<path d="${area}" fill="${l.color}" fill-opacity=".14" stroke="none"/>`;
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
  client: v => 'https://bgp.he.net/ip/' + encodeURIComponent(v),
  server: v => 'https://bgp.he.net/ip/' + encodeURIComponent(v),
  asn: v => 'https://bgp.he.net/AS' + encodeURIComponent(v.replace(/^AS/i, '')),
  port: v => 'https://www.speedguide.net/port.php?port=' + encodeURIComponent(v.split('/')[0]),
};
function openPop(c, rect) {
  ctx = c;
  pop.innerHTML = `<div class="who">${esc(t('field.' + c.k))}<strong>${esc(c.label)}</strong></div>
    <button role="menuitem" data-a="only">${t('pop.only')}<kbd>F</kbd></button>
    <button role="menuitem" data-a="not">${t('pop.not')}<kbd>X</kbd></button>
    <button role="menuitem" data-a="records">${t('pop.records')}<kbd>↵</kbd></button>
    ${DETAIL.has(c.k) ? `<button role="menuitem" data-a="detail">${t('pop.detail')}<kbd>D</kbd></button>` : ''}
    ${NAMEABLE.has(c.k) ? `<button role="menuitem" data-a="name">${t('pop.name')}<kbd>N</kbd></button>` : ''}
    ${LOOKUP[c.k] ? `<button role="menuitem" data-a="lookup">${t('pop.lookup')}</button>` : ''}`;
  pop.style.display = 'block';
  const pr = pop.getBoundingClientRect();
  pop.style.left = Math.max(8, Math.min(rect.left, innerWidth - pr.width - 8)) + 'px';
  pop.style.top = (rect.bottom + pr.height + 8 > innerHeight ? rect.top - pr.height - 6 : rect.bottom + 6) + 'px';
  pop.querySelector('button').focus();
}
// what a value's popup offers besides filtering
const DETAIL = new Set(['ip', 'client', 'server', 'exporter', 'port']);
const NAMEABLE = new Set(['ip', 'client', 'server', 'exporter']);
function showDetail(f, v) { state.det = {f: ['exporter', 'client', 'server'].includes(f) ? 'ip' : f, v}; go('detail'); }
// Name an address from anywhere: the inventory line for it is replaced
// (host for an address, device for a flow exporter) and saved at once.
function nameForm() {
  const kind = ctx.k === 'exporter' ? 'device' : 'host', ip = ctx.v, cur = names.get(ip) || '';
  pop.innerHTML = `<div class="who">${esc(t('pop.name'))}<strong>${esc(ip)}</strong></div>
    <form><input name="n" maxlength="80" value="${esc(cur)}" placeholder="${esc(t('pop.name_ph'))}"><button>${esc(t('src.save'))}</button></form>`;
  const inp = pop.querySelector('input'); inp.focus(); inp.select();
  pop.querySelector('form').onsubmit = async e => {
    e.preventDefault();
    const name = inp.value.trim();
    try {
      const inv = await api('inventory');
      const re = new RegExp('^\\s*(host|device)\\s+' + ip.replace(/[.:]/g, '\\$&') + '(\\s|$)', 'i');
      const lines = (inv.text || '').split('\n').filter(l => !re.test(l));
      while (lines.length && lines[lines.length - 1].trim() === '') lines.pop();
      if (name) lines.push(`${kind} ${ip} ${name}`);
      const res = await fetch('/api/inventory', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({text: lines.join('\n') + '\n'})});
      if (!res.ok) { const j = await res.json().catch(() => ({})); toast(j.error || res.statusText); return; }
      if (name) names.set(ip, name); else names.delete(ip);
      pop.style.display = 'none';
      toast(t('pop.named'));
      await loadStatus(); render();
    } catch (err) { toast(errMsg(err)); }
  };
}
function act(a) {
  if (!ctx) return;
  if (a === 'name') { nameForm(); return; }
  pop.style.display = 'none';
  if (a === 'detail') { showDetail(ctx.k, ctx.v); return; }
  if (a === 'lookup') { window.open(LOOKUP[ctx.k](ctx.v), '_blank', 'noopener'); return; }
  state.f = state.f.filter(x => !(x.f === ctx.k && x.v === ctx.v));
  if (a === 'records') { state.f = state.f.filter(x => x.f !== ctx.k); state.f.push({f: ctx.k, v: ctx.v, neg: false}); go('records'); return; }
  state.f.push({f: ctx.k, v: ctx.v, neg: a === 'not'});
  render();
}
pop.addEventListener('click', e => { e.stopPropagation(); const b = e.target.closest('button[data-a]'); if (b) act(b.dataset.a); });
document.addEventListener('keydown', e => {
  if (pop.style.display !== 'block' || e.target.closest?.('#pop form')) return;
  const k = e.key.toLowerCase();
  if (k === 'escape') pop.style.display = 'none';
  else if (k === 'f') act('only');
  else if (k === 'x') act('not');
  else if (k === 'd' && DETAIL.has(ctx?.k)) act('detail');
  else if (k === 'n' && NAMEABLE.has(ctx?.k)) { e.preventDefault(); act('name'); }
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
// Filter dropdowns: device, client, server, service. Each offers the busiest
// values of the time range (given the other filters) and also takes typed
// values; choosing one replaces any filter on that field, emptying it
// removes the filter.
const FBAR = [['exporter', 'field.exporter'], ['client', 'field.client'], ['server', 'field.server'], ['port', 'col.service']];
// The search box and the filter boxes are on Top 66 and Traffic details
// only; elsewhere filters come from clicking values, and their chips show
// when there are any.
const FBAR_VIEWS = new Set(['topn', 'traffic']);
function renderFbar() {
  const bar = $('#fbar'), on = FBAR_VIEWS.has(state.v);
  bar.hidden = !on;
  $('.search').hidden = !on;
  $('#filters').hidden = !on && !state.f.length;
  if (bar.hidden) return;
  bar.innerHTML = FBAR.map(([f, lab]) => {
    const cur = state.f.find(x => x.f === f && !x.neg);
    return `<label><span>${esc(t(lab))}</span><input data-fb="${f}" list="dl-${f}" autocomplete="off" spellcheck="false" class="${cur ? 'on' : ''}" value="${esc(cur ? cur.v : '')}" placeholder="${esc(t('fbar.any'))}"><datalist id="dl-${f}"></datalist></label>`;
  }).join('');
  bar.querySelectorAll('input').forEach(inp => {
    const f = inp.dataset.fb;
    inp.onfocus = async () => {
      const dl = $('#dl-' + f);
      if (dl.dataset.k === state.r + JSON.stringify(state.f)) return;
      dl.dataset.k = state.r + JSON.stringify(state.f);
      try {
        const d = await api('topn', {dim: f, limit: 200}, state.f.filter(x => x.f !== f));
        const rows = d.rows || [];
        if (f !== 'port') await lookupNames(rows.map(r => r.key));
        dl.innerHTML = rows.map(r => `<option value="${esc(r.key)}">${esc([f === 'port' ? r.extra : names.get(r.key), fmtBytes(r.wire)].filter(Boolean).join(' · '))}</option>`).join('');
      } catch (e) {}
    };
    const apply = () => {
      const v = inp.value.trim(), cur = state.f.find(x => x.f === f && !x.neg);
      if ((cur ? cur.v : '') === v) return;
      state.f = state.f.filter(x => x.f !== f);
      if (v) state.f.push({f, v, neg: false});
      render();
    };
    inp.onchange = apply;
    inp.onkeydown = e => { if (e.key === 'Enter') { e.preventDefault(); apply(); } };
  });
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
  const [d, ifs, fd] = await Promise.all([api('overview'), api('ifaces').catch(() => null), api('findings', {limit: 3}).catch(() => null)]);
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
  const areas = series.names.map((n, i) => ({name: appLabel(n), color: color(i, n), data: series.values[i]}));
  const lines = d.baseline ? [{name: t(basis === 'week' ? 'last_week' : 'prev_period'), color: 'var(--base)', dash: true, data: d.baseline}] : [];
  const dirParts = (d.dir || []).map((p, i) => ({n: t('dir.' + p.key), v: p.wire, c: color(i), val: DIRS[p.key]}));
  const protoParts = (d.proto || []).map((p, i) => ({n: p.key === '__other__' ? t('other') : proto(+p.key), v: p.wire, c: color(i, p.key), val: p.key === '__other__' ? null : p.key}));
  const cl = d.top_clients || [], cmax = Math.max(1, ...cl.map(r => r.wire));
  const sv = d.top_services || [], smax = Math.max(1, ...sv.map(r => r.wire));
  el.innerHTML = `<div class="grid">
    <div class="panel c12"><div class="kpis">
      ${inSB() ? `<div class="kpi"><div class="lab">${t('sb.avg')}</div><div class="n">${fmtBps(tot.wire * 8 / (spanMs() / 1000))}</div><div class="d muted">&nbsp;</div></div>`
        : `<div class="kpi"><div class="lab">${t('kpi.now')}</div><div class="n">${fmtBps(d.now_bps)}</div><div class="d ${change > 0.1 ? 'up' : 'muted'}">${change == null ? '&nbsp;' : esc(t(basis === 'week' ? 'kpi.vs_week' : 'kpi.vs_prev', {p: fmtPct(change, 0)}))}</div></div>`}
      <div class="kpi"><div class="lab">${t('kpi.total', {r: rangeLabel()})}</div><div class="n">${fmtBytes(tot.wire)}</div><div class="d muted">${esc(peakTxt)}</div></div>
      <div class="kpi"><div class="lab">${t('kpi.hosts')}</div><div class="n">${nf(tot.hosts)}</div><div class="d muted">&nbsp;</div></div>
      <div class="kpi"><div class="lab">${t('kpi.peers')}</div><div class="n">${nf(tot.peers)}</div><div class="d muted">${esc(t('kpi.countries', {n: nf(tot.countries)}))}</div></div>
      <div class="kpi"><div class="lab">${t('kpi.accuracy')}</div><div class="n">${acc.n}</div><div class="d">${acc.kind ? status(acc.kind, acc.d) : `<span class="muted">${esc(acc.d)}</span>`}</div></div>
    </div></div>
    <div class="panel c12"><div class="ph"><h2>${t('ov.bw_title')}</h2><span class="sub">${lines.length ? t(basis === 'week' ? 'ov.bw_sub_week' : 'ov.bw_sub_prev') : ''}</span></div>
      <div class="chart" id="chStack" style="height:250px" aria-label="${esc(t('ov.bw_title'))}"></div>
      <div class="legend">${areas.map((a, i) => `<span><i style="background:${a.color}"></i>${series.names[i] === '__other__' ? esc(a.name) : V('app', series.names[i], a.name)}</span>`).join('')}${lines.length ? `<span><i class="dash"></i>${esc(lines[0].name)}</span>` : ''}</div></div>
    ${fd ? findingsPanel(fd) : ''}
    ${panel('c6', t('ov.dir'), '', '<div class="donut" id="dDir"></div>')}
    ${panel('c6', t('ov.proto'), '', '<div class="donut" id="dProto"></div>')}
    ${panel('c6', t('ov.top_clients'), t('ov.top_clients_sub'), `<table><tr><th></th><th>${t('col.client')}</th><th class="num">${t('col.traffic')}</th><th style="width:34%">${t('col.share')}</th></tr>
      ${cl.map((r, i) => `<tr><td class="rank">${i + 1}</td><td>${ipCell(r.key)}</td><td class="num">${fmtBytes(r.wire)}</td><td>${bar(r.wire, cmax)}</td></tr>`).join('') || `<tr><td colspan="4" class="empty">${t('empty.nodata')}</td></tr>`}</table>`)}
    ${panel('c6', t('ov.top_services'), t('ov.top_services_sub'), `<table><tr><th></th><th>${t('col.server')}</th><th>${t('col.service')}</th><th class="num">${t('col.traffic')}</th><th style="width:24%">${t('col.share')}</th></tr>
      ${sv.map((r, i) => `<tr><td class="rank">${i + 1}</td><td>${ipCell(r.key)}</td><td class="nw">${V('port', r.key2, r.key2)} <span class="muted">${esc(r.extra)}</span></td><td class="num">${fmtBytes(r.wire)}</td><td>${bar(r.wire, smax)}</td></tr>`).join('') || `<tr><td colspan="5" class="empty">${t('empty.nodata')}</td></tr>`}</table>`)}
  </div>`;
  tsChart($('#chStack'), {times: series.times, areas, lines});
  bindFindings(el);
  donut($('#dDir'), dirParts, 'dir');
  donut($('#dProto'), protoParts, 'proto');
};


// ------------------------------------------------------------ findings
// What the detection rules found, as sentences a person can act on.
const SEV = {3: 'high', 2: 'medium', 1: 'low'};
// t() with values that are HTML (buttons for addresses): the text is escaped
// first, then the placeholders are filled.
function tHtml(k, p) {
  let s = esc(t(k));
  for (const [a, b] of Object.entries(p || {})) s = s.split('{' + a + '}').join(b);
  // keep punctuation next to a value on the same line: "203.0.113.200," must
  // not break before the comma
  const B = '<button[^>]*>(?:[^<]|<span[^>]*>[^<]*</span>)*</button>';
  return s.replace(new RegExp(`((?:[(（「“«])*${B}(?:[,，、。.:;)）」”»])+|(?:[(（「“«])+${B})`, 'g'), '<span class="nw">$1</span>');
}
function fmtSpan(f) {
  const a = Date.parse(f.first), b = Date.parse(f.last), min = Math.max(1, Math.round((b - a) / 60000));
  const day = new Date(a).toDateString() !== new Date().toDateString();
  const tf = new Intl.DateTimeFormat(LANG, day ? {month: 'numeric', day: 'numeric', hour: '2-digit', minute: '2-digit'} : {hour: '2-digit', minute: '2-digit'});
  const tt = new Intl.DateTimeFormat(LANG, {hour: '2-digit', minute: '2-digit'});
  const dur = min < 60 ? t('fd.min', {n: nf(min)}) : t('fd.hm', {h: nf(Math.floor(min / 60)), m: nf(min % 60)});
  return `${tf.format(new Date(a))}–${tt.format(new Date(b))} · ${dur}`;
}
const portV = p => p && p !== 'icmp' ? V('port', p, p) : esc(p === 'icmp' ? 'ICMP' : p || '');
function findingText(f) {
  const e = f.ev || {}, src = ipCell(f.src), dst = ipCell(f.dst), port = portV(f.port);
  switch (f.kind) {
    case 'scan': return tHtml('fd.k.scan', {src, port, n: nf(e.targets)});
    case 'portscan': return tHtml('fd.k.portscan', {src, dst, n: nf(e.ports)});
    case 'brute': return tHtml('fd.k.brute', {src, dst, port});
    case 'lateral': return tHtml('fd.k.lateral', {src, port, n: nf(e.hosts)});
    case 'exfil': return tHtml('fd.k.exfil', {src, dst, bytes: esc(fmtBytes(e.up))});
    case 'flood': return tHtml('fd.k.flood', {dst, port, pps: esc(nf(e.pps))});
    case 'threat': return tHtml(e.dir === 2 ? 'fd.k.threat_in' : 'fd.k.threat_out', {src, dst, list: esc(f.port)});
  }
  return esc(f.kind);
}
// The numbers behind a finding, and how the data was sampled.
function findingEvidence(f) {
  const e = f.ev || {}, parts = [];
  const list = xs => (xs || []).slice(0, 4).map(x => f.kind === 'portscan' ? esc(x) : ipCell(x)).join(', ') + ((xs || []).length > 4 ? ' …' : '');
  switch (f.kind) {
    case 'scan': case 'portscan':
      parts.push(esc(t('fd.e.pkts', {n: nf(e.pkts)})), esc(t('fd.e.avg', {n: nf(e.avg)})));
      if ((e.examples || []).length) parts.push(tHtml('fd.e.examples', {list: list(e.examples)}));
      break;
    case 'brute': parts.push(esc(t('fd.e.conns', {n: nf(e.conns)})), esc(t('fd.e.pkts', {n: nf(e.pkts)})), esc(t('fd.e.avg', {n: nf(e.avg)}))); break;
    case 'lateral': parts.push(tHtml('fd.e.targets', {list: list(e.targets)}), esc(t('fd.e.bytes', {n: fmtBytes(e.bytes)}))); break;
    case 'exfil': parts.push(esc(t('fd.e.down', {n: fmtBytes(e.down)}))); if (e.cc || e.org) parts.push(esc([e.cc ? country(e.cc) : '', e.org].filter(Boolean).join(', '))); break;
    case 'flood': parts.push(esc(t('fd.e.sources', {n: nf(e.sources)})), esc(t('fd.e.avg', {n: nf(e.avg)})), esc(t('fd.e.usual', {n: nf(e.usual)}))); break;
    case 'threat': parts.push(esc(t('fd.e.up', {n: fmtBytes(e.up)})), esc(t('fd.e.down', {n: fmtBytes(e.down)}))); break;
  }
  parts.push(esc(e.sampling > 1 ? t('fd.e.sampling', {n: nf(e.sampling)}) : t('fd.e.unsampled')));
  return parts.join(' · ');
}
function findingRows(list, actions, here) {
  return list.map(f => {
    const open = f.status === 'open', who = f.src || f.dst;
    const acts = !actions ? '' : `<td class="acts">${who !== here ? `<button class="btn" data-fd-view="${esc(who)}">${t('fd.view')}</button>` : ''}${open
      ? `<button class="btn" data-fd-set="done" data-id="${f.id}">${t('fd.done')}</button><button class="btn" data-fd-set="false" data-id="${f.id}" title="${esc(t('fd.false_tip'))}">${t('fd.false')}</button>`
      : `<button class="btn" data-fd-set="open" data-id="${f.id}">${t('fd.reopen')}</button>`}</td>`;
    return `<tr class="${open ? '' : 'closed'}"><td><span class="sev s${f.sev}">${t('sev.' + SEV[f.sev])}</span></td>
      <td><div class="kind">${t('fd.kind.' + f.kind)}${open ? '' : `<span class="stl">${t('fd.st_' + f.status)}</span>`}</div><div class="what">${findingText(f)}</div><div class="ev">${findingEvidence(f)}</div></td>
      <td class="when">${esc(fmtSpan(f))}</td>${acts}</tr>`;
  }).join('');
}
function findingCounts(d) {
  const o = d.open || {};
  return ['high', 'medium', 'low'].map(k => `<span class="cnt"><span class="sev s${{high: 3, medium: 2, low: 1}[k]}">${t('sev.' + k)}</span><b>${nf(o[k] || 0)}</b></span>`).join('');
}
// The overview's panel: open findings first, or a line saying there are none.
function findingsPanel(d) {
  const fs = d.findings || [], o = d.open || {}, n = (o.high || 0) + (o.medium || 0) + (o.low || 0);
  const body = fs.length
    ? `<table class="fd">${findingRows(fs, false)}</table>${n > fs.length ? `<p class="fdnote"><button class="btn" data-go="findings">${t('ov.findings_all', {n: nf(n)})}</button></p>` : ''}`
    : `<div style="padding:4px 0">${status('ok', t('fd.none_open'))}</div>`;
  return `<div class="panel c12"><div class="ph"><h2>${t('nav.findings')}</h2><div class="fdhead" style="margin-inline-start:auto">${findingCounts(d)}</div></div>${body}</div>`;
}
function bindFindings(el) {
  el.querySelectorAll('[data-go]').forEach(b => b.onclick = () => go(b.dataset.go));
  el.querySelectorAll('[data-fd-view]').forEach(b => b.onclick = () => showDetail('ip', b.dataset.fdView));
  el.querySelectorAll('[data-fd-set]').forEach(b => b.onclick = async () => {
    const res = await fetch('/api/findings' + (inSB() ? '?ds=sb' : ''), {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({ids: [+b.dataset.id], status: b.dataset.fdSet})});
    if (!res.ok) { const j = await res.json().catch(() => ({})); toast(j.error || res.statusText); return; }
    loadStatus(); render();
  });
}
views.findings = async (el) => {
  const all = state.fd === 'all';
  const d = await api('findings', {status: all ? 'all' : 'open'});
  const fs = d.findings || [], det = d.detector || {};
  const learning = det.learning_until && !det.learning_until.startsWith('0001') ? Date.parse(det.learning_until) : 0;
  el.innerHTML = `<div class="grid"><div class="panel c12">
    <div class="ph"><div class="fdhead">${findingCounts(d)}</div>
      <div class="tools" style="margin-inline-start:auto;display:flex;gap:12px;align-items:center">${det.last_run && !det.last_run.startsWith('0001') ? `<span class="muted" style="font-size:12.5px">${esc(t('fd.last_run', {t: ago(Date.parse(det.last_run))}))}</span>` : ''}
      <div class="seg" role="group"><button data-fd="open" aria-pressed="${!all}">${t('fd.open')}</button><button data-fd="all" aria-pressed="${all}">${t('fd.all')}</button></div></div></div>
    ${learning > Date.now() ? `<div class="verdict warn" style="margin:0 0 10px">${esc(t('fd.learning', {t: new Intl.DateTimeFormat(LANG, {dateStyle: 'medium', timeStyle: 'short'}).format(new Date(learning))}))}</div>` : ''}
    ${fs.length ? `<table class="fd">${findingRows(fs, true)}</table>` : `<div class="empty">${all ? t('fd.none') : status('ok', t('fd.none_open'))}</div>`}
    <p class="fdnote">${esc(t('fd.note'))}</p></div></div>`;
  el.querySelectorAll('[data-fd]').forEach(b => b.onclick = () => { state.fd = b.dataset.fd; render(); });
  bindFindings(el);
};


// ------------------------------------------------------------ series charts
// One measure over time, split by a dimension: the top 8 values in the
// fixed colours, the rest as Other. The legend names each value with its
// total and opens the usual menu.
const ifaceNames = new Map(); // "exporter/ifindex" -> "name · device"
function seriesLabel(by, k) {
  if (k === '__other__') return t('other');
  if (by === 'client' || by === 'server') return names.get(k) || k;
  if (by === 'service') { const [p, a] = k.split('\t'); return a ? p + ' ' + a : p; }
  if (by.startsWith('asn')) { const [n, o] = k.split('\t'); return 'AS' + n + (o ? ' ' + o : ''); }
  if (by.startsWith('if_')) return ifaceNames.get(k) || k;
  return appLabel(k);
}
function seriesValue(by, k) {
  if (by === 'client' || by === 'server') return [by, k];
  if (by === 'service') return ['port', k.split('\t')[0]];
  if (by.startsWith('asn')) return ['asn', k.split('\t')[0]];
  if (by.startsWith('if_')) return ['iface', k];
  return ['app', k];
}
const seriesPanel = (id, cls, title, sub = '') => `<div class="panel ${cls}" id="${id}"><div class="ph"><h2>${title}</h2><span class="sub">${sub}</span></div>
  <div class="chart" style="height:200px" aria-label="${esc(title)}"></div><div class="legend"></div></div>`;
function fillSeries(id, d, by, measure) {
  const el = document.getElementById(id); if (!el || !d) return;
  const pk = measure === 'pkts';
  if (!d.names.length) { el.querySelector('.chart').innerHTML = `<div class="empty">${t('empty.nodata')}</div>`; return; }
  // with thousands of hosts the rest dwarfs the top 8: draw only those and
  // give the rest's total in the legend
  const hideOther = by === 'client' || by === 'server';
  const all = d.names.map((n, i) => ({n, name: seriesLabel(by, n), color: color(i, n), data: d.values[i], tot: d.totals?.[i]}));
  const areas = all.filter(a => !(hideOther && a.n === '__other__'));
  tsChart(el.querySelector('.chart'), {times: d.times, areas, fmtV: pk ? fmtPps : fmtBps});
  el.querySelector('.legend').innerHTML = all.map(a => {
    const [f, v] = seriesValue(by, a.n), tot = a.tot == null ? '' : pk ? nf(a.tot) : fmtBytes(a.tot);
    if (a.n === '__other__') return `<span>${hideOther ? '' : `<i style="background:${a.color}"></i>`}${esc(hideOther ? t('ch.rest') : a.name)}<span class="muted">${tot}</span></span>`;
    return `<span><i style="background:${a.color}"></i>${V(f, v, a.name)}<span class="muted">${tot}</span></span>`;
  }).join('');
}
async function seriesData(by, measure, filters) {
  const d = await api('series', {by, measure, top: 8}, filters);
  if (by === 'client' || by === 'server') await lookupNames(d.names.filter(n => n !== '__other__'));
  return d;
}
// Bars over time (flow record counts).
function barChart(el, times, values, fmtV) {
  const draw = () => {
    const W = el.clientWidth, H = el.clientHeight; if (W < 120 || !times.length) { el.innerHTML = ''; return; }
    const rtl = document.documentElement.dir === 'rtl', L = rtl ? 8 : 50, R = rtl ? 50 : 8, T = 8, B = 22, n = times.length;
    const Y = niceMax(Math.max(...values) * 1.05), y = v => T + (H - T - B) * (1 - v / Y);
    const bw = (W - L - R) / n, x = i => rtl ? W - R - bw * (i + 1) : L + bw * i, span = times[n - 1] - times[0];
    let s = `<svg viewBox="0 0 ${W} ${H}" height="${H}" role="img" aria-label="${esc(el.getAttribute('aria-label') || '')}">`;
    for (let k = 0; k <= 4; k++) { const yy = y(Y * k / 4); s += `<line class="gl" x1="${L}" x2="${W - R}" y1="${yy}" y2="${yy}"/><text class="ax" x="${rtl ? W - R + 8 : L - 8}" y="${yy + 4}" text-anchor="${rtl ? 'start' : 'end'}">${fmtAxisBps(Y * k / 4)}</text>`; }
    const ticks = W < 600 ? 4 : 7, step = Math.max(1, Math.round(n / ticks));
    for (let i = 0; i < n; i += step) s += `<text class="ax" x="${x(i) + bw / 2}" y="${H - 5}" text-anchor="middle">${fmtAxisTime(times[i], span)}</text>`;
    values.forEach((v, i) => { if (v > 0) s += `<rect x="${(x(i) + 1).toFixed(1)}" y="${y(v).toFixed(1)}" width="${Math.max(1, bw - 2).toFixed(1)}" height="${(H - B - y(v)).toFixed(1)}" rx="${Math.min(2, bw / 4)}" fill="var(--c1)"/>`; });
    s += `<rect class="hit" x="${Math.min(L, R)}" y="${T}" width="${W - L - R}" height="${H - T - B}" fill="transparent"/></svg>`;
    el.innerHTML = s;
    const svg = el.querySelector('svg');
    svg.querySelector('.hit').onmousemove = e => {
      const r = svg.getBoundingClientRect(); let fx = (e.clientX - r.left - L) / (W - L - R); if (rtl) fx = 1 - fx;
      const i = Math.max(0, Math.min(n - 1, Math.floor(fx * n)));
      showTip(e, `<div class="t">${fmtTime(times[i], span)}</div><b>${fmtV(values[i])}</b>`);
    };
    svg.querySelector('.hit').onmouseleave = hideTip;
  };
  el._draw = draw; draw();
  if (!el._ro) { el._ro = new ResizeObserver(() => el._draw && el._draw()); el._ro.observe(el); }
}

// Traffic details: clients, servers and services over time, in bits and
// packets per second.
// Traffic details: two two-ring charts. Servers and their clients (or,
// swapped, clients and the servers they use), and services and their
// servers (or servers and their services). Seen from either side, like
// pull and push.
const RING_FIELD = {server: 'server', client: 'client', service: 'port'};
const ringVal = (dim, k) => dim === 'service' ? k.split('\t')[0] : k;
views.traffic = async (el) => {
  const ra = state.ra === 'client' ? ['client', 'server'] : ['server', 'client'];
  const rb = state.rb === 'server' ? ['server', 'service'] : ['service', 'server'];
  const [da, db] = await Promise.all([api('rings', {inner: ra[0], outer: ra[1]}), api('rings', {inner: rb[0], outer: rb[1]})]);
  await lookupNames([...da.inner, ...da.outer, ...db.inner, ...db.outer].map(r => r.k).filter(k => /^[0-9a-f.:]+$/i.test(k)));
  const swap = (id, cur, opts) => `<div class="seg" role="group">${opts.map(([v, lab]) => `<button data-${id}="${v}" aria-pressed="${cur === v}">${esc(lab)}</button>`).join('')}</div>`;
  el.innerHTML = `<div class="grid">
    <div class="panel c6"><div class="ph"><h2>${t('rg.hosts')}</h2><div class="tools" style="margin-inline-start:auto">${swap('ra', ra[0], [['server', t('rg.in_server')], ['client', t('rg.in_client')]])}</div></div>
      <div class="sub" style="margin:-4px 0 6px">${esc(t(ra[0] === 'server' ? 'rg.hosts_sub_s' : 'rg.hosts_sub_c'))}</div><div class="rings" id="rgA"></div></div>
    <div class="panel c6"><div class="ph"><h2>${t('rg.services')}</h2><div class="tools" style="margin-inline-start:auto">${swap('rb', rb[0], [['service', t('rg.in_service')], ['server', t('rg.in_server')]])}</div></div>
      <div class="sub" style="margin:-4px 0 6px">${esc(t(rb[0] === 'service' ? 'rg.svc_sub_v' : 'rg.svc_sub_s'))}</div><div class="rings" id="rgB"></div></div></div>`;
  el.querySelectorAll('[data-ra]').forEach(b => b.onclick = () => { state.ra = b.dataset.ra; render(); });
  el.querySelectorAll('[data-rb]').forEach(b => b.onclick = () => { state.rb = b.dataset.rb; render(); });
  drawRings($('#rgA'), da, ra[0], ra[1]);
  drawRings($('#rgB'), db, rb[0], rb[1]);
};
function drawRings(el, d, inDim, outDim) {
  if (!d.total) { el.innerHTML = `<div class="empty">${t('empty.nodata')}</div>`; return; }
  const draw = () => {
    const W = el.clientWidth; if (W < 200) return;
    // room for the outer labels on both sides
    const SW = Math.min(W, 600), R = Math.max(80, Math.min(170, SW / 2 - 120)), SH = 2 * R + 60, c = SW / 2, cy = SH / 2, r0 = R * .38, r1 = R * .68;
    const size = SW;
    const pt = (r, a) => [c + r * Math.sin(a), cy - r * Math.cos(a)];
    const arc = (ri, ro, a0, a1) => {
      a1 = Math.min(a1, a0 + Math.PI * 2 - 1e-4);
      const big = a1 - a0 > Math.PI ? 1 : 0, [x0, y0] = pt(ro, a0), [x1, y1] = pt(ro, a1), [x2, y2] = pt(ri, a1), [x3, y3] = pt(ri, a0);
      return `M${x0.toFixed(2)} ${y0.toFixed(2)}A${ro} ${ro} 0 ${big} 1 ${x1.toFixed(2)} ${y1.toFixed(2)}L${x2.toFixed(2)} ${y2.toFixed(2)}A${ri} ${ri} 0 ${big} 0 ${x3.toFixed(2)} ${y3.toFixed(2)}Z`;
    };
    const short = (v, n) => { const a = [...v]; return a.length > n ? a.slice(0, n - 1).join('') + '…' : v; };
    const col = {}, tot = d.total;
    d.inner.forEach((r, i) => col[r.k] = r.k === '__other__' ? 'var(--other)' : color(i, r.k));
    let s = `<svg viewBox="0 0 ${SW} ${SH}" width="${SW}" height="${SH}" role="img">`, a = 0, labels = '';
    const lastY = {l: -1e9, r: -1e9};
    const segs = [];
    d.inner.forEach(r => {
      const a1 = a + r.v / tot * Math.PI * 2;
      segs.push({ring: 0, k: r.k, p: '', v: r.v, a0: a, a1});
      let b = a;
      d.outer.filter(o => o.p === r.k).forEach((o, j) => { const b1 = b + o.v / tot * Math.PI * 2; segs.push({ring: 1, k: o.k, p: r.k, v: o.v, a0: b, a1: b1, j}); b = b1; });
      a = a1;
    });
    segs.forEach((g, n) => {
      const fill = col[g.ring ? g.p : g.k], op = g.ring ? (g.k === '__other__' ? .28 : (g.j % 2 ? .55 : .75)) : 1;
      s += `<path d="${arc(g.ring ? r1 : r0, g.ring ? R : r1, g.a0, g.a1)}" fill="${fill}" fill-opacity="${op}" stroke="var(--surface)" stroke-width="1" data-n="${n}"/>`;
      const mid = (g.a0 + g.a1) / 2, span = g.a1 - g.a0;
      if (!g.ring && span > .45 && g.k !== '__other__') {
        const [x, y] = pt((r0 + r1) / 2, mid);
        labels += `<text x="${x}" y="${y}" text-anchor="middle" dominant-baseline="middle" class="rgin">${esc(short(seriesLabel(inDim, g.k), 11))}</text>`;
      } else if (g.ring && span > .06 && g.k !== '__other__') {
        // outer labels go down each side; one that would overlap the previous is left to the tooltip
        const [x0, y0] = pt(R + 2, mid), [x1, y1] = pt(R + 14, mid), right = Math.sin(mid) >= 0, side = right ? 'r' : 'l';
        if (Math.abs(y1 - lastY[side]) >= 13) {
          lastY[side] = y1;
          const x2 = x1 + (right ? 6 : -6);
          labels += `<path d="M${x0} ${y0}L${x1} ${y1}L${x2} ${y1}" class="rgline"/><text x="${x2 + (right ? 3 : -3)}" y="${y1}" text-anchor="${right ? 'start' : 'end'}" dominant-baseline="middle" class="rgout">${esc(short(seriesLabel(outDim, g.k), 16))}</text>`;
        }
      }
    });
    s += labels + `<text x="${c}" y="${cy - 4}" text-anchor="middle" class="rgtot">${fmtBytes(tot)}</text><text x="${c}" y="${cy + 14}" text-anchor="middle" class="rgsub">${esc(t('col.traffic'))}</text></svg>`;
    const legend = d.inner.map(r => `<div class="rgleg"><i style="background:${col[r.k]}"></i>${r.k === '__other__' ? `<span>${t('other')}</span>` : V(RING_FIELD[inDim], ringVal(inDim, r.k), seriesLabel(inDim, r.k))}<b>${fmtBytes(r.v)}</b><span class="muted">${nf(100 * r.v / tot, 1)}%</span></div>`).join('');
    el.innerHTML = `<div class="rgwrap">${s}<div class="rglegs">${legend}</div></div>`;
    el.querySelectorAll('path[data-n]').forEach(p => {
      const g = segs[+p.dataset.n];
      p.onmousemove = e => showTip(e, g.ring
        ? `<div class="t">${esc(seriesLabel(inDim, g.p))} → ${esc(seriesLabel(outDim, g.k))}</div><b>${fmtBytes(g.v)}</b> <span class="muted">${nf(100 * g.v / tot, 1)}%</span>`
        : `<div class="t">${esc(seriesLabel(inDim, g.k))}</div><b>${fmtBytes(g.v)}</b> <span class="muted">${nf(100 * g.v / tot, 1)}%</span>`);
      p.onmouseleave = hideTip;
      const dim = g.ring ? outDim : inDim;
      if (g.k !== '__other__') p.onclick = e => { e.stopPropagation(); hideTip(); openPop({k: RING_FIELD[dim], v: ringVal(dim, g.k), label: seriesLabel(dim, g.k)}, {left: e.clientX, right: e.clientX, top: e.clientY, bottom: e.clientY}); };
    });
  };
  draw();
  if (!el._ro) { el._ro = new ResizeObserver(() => draw()); el._ro.observe(el); }
}

const DIMS = ['conv', 'client', 'server', 'app', 'port', 'country', 'asn', 'segment', 'exporter', 'encap', 'vlan'];
// Top-N is one table. It starts with conversations (client, server,
// service, country); the select switches the grouping. Every column heading
// sorts: number columns rank on the server, so "by packets" is the real top
// by packets and not the top by traffic re-ordered; text columns sort the
// rows shown.
const NUMK = ['wire', 'pkts', 'avg', 'flows', 'peers'];
const TEXT_OF = {client: ['key'], server: ['key', 'extra'], conv: ['key', 'key2', 'key3', 'extra'], app: ['key'], port: ['key', 'extra'],
  country: ['key', ''], asn: ['key', 'extra'], segment: ['key'], exporter: ['key'], encap: ['key'], vlan: ['key']};
// The Top 66 page opens on "Talkers": traffic by service over time, and the
// top 30 clients and servers side by side with bytes, packets and flow
// records, over a total row for all traffic. "Table" is the single
// regroupable table of the top 66.
const modeSeg = () => `<div class="seg" role="group"><button data-tm="talkers" aria-pressed="${state.tm !== 'table'}">${t('topn.talkers')}</button><button data-tm="table" aria-pressed="${state.tm === 'table'}">${t('topn.table')}</button></div>`;
function bindMode(el) { el.querySelectorAll('[data-tm]').forEach(b => b.onclick = () => { state.tm = b.dataset.tm; render(); }); }
views.topn = el => {
  if (!DIMS.includes(state.dim)) state.dim = 'conv';
  return state.tm === 'table' ? topTable(el, state.dim) : talkers(el);
};
async function talkers(el) {
  const [cl, srv, ov] = await Promise.all([api('topn', {dim: 'client', limit: 30}), api('topn', {dim: 'server', limit: 30}), api('overview')]);
  const tot = ov.totals || {};
  const table = (rows, head, f) => `<table class="talk"><tr><th></th><th>${t(head)}</th><th class="num">${t('col.traffic')}</th><th class="num">${t('col.pkts')}</th><th class="num">${t('col.flows')}</th></tr>
    ${(rows || []).map((r, i) => `<tr><td class="rank">${i + 1}</td><td>${ipCell(r.key, f)}</td><td class="num">${fmtBytes(r.wire)}</td><td class="num">${nf(r.pkts)}</td><td class="num">${nf(r.flows)}</td></tr>`).join('') || `<tr><td colspan="5" class="empty">${t('empty.nodata')}</td></tr>`}
    <tr class="sum"><td></td><td>${t('topn.all')}</td><td class="num">${fmtBytes(tot.wire)}</td><td class="num">${nf(tot.pkts)}</td><td class="num">${nf(tot.flows)}</td></tr></table>`;
  el.innerHTML = `<div class="grid">
    <div class="c12"><div style="display:inline-block">${modeSeg()}</div></div>
    ${panel('c6', t('ov.top_clients'), esc(t('topn.top_n', {n: 30})), table(cl.rows, 'col.client', 'client'))}
    ${panel('c6', t('topn.top_servers'), esc(t('topn.top_n', {n: 30})), table(srv.rows, 'col.server', 'server'))}
  </div>`;
  bindMode(el);
}
async function topTable(el, dim) {
  const sk = state.sort.k, asc = state.sort.asc, num = NUMK.includes(sk);
  const d = await api('topn', num ? {dim, limit: 66, by: sk, asc: asc ? 1 : 0} : {dim, limit: 66});
  let rows = d.rows || [];
  const max = Math.max(1, ...rows.map(r => r.wire)), total = rows.reduce((s, r) => s + r.wire, 0);
  if (!num) {
    const f = TEXT_OF[dim][+sk.slice(1)] || 'key';
    rows.sort((a, b) => (asc ? 1 : -1) * String(a[f] || '').localeCompare(String(b[f] || ''), undefined, {numeric: true}));
  }
  const avg = r => r.pkts ? r.wire / r.pkts : 0;
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
  // peer counts exist only for clients and servers over the detail data
  const showPeers = ['client', 'server'].includes(dim) && (rows.some(r => r.peers) || sk === 'peers');
  const sortTh = (k, label, cls = 'num') => {
    if (!label) return '<th></th>';
    const on = sk === k, ar = on ? (asc ? '▲' : '▼') : '';
    return `<th class="${cls} sk" data-s="${k}" ${on ? `aria-sort="${asc ? 'ascending' : 'descending'}"` : ''}>${label}<span class="ar">${ar}</span></th>`;
  };
  const slow = dim === 'conv' && spanMs() > 216e5 ? ` · ${esc(t('conv.slow'))}` : '';
  el.innerHTML = `<div class="panel">
    <div class="ph"><h2>${t('topn.title', {n: 66})}</h2><span class="sub">${esc(t('topn.hint'))}${slow}</span>
      <div class="tools" style="display:flex;gap:12px;align-items:center">${modeSeg()}<label>${t('topn.group')} <select class="dimsel" id="dimSel">${DIMS.map(k => `<option value="${k}" ${k === dim ? 'selected' : ''}>${t('dim.' + k)}</option>`).join('')}</select></label></div></div>
    <table><tr><th></th>${heads.map((h, i) => sortTh('t' + i, h && t(h), '')).join('')}${sortTh('wire', t('col.traffic'))}<th class="num">%</th><th style="width:16%">${t('col.share')}</th>${sortTh('pkts', t('col.pkts'))}${sortTh('avg', t('col.avg'))}${showPeers ? sortTh('peers', t('col.peers')) : ''}${sortTh('flows', t('col.flows'))}</tr>
    ${rows.map((r, i) => `<tr><td class="rank">${i + 1}</td>${cell(r).map(c => `<td>${c}</td>`).join('')}<td class="num">${fmtBytes(r.wire)}</td><td class="num muted">${nf(r.wire / Math.max(1, total) * 100, 1)}</td><td>${bar(r.wire, max)}</td><td class="num">${nf(r.pkts)}</td><td class="num">${nf(avg(r))}&nbsp;B</td>${showPeers ? `<td class="num">${nf(r.peers)}</td>` : ''}<td class="num">${nf(r.flows)}</td></tr>`).join('') || `<tr><td colspan="11" class="empty">${t('empty.nodata')}</td></tr>`}
    </table></div>`;
  $('#dimSel').onchange = e => { state.dim = e.target.value; state.sort = {k: 'wire', asc: false}; render(); };
  bindMode(el);
  el.querySelectorAll('th[data-s]').forEach(h => h.onclick = () => {
    const k = h.dataset.s;
    // a new column starts at its interesting end: largest first, except the
    // average packet size (small packets: scans, floods) and text (A to Z)
    state.sort = state.sort.k === k ? {k, asc: !state.sort.asc} : {k, asc: k === 'avg' || !NUMK.includes(k)};
    render();
  });
}

// Flow paths start from each internal host (default) or from each network
// segment, through the application to the remote country.
async function lookupNames(ips) {
  const ask = ips.filter(ip => ip && !names.has(ip) && !asked.has(ip));
  if (!ask.length) return;
  ask.forEach(ip => asked.add(ip));
  try {
    const res = await fetch('/api/resolve', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({ips: ask})});
    if (res.ok) for (const [ip, n] of Object.entries(await res.json())) names.set(ip, n);
  } catch (e) {}
}
views.sankey = async (el) => {
  // host: host → application → country; segment: network → application →
  // country; conv: client → service → server
  const mode = ['segment', 'conv'].includes(state.sk) ? state.sk : 'host', byHost = mode === 'host', conv = mode === 'conv';
  const d = await api('sankey', {by: mode});
  const ipCols = conv ? [0, 2] : byHost ? [0] : [];
  await lookupNames([...(d.seg_app || []).map(l => l.S), ...(conv ? (d.app_cc || []).map(l => l.T) : [])].filter(k => k !== '__other__'));
  const title = t({host: 'sankey.title_host', segment: 'sankey.title', conv: 'sankey.title_conv'}[mode]);
  el.innerHTML = `<div class="panel"><div class="ph"><h2>${title}</h2><span class="sub">${t('sankey.hint')}</span>
      <div class="tools" style="margin-inline-start:auto"><div class="seg" role="group">${['host', 'conv', 'segment'].map(m => `<button data-sk="${m}" aria-pressed="${m === mode}">${t({host: 'sankey.by_host', conv: 'sankey.by_conv', segment: 'sankey.by_seg'}[m])}</button>`).join('')}</div></div></div>
    <div class="chart sankey" id="chSankey" style="height:${mode === 'segment' ? 440 : 520}px" aria-label="${esc(title)}"></div></div>`;
  el.querySelectorAll('[data-sk]').forEach(b => b.onclick = () => { state.sk = b.dataset.sk; render(); });
  const sEl = $('#chSankey');
  if (!(d.seg_app || []).length) { sEl.innerHTML = `<div class="empty">${t('empty.nodata')}</div>`; return; }
  const cols = [[], [], []], ids = [{}, {}, {}];
  const node = (c, k) => { if (!(k in ids[c])) { ids[c][k] = cols[c].length; cols[c].push({k, in: 0, out: 0}); } return cols[c][ids[c][k]]; };
  d.seg_app.forEach(l => { node(0, l.S).out += l.V; node(1, l.T).in += l.V; });
  d.app_cc.forEach(l => { node(1, l.S).out += l.V; node(2, l.T).in += l.V; });
  cols.forEach(c => c.sort((a, b) => (a.k === '__other__') - (b.k === '__other__') || Math.max(b.in, b.out) - Math.max(a.in, a.out)));
  const label = (c, k) => k === '__other__' ? t('other') : ipCols.includes(c) ? (names.get(k) || k) : c === 2 ? country(k) : appLabel(k);
  // long host names are cut to 22 characters; the full name shows on hover
  const short = v => { const a = [...v]; return a.length > 22 ? a.slice(0, 21).join('') + '…' : v; };
  const field = {host: ['ip', 'app', 'country'], segment: ['segment', 'app', 'country'], conv: ['client', 'port', 'server']}[mode];
  const draw = () => {
    const W = sEl.clientWidth, H = sEl.clientHeight; if (W < 300) return;
    const rtl = document.documentElement.dir === 'rtl';
    const nodeW = 12, pad = 14, lw = mode === 'segment' ? 118 : 200, rw = conv ? 200 : 130, colX = rtl ? [W - lw - 12, Math.round(W / 2 - 6), rw - 12] : [lw, Math.round(W / 2 - 6), W - rw];
    const nv = n => Math.max(n.in, n.out);
    const k = Math.min(...cols.map(c => (H - pad * (c.length - 1)) / Math.max(1, c.reduce((s, n) => s + nv(n), 0))));
    cols.forEach((c, ci) => { let yy = 0; c.forEach((n, i) => { n.ci = ci; n.x = colX[ci]; n.y = yy; n.h = nv(n) * k; n.o = 0; n.i = 0; n.color = ci === 0 ? color(i, n.k) : 'var(--ink-2)'; yy += n.h + pad; }); });
    let s = `<svg viewBox="0 0 ${W} ${H}" height="${H}" role="img" aria-label="${esc(title)}">`;
    const link = (A, B, v, col, lab) => {
      const w = v * k, y0 = A.y + A.o + w / 2, y1 = B.y + B.i + w / 2; A.o += w; B.i += w;
      const x0 = rtl ? A.x : A.x + nodeW, x1 = rtl ? B.x + nodeW : B.x, mx = (x0 + x1) / 2;
      return `<path d="M${x0} ${y0}C${mx} ${y0} ${mx} ${y1} ${x1} ${y1}" fill="none" stroke="${col}" stroke-opacity=".3" stroke-width="${Math.max(1, w - 1)}" data-l="${esc(lab)}" data-v="${v}" data-c="${B.ci}" data-k="${esc(B.k)}"/>`;
    };
    const segColor = {}; cols[0].forEach(n => segColor[n.k] = n.color);
    [...d.seg_app].sort((a, b) => ids[0][a.S] - ids[0][b.S] || ids[1][a.T] - ids[1][b.T]).forEach(l => s += link(cols[0][ids[0][l.S]], cols[1][ids[1][l.T]], l.V, segColor[l.S], label(0, l.S) + ' → ' + label(1, l.T)));
    [...d.app_cc].sort((a, b) => ids[1][a.S] - ids[1][b.S] || ids[2][a.T] - ids[2][b.T]).forEach(l => s += link(cols[1][ids[1][l.S]], cols[2][ids[2][l.T]], l.V, 'var(--ink-3)', label(1, l.S) + ' → ' + label(2, l.T)));
    cols.forEach((c, ci) => c.forEach(n => {
      const dk = `data-c="${ci}" data-k="${esc(n.k)}"`;
      s += `<rect x="${n.x}" y="${n.y}" width="${nodeW}" height="${Math.max(2, n.h)}" rx="2" fill="${n.color}" ${dk}><title>${esc(label(ci, n.k))} ${fmtBytes(nv(n))}</title></rect>`;
      s += `<rect class="hit" x="${n.x - 8}" y="${n.y - 2}" width="${nodeW + 16}" height="${Math.max(6, n.h + 4)}" ${dk}/>`;
      const right = (ci === 2) !== rtl, tx = right ? n.x + nodeW + 8 : n.x - 8, anc = right ? 'start' : 'end';
      const full = `<title>${esc(label(ci, n.k))} · ${fmtBytes(nv(n))}</title>`;
      if (n.h >= 26) {
        s += `<text x="${tx}" y="${n.y + n.h / 2 - 6}" text-anchor="${anc}" dominant-baseline="middle" ${dk}>${full}${esc(short(label(ci, n.k)))}</text>`;
        s += `<text class="nv" x="${tx}" y="${n.y + n.h / 2 + 9}" text-anchor="${anc}" dominant-baseline="middle" ${dk}>${fmtBytes(nv(n))}</text>`;
      } else if (n.h >= 9 || ci !== 1) {
        s += `<text x="${tx}" y="${n.y + n.h / 2}" text-anchor="${anc}" dominant-baseline="middle" ${dk}>${full}${esc(short(label(ci, n.k)))} <tspan class="nv">${fmtBytes(nv(n))}</tspan></text>`;
      }
    }));
    sEl.innerHTML = s + '</svg>';
    sEl.querySelectorAll('path[data-l]').forEach(p => { p.onmousemove = e => showTip(e, `<div class="t">${esc(p.dataset.l)}</div><b>${fmtBytes(+p.dataset.v)}</b>`); p.onmouseleave = hideTip; });
    sEl.querySelectorAll('[data-k]').forEach(r => r.onclick = e => {
      e.stopPropagation(); const ci = +r.dataset.c, kk = r.dataset.k; if (kk === '__other__') return;
      openPop({k: field[ci], v: kk, label: label(ci, kk)}, r.getBoundingClientRect());
    });
  };
  draw();
  if (!sEl._ro) { sEl._ro = new ResizeObserver(draw); sEl._ro.observe(sEl); }
};

// ------------------------------------------------------------ world map
let WORLD;
const worldMap = () => WORLD || (WORLD = fetch('world.json').then(r => r.json()).catch(e => { WORLD = null; throw e; }));
// five steps of the sequential blue, light to dark; no traffic is grey
const MAP_STEPS = ['#cde2fb', '#86b6ef', '#3987e5', '#1c5cab', '#0d366b'];
function mapBins(vals) {
  const v = vals.filter(x => x > 0).sort((a, b) => a - b);
  if (!v.length) return [];
  // log-spaced edges between the smallest and largest country
  const lo = Math.log10(v[0]), hi = Math.log10(v[v.length - 1]);
  return MAP_STEPS.slice(1).map((_, i) => Math.pow(10, lo + (hi - lo) * (i + 1) / MAP_STEPS.length));
}
const mapStep = (v, edges) => { let i = 0; while (i < edges.length && v >= edges[i]) i++; return i; };
function drawMap(el, w, rows, gl) {
  const by = new Map(rows.map(r => [r.key, r.wire]));
  const edges = mapBins(rows.map(r => r.wire));
  const fill = cc => by.get(cc) > 0 ? MAP_STEPS[mapStep(by.get(cc), edges)] : 'var(--map-none)';
  const dots = Object.entries(w.s).filter(([cc]) => by.get(cc) > 0);
  el.innerHTML = `<svg viewBox="0 0 ${w.w} ${w.h}" class="wmap" role="img" aria-label="${esc(t('geo.map'))}">
    ${Object.entries(w.c).map(([cc, d]) => `<path d="${d}" data-cc="${cc}" fill="${fill(cc)}"/>`).join('')}
    ${dots.map(([cc, [x, y]]) => `<circle cx="${x}" cy="${y}" r="4" data-cc="${cc}" fill="${fill(cc)}"/>`).join('')}${mapLines(w, gl)}</svg>
    <div class="maplegend">${edges.length ? `<span class="muted">${t('geo.map_less')}</span>${MAP_STEPS.map((c, i) => `<i style="background:${c}" title="${esc(i ? '≥ ' + fmtBytes(edges[i - 1]) : '< ' + fmtBytes(edges[0]))}"></i>`).join('')}<span class="muted">${t('geo.map_more')}</span>
      <span class="muted" style="margin-inline-start:12px">${esc(fmtBytes(Math.min(...rows.map(r => r.wire).filter(x => x > 0))))} – ${esc(fmtBytes(Math.max(...rows.map(r => r.wire))))}</span>` : ''}
      <span style="margin-inline-start:12px"><i style="background:var(--map-none)"></i> <span class="muted">${t('geo.map_none')}</span></span></div>`;
  const total = rows.reduce((a, r) => a + r.wire, 0) || 1;
  el.querySelectorAll('[data-cc]').forEach(p => {
    const cc = p.dataset.cc, v = by.get(cc) || 0;
    p.onmousemove = e => showTip(e, `<div class="t">${esc(country(cc))}</div><b>${v ? fmtBytes(v) : t('geo.map_none')}</b>${v ? ` <span class="muted">${nf(100 * v / total, 1)}%</span>` : ''}`);
    p.onmouseleave = hideTip;
    if (v) p.onclick = e => { e.stopPropagation(); hideTip(); openPop({k: 'country', v: cc, label: country(cc)}, {left: e.clientX, right: e.clientX, top: e.clientY, bottom: e.clientY}); };
    else p.style.cursor = 'default';
  });
  el.querySelectorAll('.mlines path').forEach(p => {
    p.onmousemove = e => showTip(e, `<div class="t">${esc(country(p.dataset.from))} ↔ ${esc(country(p.dataset.to))}</div><b>${fmtBytes(+p.dataset.v)}</b>`);
    p.onmouseleave = hideTip;
  });
  if (gl && !(gl.origins || []).length) el.insertAdjacentHTML('beforeend', `<p class="muted maphint">${esc(t('geo.lines_hint'))}</p>`);
}
// Lines from the countries of the internal networks to the remote
// countries, thicker for more traffic.
function mapLines(w, gl) {
  const lines = (gl && gl.lines || []).filter(l => w.p[l.from] && w.p[l.to]).slice(0, 60);
  if (!lines.length) return '';
  const max = Math.max(...lines.map(l => l.v)), min = Math.min(...lines.map(l => l.v));
  const k = v => max === min ? 1 : Math.log(v / min) / Math.log(max / min);
  let s = '<g class="mlines">';
  [...lines].reverse().forEach(l => {
    const [x0, y0] = w.p[l.from], [x1, y1] = w.p[l.to], dx = x1 - x0, dy = y1 - y0, dist = Math.hypot(dx, dy);
    // bend each line upwards by a fifth of its length
    const mx = (x0 + x1) / 2 + dy * .2 * Math.sign(dx || 1) * -1, my = (y0 + y1) / 2 - Math.abs(dx) * .2 - dist * .05;
    const f = k(l.v);
    s += `<path d="M${x0} ${y0}Q${mx.toFixed(1)} ${my.toFixed(1)} ${x1} ${y1}" stroke-width="${(2 + 7 * f).toFixed(2)}" stroke-opacity="${(0.55 + 0.4 * f).toFixed(2)}" data-from="${l.from}" data-to="${l.to}" data-v="${l.v}"/>`;
    s += `<circle cx="${x1}" cy="${y1}" r="${(2.5 + 2.5 * f).toFixed(1)}" class="mend"/>`;
  });
  (gl.origins || []).filter(cc => w.p[cc]).forEach(cc => { const [x, y] = w.p[cc]; s += `<circle cx="${x}" cy="${y}" r="9" class="morigin-ring"/><circle cx="${x}" cy="${y}" r="5" class="morigin"/>`; });
  return s + '</g>';
}
const ATTRIB = {
  dbip: '<a href="https://db-ip.com" target="_blank" rel="noopener">IP Geolocation by DB-IP</a>',
  maxmind: 'This product includes GeoLite2 data created by MaxMind, available from <a href="https://www.maxmind.com" target="_blank" rel="noopener">https://www.maxmind.com</a>',
  ipinfo: '<a href="https://ipinfo.io" target="_blank" rel="noopener">IP address data powered by IPinfo</a>',
  iptoasn: '<a href="https://iptoasn.com" target="_blank" rel="noopener">IPtoASN</a>'
};
const attribution = srcs => [...new Set((srcs || []).map(g => g.vendor).filter(v => ATTRIB[v]))].map(v => ATTRIB[v]).join(' · ');

views.geo = async (el) => {
  const asCharts = [['asn_src', 'wire'], ['asn_dst', 'wire'], ['asn_src', 'pkts'], ['asn_dst', 'pkts']];
  const [c, a, gd, w, gl, ...ad] = await Promise.all([api('topn', {dim: 'country', limit: 300}), api('topn', {dim: 'asn', limit: 66}), api('geo').catch(() => ({})), worldMap().catch(() => null), api('geolines').catch(() => null), ...asCharts.map(([by, m]) => seriesData(by, m))]);
  const all = (c.rows || []).filter(r => r.key !== '__internal__' && r.key), cr = all.slice(0, 66), ar = a.rows || [];
  const cm = Math.max(1, ...cr.map(r => r.wire)), am = Math.max(1, ...ar.map(r => r.wire));
  const attr = attribution(gd.sources);
  el.innerHTML = `<div class="grid" style="margin-bottom:16px">${panel('c12', t('geo.map'), t('geo.map_sub', {n: nf(all.length)}), `<div id="wmap"></div>
      <p class="muted attrib">${attr ? `${t('geo.data_from')} ${attr} · ` : ''}${t('geo.map_from')} <a href="https://www.naturalearthdata.com" target="_blank" rel="noopener">Natural Earth</a></p>`)}</div>
    <div class="grid" style="margin-bottom:16px">${asCharts.map(([by, m], i) => seriesPanel('asr' + i, 'c6', `${t(by === 'asn_src' ? 'geo.as_src' : 'geo.as_dst')} · ${t(m === 'wire' ? 'ch.bps' : 'ch.pps')}`, i < 2 ? esc(t(by === 'asn_src' ? 'geo.as_src_sub' : 'geo.as_dst_sub')) : '')).join('')}</div>
    <div class="grid">
    ${panel('c6', t('geo.countries'), t('geo.countries_sub'), `<table><tr><th></th><th>${t('col.country')}</th><th class="num">${t('col.traffic')}</th><th style="width:40%">${t('col.share')}</th></tr>
      ${cr.map((r, i) => `<tr><td class="rank">${i + 1}</td><td>${V('country', r.key, country(r.key))} <span class="muted">${esc(r.key)}</span></td><td class="num">${fmtBytes(r.wire)}</td><td>${bar(r.wire, cm)}</td></tr>`).join('') || `<tr><td colspan="4" class="empty">${t('geo.no_table')}</td></tr>`}</table>`)}
    ${panel('c6', t('geo.as'), '', `<table><tr><th></th><th>${t('col.asn')}</th><th>${t('col.org')}</th><th class="num">${t('col.traffic')}</th><th style="width:32%">${t('col.share')}</th></tr>
      ${ar.map((r, i) => `<tr><td class="rank">${i + 1}</td><td class="nw">${r.key === '0' ? `<span class="muted">${t('unknown')}</span>` : V('asn', r.key, 'AS' + r.key)}</td><td>${esc(r.extra)}</td><td class="num">${fmtBytes(r.wire)}</td><td>${bar(r.wire, am)}</td></tr>`).join('') || `<tr><td colspan="5" class="empty">${t('geo.no_table')}</td></tr>`}</table>`)}
  </div>`;
  if (w) drawMap($('#wmap'), w, all, gl); else $('#wmap').innerHTML = `<div class="empty">${t('empty.nodata')}</div>`;
  asCharts.forEach(([by, m], i) => fillSeries('asr' + i, ad[i], by, m));
};

// ------------------------------------------------------------ data cleanup
// Deletes flow records, summaries, counters and findings older than a number
// of days, or everything; each shows what it frees and asks first.
views.cleanup = async (el) => {
  const d = await (await fetch('/api/cleanup')).json();
  const opts = d.options || [];
  const row = o => {
    const what = o.days ? t('cl.days', {n: o.days}) : t('cl.all');
    const has = o.rows > 0 || !o.days;
    return `<tr><td class="nw"><b>${esc(what)}</b></td><td class="num">${has && o.rows ? esc(t('cl.rows', {n: nf(o.rows)})) : `<span class="muted">${t('cl.none')}</span>`}</td>
      <td class="num">${o.bytes ? fmtBytes(o.bytes) : '—'}</td>
      <td style="text-align:end"><button class="btn ${o.days ? '' : 'danger'}" data-cl="${o.days}" ${has ? '' : 'disabled'}>${t('cl.run')}</button></td></tr>`;
  };
  el.innerHTML = `<div class="grid">${panel('c12', t('cl.title'), t('cl.sub'), `<p class="muted" style="margin:0 0 12px;font-size:13px">${esc(t('cl.what'))}</p>
    <div class="kpis" style="grid-template-columns:repeat(2,1fr);margin-bottom:14px"><div class="kpi"><div class="lab">${t('cl.disk')}</div><div class="n">${fmtBytes(d.disk_bytes)}</div></div>
      <div class="kpi"><div class="lab">${t('cl.oldest')}</div><div class="n" style="font-size:20px">${d.oldest && !d.oldest.startsWith('0001') ? esc(new Intl.DateTimeFormat(LANG, {dateStyle: 'medium', timeStyle: 'short'}).format(new Date(d.oldest))) : '—'}</div></div></div>
    <table><tr><th></th><th class="num">${t('cl.col_rows')}</th><th class="num">${t('cl.col_size')}</th><th></th></tr>${opts.map(row).join('')}</table>
    <div id="clMsg" style="margin-top:10px;font-size:13px"></div>`)}</div>`;
  el.querySelectorAll('[data-cl]').forEach(b => b.onclick = async () => {
    const days = +b.dataset.cl;
    if (!confirm(days ? t('cl.confirm_days', {n: days}) : t('cl.confirm_all'))) return;
    if (!days && !confirm(t('cl.confirm_all2'))) return;
    b.disabled = true;
    const res = await fetch('/api/cleanup', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({days})});
    const j = await res.json().catch(() => ({}));
    if (!res.ok) { const m = $('#clMsg'); m.style.color = 'var(--crit)'; m.textContent = j.error || res.statusText; b.disabled = false; return; }
    clDone = t('cl.done', {n: nf(j.rows || 0), s: fmtBytes(j.bytes || 0)});
    loadStatus(); render();
  });
  if (clDone) { const m = $('#clMsg'); m.style.color = 'var(--good)'; m.textContent = clDone; clDone = ''; }
};
let clDone = '';

// ------------------------------------------------------------ offline analysis
// Capture files are imported into a database of their own; "Analyse" shows
// them on all the other pages, apart from the live data.
let sbMsg = '';
views.sandbox = async (el) => {
  await loadSB();
  const fs = sbInfo.files || [], own = fs.filter(f => !f.sample).length, mb = Math.round(sbInfo.max_file_size / 1048576);
  const df = new Intl.DateTimeFormat(LANG, {dateStyle: 'short', timeStyle: 'medium'});
  const pending = fs.some(f => f.status !== 'done' && f.status !== 'error') || sbInfo.busy;
  const rows = fs.map(f => `<tr><td>${esc(f.name)}${f.sample ? ` <span class="tag">${t('sb.sample')}</span>` : ''}</td><td class="num">${fmtBytes(f.size)}</td>
      <td class="num">${f.packets ? nf(f.packets) : '—'}</td><td class="num">${f.flows ? nf(f.flows) : '—'}</td>
      <td class="nw muted">${f.flows ? esc(df.format(new Date(f.first))) + ' – ' + esc(df.format(new Date(f.last))) : ''}</td>
      <td>${f.status === 'error' ? status('bad', f.error || t('sb.st_error')) : f.status === 'done' ? status('ok', t('sb.st_done')) : `<span class="muted">${t('sb.st_' + f.status)}…</span>`}</td>
      <td><button class="btn" data-sbdel="${esc(f.name)}">${t('sb.delete')}</button></td></tr>`).join('');
  el.innerHTML = `<div class="grid">${panel('c12', t('sb.title'), t('sb.sub'), `<p class="sbexplain">${esc(t('sb.explain'))}</p>
    <table><tr><th>${t('sb.col_file')}</th><th class="num">${t('sb.col_size')}</th><th class="num">${t('sb.col_packets')}</th><th class="num">${t('sb.col_flows')}</th><th>${t('sb.col_span')}</th><th>${t('sb.col_status')}</th><th></th></tr>
    ${rows || `<tr><td colspan="7" class="empty">${t('sb.empty')}</td></tr>`}</table>
    <div class="sbactions"><button class="primary" id="sbGo" ${sbInfo.ready && !pending ? '' : 'disabled'}>${t('sb.analyze')}</button>
      <label class="btn ${own >= sbInfo.max_files ? 'disabled' : ''}" style="cursor:pointer">${t('sb.upload')}<input type="file" id="sbFile" accept=".pcap,.pcapng,.cap" multiple hidden ${own >= sbInfo.max_files ? 'disabled' : ''}></label>
      ${fs.length ? `<button class="btn" id="sbAll">${t('sb.delete_all')}</button>` : ''}<span id="sbMsg" style="font-size:13px"></span></div>
    <p class="muted" style="font-size:12.5px;margin:10px 0 0">${esc(sbInfo.max_total >= sbInfo.max_file_size * sbInfo.max_files ? t('sb.limits', {n: sbInfo.max_files, mb}) : sbInfo.max_total <= sbInfo.max_file_size ? t('sb.limits_total', {n: sbInfo.max_files, gb: fmtBytes(sbInfo.max_total)}) : t('sb.limits', {n: sbInfo.max_files, mb}) + ' ' + t('sb.limits_total', {n: sbInfo.max_files, gb: fmtBytes(sbInfo.max_total)}))} ${esc(t('sb.formats'))}</p>`)}</div>`;
  // the last upload problem stays shown after the page refreshes
  const msg = (ok, text) => { sbMsg = ok === false ? text : ''; const m = $('#sbMsg'); m.style.color = ok === null ? 'var(--ink-3)' : ok ? 'var(--good)' : 'var(--crit)'; m.textContent = text; };
  if (sbMsg) msg(false, sbMsg);
  $('#sbGo').onclick = () => { state.ds = 'sb'; go('overview'); };
  el.querySelectorAll('[data-sbdel]').forEach(b => b.onclick = async () => {
    if (!confirm(t('sb.confirm', {f: b.dataset.sbdel}))) return;
    await fetch('/api/sandbox/files?name=' + encodeURIComponent(b.dataset.sbdel), {method: 'DELETE'});
    render();
  });
  if ($('#sbAll')) $('#sbAll').onclick = async () => {
    if (!confirm(t('sb.confirm_all'))) return;
    await fetch('/api/sandbox/files', {method: 'DELETE'});
    state.ds = ''; render();
  };
  $('#sbFile').onchange = async e => {
    const files = [...e.target.files];
    msg(null, '');
    let left = sbInfo.max_files - own;
    for (const f of files) {
      if (left <= 0) { msg(false, t('sb.too_many', {n: sbInfo.max_files})); break; }
      if (f.size > sbInfo.max_file_size) { msg(false, t('sb.too_big', {f: f.name, mb})); continue; }
      const ok = await new Promise(done => {
        const x = new XMLHttpRequest();
        x.open('POST', '/api/sandbox/files?name=' + encodeURIComponent(f.name));
        x.upload.onprogress = ev => ev.lengthComputable && msg(null, t('sb.uploading', {f: f.name, p: Math.round(100 * ev.loaded / ev.total)}));
        x.onload = () => {
          if (x.status === 200) return done(true);
          let j = {}; try { j = JSON.parse(x.responseText); } catch (err) {}
          msg(false, x.status === 413 ? (j.max_files && !/MB/.test(j.error) ? t('sb.too_many', {n: sbInfo.max_files}) : t('sb.too_big', {f: f.name, mb})) : j.code === 'not_capture' ? t('sb.not_capture', {f: f.name}) : (j.error || x.statusText)); done(false);
        };
        x.onerror = () => { msg(false, t('sb.failed')); done(false); };
        x.send(f);
      });
      if (ok) left--;
    }
    render();
  };
  if (pending) { setTimeout(() => { if (state.v === 'sandbox') render(); }, 1500); sbWaited = true; }
  else if (offlineMode && sbInfo.ready && sbWaited) { sbWaited = false; state.ds = 'sb'; go('overview'); }
};
// in offline mode the first visit goes on to the overview once the files are in
let sbWaited = true;

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
  const key = state.r + JSON.stringify(state.f);
  if (state.rk !== key) { state.rk = key; state.rp = 0; }
  const per = state.rps || 50, page = state.rp || 0;
  const d = await api('records', {limit: per, offset: page * per, stats: 1});
  const rows = d.rows || [], on = recCols(), total = d.total || 0, pages = Math.max(1, Math.ceil(total / per));
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
  const pager = `<div class="pager"><label>${t('rec.per_page')} <select id="recPer">${[50, 100, 200].map(n => `<option ${n === per ? 'selected' : ''}>${n}</option>`).join('')}</select></label>
      <span>${esc(t('rec.range', {a: nf(total ? page * per + 1 : 0), b: nf(Math.min(total, page * per + rows.length)), n: nf(total)}))}</span>
      <button class="btn" id="recPrev" ${page ? '' : 'disabled'}>‹ ${t('rec.prev')}</button><button class="btn" id="recNext" ${page + 1 < pages ? '' : 'disabled'}>${t('rec.next')} ›</button></div>`;
  el.innerHTML = `<div class="panel" style="margin-bottom:16px"><div class="rechead"><div class="kpi"><div class="lab">${t('rec.title')}</div><div class="n big">${nf(total)}</div><div class="d muted">${esc(t('rec.in', {r: rangeLabel()}))}</div></div>
      <div class="chart" id="chRec" style="height:150px" aria-label="${esc(t('rec.title'))}"></div></div></div>
    <div class="panel"><div class="ph"><h2>${t('rec.title')}</h2><span class="sub">${esc(t('rec.sorted'))}</span>
      <div class="cols"><button class="btn" id="colBtn">${t('rec.columns')}</button><div class="menu" id="colMenu" hidden>${REC_COLS.map(c => `<label><input type="checkbox" value="${c[0]}" ${on.has(c[0]) ? 'checked' : ''}>${t('rc.' + c[0])}</label>`).join('')}</div></div></div>
    <div style="overflow-x:auto"><table><tr>${cols.map(c => `<th class="${num.has(c) ? 'num' : ''}">${t('rc.' + c)}</th>`).join('')}</tr>
    ${rows.map(r => `<tr>${cols.map(c => `<td class="${num.has(c) ? 'num' : ''}">${cell(c, r)}</td>`).join('')}</tr>`).join('') || `<tr><td colspan="${cols.length}" class="empty">${t('empty.nodata')}</td></tr>`}</table></div>${pager}</div>`;
  if (d.hist) barChart($('#chRec'), d.hist.times, d.hist.values[0], v => t('rec.n', {n: nf(v)}));
  $('#recPer').onchange = e => { state.rps = +e.target.value; state.rp = 0; render(); };
  $('#recPrev').onclick = () => { state.rp = Math.max(0, page - 1); render(); };
  $('#recNext').onclick = () => { state.rp = page + 1; render(); };
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
// Interface check: the selected interface's traffic in both directions
// (ingress green, egress blue) in bits/s and packets/s, the list of
// interfaces, and flow numbers next to the interface counters.
const IF_IN = 'var(--c3)', IF_OUT = 'var(--c1)';
function ifVerdict(r, rc, dirIn) {
  const dev = dirIn ? r.in_dev : r.out_dev, err = dirIn ? r.in_stat_err : r.out_stat_err;
  if (!r.has_counters) return {text: t('if.no_counters'), kind: 'none', dev, err};
  if (Math.abs(dev) <= Math.max(0.02, 2 * err)) return {text: t('if.ok', {dev: fmtPct(dev), err: '±' + nf(err * 100, 1) + '%'}), kind: 'ok', dev, err};
  const causes = [], src = rc.source;
  if (src && src.lost_pct >= 0.5) causes.push(t('cause.loss', {p: nf(src.lost_pct, 1) + '%'}));
  if (src && src.sampling_state === 'waiting') causes.push(t('cause.sampling'));
  if (dev < 0) causes.push(t('cause.unsampled_iface'));
  if (dev > 0) causes.push(t('cause.dup'));
  return {text: t(dev < 0 ? 'if.low' : 'if.high', {dev: nf(Math.abs(dev) * 100, 1) + '%', causes: causes.join('; ')}), kind: 'warn', dev, err};
}
views.ifaces = async (el) => {
  const d = await api('ifaces');
  (d.ifaces || []).forEach(f => ifaceNames.set(f.exporter + '/' + f.ifindex, ifName(f) + ' · ' + (f.device || f.exporter)));
  const list = (d.ifaces || []).slice().sort((a, b) => (!!b.has_counters - !!a.has_counters) || ((devKind(b) === 'warn') - (devKind(a) === 'warn')));
  if (!list.length) { el.innerHTML = `<div class="panel"><div class="empty">${t('empty.nodata')}</div></div>`; return; }
  if (!state.ifc || !list.some(f => f.exporter === state.ifc.exporter && f.ifindex === state.ifc.ifindex)) state.ifc = {exporter: list[0].exporter, ifindex: list[0].ifindex};
  const key = state.ifc.exporter + '/' + state.ifc.ifindex, sel = list.find(f => f.exporter === state.ifc.exporter && f.ifindex === state.ifc.ifindex);
  const fil = [...state.f, {f: 'iface', v: key, neg: false}];
  const [ri, ro, pi, po, rc] = await Promise.all([['if_in', 'wire'], ['if_out', 'wire'], ['if_in', 'pkts'], ['if_out', 'pkts']].map(([by, measure]) => api('series', {by, measure, top: 20}, fil))
    .concat([api('recon', {exporter: state.ifc.exporter, ifindex: state.ifc.ifindex})]));
  const pick = s => { const i = (s.names || []).indexOf(key); return i < 0 ? (s.times || []).map(() => 0) : s.values[i]; };
  const sum = a => a.reduce((x, y) => x + y, 0);
  const name = `${ifName(sel)} · ${sel.device || sel.exporter}`;
  const leg = (vi, vo, fmt) => `<div class="legend"><span><i style="background:${IF_IN}"></i>${t('if.in')}</span><span><i style="background:${IF_OUT}"></i>${t('if.out')}</span></div>`;
  el.innerHTML = `<div class="grid" style="margin-bottom:16px">
    ${panel('c6', `${esc(name)} · ${t('ch.bps')}`, '', `<div class="chart" id="ifB" style="height:220px" aria-label="${esc(t('ch.bps'))}"></div>${leg()}`)}
    ${panel('c6', `${esc(name)} · ${t('ch.pps')}`, '', `<div class="chart" id="ifP" style="height:220px" aria-label="${esc(t('ch.pps'))}"></div>${leg()}`)}</div>
    <div class="grid">
    ${panel('c4', t('if.title'), t('if.sub'), `<div class="iflist">${list.map(f => {
      const k = devKind(f), cur = f.exporter === state.ifc.exporter && f.ifindex === state.ifc.ifindex;
      return `<button data-e="${esc(f.exporter)}" data-i="${f.ifindex}" aria-current="${cur}"><span>${esc(ifName(f))}<br><span class="muted" style="font-size:12.5px">${esc(f.device || f.exporter)} · ${f.ifindex}</span></span>${k ? status(k, nf(Math.max(Math.abs(f.in_dev), Math.abs(f.out_dev)) * 100, 1) + '%') : `<span class="muted" style="font-size:12.5px">${t('if.no_ctr_short')}</span>`}</button>`;
    }).join('')}</div>`)}
    <div class="panel c8" id="recon"></div></div>`;
  const bitsIn = pick(ri), bitsOut = pick(ro);
  tsChart($('#ifB'), {times: ri.times, lines: [{name: t('if.in'), color: IF_IN, data: bitsIn, fill: true}, {name: t('if.out'), color: IF_OUT, data: bitsOut, fill: true}]});
  tsChart($('#ifP'), {times: pi.times, fmtY: fmtAxisBps, fmtV: fmtPps, lines: [{name: t('if.in'), color: IF_IN, data: pick(pi), fill: true}, {name: t('if.out'), color: IF_OUT, data: pick(po), fill: true}]});
  el.querySelectorAll('.iflist button').forEach(b => b.onclick = () => { state.ifc = {exporter: b.dataset.e, ifindex: +b.dataset.i}; render(); });

  const r = rc.recon, dir = state.ifdir || 'both';
  const vin = ifVerdict(r, rc, true), vout = ifVerdict(r, rc, false);
  const head = v => r.has_counters ? t('if.dev', {v: fmtPct(v.dev)}) : '—';
  const lines = [];
  if (dir !== 'out') { if (r.has_counters) lines.push({name: t('if.in') + ' · ' + t('if.counter'), color: IF_IN, dash: true, data: r.in_counter}); lines.push({name: t('if.in') + ' · ' + t('if.flow'), color: IF_IN, data: r.in_flow}); }
  if (dir !== 'in') { if (r.has_counters) lines.push({name: t('if.out') + ' · ' + t('if.counter'), color: IF_OUT, dash: true, data: r.out_counter}); lines.push({name: t('if.out') + ' · ' + t('if.flow'), color: IF_OUT, data: r.out_flow}); }
  const kinds = (dir === 'both' ? [vin, vout] : [dir === 'in' ? vin : vout]);
  const vk = kinds.some(v => v.kind === 'warn') ? 'warn' : kinds.every(v => v.kind === 'none') ? 'none' : '';
  $('#recon').innerHTML = `<div class="ph"><div><div class="sub">${esc(name)}</div>
      <div style="font-size:20px;font-weight:650">${dir === 'both' ? `<span style="color:${IF_IN}">${t('if.in')}</span> ${head(vin)} · <span style="color:${IF_OUT}">${t('if.out')}</span> ${head(vout)}` : head(dir === 'in' ? vin : vout)}</div></div>
      <div class="seg" role="group">${['both', 'in', 'out'].map(k => `<button data-d="${k}" aria-pressed="${dir === k}">${t('if.' + k)}</button>`).join('')}</div></div>
    <div class="chart" id="chRecon" style="height:240px" aria-label="${esc(t('if.title'))}"></div>
    <div class="legend"><span><i class="line" style="background:var(--ink-2)"></i>${t('if.flow')}</span>${r.has_counters ? `<span><i class="line dash" style="border-top:2px dashed var(--ink-2);background:none"></i>${t('if.counter')}</span>` : ''}
      <span><i style="background:${IF_IN}"></i>${t('if.in')}</span><span><i style="background:${IF_OUT}"></i>${t('if.out')}</span></div>
    <div class="verdict ${vk}">${dir === 'both' ? `<b>${t('if.in')}</b>: ${vin.text}<br><b>${t('if.out')}</b>: ${vout.text}` : (dir === 'in' ? vin : vout).text}</div>`;
  $('#recon').querySelectorAll('[data-d]').forEach(b => b.onclick = () => { state.ifdir = b.dataset.d; render(); });
  tsChart($('#chRecon'), {times: r.times, lines});
};

views.sources = async (el) => {
  const [d, inv, gd] = await Promise.all([api('sources'), api('inventory').catch(() => ({text: ''})), api('geo').catch(() => ({sources: []}))]);
  const dfmt = d => d && !d.startsWith('0001') ? new Intl.DateTimeFormat(LANG, {dateStyle: 'medium'}).format(new Date(d)) : '';
  const VENDOR = {dbip: 'DB-IP Lite', maxmind: 'MaxMind GeoLite2', ipinfo: 'IPinfo Lite', iptoasn: 'IPtoASN'};
  const geoRows = (gd.sources || []).map(g => `<tr><td>${status('ok', t('geo.kind_' + g.kind))}</td>
      <td>${esc(VENDOR[g.vendor] || g.type)} <span class="muted">${esc(g.file === 'built-in' ? t('geo.builtin') : g.file)}${g.entries ? ' · ' + esc(t('geo.entries', {n: nf(g.entries)})) : ''}</span></td>
      <td class="nw">${esc(dfmt(g.built))}</td><td class="muted">${t(g.fallback ? 'geo.role_fallback' : 'geo.role_main')}</td>
      <td>${g.file !== 'built-in' ? `<button class="btn" data-geodel="${esc(g.file)}">${t('geo.remove')}</button>` : ''}</td></tr>`).join('');
  const FREE = [
    ['DB-IP Lite', 'geo.f_dbip', 'CC BY 4.0', 'https://db-ip.com/db/lite.php'],
    ['MaxMind GeoLite2', 'geo.f_maxmind', 'GeoLite2 EULA', 'https://www.maxmind.com/en/geolite2/signup'],
    ['IPinfo Lite', 'geo.f_ipinfo', 'CC BY-SA 4.0', 'https://ipinfo.io/lite'],
    ['IPtoASN', 'geo.f_iptoasn', 'PDDL 1.0', 'https://iptoasn.com']];
  const freeRows = FREE.map(([n, k, lic, url]) => `<tr><td class="nw"><a href="${url}" target="_blank" rel="noopener">${n}</a></td><td>${esc(t(k))}</td><td class="nw muted">${lic}</td></tr>`).join('');
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
    ${panel('c12', t('geo.title'), t('geo.sub'), `<table><tr><th>${t('geo.col_holds')}</th><th>${t('geo.col_db')}</th><th>${t('geo.col_built')}</th><th>${t('geo.col_use')}</th><th></th></tr>${geoRows}</table>
      <div style="display:flex;gap:10px;align-items:center;flex-wrap:wrap;margin-top:12px"><button class="primary" id="geoDbip">${t('geo.update')}</button><label class="btn" style="cursor:pointer">${t('geo.upload')}<input type="file" id="geoFile" accept=".mmdb,.tsv,.gz,.txt,.csv" hidden></label><span id="geoMsg" class="muted" style="font-size:13px"></span></div>
      <h3 class="subhead">${t('geo.free')}</h3><p class="muted" style="font-size:12.5px;margin:0 0 6px">${t('geo.free_sub')}</p>
      <table><tr><th>${t('geo.col_db')}</th><th>${t('geo.col_holds')}</th><th>${t('geo.col_lic')}</th></tr>${freeRows}</table>
      <p class="muted" style="font-size:12.5px;margin:10px 0 0">${t('geo.where')}</p>`)}
    ${panel('c12', t('logo.title'), t('logo.sub'), `<div class="logoprev"><img id="logoPrev" src="/logo?v=${Date.now()}" alt="logo">
      <div style="display:grid;gap:8px"><div style="display:flex;gap:10px;align-items:center;flex-wrap:wrap"><label class="primary" style="cursor:pointer;display:inline-block;padding:6px 14px;border-radius:6px;background:var(--accent);color:#fff;font-weight:600">${t('logo.upload')}<input type="file" id="logoFile" accept=".png,.svg,.jpg,.jpeg,.webp,.gif,image/*" hidden></label><button class="btn" id="logoReset">${t('logo.reset')}</button></div>
      <span class="muted" style="font-size:12.5px">${t('logo.hint')}</span><span id="logoMsg" style="font-size:13px"></span></div></div>`)}
    ${panel('c12', t('src.names'), t('src.names_sub'), `<div class="nmform" id="nmForm">
        <select id="nmK">${NM_KINDS.map(k => `<option value="${k}">${t('nm.k_' + k)}</option>`).join('')}</select>
        <input id="nmA" autocomplete="off" spellcheck="false"><input id="nmI" autocomplete="off" inputmode="numeric" placeholder="${esc(t('nm.ifindex'))}" hidden>
        <input id="nmN" autocomplete="off" spellcheck="false"><select id="nmC" hidden></select>
        <button class="primary" id="nmAdd">${t('nm.add')}</button><button class="btn" id="nmCancel" hidden>${t('nm.cancel')}</button></div>
      <div id="nmMsg" style="font-size:13px;min-height:1.2em;margin:4px 0 6px"></div>
      <table class="nmtab" id="nmTab"></table>
      <details class="nmadv"><summary>${t('nm.advanced')}</summary><div class="helpbox">${esc(t('src.names_help'))}<pre>host   192.168.3.28    ${esc(t('src.ex_host'))}
net    192.168.3.0/24  ${esc(t('src.ex_net'))}
device 192.168.1.1     ${esc(t('src.ex_device'))}
iface  192.168.1.1 3   ${esc(t('src.ex_iface'))}
snmp   192.168.1.1     public</pre></div><textarea class="inv" id="inv" spellcheck="false">${esc(inv.text || '')}</textarea>
      <div style="display:flex;gap:10px;align-items:center;margin-top:8px"><button class="primary" id="invSave">${t('src.save')}</button><span id="invMsg" class="muted" style="font-size:13px"></span></div></details>`)}
  </div>`;
  const geoMsg = (ok, text) => { const m = $('#geoMsg'); m.style.color = ok === null ? '' : ok ? 'var(--good)' : 'var(--crit)'; m.textContent = text; };
  $('#geoDbip').onclick = async e => {
    e.target.disabled = true; geoMsg(null, t('geo.updating'));
    try {
      const res = await fetch('/api/geo/dbip', {method: 'POST'});
      const j = await res.json().catch(() => ({}));
      if (!res.ok) { geoMsg(false, t('geo.update_failed', {e: j.error || res.statusText})); e.target.disabled = false; return; }
      geoMsg(true, t('geo.updated')); setTimeout(render, 1500);
    } catch (err) { geoMsg(false, String(err.message || err)); e.target.disabled = false; }
  };
  el.querySelectorAll('[data-geodel]').forEach(b => b.onclick = async () => {
    if (!confirm(t('geo.remove_confirm', {f: b.dataset.geodel}))) return;
    const res = await fetch('/api/geo?file=' + encodeURIComponent(b.dataset.geodel), {method: 'DELETE'});
    if (!res.ok) { const j = await res.json().catch(() => ({})); geoMsg(false, j.error || res.statusText); return; }
    render();
  });
  $('#geoFile').onchange = async e => {
    const f = e.target.files[0]; if (!f) return;
    const msg = $('#geoMsg'); msg.style.color = ''; msg.textContent = t('geo.uploading');
    try {
      const res = await fetch('/api/geo', {method: 'POST', body: f});
      const j = await res.json().catch(() => ({}));
      if (!res.ok) { msg.style.color = 'var(--crit)'; msg.textContent = j.error || res.statusText; return; }
      msg.style.color = 'var(--good)'; msg.textContent = t('geo.installed', {f: j.installed});
      setTimeout(render, 1500);
    } catch (err) { msg.style.color = 'var(--crit)'; msg.textContent = String(err.message || err); }
  };
  const logoDone = (ok, text) => {
    const m = $('#logoMsg'); m.style.color = ok ? 'var(--good)' : 'var(--crit)'; m.textContent = text;
    if (ok) { const v = Date.now(); $('#logoPrev').src = '/logo?v=' + v; document.querySelectorAll('img.logo').forEach(i => i.src = '/logo?v=' + v); }
  };
  $('#logoFile').onchange = async e => {
    const f = e.target.files[0]; if (!f) return;
    const res = await fetch('/api/logo', {method: 'POST', body: f});
    const j = await res.json().catch(() => ({}));
    logoDone(res.ok, res.ok ? t('logo.saved') : (j.error || res.statusText));
  };
  $('#logoReset').onclick = async () => {
    const res = await fetch('/api/logo', {method: 'DELETE'});
    logoDone(res.ok, res.ok ? t('logo.saved') : res.statusText);
  };
  $('#invSave').onclick = async () => {
    const res = await fetch('/api/inventory', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({text: $('#inv').value})});
    const j = await res.json().catch(() => ({}));
    $('#invMsg').textContent = res.ok ? t('src.saved') : (j.error || res.statusText);
    $('#invMsg').style.color = res.ok ? 'var(--good)' : 'var(--crit)';
    if (res.ok) { loadStatus(); namesForm(); }
  };
  namesForm();
};

// Names: the inventory as a form and a table. Each row is one line of the
// inventory text; comments and other lines are kept as they are.
const NM_KINDS = ['host', 'net', 'device', 'iface', 'snmp'];
function nmParse(text) {
  return text.split('\n').map((line, i) => {
    const f = line.trim().split(/\s+/), k = (f[0] || '').toLowerCase();
    if (!NM_KINDS.includes(k)) return null;
    let words = f.slice(k === 'iface' ? 3 : 2), c = '';
    if (k === 'net') words = words.filter(w => { const m = /^country=(\w\w)$/i.exec(w); if (m) c = m[1].toUpperCase(); return !m; });
    return {i, k, a: f[1] || '', x: k === 'iface' ? (f[2] || '') : '', n: words.join(' '), c};
  }).filter(Boolean);
}
const nmLine = r => [r.k, r.a, r.k === 'iface' ? r.x : '', r.n, r.k === 'net' && r.c ? 'country=' + r.c : ''].filter(Boolean).join(' ');
function namesForm() {
  const tab = $('#nmTab'); if (!tab) return;
  let editing = null;
  const rows = nmParse($('#inv').value);
  const sync = () => {
    const k = $('#nmK').value;
    $('#nmI').hidden = k !== 'iface';
    $('#nmA').placeholder = t(k === 'net' ? 'nm.ph_net' : 'nm.ph_addr');
    $('#nmN').placeholder = t(k === 'snmp' ? 'nm.community' : 'nm.name');
    $('#nmC').hidden = k !== 'net';
  };
  // countries for networks, named in the reader's language
  worldMap().then(w => {
    const cur = $('#nmC').value;
    $('#nmC').innerHTML = `<option value="">${esc(t('nm.country_none'))}</option>` + Object.keys(w.c).map(cc => [cc, country(cc)]).sort((x, y) => x[1].localeCompare(y[1], LANG)).map(([cc, n]) => `<option value="${cc}">${esc(n)}</option>`).join('');
    $('#nmC').value = cur;
  }).catch(() => {});
  tab.innerHTML = `<tr><th>${t('nm.type')}</th><th>${t('nm.address')}</th><th>${t('nm.name')}</th><th></th></tr>` + (rows.map(r => `<tr><td class="nw">${t('nm.k_' + r.k)}</td>
      <td class="nw"><code>${esc(r.a)}${r.k === 'iface' ? ' #' + esc(r.x) : ''}</code></td><td>${r.c ? `<span class="tag">${esc(country(r.c))}</span> ` : ''}${r.k === 'snmp' ? `<span class="muted">${esc(t('nm.community'))}:</span> ` : ''}${esc(r.n.replace(/\s*\bspeed=\d+/, ''))}${/\bspeed=(\d+)/.test(r.n) ? ` <span class="muted">${fmtBps(+r.n.match(/\bspeed=(\d+)/)[1])}</span>` : ''}</td>
      <td class="nw" style="text-align:end"><button class="btn" data-nme="${r.i}">${t('nm.edit')}</button> <button class="btn" data-nmd="${r.i}">${t('nm.delete')}</button></td></tr>`).join('') ||
    `<tr><td colspan="4" class="empty">${t('nm.empty')}</td></tr>`);
  const msg = (ok, text) => { const m = $('#nmMsg'); m.style.color = ok ? 'var(--good)' : 'var(--crit)'; m.textContent = text; };
  const save = async lines => {
    const text = lines.join('\n');
    const res = await fetch('/api/inventory', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({text})});
    const j = await res.json().catch(() => ({}));
    if (!res.ok) { msg(false, j.error || res.statusText); return false; }
    $('#inv').value = text; msg(true, t('src.saved')); loadStatus(); namesForm(); return true;
  };
  const lines = () => $('#inv').value.split('\n');
  const reset = () => { editing = null; $('#nmAdd').textContent = t('nm.add'); $('#nmCancel').hidden = true; ['#nmA', '#nmI', '#nmN', '#nmC'].forEach(x => $(x).value = ''); };
  $('#nmK').onchange = sync; sync();
  $('#nmCancel').onclick = reset;
  $('#nmAdd').onclick = async () => {
    const r = {k: $('#nmK').value, a: $('#nmA').value.trim(), x: $('#nmI').value.trim(), n: $('#nmN').value.trim().replace(/\s+/g, ' '), c: $('#nmC').value};
    if (!r.a || !r.n || (r.k === 'iface' && !r.x)) { msg(false, t('nm.missing')); return; }
    const ip = /^(\d{1,3}(\.\d{1,3}){3}|[0-9a-fA-F:]*:[0-9a-fA-F:.]*)$/;
    if (r.k === 'net' ? !(r.a.includes('/') && ip.test(r.a.split('/')[0]) && /^\d{1,3}$/.test(r.a.split('/')[1])) : !ip.test(r.a)) { msg(false, t(r.k === 'net' ? 'nm.bad_net' : 'nm.bad_addr', {a: r.a})); return; }
    if (r.k === 'iface' && !/^\d+$/.test(r.x)) { msg(false, t('nm.bad_index', {x: r.x})); return; }
    const ls = lines();
    // a name for something already named replaces the old one
    const same = rows.find(x => x.k === r.k && x.a === r.a && x.x === r.x && x.i !== editing);
    if (editing != null) { ls[editing] = nmLine(r); if (same) ls.splice(same.i, 1); }
    else if (same) ls[same.i] = nmLine(r);
    else { while (ls.length && ls[ls.length - 1].trim() === '') ls.pop(); ls.push(nmLine(r)); }
    if (await save(ls)) reset();
  };
  tab.querySelectorAll('[data-nmd]').forEach(b => b.onclick = async () => {
    const i = +b.dataset.nmd, r = rows.find(x => x.i === i);
    if (!confirm(t('nm.confirm', {x: nmLine(r)}))) return;
    const ls = lines(); ls.splice(i, 1); save(ls);
  });
  tab.querySelectorAll('[data-nme]').forEach(b => b.onclick = () => {
    const r = rows.find(x => x.i === +b.dataset.nme);
    editing = r.i; $('#nmK').value = r.k; sync();
    $('#nmA').value = r.a; $('#nmI').value = r.x; $('#nmN').value = r.n; $('#nmC').value = r.c || '';
    $('#nmAdd').textContent = t('nm.save_edit'); $('#nmCancel').hidden = false; $('#nmA').focus();
  });
}

// Drill-down: one host or one service, everything about it on one page.
// Every value on it opens the same menu, so one can keep drilling.
views.detail = async (el) => {
  const det = state.det;
  if (!det) { go('overview'); return; }
  const fs = [...state.f.filter(x => !(x.f === det.f && x.v === det.v)), {f: det.f, v: det.v, neg: false}];
  const isIP = det.f === 'ip';
  const [d, conv, side1, side2, rec, fd] = await Promise.all([
    api('overview', {}, fs), api('topn', {dim: 'conv', limit: 30}, fs),
    api('topn', {dim: isIP ? 'port' : 'client', limit: 15}, fs), api('topn', {dim: isIP ? 'country' : 'server', limit: 15}, fs),
    api('records', {limit: 50}, fs), isIP ? api('findings', {status: 'all', ip: det.v}, []).catch(() => null) : null]);
  const title = isIP ? (names.get(det.v) ? `${esc(names.get(det.v))} <span class="sub">${esc(det.v)}</span>` : esc(det.v)) : esc(det.v);
  $('#title').textContent = t('nav.detail') + ': ' + (isIP ? (names.get(det.v) || det.v) : det.v);
  const tot = d.totals, series = d.series || {times: [], names: [], values: []};
  const areas = series.names.map((n, i) => ({name: appLabel(n), color: color(i, n), data: series.values[i]}));
  const lines = d.baseline ? [{name: t(d.basis === 'week' ? 'last_week' : 'prev_period'), color: 'var(--base)', dash: true, data: d.baseline}] : [];
  const table = (rows, heads, cell) => { const m = Math.max(1, ...rows.map(r => r.wire)); return `<table><tr><th></th>${heads.map(h => `<th>${t(h)}</th>`).join('')}<th class="num">${t('col.traffic')}</th><th style="width:26%">${t('col.share')}</th></tr>
    ${rows.map((r, i) => `<tr><td class="rank">${i + 1}</td>${cell(r).map(c => `<td>${c}</td>`).join('')}<td class="num">${fmtBytes(r.wire)}</td><td>${bar(r.wire, m)}</td></tr>`).join('') || `<tr><td colspan="${heads.length + 3}" class="empty">${t('empty.nodata')}</td></tr>`}</table>`; };
  const s1 = side1.rows || [], s2 = side2.rows || [], cv = conv.rows || [], rr = rec.rows || [];
  el.innerHTML = `<div class="grid">
    <div class="panel c12"><div class="dhead"><h2>${title}</h2>
      <button class="v btn" data-k="${isIP ? 'ip' : 'port'}" data-val="${esc(det.v)}" data-label="${esc(isIP ? (names.get(det.v) || det.v) : det.v)}" aria-haspopup="menu">${esc(t('pop.actions'))} ▾</button></div>
      <div class="kpis" style="grid-template-columns:repeat(4,1fr);margin-top:10px">
        <div class="kpi"><div class="lab">${t('kpi.total', {r: rangeLabel()})}</div><div class="n">${fmtBytes(tot.wire)}</div><div class="d muted">${d.peak_at ? esc(t('kpi.peak', {v: fmtBps(d.peak_bps), t: fmtTime(d.peak_at, spanMs())})) : '&nbsp;'}</div></div>
        <div class="kpi"><div class="lab">${t(inSB() ? 'sb.avg' : 'kpi.now')}</div><div class="n">${fmtBps(inSB() ? tot.wire * 8 / (spanMs() / 1000) : d.now_bps)}</div><div class="d">&nbsp;</div></div>
        <div class="kpi"><div class="lab">${t(isIP ? 'det.peers' : 'det.clients')}</div><div class="n">${nf(isIP ? tot.peers + tot.hosts - 1 : tot.hosts)}</div><div class="d">&nbsp;</div></div>
        <div class="kpi"><div class="lab">${t('kpi.peers')}</div><div class="n">${nf(tot.peers)}</div><div class="d muted">${esc(t('kpi.countries', {n: nf(tot.countries)}))}</div></div>
      </div></div>
    ${fd && (fd.findings || []).length ? panel('c12', t('det.findings'), '', `<table class="fd">${findingRows(fd.findings, true, det.v)}</table>`) : ''}
    <div class="panel c12"><div class="ph"><h2>${t('ov.bw_title')}</h2><span class="sub">${lines.length ? t(d.basis === 'week' ? 'ov.bw_sub_week' : 'ov.bw_sub_prev') : ''}</span></div>
      <div class="chart" id="chDet" style="height:220px" aria-label="${esc(t('ov.bw_title'))}"></div>
      <div class="legend">${areas.map((a, i) => `<span><i style="background:${a.color}"></i>${series.names[i] === '__other__' ? esc(a.name) : V('app', series.names[i], a.name)}</span>`).join('')}</div></div>
    ${panel('c12', t('det.conv'), '', table(cv, ['col.client', 'col.server', 'col.service', 'col.country'], r => [ipCell(r.key), ipCell(r.key2), V('port', r.key3, r.key3), esc(country(r.extra))]))}
    ${isIP
      ? panel('c6', t('det.services'), '', table(s1, ['col.port', 'col.app'], r => [V('port', r.key, r.key), esc(r.extra)])) + panel('c6', t('ov.country'), '', table(s2, ['col.country'], r => [V('country', r.key, country(r.key))]))
      : panel('c6', t('det.clients'), '', table(s1, ['col.client'], r => [ipCell(r.key)])) + panel('c6', t('det.servers'), '', table(s2, ['col.server', 'col.country'], r => [ipCell(r.key), r.extra ? esc(country(r.extra)) : '']))}
    ${panel('c12', t('det.flows'), esc(t('rec.latest', {n: rr.length})), `<div style="overflow-x:auto"><table><tr><th>${t('rc.time')}</th><th>${t('rc.client')}</th><th>${t('rc.server')}</th><th class="num">${t('rc.port')}</th><th>${t('rc.app')}</th><th>${t('rc.country')}</th><th class="num">${t('rc.traffic')}</th></tr>
      ${rr.map(r => `<tr><td class="muted nw">${new Intl.DateTimeFormat(LANG, {hour: '2-digit', minute: '2-digit', second: '2-digit'}).format(new Date(r.ts))}</td><td>${ipCell(r.client)}</td><td>${ipCell(r.server)}</td><td class="num">${r.port ? V('port', r.port + '/' + (r.proto === 17 ? 'udp' : 'tcp'), r.port) : '—'}</td><td>${V('app', r.app, r.app)}</td><td>${r.dir === 3 ? `<span class="muted">${t('internal')}</span>` : (r.cc ? V('country', r.cc, country(r.cc)) : '—')}</td><td class="num">${fmtBytes(r.wire)}</td></tr>`).join('') || `<tr><td colspan="7" class="empty">${t('empty.nodata')}</td></tr>`}</table></div>`)}
  </div>`;
  tsChart($('#chDet'), {times: series.times, areas, lines});
  bindFindings(el);
};

// ------------------------------------------------------------ shell
// The sandbox bar replaces the time range while capture files are shown.
function renderSB() {
  const on = inSB() && !['ifaces', 'sources'].includes(state.v);
  $('#sbbar').hidden = !on;
  $('#range').hidden = on;
  $('#refresh').hidden = on;
  document.querySelectorAll('#nav [data-v=ifaces], #nav [data-v=sources], #nav [data-v=cleanup]').forEach(b => b.hidden = inSB() || offlineMode);
  if (!on) return;
  const r = sbRange(), span = r.to - r.from;
  const df = new Intl.DateTimeFormat(LANG, {dateStyle: 'medium', timeStyle: 'short'}), tf = new Intl.DateTimeFormat(LANG, {timeStyle: 'short'});
  const sameDay = new Date(r.from).toDateString() === new Date(r.to).toDateString();
  const names = sbInfo.files.filter(f => f.status === 'done').map(f => f.name);
  $('#sbbar').innerHTML = `<span class="sbtag">${t('sb.banner')}</span><span class="sbfiles">${names.map(esc).join(' · ')}</span>
    <span class="muted nw">${esc(df.format(r.from))} – ${esc(sameDay ? tf.format(r.to) : df.format(r.to))} (${esc(fmtDur(span))})</span>
    ${offlineMode ? '' : `<button class="btn" id="sbBack">${t('sb.back')}</button>`}`;
  if ($('#sbBack')) $('#sbBack').onclick = () => { state.ds = ''; render(); };
}
function fmtDur(ms) {
  const m = Math.round(ms / 6e4);
  return m < 120 ? t('sb.min', {n: nf(m)}) : t('sb.hours', {n: nf(m / 60, 1)});
}
async function loadSB() {
  try { const res = await fetch('/api/sandbox'); if (res.ok) sbInfo = await res.json(); } catch (e) {}
  if (state.ds && !sbInfo.ready) state.ds = '';
  if (offlineMode && sbInfo.ready) state.ds = 'sb';  // nothing else to show
}

async function render(push) {
  renderSB();
  renderFilters();
  renderFbar();
  writeHash(push);
  $('#title').textContent = t('nav.' + state.v);  // the detail view refines it
  document.title = t('nav.' + state.v) + ' · traffic66';
  document.querySelectorAll('#nav button').forEach(b => b.dataset.v === state.v ? b.setAttribute('aria-current', 'page') : b.removeAttribute('aria-current'));
  renderRange();
  document.querySelectorAll('.view').forEach(s => s.classList.toggle('on', s.id === 'v-' + state.v));
  const el = $('#v-' + state.v), seq = ++loadSeq;
  try {
    await views[state.v](el);
  } catch (e) {
    if (e.message !== 'login' && seq === loadSeq) el.innerHTML = errorBox(e);
  }
  if (seq === loadSeq) resolveNames();
}
function go(v) { state.v = v; render(true); window.scrollTo(0, 0); }

async function loadStatus() {
  try {
    const res = await fetch('/api/status');
    if (res.status === 401) { showLogin(); return; }
    const s = await res.json();
    for (const [ip, n] of Object.entries(s.hosts || {})) names.set(ip, n);
    $('#demo').hidden = !s.demo; $('#demo').textContent = t('demo.badge');
    offlineMode = !!s.offline;
    const L = s.license;
    if (L) {
      const id = `<span class="muted">${esc(t('lic.id', {id: L.installation_id}))}</span>`;
      $('#lic').innerHTML = L.kind === 'licensed' ? `<b>${esc(t('lic.ok', {c: L.customer, n: L.days}))}</b> ${id}`
        : `<b>${esc(t(L.kind === 'trial' ? 'lic.trial' : 'lic.over', {n: L.days}))}</b><br>${esc(t('lic.free'))} ${id}`;
      $('#lic').className = 'lic ' + L.kind; $('#lic').hidden = false;
    }
    $('#verNo').textContent = 'v' + s.version;
    if (s.now) clockSkew = s.now - Date.now();
    $('#self').hidden = offlineMode;  // nothing is collected
    const live = s.records_per_sec > 0.2;
    $('#self').innerHTML = `<div class="live ${live ? 'ok' : 'idle'}"><i></i>${t(live ? 'self.receiving' : 'self.idle')}</div>
      <span class="k">${t('self.ingest')}</span><span class="val">${esc(t('self.per_sec', {n: nf(s.records_per_sec, s.records_per_sec < 10 ? 1 : 0)}))}</span>
      <span class="k">${t('self.dropped')}</span><span class="val" style="${s.dropped ? 'color:var(--crit)' : ''}">${nf(s.dropped)}</span>
      <span class="k">${t('self.disk')}</span><span class="val">${fmtBytes(s.disk_bytes)}</span>
      <span class="k">${t('self.free')}</span><span class="val" ${s.disk_need > s.disk_free ? 'style="color:var(--crit)"' : ''} title="${esc(s.disk_need >= 0 ? t('self.need', {d: nf(s.retention_days), n: fmtBytes(s.disk_need)}) : t('self.need_later', {d: nf(s.retention_days)}))}">${fmtBytes(s.disk_free)}</span>
      ${s.write_errors ? `<span class="k" style="color:var(--crit)">!</span><span class="val" style="color:var(--crit)" title="${esc(s.last_error)}">${nf(s.write_errors)}</span>` : ''}`;
    $('#srcBadge').hidden = !s.source_warnings; $('#srcBadge').textContent = s.source_warnings;
    $('#findBadge').hidden = !s.findings_open; $('#findBadge').textContent = s.findings_open;
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
    loadStatus(); render();
  };
  $('#range').onclick = e => {
    const b = e.target.closest('[data-r]'); if (!b) return;
    if (b.dataset.r === 'custom') { e.stopPropagation(); openRangePop(b); return; }
    state.r = b.dataset.r; render();
  };
  document.addEventListener('click', e => { const p = $('#rangePop'); if (p && !p.hidden && !e.target.closest('#rangePop')) p.hidden = true; });
  $('#nav').onclick = e => { const b = e.target.closest('[data-v]'); if (b) go(b.dataset.v); };
  $('#share').onclick = () => { navigator.clipboard?.writeText(location.href).then(() => toast(t('top.copied')), () => toast(location.href)); };
  document.addEventListener('click', e => { if (!e.target.closest('.cols')) { const m = $('#colMenu'); if (m) m.hidden = true; } });
  readHash();
  window.addEventListener('hashchange', () => { const before = location.hash; readHash(); if (before) render(); });
  // Back and Forward return to the previous page, e.g. out of a drill-down
  window.addEventListener('popstate', () => { state.f = []; state.det = null; readHash(); render(); });
  const res = await fetch('/api/status');
  if (res.status === 401) { showLogin(); return; }
  $('#app').hidden = false;
  await Promise.all([loadStatus(), loadSB()]);
  render();
  setInterval(loadStatus, 10000);
  setInterval(() => {
    if (document.hidden || pop.style.display === 'block' || document.activeElement?.tagName === 'TEXTAREA' || document.activeElement?.id === 'q') return;
    if (state.v === 'sources' || state.v === 'records' || state.v === 'overview' || state.v === 'findings' || state.r === '15m' || state.r === '1h') render();
  }, 30000);
}
init();
})();
