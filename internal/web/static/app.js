(() => {
'use strict';

// ------------------------------------------------------------ the evaluation mark
// Every page an unregistered installation might hand to somebody else
// carries the mark; these three do not. The overview is the first thing a
// new installation sees and the mark would be the first thing it says, and
// the settings and the account are a person working on their own machine
// rather than a page anybody reads twice.
//
// It is a list of what is NOT marked on purpose: a page added later is
// marked unless somebody decides otherwise, which is the right way round —
// the cost of forgetting is then an extra mark, not a report that leaves
// the building looking finished when it is not.
const NOMARK = new Set(['overview', 'sources', 'account']);
let marked = false;  // from the licence: true until the installation is registered
let markNotice = '';  // the server's wording, so a copy says what a file says

// markTile draws the diagonal text once, as an SVG tile the page repeats.
// It is built here rather than kept as a file because the wording is
// translated, and because the colour has to follow the theme.
function markTile() {
  const text = t('mark.text');
  // A background image is its own document, so currentColor would come out
  // black there whatever the theme. The muted ink of the current theme is
  // read here and written into the tile instead, which is also why the
  // tile is rebuilt whenever the theme changes.
  const ink = getComputedStyle(document.body).getPropertyValue('--ink-3').trim() || '#8a8a8a';
  // One large glyph per tile rather than many small ones: the tile is
  // repeated across the whole view.
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="460" height="300">` +
    `<text x="230" y="150" transform="rotate(-24 230 150)" text-anchor="middle" ` +
    `font-family="system-ui,sans-serif" font-size="23" font-weight="600" ` +
    `fill="${esc(ink)}">${esc(text)}</text></svg>`;
  return `url("data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}")`;
}

// applyMark decides whether the view on screen carries the mark.
function applyMark() {
  const on = marked && !NOMARK.has(state.v);
  document.body.classList.toggle('marked', on);
  if (on) document.body.style.setProperty('--mark', markTile());
  markChip(on);
}

// The tile cannot be clicked: it lies over the whole view, and anything
// that caught clicks there would take them away from the page underneath.
// So the part somebody can act on is a small chip in the corner of the
// marked view, which says the same thing and opens the registration note.
function markChip(on) {
  let chip = $('#markChip');
  if (!on) { if (chip) chip.remove(); return; }
  if (!chip) {
    chip = document.createElement('button');
    chip.id = 'markChip';
    chip.type = 'button';
    chip.className = 'markchip';
    chip.onclick = markPrompt;
    document.body.appendChild(chip);
  }
  chip.textContent = t('mark.chip');
  chip.title = markNotice || t('mark.text');
}

// How long the full note stays away after it has been shown once. Somebody
// evaluating the program works under the mark all day; a note that opened
// on every click would be a reason to stop evaluating. Within the pause a
// click gives the one-line version instead, so the click still answers.
const MARK_PROMPT_PAUSE = 6 * 3600 * 1000;
const MARK_PROMPT_KEY = 't66.markPromptAt';

function markPromptDue(now) {
  try {
    const last = Number(localStorage.getItem(MARK_PROMPT_KEY) || 0);
    return !(last > 0 && now - last < MARK_PROMPT_PAUSE);
  } catch { return true; }  // private windows have no storage; still answer
}

function markPrompt() {
  const now = Date.now();
  if (!markPromptDue(now)) { toast(t('mark.chip_hint')); return; }
  try { localStorage.setItem(MARK_PROMPT_KEY, String(now)); } catch {}
  showMarkNote();
}

// The note itself: what the mark means, and where registering happens.
function showMarkNote() {
  let box = $('#markNote');
  if (box) box.remove();
  box = document.createElement('div');
  box.id = 'markNote';
  box.className = 'marknote';
  box.setAttribute('role', 'dialog');
  box.setAttribute('aria-modal', 'true');
  box.innerHTML = `<div class="marknote-in">
    <h3></h3><p class="muted"></p><p class="small muted"></p>
    <div class="marknote-act"><button class="btn" data-close></button>
    <a class="btn primary" href="#v=sources"></a></div></div>`;
  box.querySelector('h3').textContent = t('mark.note_title');
  box.querySelector('p.muted').textContent = t('mark.note_body');
  box.querySelector('p.small').textContent = markNotice;
  box.querySelector('[data-close]').textContent = t('mark.note_close');
  const go = box.querySelector('a.primary');
  go.textContent = t('mark.note_go');
  const close = () => box.remove();
  box.querySelector('[data-close]').onclick = close;
  go.onclick = close;
  box.onclick = e => { if (e.target === box) close(); };
  document.addEventListener('keydown', function esc(e) {
    if (e.key === 'Escape') { close(); document.removeEventListener('keydown', esc); }
  });
  document.body.appendChild(box);
  box.querySelector('[data-close]').focus();
}

// A table copied out of the page used to paste in clean, which made the
// mark a matter of not taking a screenshot. What is copied now carries the
// notice, in both the text and the HTML flavour, so it arrives wherever it
// is pasted. A copy of one short run of text — an address somebody is
// pasting into a terminal — is left alone; the notice is for a copy big
// enough to be a document.
const MARK_COPY_MIN = 120;
function onCopy(e) {
  if (!marked || !markNotice) return;
  const sel = String(document.getSelection() || '');
  if (sel.length < MARK_COPY_MIN) return;
  const note = '\n\n' + markNotice + '\n';
  try {
    e.clipboardData.setData('text/plain', sel + note);
    e.clipboardData.setData('text/html', '<pre>' + esc(sel) + '</pre><p>' + esc(markNotice) + '</p>');
    e.preventDefault();
  } catch {}  // a browser that will not let us rewrite it still copies the text
}
document.addEventListener('copy', onCopy);

// ------------------------------------------------------------ i18n
// Ordered by total number of speakers (Ethnologue).
const LANGS = [['en','English'],['zh','中文'],['hi','हिन्दी'],['es','Español'],['ar','العربية'],['fr','Français'],
  ['bn','বাংলা'],['pt','Português'],['ru','Русский'],['id','Bahasa Indonesia'],['ur','اردو'],['de','Deutsch'],['ja','日本語'],['vi','Tiếng Việt'],['ko','한국어']];
const RTL = new Set(['ar', 'ur']);
let LANG = 'en', DICT = {}, EN = {};
const t = (k, p) => {
  let s = DICT[k] ?? EN[k] ?? k;
  if (p) for (const [a, b] of Object.entries(p)) s = s.split('{' + a + '}').join(b);
  return s;
};
// the key for n of something: k_one, k_few, ... where the language has
// that form, else k itself
const plural = (k, n) => {
  let f = '';
  try { f = new Intl.PluralRules(LANG).select(n); } catch (e) {}
  return f && f !== 'other' && (DICT[k + '_' + f] ?? (LANG === 'en' ? EN[k + '_' + f] : undefined)) != null ? k + '_' + f : k;
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
  $('#refresh').textContent = t('top.refresh');
  themeLabel();
}
// the theme button shows the current theme; a click goes to the next one
function themeLabel() {
  const b = $('#theme'), cur = window.t66Theme ? t66Theme.get() : 'light';
  b.querySelector('span').textContent = t('theme.' + cur);
  b.title = t('theme.label') + ': ' + t('theme.' + cur) + ' · ' + t('theme.next');
  b.setAttribute('aria-label', b.title);
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
// emptyText is what a page with nothing to show says. For the first
// several minutes of a new installation the honest answer is not "no
// data" but "not yet": the collector is waiting for a template, for a
// sampling rate, or for the most recent minutes to settle. Saying only
// "no data" there reads as a fault, and sends somebody to change ports
// and firewall rules while the collector is about to finish.
// waitedFor says how long the collector has been at this stage, from the
// server's elapsed_sec, so "up to about 15 minutes" can be read against it.
function waitedFor(w) {
  return w.elapsed_sec > 0 ? ' ' + t('wait.elapsed', {m: Math.max(1, Math.floor(w.elapsed_sec / 60))}) : '';
}
// waitHint wraps a first-run waiting message in a prominent callout, so a
// new installation reads "not yet, and here is why" at a glance rather than
// a grey "no data". Ordinary empty periods keep the plain muted text.
function waitHint(msg, elapsed) {
  return `<span class="waithint" role="status"><span class="waithint-ic" aria-hidden="true">⏳</span><span>${msg}${elapsed ? `<span class="waithint-el">${elapsed}</span>` : ''}</span></span>`;
}
function emptyText() {
  const w = waiting;
  if (!w || !w.reason) return t('empty.nodata');
  if (w.reason === 'nothing') return waitHint(firstText());
  if (w.reason === 'templates') return waitHint(t('wait.templates', {n: nf(w.records || 0)}), waitedFor(w));
  if (w.reason === 'sampling') return waitHint(t('wait.sampling', {n: nf(w.records || 0)}), waitedFor(w));
  if (w.reason === 'settling') { const n = Math.max(1, Math.ceil((w.seconds || 0) / 60)); return t(plural('wait.settling', n), {n}); }
  return t('empty.nodata');
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
const VIEWS = ['account', 'overview', 'findings', 'topn', 'traffic', 'sankey', 'geo', 'threats', 'records', 'racks', 'devices', 'ipam', 'cleanup', 'sandbox', 'ifaces', 'billing', 'names', 'sources', 'detail'];
const RANGES = ['15m', '1h', '6h', '24h', '7d', '30d'];
const state = {v: 'overview', r: '24h', f: [], dim: 'conv', ifc: null, ifdir: 'both', sort: {k: 'wire', asc: false}, det: null};
function readHash() {
  const p = new URLSearchParams(location.hash.slice(1));
  if (VIEWS.includes(p.get('v'))) state.v = p.get('v');
  if (RANGES.includes(p.get('r'))) state.r = p.get('r');
  if (p.get('r') === 'custom' && +p.get('from') > 0 && +p.get('to') > +p.get('from')) { state.r = 'custom'; state.cf = +p.get('from'); state.ct = +p.get('to'); }
  if (p.get('dim')) state.dim = p.get('dim');
  state.tm = p.get('tm') === 'talkers' ? 'talkers' : 'table';  // Top 66 opens on the table
  state.sk = ['segment', 'conv'].includes(p.get('by')) ? p.get('by') : 'host';
  state.ra = p.get('ra') === 'client' ? 'client' : 'server';
  state.ds = p.get('ds') === 'sb' ? 'sb' : '';
  state.rl = p.get('live') === '1';
  if (p.get('if')) state.ifv = p.get('if');
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
  if (state.v === 'topn') { if (state.tm === 'talkers') p.set('tm', 'talkers'); else p.set('dim', state.dim); }
  if (state.v === 'sankey' && state.sk !== 'host') p.set('by', state.sk);
  if (state.v === 'records' && state.rl) p.set('live', '1');
  if (state.ifv && state.ifv !== 'all' && IFV_VIEWS.has(state.v)) p.set('if', state.ifv);
  if (state.v === 'traffic') { if (state.ra === 'client') p.set('ra', 'client'); }
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
// The interface picked at the top (or the one marked default) limits the
// traffic pages to the flows through it; findings, threat intel, interface
// check and settings stay as they are.
const IFV_VIEWS = new Set(['overview', 'topn', 'traffic', 'sankey', 'geo', 'records', 'detail']);
const IFV_DATA = new Set(['overview', 'topn', 'series', 'sankey', 'records', 'rings', 'geolines']);
const ifScoped = path => state.ifv && state.ifv !== 'all' && IFV_VIEWS.has(state.v) && IFV_DATA.has(path) && !inSB();
// What the licence grants, for the pages that have to grey a
// registered-only option out. The server checks it too and is the only
// thing that decides: this keeps the page from offering what would be
// refused.
let lic = null;
let waiting = null;  // what the collector is still waiting for, from /api/status
let setup = null;    // sources, UDP ports and addresses, for the first-run guide
const licOK = f => !!lic && lic.registered && (lic.features || []).includes(f);

let ifList = null, ifListKey = '';
// the interfaces of the range (busiest first) and the default one, cached a minute
async function loadIfaces() {
  const k = state.r + (state.r === 'custom' ? state.cf + '-' + state.ct : '') + Math.floor(Date.now() / 6e4);
  if (ifList && ifListKey === k) return ifList;
  try { ifList = await api('ifaces'); ifListKey = k; } catch (e) { ifList = ifList || {ifaces: []}; }
  return ifList;
}
async function renderIfsel() {
  const w = $('#ifselWrap'), show = IFV_VIEWS.has(state.v) && !inSB() && !offlineMode;
  w.hidden = !show;
  if (!show) return;
  const d = await loadIfaces(), list = d.ifaces || [];
  if (!list.length) { w.hidden = true; return; }  // e.g. local capture only: nothing to choose
  if (state.ifv === undefined) state.ifv = d.default || 'all';
  const key = f => f.exporter + '/' + f.ifindex;
  const opt = (v, l, tt) => `<option value="${esc(v)}"${tt ? ` title="${esc(tt)}"` : ''}>${esc(l)}</option>`;
  // sampled interfaces only, grouped by device; a peer only when it is the choice
  const shown = list.filter(f => !f.peer || f.default || key(f) === state.ifv), groups = new Map();
  shown.forEach(f => { const g = f.device || f.exporter; if (!groups.has(g)) groups.set(g, []); groups.get(g).push(f); });
  let html = opt('all', t('if.all'));
  groups.forEach((fs, g) => { html += `<optgroup label="${esc(g)}">${fs.map(f => opt(key(f), (f.default ? '★ ' : '') + ifName(f) + (f.tag ? ' [' + f.tag + ']' : ''), (f.device || f.exporter) + ' · ' + f.ifindex)).join('')}</optgroup>`; });
  if (state.ifv !== 'all' && !list.some(f => key(f) === state.ifv)) html += opt(state.ifv, state.ifv);
  const sel = $('#ifsel');
  sel.innerHTML = html;
  sel.value = state.ifv;
  sel.classList.toggle('on', state.ifv !== 'all');
  sel.title = sel.selectedOptions[0]?.title || '';
}
async function api(path, extra = {}, filters = state.f) {
  if (inSB() && (path === 'ifaces' || path === 'recon')) return {ifaces: []};
  // a capture file holds no device counters, so there is no bill in it
  if (inSB() && path === 'billing') return {bills: []};
  const p = new URLSearchParams(inSB() && SB_DATA.has(path) ? {ds: 'sb', ...sbRange(), ...extra} : extra.range ? extra : state.r === 'custom' ? {from: state.cf, to: state.ct, ...extra} : {range: state.r, ...extra});
  // Which clock the day and six-hour columns are cut on. Without it they
  // were cut on UTC, so a day in Tokyo arrived split across two of them
  // and neither column was a day's traffic. getTimezoneOffset counts
  // minutes behind UTC, so the sign is turned round here.
  p.set('tz', String(-new Date().getTimezoneOffset()));
  if (ifScoped(path) && !filters.some(x => x.f === 'iface')) filters = [...filters, {f: 'iface', v: state.ifv, neg: false}];
  if (filters.length) p.set('f', JSON.stringify(filters.map(x => ({f: x.f, v: x.v, neg: x.neg}))));
  const res = await fetch('/api/' + path + '?' + p);
  if (res.status === 401) { showLogin(); throw new Error('login'); }
  if (res.status === 403 && me) { const j = await res.clone().json().catch(() => ({})); if (j.error === 'change_password') { me.must_change = true; go('account'); throw new Error('login'); } }
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
// axisTicks returns round local times between from and to for the time
// axis: the smallest interval that gives at most max labels.
function axisTicks(from, to, max, span) {
  const M = 6e4, D = 864e5;
  const iv = [M, 2 * M, 5 * M, 10 * M, 15 * M, 30 * M, 60 * M, 120 * M, 180 * M, 360 * M, 720 * M, D, 2 * D, 5 * D, 7 * D, 10 * D, 15 * D];
  const fine = span > 36 * 3600e3 ? iv.filter(v => v >= D) : iv; // day labels only once a day
  const I = fine.find(v => (to - from) / v <= max) || fine[fine.length - 1];
  const off = ms => new Date(ms).getTimezoneOffset() * M; // local = utc - off
  const out = [];
  let t = Math.ceil((from - off(from)) / I) * I + off(from);
  for (; t <= to && out.length < 50; t += I) { const a = Math.round((t - off(t)) / I) * I + off(t); out.push(a); }
  return out;
}
// settledTo marks where the data stops being complete: buckets after it
// are drawn, because leaving the newest data off a live chart is its own
// kind of lie, but they are still filling. Without the mark a minute that
// is half counted looks like a fall in traffic.
// dataFrom is when the stored data begins: the buckets that end before it
// are left out (a gap), not drawn as traffic of zero. Lines marked keep
// (a baseline from another day, device counters) are not cut.
function tsChart(el, {times, areas = [], lines = [], fmtY = fmtAxisBps, fmtV = fmtBps, settledTo = 0, dataFrom = 0}) {
  const stepMs = times && times.length > 1 ? times[1] - times[0] : 0;
  if (dataFrom && times) {
    const cut = d => (d || []).map((v, i) => times[i] + stepMs <= dataFrom ? null : v);
    areas = areas.map(a => ({...a, data: cut(a.data)}));
    lines = lines.map(l => l.keep ? l : {...l, data: cut(l.data)});
  }
  const draw = () => {
    const W = el.clientWidth, H = el.clientHeight; if (W < 120 || H < 60 || !times || !times.length) { el.innerHTML = ''; return; }
    // nothing at all to draw: say so instead of an axis from 0 to 1
    const has = d => (d || []).some(v => v != null && v >= 0);
    if (!areas.some(a => has(a.data)) && !lines.some(l => has(l.data))) { el.innerHTML = `<div class="empty">${emptyText()}</div>`; return; }
    const rtl = document.documentElement.dir === 'rtl';
    const L = rtl ? 8 : 50, R = rtl ? 50 : 8, T = 8, B = 22, n = times.length;
    const x = i => rtl ? W - R - (W - L - R) * i / Math.max(1, n - 1) : L + (W - L - R) * i / Math.max(1, n - 1);
    const cum = areas.map(() => new Array(n).fill(null));
    for (let i = 0; i < n; i++) { let s = 0; areas.forEach((a, k) => { const v = a.data[i]; if (v == null || v < 0) return; s += v; cum[k][i] = s; }); }
    let top = 0;
    cum.forEach(c => c.forEach(v => { if (v != null && v > top) top = v; }));
    lines.forEach(l => l.data.forEach(v => { if (v != null && v > top) top = v; }));
    // all zero: a scale up to 1K reads as "nothing", 0..1 rounds to 0,0,1,1,1
    const Y = top > 0 ? niceMax(top * 1.05) : 1000, y = v => T + (H - T - B) * (1 - v / Y);
    const span = times[n - 1] - times[0];
    let s = `<svg viewBox="0 0 ${W} ${H}" height="${H}" role="img" aria-label="${esc(el.getAttribute('aria-label') || '')}">`;
    for (let k = 0; k <= 4; k++) {
      const yy = y(Y * k / 4);
      s += `<line class="gl" x1="${L}" x2="${W - R}" y1="${yy}" y2="${yy}"/><text class="ax" x="${rtl ? W - R + 8 : L - 8}" y="${yy + 4}" text-anchor="${rtl ? 'start' : 'end'}">${fmtY(Y * k / 4)}</text>`;
    }
    // time labels on round times (every 5, 10, 15, 30 minutes, hours,
    // days at local midnight), placed where that time falls
    const xt = ms => { const f = stepMs ? (ms - times[0]) / stepMs : 0; return rtl ? W - R - (W - L - R) * f / Math.max(1, n - 1) : L + (W - L - R) * f / Math.max(1, n - 1); };
    for (const ms of axisTicks(times[0], times[n - 1], W < 600 ? 4 : 7, span)) s += `<text class="ax" x="${xt(ms).toFixed(1)}" y="${H - 5}" text-anchor="middle">${fmtAxisTime(ms, span)}</text>`;
    // The buckets still filling are not stacked: a stack that falls off a
    // cliff at the right edge reads as traffic stopping. They stay in the
    // tooltip and under the hatching; lines draw them dashed.
    const firstUnsettled = settledTo ? times.findIndex(v => v + stepMs > settledTo) : -1;
    const stackTo = firstUnsettled > 0 ? firstUnsettled : n;
    areas.forEach((a, k) => {
      let seg = [], paths = '';
      const flush = () => {
        if (seg.length < 1) { seg = []; return; }
        let up = '', lo = '';
        seg.forEach(i => { up += (up ? 'L' : 'M') + x(i).toFixed(1) + ' ' + y(cum[k][i]).toFixed(1); });
        for (let j = seg.length - 1; j >= 0; j--) { const i = seg[j]; const b = k ? (cum[k - 1][i] || 0) : 0; lo += 'L' + x(i).toFixed(1) + ' ' + y(b).toFixed(1); }
        paths += up + lo + 'Z'; seg = [];
      };
      for (let i = 0; i < stackTo; i++) { if (cum[k][i] == null) flush(); else seg.push(i); }
      flush();
      s += `<path d="${paths}" fill="${a.color}" fill-opacity=".88" stroke="var(--surface)" stroke-width="1"/>`;
    });
    lines.forEach(l => {
      let d = '', pen = false, area = '', seg = [];
      const close = () => { if (seg.length > 1) area += 'M' + seg.map(i => x(i).toFixed(1) + ' ' + y(l.data[i]).toFixed(1)).join('L') + `L${x(seg[seg.length - 1]).toFixed(1)} ${y(0)}L${x(seg[0]).toFixed(1)} ${y(0)}Z`; seg = []; };
      const lineTo = l.keep ? n : stackTo;
      l.data.forEach((v, i) => { if (i >= lineTo) return; if (v == null || v < 0) { pen = false; close(); return; } d += (pen ? 'L' : 'M') + x(i).toFixed(1) + ' ' + y(v).toFixed(1); pen = true; seg.push(i); });
      close();
      // the unsettled tail, from the last settled point, thin and dashed
      let tail = '';
      if (lineTo < n) { let tp = false; for (let i = lineTo - 1; i < n; i++) { const v = l.data[i]; if (v == null || v < 0) { tp = false; continue; } tail += (tp ? 'L' : 'M') + x(i).toFixed(1) + ' ' + y(v).toFixed(1); tp = true; } }
      if (l.fill && area) s += `<path d="${area}" fill="${l.color}" fill-opacity=".14" stroke="none"/>`;
      s += `<path d="${d}" fill="none" stroke="${l.color}" stroke-width="2" ${l.dash ? 'stroke-dasharray="4 3"' : ''}/>`;
      if (tail) s += `<path class="tail" d="${tail}" fill="none" stroke="${l.color}" stroke-width="1.5" stroke-opacity=".6" stroke-dasharray="2 3"/>`;
    });
    // The tail that is still filling: a hatched wash and a line where the
    // complete data ends, rather than a chart that stops short of now.
    // A bucket is complete only when the whole of it is behind the
    // boundary, so it is the bucket's end that is compared, not its start:
    // at 10:58 with a settle of a minute, the 10:55 bucket still has two
    // minutes to go.
    // starting half a bucket back, so a single filling bucket is visible
    if (firstUnsettled >= 0 && firstUnsettled < n) {
      const xs = n > 1 ? (x(firstUnsettled) + x(Math.max(0, firstUnsettled - 1))) / 2 : x(0);
      const xa = Math.min(xs, x(n - 1)), xb = Math.max(xs, x(n - 1));
      const xEdge = rtl ? xb : xa;
      s += `<defs><pattern id="uns" width="7" height="7" patternUnits="userSpaceOnUse" patternTransform="rotate(45)">` +
        `<line x1="0" y1="0" x2="0" y2="7" stroke="var(--ink-3)" stroke-width="1" opacity=".5"/></pattern></defs>` +
        `<rect x="${xa.toFixed(1)}" y="${T}" width="${Math.max(0, xb - xa).toFixed(1)}" height="${H - T - B}" fill="url(#uns)" opacity=".5" pointer-events="none"/>` +
        `<line x1="${xEdge.toFixed(1)}" x2="${xEdge.toFixed(1)}" y1="${T}" y2="${H - B}" stroke="var(--ink-3)" stroke-width="1" stroke-dasharray="3 3"/>`;
    }
    s += `<line class="xh" x1="0" x2="0" y1="${T}" y2="${H - B}" stroke="var(--ink-2)" stroke-width="1" style="display:none"/><rect class="hit" x="${L}" y="${T}" width="${W - L - R}" height="${H - T - B}" fill="transparent"/></svg>`;
    el.innerHTML = s;
    const svg = el.querySelector('svg'), xh = svg.querySelector('.xh'), hit = svg.querySelector('.hit');
    hit.onmousemove = e => {
      const r = svg.getBoundingClientRect();
      let fx = (e.clientX - r.left - L) / (W - L - R); if (rtl) fx = (W - R - (e.clientX - r.left)) / (W - L - R);
      const i = Math.max(0, Math.min(n - 1, Math.round(fx * (n - 1))));
      xh.setAttribute('x1', x(i)); xh.setAttribute('x2', x(i)); xh.style.display = '';
      let h = `<div class="t">${fmtTime(times[i], span)}${settledTo && times[i] + (n > 1 ? times[1] - times[0] : 0) > settledTo ? ` <span class="muted">· ${esc(t('ch.unsettled'))}</span>` : ''}</div>`, tot = 0, any = false;
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
  if (!tot) { el.innerHTML = `<div class="empty">${emptyText()}</div>`; return; }
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
  if (a === 'records') { state.f = state.f.filter(x => x.f !== ctx.k); state.f.push({f: ctx.k, v: ctx.v, neg: false}); state.rl = false; go('records'); return; }
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

// what the dashed line and the change compare with: the same time
// yesterday (up to a day), last week (up to a week), or the days before
const baseText = (d, kind) => {
  const k = {day: 'day', week: 'week', prev: 'prev'}[d.basis];
  return k ? t({line: 'base.', kpi: 'kpi.vs_', sub: 'ov.bw_sub_'}[kind] + k, {n: d.base_days || ''}) : '';
};
// "Nothing received yet": the default ports are in the text; other ports
// are named as they were given to -listen.
function firstText() {
  const l = setup?.listen || [], ports = l.map(x => x.port).sort().join(',');
  if (!l.length || ports === '2055,4739,6343') return t('empty.first');
  return t('ob.step1') + ' ' + t('ob.ports') + ' ' + l.map(x => x.name + ' ' + x.port).join(', ');
}
// A new installation with no device yet: say exactly where to send flows,
// with this machine's addresses and the collectors' ports, ready to copy.
function onboardGuide(tot) {
  const s = setup;
  if (!s || s.demo || s.offline || inSB() || s.sources > 0 || (tot && tot.wire > 0) || !(s.listen || []).length) return '';
  const ips = (s.addrs || []).length ? s.addrs : [location.hostname];
  const code = v => `<code class="ob-c">${esc(v)}</code><button class="ob-copy" data-copy="${esc(v)}">${esc(t('ob.copy'))}</button>`;
  const ports = s.listen.map(l => `<span class="ob-p">${esc(l.name)} ${code(l.port)}</span>`).join(' ');
  // ready-made export commands for this machine's address and ports
  const port = n => (s.listen.find(l => l.name === n) || s.listen[0]).port, ip = ips[0];
  const cfg = [
    `/ip traffic-flow target add dst-address=${ip} port=${port('netflow')} version=9`,
    `set protocols sflow collector ${ip} udp-port ${port('sflow')}`,
    `sflow destination ${ip} ${port('sflow')}`,
    `softflowd -i eth0 -n ${ip}:${port('ipfix')} -v 10 -t maxlife=60`,
  ];
  const lab = ['MikroTik, NetFlow v9', 'Juniper, sFlow', 'Arista, sFlow', 'Linux, IPFIX'];
  return `<div class="panel c12 onboard"><div class="ph"><h2>${esc(t('ob.title'))}</h2></div>
    <ol class="ob">
      <li>${esc(t('ob.step1'))}<div class="ob-row">${ips.map(code).join(' ')}</div><div class="ob-row">${esc(t('ob.ports'))} ${ports}</div></li>
      <li>${esc(t('ob.try'))}${cfg.map((c, i) => `<div class="ob-row"><span class="muted">${esc(lab[i])}</span> ${code(c)}</div>`).join('')}
        <div class="ob-row"><a href="https://github.com/githubflyideas/traffic66#device-configuration" target="_blank" rel="noopener">Cisco, Huawei NetStream, Arista …</a></div></li>
      <li>${esc(t('ob.step2'))} <a href="#v=sources" data-obgo="sources">${esc(t('nav.sources'))}</a></li>
    </ol></div>`;
}
document.addEventListener('click', e => {
  const b = e.target.closest('.ob-copy');
  if (b) navigator.clipboard?.writeText(b.dataset.copy).then(() => toast(t('top.copied')), () => toast(b.dataset.copy));
  const g = e.target.closest('[data-obgo]');
  if (g) { e.preventDefault(); go(g.dataset.obgo); }
});
views.overview = async (el) => {
  // total, or only inbound or only outbound traffic (direction from your networks' point of view)
  const od = ['inbound', 'outbound'].includes(state.od) ? state.od : '';
  const [d, ifs, fd] = await Promise.all([api('overview', {}, od ? [...state.f, {f: 'dir', v: od, neg: false}] : state.f), inSB() ? null : loadIfaces(), api('findings', {limit: 3}).catch(() => null)]);
  // the interface chart: the interface picked at the top, else the default, else the busiest
  const ifl = ifs?.ifaces || [], ikey = f => f.exporter + '/' + f.ifindex;
  const ifk = state.ifv && state.ifv !== 'all' ? state.ifv : (ifs?.default || ((ifl.find(f => !f.peer) || ifl[0]) ? ikey(ifl.find(f => !f.peer) || ifl[0]) : ''));
  const ifSel = ifl.find(f => ikey(f) === ifk);
  const [iIn, iOut] = ifk && !inSB() ? await Promise.all(['if_in', 'if_out'].map(by => api('series', {by, measure: 'wire', top: 20, dup: 1}, [...state.f, {f: 'iface', v: ifk, neg: false}]).catch(() => null))) : [null, null];
  const tot = d.totals, base = d.base_totals, basis = d.basis;
  // a baseline that was nearly idle gives figures like +57,695%: more than
  // ten times the baseline reads ">10×", and a baseline under 1 kB is no
  // baseline at all (as in the terminal UI)
  const change = basis && base.wire >= 1000 ? (tot.wire - base.wire) / base.wire : null;
  const changeTxt = change == null ? '' : change > 9 ? '>10×' : fmtPct(change, 0);
  // accuracy: worst interface deviation relative to its statistical error
  let acc = {n: '—', d: t('kpi.no_counters'), kind: null};
  const withCtr = (ifs?.ifaces || []).filter(f => f.has_counters);
  if (withCtr.length) {
    // a direction with counters but no flows at all is an export that is
    // not happening, not a deviation: it is left out of the worst figure
    let worst = 0, bad = false, any = false;
    withCtr.forEach(f => { const dv = ifDev(f); if (dv == null) return; any = true; worst = Math.max(worst, dv); if (dv > Math.max(0.02, 2 * f.stat_err)) bad = true; });
    acc = any ? {n: nf(worst * 100, 1) + '<small>%</small>', d: bad ? t('kpi.check') : t('kpi.trust'), kind: bad ? 'warn' : 'ok'} : {n: '—', d: t('kpi.no_flow'), kind: null};
  }
  const peakTxt = d.peak_at ? t('kpi.peak', {v: fmtBps(d.peak_bps), t: fmtTime(d.peak_at, spanMs())}) : '';
  const series = d.series || {times: [], names: [], values: []};
  const areas = series.names.map((n, i) => ({name: appLabel(n), color: color(i, n), data: series.values[i]}));
  const lines = d.baseline && basis ? [{name: baseText(d, 'line'), color: 'var(--base)', dash: true, keep: true, data: d.baseline}] : [];
  const dirParts = (d.dir || []).map((p, i) => ({n: t('dir.' + p.key), v: p.wire, c: color(i), val: DIRS[p.key]}));
  const protoParts = (d.proto || []).map((p, i) => ({n: p.key === '__other__' ? t('other') : proto(+p.key), v: p.wire, c: color(i, p.key), val: p.key === '__other__' ? null : p.key}));
  const cl = d.top_clients || [], cmax = Math.max(1, ...cl.map(r => r.wire));
  const sv = d.top_services || [], smax = Math.max(1, ...sv.map(r => r.wire));
  el.innerHTML = `<div class="grid">
    ${onboardGuide(tot)}
    <div class="panel c12"><div class="kpis">
      ${inSB() ? `<div class="kpi"><div class="lab">${t('sb.avg')}</div><div class="n">${fmtBps(tot.wire * 8 / (spanMs() / 1000))}</div><div class="d muted">&nbsp;</div></div>`
        : `<div class="kpi"><div class="lab">${t('kpi.now')}</div><div class="n">${fmtBps(d.now_bps)}</div><div class="d ${change > 0.1 ? 'up' : 'muted'}">${change == null ? '&nbsp;' : esc(baseText(d, 'kpi').split('{p}').join(changeTxt))}</div></div>`}
      <div class="kpi"><div class="lab">${t('kpi.total', {r: rangeLabel()})}</div><div class="n">${fmtBytes(tot.wire)}</div><div class="d muted">${esc(peakTxt)}</div></div>
      <div class="kpi"><div class="lab">${t('kpi.hosts')}</div><div class="n">${nf(tot.hosts)}</div><div class="d muted">&nbsp;</div></div>
      <div class="kpi"><div class="lab">${t('kpi.peers')}</div><div class="n">${nf(tot.peers)}</div><div class="d muted">${esc(t('kpi.countries', {n: nf(tot.countries)}))}</div></div>
      <div class="kpi"><div class="lab">${t('kpi.accuracy')}</div><div class="n">${acc.n}</div><div class="d">${acc.kind ? status(acc.kind, acc.d) : `<span class="muted">${esc(acc.d)}</span>`}</div></div>
    </div></div>
    ${iIn ? `<div class="panel c12"><div class="ph"><h2>${t('if.bw')} · ${esc(ifSel ? ifName(ifSel) : ifk)}${ifSel?.tag ? `<span class="iftag">${esc(ifSel.tag)}</span>` : ''}</h2><span class="sub">${esc(t('ov.if_sub'))}</span></div>
      <div class="chart" id="chIf" style="height:220px" aria-label="${esc(t('if.bw'))}"></div>
      <div class="legend"><span><i style="background:${IF_IN}"></i>${t('if.in')}</span><span><i style="background:${IF_OUT}"></i>${t('if.out')}</span></div></div>` : ''}
    <div class="panel c12"><div class="ph"><h2>${t({'': 'ov.bw_all', inbound: 'ov.bw_in', outbound: 'ov.bw_out'}[od])}</h2><span class="sub">${lines.length ? esc(baseText(d, 'sub')) : ''}</span>
      <div class="seg" role="group" style="margin-inline-start:auto">${[['', 'ov.dir_total'], ['inbound', 'dir.2'], ['outbound', 'dir.1']].map(([k, l]) => `<button data-od="${k}" aria-pressed="${od === k}">${t(l)}</button>`).join('')}</div></div>
      <div class="chart" id="chStack" style="height:250px" aria-label="${esc(t('ov.bw_title'))}"></div>
      <div class="legend">${areas.map((a, i) => `<span><i style="background:${a.color}"></i>${series.names[i] === '__other__' ? esc(a.name) : V('app', series.names[i], a.name)}</span>`).join('')}${lines.length ? `<span><i class="dash"></i>${esc(lines[0].name)}</span>` : ''}</div></div>
    ${panel('c6', t('ov.dir'), '', '<div class="donut" id="dDir"></div>')}
    ${panel('c6', t('ov.proto'), '', '<div class="donut" id="dProto"></div>')}
    ${panel('c6', t('ov.top_clients'), t('ov.top_clients_sub'), `<table><tr><th></th><th>${t('col.client')}</th><th class="num">${t('col.traffic')}</th><th style="width:34%">${t('col.share')}</th></tr>
      ${cl.map((r, i) => `<tr><td class="rank">${i + 1}</td><td>${ipCell(r.key)}</td><td class="num">${fmtBytes(r.wire)}</td><td>${bar(r.wire, cmax)}</td></tr>`).join('') || `<tr><td colspan="4" class="empty">${emptyText()}</td></tr>`}</table>`)}
    ${panel('c6', t('ov.top_services'), t('ov.top_services_sub'), `<table><tr><th></th><th>${t('col.server')}</th><th>${t('col.service')}</th><th class="num">${t('col.traffic')}</th><th style="width:24%">${t('col.share')}</th></tr>
      ${sv.map((r, i) => `<tr><td class="rank">${i + 1}</td><td>${ipCell(r.key)}</td><td class="nw">${V('port', r.key2, r.key2)} <span class="muted">${esc(r.extra)}</span></td><td class="num">${fmtBytes(r.wire)}</td><td>${bar(r.wire, smax)}</td></tr>`).join('') || `<tr><td colspan="5" class="empty">${emptyText()}</td></tr>`}</table>`)}
    ${fd ? findingsPanel(fd) : ''}
  </div>`;
  tsChart($('#chStack'), {times: series.times, areas, lines, settledTo: series.settled_to, dataFrom: series.data_from});
  if (iIn) {
    const pk = s => { const i = (s?.names || []).indexOf(ifk); return i < 0 ? (s?.times || []).map(() => 0) : s.values[i]; };
    tsChart($('#chIf'), {times: iIn.times, settledTo: iIn.settled_to, dataFrom: iIn.data_from, lines: [{name: t('if.in'), color: IF_IN, data: pk(iIn), fill: true}, {name: t('if.out'), color: IF_OUT, data: pk(iOut), fill: true}]});
  }
  el.querySelectorAll('[data-od]').forEach(b => b.onclick = () => { state.od = b.dataset.od; render(); });
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
    case 'fastscan': return tHtml('fd.k.fastscan', {src, n: nf(e.targets), p: nf(e.ports)});
    case 'beacon': return tHtml('fd.k.beacon', {src, dst, s: nf(Math.round(e.period_s || 0)), n: nf(e.checkins)});
    case 'chain': return tHtml('fd.k.chain', {b: ipCell(e.via), n: nf(e.onward), port});
    case 'llm': return tHtml('fd.k.llm', {src, org: esc(e.org || ''), bytes: esc(fmtBytes(e.bytes))});
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
    case 'fastscan':
      parts.push(esc(t('fd.e.pkts', {n: nf(e.pkts)})), esc(t('fd.e.avg', {n: nf(e.avg)})));
      if ((e.examples || []).length) parts.push(tHtml('fd.e.examples', {list: list(e.examples)}));
      break;
    case 'beacon': parts.push(esc(t('fd.e.jitter', {n: nf((e.jitter || 0) * 100, 1)})), esc(t('fd.e.bytes', {n: fmtBytes(e.bytes)}))); if (e.org) parts.push(esc(e.org)); break;
    case 'chain':
      parts.push(tHtml('fd.e.reached_by', {list: list(e.from)}), tHtml('fd.e.targets', {list: list(e.to)}));
      parts.push(esc(t('fd.e.hops', {a: e.in_port, b: e.out_port})));
      break;
    case 'llm': parts.push(esc(t('fd.e.up', {n: fmtBytes(e.bytes)})), esc(t('fd.e.minutes', {n: nf(e.minutes)}))); break;
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
  if (!d.names.length) { el.querySelector('.chart').innerHTML = `<div class="empty">${emptyText()}</div>`; return; }
  // with thousands of hosts the rest dwarfs the top 8: draw only those and
  // give the rest's total in the legend
  const hideOther = by === 'client' || by === 'server';
  const all = d.names.map((n, i) => ({n, name: seriesLabel(by, n), color: color(i, n), data: d.values[i], tot: d.totals?.[i]}));
  const areas = all.filter(a => !(hideOther && a.n === '__other__'));
  tsChart(el.querySelector('.chart'), {times: d.times, areas, settledTo: d.settled_to, dataFrom: d.data_from, fmtV: pk ? fmtPps : fmtBps});
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
    s += `<rect class="hit" x="${L}" y="${T}" width="${W - L - R}" height="${H - T - B}" fill="transparent"/></svg>`;
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
  const [da, db] = await Promise.all([api('rings', {inner: ra[0], outer: ra[1]}), api('rings', {inner: 'service', outer: 'server'})]);
  await lookupNames([...da.inner, ...da.outer, ...db.inner, ...db.outer].map(r => r.k).filter(k => /^[0-9a-f.:]+$/i.test(k)));
  const swap = (id, cur, opts) => `<div class="seg" role="group">${opts.map(([v, lab]) => `<button data-${id}="${v}" aria-pressed="${cur === v}">${esc(lab)}</button>`).join('')}</div>`;
  el.innerHTML = `<div class="grid">
    <div class="panel c6"><div class="ph"><h2>${t('rg.hosts')}</h2><div class="tools" style="margin-inline-start:auto">${swap('ra', ra[0], [['server', t('rg.in_server')], ['client', t('rg.in_client')]])}</div></div>
      <div class="sub" style="margin:-4px 0 6px">${esc(t(ra[0] === 'server' ? 'rg.hosts_sub_s' : 'rg.hosts_sub_c'))}</div><div class="rings" id="rgA"></div></div>
    <div class="panel c6"><div class="ph"><h2>${t('rg.svc_title')}</h2></div>
      <div class="sub" style="margin:-4px 0 6px">${esc(t('rg.svc_one'))}</div><div class="rings" id="rgB"></div></div></div>`;
  el.querySelectorAll('[data-ra]').forEach(b => b.onclick = () => { state.ra = b.dataset.ra; render(); });
  drawRings($('#rgA'), da, ra[0], ra[1]);
  drawRings($('#rgB'), db, 'service', null);  // one ring: the services
};
function drawRings(el, d, inDim, outDim) {
  if (!d.total) { el.innerHTML = `<div class="empty">${emptyText()}</div>`; return; }
  const draw = () => {
    const W = el.clientWidth; if (W < 200) return;
    // room for the outer labels on both sides
    const SW = Math.min(W, 600), R = Math.max(80, Math.min(170, SW / 2 - 120)), SH = 2 * R + 60, c = SW / 2, cy = SH / 2, r0 = outDim ? R * .34 : R * .5, r1 = outDim ? R * .70 : R;
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
      if (outDim) d.outer.filter(o => o.p === r.k).forEach((o, j) => { const b1 = b + o.v / tot * Math.PI * 2; segs.push({ring: 1, k: o.k, p: r.k, v: o.v, a0: b, a1: b1, j}); b = b1; });
      a = a1;
    });
    segs.forEach((g, n) => {
      const fill = col[g.ring ? g.p : g.k], op = g.ring ? (g.k === '__other__' ? .28 : (g.j % 2 ? .55 : .75)) : 1;
      s += `<path d="${arc(g.ring ? r1 : r0, g.ring ? R : r1, g.a0, g.a1)}" fill="${fill}" fill-opacity="${op}" stroke="var(--surface)" stroke-width="1" data-n="${n}"/>`;
      const mid = (g.a0 + g.a1) / 2, span = g.a1 - g.a0;
      // a horizontal label inside the inner ring: as wide as the band allows at that angle
      // where in the segment the band is widest horizontally: its middle, or the top or bottom of the ring if it spans them
      const rm = (r0 + r1) / 2, room = x => Math.min((r1 - r0) / Math.max(Math.abs(Math.sin(x)), .25), rm * 2 * Math.min(x - g.a0, g.a1 - x));
      let la = mid;
      for (const x of [0, Math.PI, 2 * Math.PI]) if (x > g.a0 + .25 && x < g.a1 - .25 && room(x) > room(la)) la = x;
      const fit = Math.floor((room(la) - 6) / 6.2);
      const inLab = g.ring ? '' : seriesLabel(inDim, g.k), nl = [...inLab].length;
      // whole names only, or a long one cut to at least 8 characters; the legend has the rest
      if (!g.ring && g.k !== '__other__' && (nl <= fit || fit >= 8)) {
        const [x, y] = pt(rm, la);
        labels += `<text x="${x}" y="${y}" text-anchor="middle" dominant-baseline="middle" class="rgin">${esc(short(inLab, Math.min(fit, 16)))}</text>`;
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
const modeSeg = () => `<div class="seg" role="group"><button data-tm="table" aria-pressed="${state.tm !== 'talkers'}">${t('topn.table')}</button><button data-tm="talkers" aria-pressed="${state.tm === 'talkers'}">${t('topn.talkers')}</button></div>`;
function bindMode(el) { el.querySelectorAll('[data-tm]').forEach(b => b.onclick = () => { state.tm = b.dataset.tm; render(); }); }
views.topn = el => {
  if (!DIMS.includes(state.dim)) state.dim = 'conv';
  return state.tm === 'talkers' ? talkers(el) : topTable(el, state.dim);
};
async function talkers(el) {
  const [cl, srv, ov] = await Promise.all([api('topn', {dim: 'client', limit: 30}), api('topn', {dim: 'server', limit: 30}), api('overview')]);
  const tot = ov.totals || {};
  const table = (rows, head, f) => `<table class="talk"><tr><th></th><th>${t(head)}</th><th class="num">${t('col.traffic')}</th><th class="num">${t('col.pkts')}</th><th class="num">${t('col.flows')}</th></tr>
    ${(rows || []).map((r, i) => `<tr><td class="rank">${i + 1}</td><td>${ipCell(r.key, f)}</td><td class="num">${fmtBytes(r.wire)}</td><td class="num">${nf(r.pkts)}</td><td class="num">${nf(r.flows)}</td></tr>`).join('') || `<tr><td colspan="5" class="empty">${emptyText()}</td></tr>`}
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
    ${rows.map((r, i) => `<tr><td class="rank">${i + 1}</td>${cell(r).map(c => `<td>${c}</td>`).join('')}<td class="num">${fmtBytes(r.wire)}</td><td class="num muted">${nf(r.wire / Math.max(1, total) * 100, 1)}</td><td>${bar(r.wire, max)}</td><td class="num">${nf(r.pkts)}</td><td class="num">${nf(avg(r))}&nbsp;B</td>${showPeers ? `<td class="num">${nf(r.peers)}</td>` : ''}<td class="num">${nf(r.flows)}</td></tr>`).join('') || `<tr><td colspan="11" class="empty">${emptyText()}</td></tr>`}
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
    <div class="chart sankey" id="chSankey" style="height:${mode === 'segment' ? 550 : 650}px" aria-label="${esc(title)}"></div></div>`;
  el.querySelectorAll('[data-sk]').forEach(b => b.onclick = () => { state.sk = b.dataset.sk; render(); });
  const sEl = $('#chSankey');
  if (!(d.seg_app || []).length) { sEl.innerHTML = `<div class="empty">${emptyText()}</div>`; return; }
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
    const nodeW = 12, pad = 14, lw = mode === 'segment' ? 118 : 200, rw = conv ? 200 : 130, colX = rtl ? [W - lw - 12, 0, rw - 12] : [lw, 0, W - rw];
    // the middle column (service, port) belongs to the server side: two thirds of the way across
    colX[1] = Math.round(colX[0] + (colX[2] - colX[0]) * 0.68);
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
    sEl.querySelectorAll('path[data-l]').forEach(p => { p.onmousemove = e => showTip(e, `<div class="t">${esc(p.dataset.l)}</div><b>${fmtBytes(+p.dataset.v)}</b> <span class="muted">L${+p.dataset.lv + 1} · ${esc(t('sb.avg'))} ${fmtBps(+p.dataset.v * 8 / Math.max(1, spanMs() / 1000))}</span>`); p.onmouseleave = hideTip; });
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
const MAP_STEPS = ['var(--m1)', 'var(--m2)', 'var(--m3)', 'var(--m4)', 'var(--m5)'];
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
      <span style="margin-inline-start:12px"><i style="background:var(--map-none)"></i> <span class="muted">${t('geo.map_none')}</span></span></div>${lineLegend(w, gl)}`;
  const total = rows.reduce((a, r) => a + r.wire, 0) || 1;
  el.querySelectorAll('[data-cc]').forEach(p => {
    const cc = p.dataset.cc, v = by.get(cc) || 0;
    p.onmousemove = e => showTip(e, `<div class="t">${esc(country(cc))}</div><b>${v ? fmtBytes(v) : t('geo.map_none')}</b>${v ? ` <span class="muted">${nf(100 * v / total, 1)}%</span>` : ''}`);
    p.onmouseleave = hideTip;
    if (v) p.onclick = e => { e.stopPropagation(); hideTip(); openPop({k: 'country', v: cc, label: country(cc)}, {left: e.clientX, right: e.clientX, top: e.clientY, bottom: e.clientY}); };
    else p.style.cursor = 'default';
  });
  el.querySelectorAll('.mlines path').forEach(p => {
    p.onmousemove = e => showTip(e, `<div class="t">${esc(country(p.dataset.from))} ↔ ${esc(country(p.dataset.to))}</div><b>${fmtBytes(+p.dataset.v)}</b> <span class="muted">L${+p.dataset.lv + 1} · ${esc(t('sb.avg'))} ${fmtBps(+p.dataset.v * 8 / Math.max(1, spanMs() / 1000))}</span>`);
    p.onmouseleave = hideTip;
  });
  const mp = el.querySelector('#mappal');
  if (mp) mp.onchange = () => { try { localStorage.setItem('t66.mappal', mp.value); } catch (e) {} drawMap(el, w, rows, gl); };
  if (gl && !(gl.origins || []).length) el.insertAdjacentHTML('beforeend', `<p class="muted maphint">${esc(t('geo.lines_hint'))}</p>`);
}
// Lines from the countries of the internal networks to the remote
// countries. How much traffic is said twice: five steps of colour and
// three widths. One encoding alone does not carry it — colour has to be
// judged against a choropleth that is itself five steps of colour, and
// width alone runs out of room where the lines cross.
//
// The steps are log-spaced over the lines' own range, the same way the
// map's are over the countries': the quietest line and the busiest are
// usually orders of magnitude apart, and a linear scale would put every
// line but one in the bottom step.
// 15 levels: three widths times five shades, light to dark. A line's level
// is set on a log scale of its share of the busiest line (log10(v/vmax)
// over the lines' own range, cut into 15 equal steps), not by quantiles:
// a few huge flows then do not flatten the rest, and equal traffic always
// gets an equal level. Level l -> width LINE_W[floor(l/5)], shade l%5.
// The palette follows the theme unless one is picked in the legend.
const LINE_W = [1.5, 4, 8];
const LINE_PAL = {
  gb: ['#7fd6a0', '#3fb8a0', '#1f97b5', '#1f6fb8', '#163f8f'],
  or: ['#ffc266', '#ff9a3c', '#f26a2e', '#d63a2a', '#9e1622'],
  pp: ['#f7a8d0', '#e57ab8', '#c34fa8', '#8e3a9e', '#5a2585']
};
// theme -> palette when the choice is "follow theme"
const THEME_PAL = {light: 'or', black: 'pp', teal: 'gb', orange: 'or'};
const curTheme = () => window.t66Theme ? t66Theme.get() : 'light';
function linePalName() {
  let v = 'auto';
  try { v = localStorage.getItem('t66.mappal') || 'auto'; } catch (e) {}
  return LINE_PAL[v] ? v : THEME_PAL[curTheme()] || 'or';
}
function linePal() {
  const p = LINE_PAL[linePalName()];
  // on the black theme the two darkest shades would sink into the
  // background: lift them towards white so every level stays visible
  return curTheme() === 'black' ? p.map((c, i) => i >= 3 ? `color-mix(in oklab, ${c} ${i === 3 ? 75 : 60}%, #fff)` : c) : p;
}
function lineLevel(v, vmin, vmax) {
  if (!(v > 0) || !(vmax > vmin)) return v > 0 ? 14 : 0;
  const lo = Math.log10(vmin), hi = Math.log10(vmax);
  return Math.max(0, Math.min(14, Math.floor(15 * (Math.log10(v) - lo) / (hi - lo))));
}
// lower byte bound of level l (inverse of lineLevel)
const levelFrom = (l, vmin, vmax) => Math.pow(10, Math.log10(vmin) + (Math.log10(vmax) - Math.log10(vmin)) * l / 15);
const lineStyle = l => `--lw:${LINE_W[Math.floor(l / 5)]}px;--lc:${linePal()[l % 5]}`;
const geoLines = (w, gl) => (gl && gl.lines || []).filter(l => w.p[l.from] && w.p[l.to]).slice(0, 60);

// The lines have a scale of their own, over their own range: a line is one
// country pair's traffic and a country's fill is all of its, so the two
// scales do not meet and one legend cannot speak for both.
function lineLegend(w, gl) {
  const vals = geoLines(w, gl).map(l => l.v).filter(x => x > 0);
  if (!vals.length) return '';
  const vmax = Math.max(...vals), vmin = Math.min(...vals), one = !(vmax > vmin);
  const cur = (() => { try { return localStorage.getItem('t66.mappal') || 'auto'; } catch (e) { return 'auto'; } })();
  let g = '<div class="lgrid">';
  for (let l = 0; l < 15; l++) {
    const rng = one ? fmtBytes(vmax) : `${fmtBytes(levelFrom(l, vmin, vmax))} – ${fmtBytes(levelFrom(l + 1, vmin, vmax))}`;
    g += `<span class="lcell" title="L${l + 1}: ${esc(rng)}"><i class="lw" style="${lineStyle(l)}"></i><small>${esc(rng)}</small></span>`;
  }
  g += '</div>';
  return `<div class="maplegend mlegend"><span class="muted">${t('geo.lines_scale')}</span>
    <label class="muted" style="margin-inline-start:auto">${esc(t('geo.line_pal'))} <select id="mappal">${['auto', 'gb', 'or', 'pp'].map(k => `<option value="${k}"${k === cur ? ' selected' : ''}>${esc(t('geo.pal_' + k))}</option>`).join('')}</select></label></div>${g}`;
}
function mapLines(w, gl) {
  const lines = geoLines(w, gl);
  if (!lines.length) return '';
  const pos = lines.map(l => l.v).filter(x => x > 0);
  const vmax = Math.max(0, ...pos), vmin = pos.length ? Math.min(...pos) : 0;
  let s = '<g class="mlines">';
  [...lines].sort((a, b) => a.v - b.v).forEach(l => {
    const [x0, y0] = w.p[l.from], [x1, y1] = w.p[l.to], dx = x1 - x0, dy = y1 - y0, dist = Math.hypot(dx, dy);
    // bend each line upwards by a fifth of its length
    const mx = (x0 + x1) / 2 + dy * .2 * Math.sign(dx || 1) * -1, my = (y0 + y1) / 2 - Math.abs(dx) * .2 - dist * .05;
    const lv = lineLevel(l.v, vmin, vmax), lw = LINE_W[Math.floor(lv / 5)];
    s += `<path d="M${x0} ${y0}Q${mx.toFixed(1)} ${my.toFixed(1)} ${x1} ${y1}" class="lv${lv}" style="${lineStyle(lv)}" data-lv="${lv}" data-from="${l.from}" data-to="${l.to}" data-v="${l.v}"/>`;
    // the dot at the far end is sized from the line's own width
    s += `<circle cx="${x1}" cy="${y1}" r="${(lw / 2 + 1.5).toFixed(1)}" class="mend" style="--lc:${linePal()[lv % 5]}"/>`;
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
  if (w) drawMap($('#wmap'), w, all, gl); else $('#wmap').innerHTML = `<div class="empty">${emptyText()}</div>`;
  asCharts.forEach(([by, m], i) => fillSeries('asr' + i, ad[i], by, m));
};

// ------------------------------------------------------------ data cleanup
// Deletes flow records, summaries, counters and findings older than a number
// of days, or everything; each shows what it frees and asks first.
views.cleanup = async (el) => {
  // Every control here deletes stored data, so there is nothing of this
  // page to show someone who may not.
  if (!adm()) { el.innerHTML = `<div class="panel"><div class="empty">${esc(t('cl.admin_only'))}</div></div>`; return; }
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
  await ifaceCleanup(el);
};
let clDone = '';

// Deleting one interface's data (registered only). The summaries hold
// traffic by application, host and dimension, never by interface, so they
// cannot have one interface subtracted from them; the panel says so rather
// than letting anyone believe the interface is gone from every page.
async function ifaceCleanup(el) {
  const on = licOK('purge-iface');
  const ages = [0, ...cleanupDaysUI];
  const ageOpt = d => `<option value="${d}">${esc(d ? t('cl.days', {n: d}) : t('cl.if_any_age'))}</option>`;
  const wrap = document.createElement('div');
  wrap.className = 'grid';
  wrap.innerHTML = panel('c12' + (on ? '' : ' locked'), t('cl.if_title'), on ? t('cl.if_sub') : t('cl.reg_only'),
    `<p class="muted" style="margin:0 0 12px;font-size:13px">${esc(t('cl.if_what'))}<br>${esc(t('cl.if_both'))}</p>
     <div style="display:flex;gap:8px;flex-wrap:wrap;align-items:center">
       <select id="clIf" ${on ? '' : 'disabled'} style="max-width:420px"><option>${esc(t('cl.if_pick'))}</option></select>
       <select id="clIfAge" ${on ? '' : 'disabled'}>${ages.map(ageOpt).join('')}</select>
       <button class="btn danger" id="clIfRun" disabled>${t('cl.run')}</button>
     </div>
     <div id="clIfEst" class="muted" style="margin-top:10px;font-size:13px"></div>
     <div id="clIfMsg" style="margin-top:6px;font-size:13px"></div>`);
  el.append(...wrap.children);
  if (!on) { $('#clIfEst').textContent = t('cl.if_locked'); return; }

  const d = await api('ifaces', {from: 0, to: Date.now()}).catch(() => ({ifaces: []}));
  const list = (d.ifaces || []).filter(f => f.ifindex);
  const sel = $('#clIf');
  if (!list.length) { sel.innerHTML = `<option>${esc(t('cl.if_none'))}</option>`; sel.disabled = true; $('#clIfAge').disabled = true; return; }
  const groups = new Map();
  list.forEach(f => { const g = f.device || f.exporter; if (!groups.has(g)) groups.set(g, []); groups.get(g).push(f); });
  let html = '';
  groups.forEach((fs, g) => { html += `<optgroup label="${esc(g)}">${fs.map(f => `<option value="${esc(f.exporter + '/' + f.ifindex)}" title="${esc((f.device || f.exporter) + ' · ' + f.ifindex)}">${esc(ifName(f) + (f.tag ? ' [' + f.tag + ']' : ''))}</option>`).join('')}</optgroup>`; });
  sel.innerHTML = html;

  const target = () => { const [exporter, ifindex] = sel.value.split('/'); return {exporter, ifindex: +ifindex, days: +$('#clIfAge').value}; };
  const estimate = async () => {
    const tg = target(), est = $('#clIfEst'), run = $('#clIfRun');
    run.disabled = true;
    est.textContent = t('cl.if_estimating');
    const res = await fetch(`/api/cleanup/iface?exporter=${encodeURIComponent(tg.exporter)}&ifindex=${tg.ifindex}&days=${tg.days}`);
    const j = await res.json().catch(() => ({}));
    if (!res.ok) { est.textContent = j.error === 'not_registered' ? t('cl.if_locked') : (j.error || res.statusText); return; }
    const parts = [t('cl.rows', {n: nf(j.rows || 0)}), t('cl.if_counters', {n: nf(j.counters || 0)})];
    if (j.bytes) parts.push(t('cl.if_frees', {s: fmtBytes(j.bytes)}));
    est.textContent = parts.join(' · ') + (j.summaries ? ' — ' + t('cl.if_keep', {n: nf(j.summaries)}) : '');
    run.disabled = !(j.rows || j.counters);
  };
  sel.onchange = $('#clIfAge').onchange = estimate;
  $('#clIfRun').onclick = async () => {
    const tg = target(), name = sel.selectedOptions[0].textContent;
    if (!confirm(t(tg.days ? 'cl.if_confirm_days' : 'cl.if_confirm', {i: name, n: tg.days}))) return;
    const run = $('#clIfRun'), m = $('#clIfMsg');
    run.disabled = true;
    const res = await fetch('/api/cleanup/iface', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(tg)});
    const j = await res.json().catch(() => ({}));
    if (!res.ok) { m.style.color = 'var(--crit)'; m.textContent = j.error === 'not_registered' ? t('cl.if_locked') : (j.error || res.statusText); return; }
    m.style.color = 'var(--good)';
    m.textContent = t('cl.if_done', {i: name, n: nf(j.rows || 0), c: nf(j.counters || 0), s: fmtBytes(j.bytes || 0)});
    loadStatus();
    await estimate();
  };
  await estimate();
}
// the ages offered, mirroring the server's own list
const cleanupDaysUI = [120, 90, 60, 30, 7];

// ------------------------------------------------------- the asset ledger
// Sites, racks, devices and their ports, the cables between them, and the
// networks and addresses assigned on them. It is a ledger: everything here
// was written down by someone, never discovered. Registered installations
// only — the panels are shown and explained rather than hidden, so what
// registering buys is visible, and the server refuses every call anyway.
let assetData = null;
const assetSel = {site: 0, rack: 0, device: 0, subnet: 0};

async function loadAssets() {
  const res = await fetch('/api/assets');
  if (res.status === 401) { showLogin(); throw new Error('login'); }
  const j = await res.json().catch(() => ({}));
  if (!res.ok) return assetData = {locked: j.error === 'not_registered', error: j.error || res.statusText};
  j.by = {
    site: new Map((j.sites || []).map(x => [x.id, x])),
    rack: new Map((j.racks || []).map(x => [x.id, x])),
    device: new Map((j.devices || []).map(x => [x.id, x])),
    port: new Map((j.ports || []).map(x => [x.id, x])),
  };
  return assetData = j;
}

async function assetSave(kind, rec) {
  const res = await fetch('/api/assets/' + kind, {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(rec)});
  const j = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(j.detail || j.error || res.statusText);
  return j;
}

async function assetDelete(kind, id) {
  const res = await fetch(`/api/assets/${kind}?id=${id}`, {method: 'DELETE'});
  const j = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(j.detail || j.error || res.statusText);
}

// The locked panel every asset page shows without a licence. It says what
// the page would do, because a prospect has to see what they would buy.
function assetLocked(el, titleKey, whatKey) {
  el.innerHTML = `<div class="grid">${panel('c12 locked', t(titleKey), t('cl.reg_only'),
    `<p class="muted" style="margin:0;font-size:13px">${esc(t(whatKey))}</p>
     <p style="margin:10px 0 0;font-size:13px">${esc(t('as.locked'))}</p>`)}</div>`;
}

// assetPage loads the ledger and handles the two states that are not the
// page: no licence, and a failure worth showing.
async function assetPage(el, titleKey, whatKey) {
  const d = await loadAssets();
  if (d.locked) { assetLocked(el, titleKey, whatKey); return null; }
  if (d.error) { el.innerHTML = `<div class="grid">${panel('c12', t(titleKey), '', `<div class="empty">${esc(d.error)}</div>`)}</div>`; return null; }
  return d;
}

// ------------------------------------------------------------ the editor
// One form over one record. Fields describe what to show; saving posts the
// whole record back, so a field the form does not show is carried over
// untouched rather than cleared.
const ASSET_FIELDS = {
  site: [['name', 'text', 1], ['address', 'text'], ['note', 'area'], ['hidden', 'check']],
  rack: [['name', 'text', 1], ['site_id', 'site'], ['units', 'num'], ['note', 'area'], ['hidden', 'check']],
  device: [['name', 'text', 1], ['role', 'text'], ['vendor', 'text'], ['model', 'text'], ['serial', 'text'],
    ['mgmt_ip', 'text'], ['rack_id', 'rack'], ['pos_u', 'num'], ['height_u', 'num'], ['tags', 'tags'],
    ['note', 'area'], ['hidden', 'check']],
  port: [['name', 'text', 1], ['ifindex', 'num'], ['speed', 'num'], ['kind', 'text'], ['note', 'area']],
  circuit: [['name', 'text', 1], ['carrier', 'text'], ['kind', 'text'], ['bandwidth', 'text'],
    ['site_id', 'site'], ['port_id', 'port'], ['contract', 'text'], ['starts', 'day'], ['ends', 'day'],
    ['tags', 'tags'], ['note', 'area'], ['hidden', 'check']],
  subnet: [['prefix', 'text', 1], ['name', 'text'], ['vlan', 'num'], ['site_id', 'site'],
    ['gateway', 'text'], ['tags', 'tags'], ['note', 'area'], ['hidden', 'check']],
  addr: [['ip', 'text', 1], ['name', 'text'], ['kind', 'akind'], ['device_id', 'device'], ['port_id', 'port'],
    ['mac', 'text'], ['tags', 'tags'], ['note', 'area'], ['hidden', 'check']],
};
const ADDR_KINDS = ['static', 'dhcp', 'reserved', 'gateway', 'vip'];

function assetOptions(type, cur) {
  const none = `<option value="0">${esc(t('as.none'))}</option>`;
  const opt = (v, l) => `<option value="${v}"${+cur === +v ? ' selected' : ''}>${esc(l)}</option>`;
  if (type === 'site') return none + (assetData.sites || []).map(x => opt(x.id, x.name)).join('');
  if (type === 'rack') return none + (assetData.racks || []).map(x => opt(x.id, rackLabel(x))).join('');
  if (type === 'device') return none + (assetData.devices || []).map(x => opt(x.id, x.name)).join('');
  if (type === 'port') return none + (assetData.ports || []).map(x => opt(x.id, portLabel(x))).join('');
  if (type === 'akind') return ADDR_KINDS.map(k => opt(k, t('as.kind_' + k))).join('');
  return '';
}
const rackLabel = r => (r.site_id && assetData.by.site.get(r.site_id) ? assetData.by.site.get(r.site_id).name + ' · ' : '') + r.name;
const portLabel = p => (assetData.by.device.get(p.device_id)?.name || '?') + ' · ' + p.name;

// assetEdit opens the editor for one record and resolves when it is saved
// (to the saved record) or cancelled (to null).
function assetEdit(kind, rec) {
  return new Promise(resolve => {
    const fields = ASSET_FIELDS[kind];
    const val = f => rec[f] ?? '';
    const row = ([f, type, req]) => {
      const id = 'af_' + f, lab = `<label for="${id}">${esc(t('as.f_' + f))}${req ? ' *' : ''}</label>`;
      if (type === 'check') return `<div class="afield afwide"><label class="afcheck"><input type="checkbox" id="${id}"${rec[f] ? ' checked' : ''}> ${esc(t('as.f_' + f))}</label>
        <span class="muted" style="font-size:12px">${esc(t('as.hidden_hint'))}</span></div>`;
      if (type === 'area') return `<div class="afield afwide">${lab}<textarea id="${id}" rows="2">${esc(val(f))}</textarea></div>`;
      if (type === 'tags') return `<div class="afield afwide">${lab}<input id="${id}" value="${esc((rec[f] || []).join(', '))}" placeholder="${esc(t('as.tags_hint'))}"></div>`;
      if (type === 'num') return `<div class="afield">${lab}<input id="${id}" type="number" min="0" value="${rec[f] || ''}"></div>`;
      if (type === 'day') return `<div class="afield">${lab}<input id="${id}" type="date" value="${esc(val(f))}"></div>`;
      if (['site', 'rack', 'device', 'port', 'akind'].includes(type)) return `<div class="afield">${lab}<select id="${id}">${assetOptions(type, rec[f])}</select></div>`;
      return `<div class="afield">${lab}<input id="${id}" value="${esc(val(f))}"></div>`;
    };
    const dlg = document.createElement('dialog');
    dlg.className = 'aedit';
    dlg.innerHTML = `<form method="dialog"><h2>${esc(t(rec.id ? 'as.edit_' + kind : 'as.add_' + kind))}</h2>
      <div class="aform">${fields.map(row).join('')}</div>
      <div id="afMsg" class="afmsg"></div>
      <div class="afbtns"><button type="button" class="btn" id="afCancel">${esc(t('as.cancel'))}</button>
        <button type="button" class="btn primary" id="afSave">${esc(t('as.save'))}</button></div></form>`;
    document.body.append(dlg);
    const close = v => { dlg.close(); dlg.remove(); resolve(v); };
    dlg.querySelector('#afCancel').onclick = () => close(null);
    dlg.addEventListener('cancel', e => { e.preventDefault(); close(null); });
    dlg.querySelector('#afSave').onclick = async () => {
      const out = {...rec};
      for (const [f, type] of fields) {
        const e = dlg.querySelector('#af_' + f);
        if (type === 'check') out[f] = e.checked;
        else if (type === 'tags') out[f] = e.value.split(',').map(s => s.trim()).filter(Boolean);
        else if (type === 'num' || ['site', 'rack', 'device', 'port'].includes(type)) out[f] = +e.value || 0;
        else out[f] = e.value.trim();
      }
      const m = dlg.querySelector('#afMsg');
      m.textContent = '';
      try { close(await assetSave(kind, out)); }
      catch (err) { m.textContent = err.message; }
    };
    dlg.showModal();
    dlg.querySelector('input, select, textarea')?.focus();
  });
}

// assetRemove asks first and names what is going.
async function assetRemove(kind, rec, label) {
  if (!confirm(t('as.confirm_delete', {x: label}))) return false;
  try { await assetDelete(kind, rec.id); return true; }
  catch (err) { alert(err.message); return false; }
}

// Controls only an administrator gets. The calls behind them are gated
// server-side too (adminAPI, in internal/api/server.go) — this only keeps
// the page from offering a button that would answer 403. Tables and
// everything else worth reading stay visible to everyone.
const adm = () => !!(me && me.admin);
const admOnly = html => adm() ? html : '';

// Buttons an administrator gets and nobody else: the ledger is read by
// everyone and written by the one who keeps it.
const adminOnly = html => assetData.admin ? html : '';
const quotaLine = (have, limit) => limit > 0 ? `${nf(have)} / ${nf(limit)}` : nf(have);

// ------------------------------------------------- the asset management
// One page for where the equipment is — sites, racks and the devices in
// them — and a page of its own for a device, reached by clicking it. The
// device page is not in the navigation because it is about one device:
// there is nothing to show until one has been chosen.
const ASSET_VIEWS = new Set(['racks', 'devices', 'ipam', 'names']);

// goDevice opens one device's own page.
function goDevice(id) { assetSel.device = id; go('devices'); }

views.racks = async (el) => {
  const d = await assetPage(el, 'nav.racks', 'as.what_racks');
  if (!d) return;
  const sites = d.sites || [], racks = d.racks || [], ports = d.ports || [];
  if (assetSel.site && !d.by.site.has(assetSel.site)) assetSel.site = 0;
  const shown = assetSel.site ? racks.filter(r => r.site_id === assetSel.site) : racks;
  const devOf = r => (d.devices || []).filter(x => x.rack_id === r.id);
  const siteRow = s => `<tr${assetSel.site === s.id ? ' class="on"' : ''}><td><button class="lnk" data-site="${s.id}">${esc(s.name)}</button>
      ${s.hidden ? `<span class="tag">${esc(t('as.f_hidden'))}</span>` : ''}</td>
    <td class="muted">${esc(s.address || '')}</td>
    <td class="num">${nf(racks.filter(r => r.site_id === s.id).length)}</td>
    <td style="text-align:end">${adminOnly(`<button class="btn sm" data-esite="${s.id}">${esc(t('as.edit'))}</button>
      <button class="btn sm danger" data-dsite="${s.id}">${esc(t('as.delete'))}</button>`)}</td></tr>`;
  // the devices of the chosen site, or of every site
  const devices = assetSel.site ? (d.devices || []).filter(x => d.by.rack.get(x.rack_id)?.site_id === assetSel.site) : (d.devices || []);
  const devRow = x => `<tr><td><button class="lnk" data-dev="${x.id}">${esc(x.name)}</button>
      ${x.hidden ? `<span class="tag">${esc(t('as.f_hidden'))}</span>` : ''}
      ${(x.tags || []).map(tg => `<span class="tag">${esc(tg)}</span>`).join('')}</td>
    <td class="muted">${esc(x.role || '')}</td>
    <td class="muted">${esc([x.vendor, x.model].filter(Boolean).join(' '))}</td>
    <td>${x.mgmt_ip ? ipCell(x.mgmt_ip) : ''}</td>
    <td>${devPlace(d, x)}</td>
    <td class="num">${nf(ports.filter(p => p.device_id === x.id).length)}</td></tr>`;
  el.innerHTML = `<div class="grid">
    ${panel('c12', t('nav.racks'), assetCounts(d), `<p class="muted" style="margin:0 0 12px;font-size:13px">${esc(t('as.what_racks'))}</p>
      <table><tr><th>${esc(t('as.f_name'))}</th><th>${esc(t('as.f_address'))}</th><th class="num">${esc(t('as.racks'))}</th><th></th></tr>
      ${sites.map(siteRow).join('') || `<tr><td colspan="4" class="empty">${esc(t('as.no_sites'))}</td></tr>`}</table>
      ${adminOnly(`<div style="margin-top:10px"><button class="btn" id="asAddSite">${esc(t('as.add_site'))}</button>
        ${sites.length ? `<button class="btn" id="asAddRack">${esc(t('as.add_rack'))}</button>` : ''}</div>`)}`)}
    ${panel('c12', t('as.racks') + (assetSel.site ? ' · ' + esc(d.by.site.get(assetSel.site).name) : ''), '',
      shown.length ? `<div class="rackwrap">${shown.map(r => rackCard(r, devOf(r))).join('')}</div>`
        : `<div class="empty">${esc(t('as.no_racks'))}</div>`)}
    ${panel('c12', t('as.devices'), t('as.devices_sub'), `<table><tr><th>${esc(t('as.f_name'))}</th><th>${esc(t('as.f_role'))}</th>
        <th>${esc(t('as.f_model'))}</th><th>${esc(t('as.f_mgmt_ip'))}</th><th>${esc(t('as.place'))}</th><th class="num">${esc(t('as.ports'))}</th></tr>
      ${devices.map(devRow).join('') || `<tr><td colspan="6" class="empty">${esc(t('as.no_devices'))}</td></tr>`}</table>
      ${adminOnly(`<div style="margin-top:10px"><button class="btn" id="asAddDev">${esc(t('as.add_device'))}</button></div>`)}`)}
    ${adminOnly(pastePanel(d) + csvPanel())}
  </div>`;
  el.querySelectorAll('[data-site]').forEach(b => b.onclick = () => { assetSel.site = assetSel.site === +b.dataset.site ? 0 : +b.dataset.site; render(); });
  el.querySelectorAll('[data-esite]').forEach(b => b.onclick = async () => { if (await assetEdit('site', {...d.by.site.get(+b.dataset.esite)})) render(); });
  el.querySelectorAll('[data-dsite]').forEach(b => b.onclick = async () => {
    const s = d.by.site.get(+b.dataset.dsite);
    if (await assetRemove('site', s, s.name)) render();
  });
  el.querySelectorAll('[data-erack]').forEach(b => b.onclick = async () => { if (await assetEdit('rack', {...d.by.rack.get(+b.dataset.erack)})) render(); });
  el.querySelectorAll('[data-drack]').forEach(b => b.onclick = async () => {
    const r = d.by.rack.get(+b.dataset.drack);
    if (await assetRemove('rack', r, r.name)) render();
  });
  // a device in a rack and a device in the list both open its own page
  el.querySelectorAll('[data-rdev]').forEach(b => b.onclick = () => goDevice(+b.dataset.rdev));
  el.querySelectorAll('[data-dev]').forEach(b => b.onclick = () => goDevice(+b.dataset.dev));
  const add = $('#asAddSite'); if (add) add.onclick = async () => { if (await assetEdit('site', {})) render(); };
  const addR = $('#asAddRack'); if (addR) addR.onclick = async () => { if (await assetEdit('rack', {site_id: assetSel.site || sites[0].id, units: d.default_units || 46})) render(); };
  const addD = $('#asAddDev'); if (addD) addD.onclick = async () => {
    const made = await assetEdit('device', {height_u: 1, rack_id: 0});
    if (made) { assetData = null; goDevice(made.id); }
  };
  wirePaste();
  wireCSV();
};

const assetCounts = d => [
  t('as.c_sites', {x: quotaLine(d.counts.sites, d.quota.sites)}),
  t('as.c_racks', {x: quotaLine(d.counts.racks, d.quota.racks)}),
  t('as.c_devices', {x: quotaLine(d.counts.devices, d.quota.devices)}),
].join(' · ');

// where a device stands, for the lists
function devPlace(d, x) {
  const r = d.by.rack.get(x.rack_id);
  if (!r) return `<span class="muted">${esc(t('as.unracked'))}</span>`;
  return esc(r.name) + (x.pos_u ? ` <span class="muted">U${x.pos_u}</span>` : '');
}

// One rack drawn by the unit, numbered from the bottom as racks are. A
// device fills the units it occupies, so a gap on the page is a gap in the
// cabinet; a device in the rack whose unit nobody wrote down is listed
// under it rather than placed somewhere invented.
function rackCard(r, devs) {
  // one rack unit in pixels; .rack .ru in app.css draws the same height
  const H = 14, placed = devs.filter(x => x.pos_u > 0), loose = devs.filter(x => !x.pos_u);
  const units = [];
  for (let u = r.units; u >= 1; u--) units.push(`<div class="ru"><span>${u}</span></div>`);
  const blocks = placed.map(x => {
    const h = Math.max(1, x.height_u || 1), top = (r.units - (x.pos_u + h - 1)) * H;
    const title = `${x.name} · ${t('as.units_at', {u: x.pos_u, n: h})}${x.model ? ' · ' + x.model : ''}`;
    return `<button class="rdev" data-rdev="${x.id}" style="top:${top}px;height:${h * H - 2}px" title="${esc(title)}">${esc(x.name)}</button>`;
  });
  const free = r.units - placed.reduce((n, x) => n + Math.max(1, x.height_u || 1), 0);
  return `<div class="rackcard"><div class="rh"><b>${esc(r.name)}</b>
      <span class="muted">${esc(t('as.units_free', {n: free, t: r.units}))}</span>
      ${adminOnly(`<span class="rbtns"><button class="btn sm" data-erack="${r.id}">${esc(t('as.edit'))}</button><button class="btn sm danger" data-drack="${r.id}">${esc(t('as.delete'))}</button></span>`)}</div>
    <div class="rack" style="height:${r.units * H}px">${units.join('')}${blocks.join('')}</div>
    ${loose.length ? `<div class="rloose">${esc(t('as.unplaced'))}: ${loose.map(x => `<button class="lnk" data-rdev="${x.id}">${esc(x.name)}</button>`).join(', ')}</div>` : ''}</div>`;
}

// ------------------------------------------------------- one device
// Reached by clicking a device, never from the navigation: what it is,
// where it stands, and every port with the port at the other end of its
// cable.
views.devices = async (el) => {
  const d = await assetPage(el, 'as.device', 'as.what_devices');
  if (!d) return;
  const x = assetSel.device ? d.by.device.get(assetSel.device) : null;
  if (!x) { go('racks'); return; }
  $('#title').textContent = x.name;
  document.title = x.name + ' · traffic66';
  const ports = (d.ports || []).filter(p => p.device_id === x.id);
  const peer = new Map();
  (d.links || []).forEach(l => { peer.set(l.a_port, l.b_port); peer.set(l.b_port, l.a_port); });
  const fact = (k, v) => v ? `<div class="afact"><span class="k">${esc(t(k))}</span><span class="v">${v}</span></div>` : '';
  el.innerHTML = `<div class="grid">
    ${panel('c12', esc(x.name), esc(x.role || ''), `<div class="dtop">
        <button class="btn sm" id="asBack">← ${esc(t('nav.racks'))}</button>
        ${adminOnly(`<span style="margin-inline-start:auto;display:flex;gap:6px"><button class="btn sm" id="asEditDev">${esc(t('as.edit'))}</button>
          <button class="btn sm danger" id="asDelDev">${esc(t('as.delete'))}</button></span>`)}</div>
      <div class="afacts">
        ${fact('as.f_vendor', esc([x.vendor, x.model].filter(Boolean).join(' ')))}
        ${fact('as.f_serial', esc(x.serial || ''))}
        ${fact('as.f_mgmt_ip', x.mgmt_ip ? ipCell(x.mgmt_ip) : '')}
        ${fact('as.place', devPlace(d, x))}
        ${fact('as.f_height_u', x.height_u ? esc(x.height_u + 'U') : '')}
        ${fact('as.f_tags', (x.tags || []).map(tg => `<span class="tag">${esc(tg)}</span>`).join(''))}
        ${x.hidden ? fact('as.f_hidden', `<span class="tag">${esc(t('as.f_hidden'))}</span>`) : ''}
      </div>
      ${x.note ? `<p class="muted" style="margin:10px 0 0;font-size:13px;white-space:pre-wrap">${esc(x.note)}</p>` : ''}`)}
    ${panel('c12', t('as.ports_of', {x: x.name}), portUse(ports, peer), portTable(d, ports, peer))}
  </div>`;
  $('#asBack').onclick = () => go('racks');
  const ed = $('#asEditDev'); if (ed) ed.onclick = async () => { if (await assetEdit('device', {...x})) render(); };
  const del = $('#asDelDev'); if (del) del.onclick = async () => {
    if (await assetRemove('device', x, x.name)) { assetSel.device = 0; assetData = null; go('racks'); }
  };
  el.querySelectorAll('[data-eport]').forEach(b => b.onclick = async () => { if (await assetEdit('port', {...d.by.port.get(+b.dataset.eport)})) render(); });
  el.querySelectorAll('[data-dport]').forEach(b => b.onclick = async () => {
    const p = d.by.port.get(+b.dataset.dport);
    if (await assetRemove('port', p, portLabel(p))) render();
  });
  el.querySelectorAll('[data-cable]').forEach(b => b.onclick = () => cablePort(d, +b.dataset.cable, peer));
  const addP = $('#asAddPort'); if (addP) addP.onclick = async () => { if (await assetEdit('port', {device_id: x.id})) render(); };
};

// Port occupancy in one line: how many of the ports written down carry a
// cable. It counts the ledger, not the link state — nothing here is polled.
function portUse(ps, peer) {
  if (!ps.length) return '';
  const used = ps.filter(p => peer.has(p.id)).length;
  return t('as.port_use', {u: nf(used), n: nf(ps.length), p: nf(Math.round(used / ps.length * 100))});
}

function portTable(d, ps, peer) {
  const row = p => {
    const o = peer.get(p.id), op = o ? d.by.port.get(o) : null;
    return `<tr><td><b>${esc(p.name)}</b></td>
      <td class="muted">${p.ifindex ? nf(p.ifindex) : ''}</td>
      <td class="muted">${p.speed ? fmtBps(p.speed) : ''}</td>
      <td class="muted">${esc(p.kind || '')}</td>
      <td>${op ? esc(portLabel(op)) : `<span class="muted">${esc(t('as.port_free'))}</span>`}</td>
      <td style="text-align:end">${adminOnly(`<button class="btn sm" data-cable="${p.id}">${esc(t(op ? 'as.recable' : 'as.cable'))}</button>
        <button class="btn sm" data-eport="${p.id}">${esc(t('as.edit'))}</button>
        <button class="btn sm danger" data-dport="${p.id}">${esc(t('as.delete'))}</button>`)}</td></tr>`;
  };
  return `<table><tr><th>${esc(t('as.f_name'))}</th><th>${esc(t('as.f_ifindex'))}</th><th>${esc(t('as.f_speed'))}</th>
      <th>${esc(t('as.f_kind'))}</th><th>${esc(t('as.cabled_to'))}</th><th></th></tr>
    ${ps.map(row).join('') || `<tr><td colspan="6" class="empty">${esc(t('as.no_ports'))}</td></tr>`}</table>
    ${adminOnly(`<div style="margin-top:10px"><button class="btn" id="asAddPort">${esc(t('as.add_port'))}</button></div>`)}`;
}

// Cabling a port: pick the other end, or pull the cable out. A port holds
// one cable, so choosing a port that already has one moves it.
function cablePort(d, portID, peer) {
  const others = (d.ports || []).filter(p => p.id !== portID);
  const cur = peer.get(portID) || 0;
  const dlg = document.createElement('dialog');
  dlg.className = 'aedit';
  dlg.innerHTML = `<form method="dialog"><h2>${esc(t('as.cable_title', {x: portLabel(d.by.port.get(portID))}))}</h2>
    <div class="aform"><div class="afield afwide"><label for="afPeer">${esc(t('as.cabled_to'))}</label>
      <select id="afPeer"><option value="0">${esc(t('as.no_cable'))}</option>
        ${others.map(p => `<option value="${p.id}"${cur === p.id ? ' selected' : ''}>${esc(portLabel(p))}</option>`).join('')}</select></div></div>
    <div id="afMsg" class="afmsg"></div>
    <div class="afbtns"><button type="button" class="btn" id="afCancel">${esc(t('as.cancel'))}</button>
      <button type="button" class="btn primary" id="afSave">${esc(t('as.save'))}</button></div></form>`;
  document.body.append(dlg);
  const close = () => { dlg.close(); dlg.remove(); };
  dlg.querySelector('#afCancel').onclick = close;
  dlg.addEventListener('cancel', e => { e.preventDefault(); close(); });
  dlg.querySelector('#afSave').onclick = async () => {
    const to = +dlg.querySelector('#afPeer').value;
    const link = (d.links || []).find(l => l.a_port === portID || l.b_port === portID);
    try {
      if (!to) { if (link) await assetDelete('link', link.id); }
      else await assetSave('link', {id: link ? link.id : 0, a_port: portID, b_port: to});
      close(); render();
    } catch (err) { dlg.querySelector('#afMsg').textContent = err.message; }
  };
  dlg.showModal();
}

// ------------------------------------------------------- import and export
// Pasting what a switch printed. The operator runs the command they
// already run; there is nothing to install on the device, no SNMP
// community to hand out and no credentials to keep anywhere. Nothing is
// written until the plan has been shown and accepted, so a paste of the
// wrong buffer is seen rather than suffered.
function pastePanel(d) {
  const opt = x => `<option value="${x.id}"${assetSel.device === x.id ? ' selected' : ''}>${esc(x.name)}</option>`;
  return panel('c12', t('as.paste_title'), t('as.paste_sub'),
    `<p class="muted" style="margin:0 0 10px;font-size:13px">${esc(t('as.paste_what'))}</p>
     <div style="display:flex;gap:8px;flex-wrap:wrap;align-items:center;margin-bottom:8px">
       <label for="asPasteDev" style="font-size:13px">${esc(t('as.paste_from'))}</label>
       <select id="asPasteDev"><option value="0">${esc(t('as.none'))}</option>${(d.devices || []).map(opt).join('')}</select>
     </div>
     <textarea id="asPasteText" rows="7" class="pastebox" placeholder="${esc(t('as.paste_hint'))}"></textarea>
     <div style="margin-top:8px;display:flex;gap:8px"><button class="btn" id="asPasteCheck">${esc(t('as.preview'))}</button>
       <button class="btn primary" id="asPasteApply" disabled>${esc(t('as.apply'))}</button></div>
     <div id="asPastePlan" style="margin-top:10px"></div>`);
}

// The ledger as a spreadsheet, in and out. Export, edit, import: the id
// column ties each row back to its record, so a file that came from here
// updates what it came from instead of doubling it.
function csvPanel() {
  const kinds = ['site', 'rack', 'device', 'port', 'circuit', 'subnet', 'addr'];
  const opt = k => `<option value="${k}">${esc(t('as.kind_of_' + k))}</option>`;
  return panel('c12', t('as.csv_title'), t('as.csv_sub'),
    `<p class="muted" style="margin:0 0 10px;font-size:13px">${esc(t('as.csv_what'))}</p>
     <div style="display:flex;gap:8px;flex-wrap:wrap;align-items:center">
       <select id="asCsvKind">${kinds.map(opt).join('')}</select>
       <button class="btn" id="asCsvOut">${esc(t('as.csv_export'))}</button>
       <input type="file" id="asCsvFile" accept=".csv,text/csv" style="font-size:13px">
     </div>
     <div style="margin-top:8px;display:flex;gap:8px"><button class="btn" id="asCsvCheck" disabled>${esc(t('as.preview'))}</button>
       <button class="btn primary" id="asCsvApply" disabled>${esc(t('as.apply'))}</button></div>
     <div id="asCsvPlan" style="margin-top:10px"></div>`);
}

// A plan, as a table of what would happen. Errors are shown with the rest
// rather than instead of it: one bad line in a spreadsheet does not make
// the other fifty wrong.
function planTable(plan) {
  if (!plan) return '';
  const rows = plan.changes || [];
  if (!rows.length && !(plan.skipped || []).length) return `<div class="empty">${esc(t('as.plan_none'))}</div>`;
  const row = c => `<tr><td class="muted">${esc(t('as.kind_of_' + c.kind) || c.kind)}</td>
    <td><span class="st ${c.action === 'error' ? 'crit' : c.action === 'same' ? 'ok' : 'warn'}">${esc(t('as.act_' + c.action))}</span></td>
    <td>${esc(c.what)}</td><td class="muted">${esc(c.error || '')}</td></tr>`;
  const n = rows.filter(c => c.action !== 'error' && c.action !== 'same').length;
  return `<p style="margin:0 0 8px;font-size:13px">${esc(t(plan.applied ? 'as.plan_done' : 'as.plan_would', {n: nf(n)}))}</p>
    <table><tr><th></th><th></th><th>${esc(t('as.what'))}</th><th>${esc(t('as.detail'))}</th></tr>${rows.map(row).join('')}</table>
    ${(plan.skipped || []).length ? `<p class="muted" style="margin:8px 0 0;font-size:12.5px">${esc(t('as.plan_skipped'))}: ${esc(plan.skipped.slice(0, 8).join(' | '))}</p>` : ''}`;
}

// wirePaste and wireCSV are called after the devices page is drawn.
function wirePaste() {
  const box = $('#asPasteText'), check = $('#asPasteCheck'), apply = $('#asPasteApply'), out = $('#asPastePlan');
  if (!box) return;
  const send = async applyIt => {
    const body = {text: box.value, device_id: +$('#asPasteDev').value, apply: applyIt};
    const res = await fetch('/api/assets/paste', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(body)});
    const j = await res.json().catch(() => ({}));
    if (!res.ok) { out.innerHTML = `<div class="empty">${esc(j.detail || j.error || res.statusText)}</div>`; apply.disabled = true; return null; }
    out.innerHTML = planTable(j);
    return j;
  };
  check.onclick = async () => { const j = await send(false); apply.disabled = !j || !(j.changes || []).some(c => c.action !== 'error' && c.action !== 'same'); };
  apply.onclick = async () => { if (await send(true)) { apply.disabled = true; assetData = null; } };
}

function wireCSV() {
  const kind = $('#asCsvKind'), file = $('#asCsvFile'), check = $('#asCsvCheck'), apply = $('#asCsvApply'), out = $('#asCsvPlan');
  if (!kind) return;
  let text = '';
  $('#asCsvOut').onclick = () => { location.href = '/api/assets/csv?kind=' + encodeURIComponent(kind.value); };
  file.onchange = async () => {
    const f = file.files[0];
    if (!f) { text = ''; check.disabled = true; return; }
    text = await f.text();
    check.disabled = false;
    apply.disabled = true;
    out.innerHTML = '';
  };
  const send = async applyIt => {
    const res = await fetch('/api/assets/csv', {method: 'POST', headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({kind: kind.value, text, apply: applyIt})});
    const j = await res.json().catch(() => ({}));
    if (!res.ok) { out.innerHTML = `<div class="empty">${esc(j.detail || j.error || res.statusText)}</div>`; apply.disabled = true; return null; }
    out.innerHTML = planTable(j);
    return j;
  };
  check.onclick = async () => { const j = await send(false); apply.disabled = !j || !(j.changes || []).some(c => c.action !== 'error' && c.action !== 'same'); };
  apply.onclick = async () => { if (await send(true)) { apply.disabled = true; assetData = null; } };
}

// ------------------------------------------------------------- addresses
views.ipam = async (el) => {
  const d = await assetPage(el, 'nav.ipam', 'as.what_ipam');
  if (!d) return;
  const res = await fetch('/api/assets/ipam');
  const ip = await res.json().catch(() => ({}));
  if (!res.ok) { el.innerHTML = `<div class="grid">${panel('c12', t('nav.ipam'), '', `<div class="empty">${esc(ip.error || res.statusText)}</div>`)}</div>`; return; }
  const subs = ip.subnets || [], conflicts = ip.conflicts || [];
  if (assetSel.subnet && !subs.some(s => s.id === assetSel.subnet)) assetSel.subnet = 0;
  const sel = subs.find(s => s.id === assetSel.subnet) || null;
  const [addrs, free] = await Promise.all([
    sel ? fetch(`/api/assets/addrs?subnet=${sel.id}`).then(r => r.json()).then(j => j.addrs || []) : Promise.resolve([]),
    sel ? fetch(`/api/assets/free?subnet=${sel.id}&n=8`).then(r => r.json()).then(j => j.free || []) : Promise.resolve([]),
  ]);
  const fill = s => s.hosts > 0 ? Math.min(100, Math.round(s.assigned / s.hosts * 100)) : 0;
  const subRow = s => `<tr${assetSel.subnet === s.id ? ' class="on"' : ''}><td><button class="lnk" data-sub="${s.id}">${esc(s.prefix)}</button>
      ${s.hidden ? `<span class="tag">${esc(t('as.f_hidden'))}</span>` : ''}
      ${(s.tags || []).map(tg => `<span class="tag">${esc(tg)}</span>`).join('')}</td>
    <td class="muted">${esc(s.name || '')}</td>
    <td class="muted">${s.vlan ? nf(s.vlan) : ''}</td>
    <td class="muted">${esc(d.by.site.get(s.site_id)?.name || '')}</td>
    <td class="num">${nf(s.assigned)}${s.hosts >= 0 ? ' / ' + nf(s.hosts) : ''}</td>
    <td style="width:22%">${s.hosts >= 0 ? bar(s.assigned, s.hosts, fill(s) > 90 ? 'var(--crit)' : fill(s) > 75 ? 'var(--warn)' : '') : `<span class="muted">${esc(t('as.too_large'))}</span>`}</td>
    <td style="text-align:end">${adminOnly(`<button class="btn sm" data-esub="${s.id}">${esc(t('as.edit'))}</button>
      <button class="btn sm danger" data-dsub="${s.id}">${esc(t('as.delete'))}</button>`)}</td></tr>`;
  const addrRow = a => `<tr><td><b>${esc(a.ip)}</b>${a.hidden ? ` <span class="tag">${esc(t('as.f_hidden'))}</span>` : ''}</td>
    <td>${esc(a.name || '')}</td>
    <td class="muted">${esc(t('as.kind_' + (a.kind || 'static')))}</td>
    <td class="muted">${esc(d.by.device.get(a.device_id)?.name || '')}</td>
    <td class="muted mono">${esc(a.mac || '')}</td>
    <td>${(a.tags || []).map(tg => `<span class="tag">${esc(tg)}</span>`).join('')}</td>
    <td style="text-align:end">${adminOnly(`<button class="btn sm" data-eaddr="${a.id}">${esc(t('as.edit'))}</button>
      <button class="btn sm danger" data-daddr="${a.id}">${esc(t('as.delete'))}</button>`)}</td></tr>`;
  el.innerHTML = `<div class="grid">
    ${panel('c12', t('nav.ipam'), '', `<p class="muted" style="margin:0 0 12px;font-size:13px">${esc(t('as.what_ipam'))}</p>
      <table><tr><th>${esc(t('as.f_prefix'))}</th><th>${esc(t('as.f_name'))}</th><th>${esc(t('as.f_vlan'))}</th>
        <th>${esc(t('as.f_site_id'))}</th><th class="num">${esc(t('as.assigned'))}</th><th></th><th></th></tr>
      ${subs.map(subRow).join('') || `<tr><td colspan="7" class="empty">${esc(t('as.no_subnets'))}</td></tr>`}</table>
      ${adminOnly(`<div style="margin-top:10px"><button class="btn" id="asAddSub">${esc(t('as.add_subnet'))}</button></div>`)}`)}
    ${sel ? panel('c12', t('as.addrs_of', {x: sel.prefix}), sel.hosts >= 0 ? t('as.free_count', {n: nf(sel.free)}) : '',
      `${free.length ? `<p class="freelist">${esc(t('as.next_free'))}: ${free.map(f => `<button class="lnk mono" data-free="${esc(f)}">${esc(f)}</button>`).join('')}</p>` : ''}
       <table><tr><th>${esc(t('as.f_ip'))}</th><th>${esc(t('as.f_name'))}</th><th>${esc(t('as.f_kind'))}</th>
         <th>${esc(t('as.f_device_id'))}</th><th>${esc(t('as.f_mac'))}</th><th>${esc(t('as.f_tags'))}</th><th></th></tr>
       ${addrs.map(addrRow).join('') || `<tr><td colspan="7" class="empty">${esc(t('as.no_addrs'))}</td></tr>`}</table>
       ${adminOnly(`<div style="margin-top:10px"><button class="btn" id="asAddAddr">${esc(t('as.add_addr'))}</button></div>`)}`) : ''}
    ${conflicts.length ? panel('c12', t('as.conflicts'), t('as.conflicts_sub'),
      `<table><tr><th>${esc(t('as.f_ip'))}</th><th>${esc(t('as.what'))}</th><th>${esc(t('as.detail'))}</th></tr>
       ${conflicts.map(c => `<tr><td class="mono">${esc(c.ip)}</td><td>${esc(t('as.cf_' + c.kind))}</td><td class="muted">${esc(c.detail)}</td></tr>`).join('')}</table>`) : ''}
  </div>`;
  el.querySelectorAll('[data-sub]').forEach(b => b.onclick = () => { assetSel.subnet = assetSel.subnet === +b.dataset.sub ? 0 : +b.dataset.sub; render(); });
  el.querySelectorAll('[data-esub]').forEach(b => b.onclick = async () => { if (await assetEdit('subnet', {...subs.find(s => s.id === +b.dataset.esub)})) render(); });
  el.querySelectorAll('[data-dsub]').forEach(b => b.onclick = async () => {
    const s = subs.find(x => x.id === +b.dataset.dsub);
    if (await assetRemove('subnet', s, s.prefix)) render();
  });
  el.querySelectorAll('[data-eaddr]').forEach(b => b.onclick = async () => { if (await assetEdit('addr', {...addrs.find(a => a.id === +b.dataset.eaddr)})) render(); });
  el.querySelectorAll('[data-daddr]').forEach(b => b.onclick = async () => {
    const a = addrs.find(x => x.id === +b.dataset.daddr);
    if (await assetRemove('addr', a, a.ip)) render();
  });
  el.querySelectorAll('[data-free]').forEach(b => b.onclick = async () => { if (await assetEdit('addr', {ip: b.dataset.free, kind: 'static'})) render(); });
  const addS = $('#asAddSub'); if (addS) addS.onclick = async () => { if (await assetEdit('subnet', {})) render(); };
  const addA = $('#asAddAddr'); if (addA) addA.onclick = async () => { if (await assetEdit('addr', {ip: free[0] || '', kind: 'static'})) render(); };
};

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
      <td class="nw"><button class="${f.name === sbInfo.active ? 'primary' : 'btn'}" data-sbgo="${esc(f.name)}" ${f.status === 'done' ? '' : 'disabled'}>${t('sb.analyze')}</button> ${admOnly(`<button class="btn" data-sbdel="${esc(f.name)}">${t('sb.delete')}</button>`)}</td></tr>`).join('');
  el.innerHTML = `<div class="grid">${panel('c12', t('sb.title'), t('sb.sub'), `<p class="sbexplain">${esc(t('sb.explain'))}</p>
    <table><tr><th>${t('sb.col_file')}</th><th class="num">${t('sb.col_size')}</th><th class="num">${t('sb.col_packets')}</th><th class="num">${t('sb.col_flows')}</th><th>${t('sb.col_span')}</th><th>${t('sb.col_status')}</th><th></th></tr>
    ${rows || `<tr><td colspan="7" class="empty">${t('sb.empty')}</td></tr>`}</table>
    <div class="sbactions"><label class="btn ${own >= sbInfo.max_files ? 'disabled' : ''}" style="cursor:pointer">${t('sb.upload')}<input type="file" id="sbFile" accept=".pcap,.pcapng,.cap" multiple hidden ${own >= sbInfo.max_files ? 'disabled' : ''}></label>
      ${fs.length && adm() ? `<button class="btn" id="sbAll">${t('sb.delete_all')}</button>` : ''}<span id="sbMsg" style="font-size:13px"></span></div>
    <p class="muted" style="font-size:12.5px;margin:10px 0 0">${esc(sbInfo.max_total >= sbInfo.max_file_size * sbInfo.max_files ? t('sb.limits', {n: sbInfo.max_files, mb}) : sbInfo.max_total <= sbInfo.max_file_size ? t('sb.limits_total', {n: sbInfo.max_files, gb: fmtBytes(sbInfo.max_total)}) : t('sb.limits', {n: sbInfo.max_files, mb}) + ' ' + t('sb.limits_total', {n: sbInfo.max_files, gb: fmtBytes(sbInfo.max_total)}))} ${esc(t('sb.formats'))}</p>`)}</div>`;
  // the last upload problem stays shown after the page refreshes
  const msg = (ok, text) => { sbMsg = ok === false ? text : ''; const m = $('#sbMsg'); m.style.color = ok === null ? 'var(--ink-3)' : ok ? 'var(--good)' : 'var(--crit)'; m.textContent = text; };
  if (sbMsg) msg(false, sbMsg);
  el.querySelectorAll('[data-sbgo]').forEach(b => b.onclick = async () => { if (await sbSelect(b.dataset.sbgo)) { state.ds = 'sb'; go('overview'); } });
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

// Account: one's own password; for the administrator also the users, and
// the coming LDAP / AD sign-in.
views.account = async (el) => {
  await loadMe();
  const m = me || {};
  const pwForm = m.fixed ? `<p class="muted">${esc(t('acct.fixed'))}</p>` : `
    ${m.must_change ? `<p class="acctmust">${esc(t('acct.must'))}</p>` : ''}
    <form class="acctform" id="pwForm" autocomplete="off">
      <label ${m.must_change ? 'hidden' : ''}><span>${t('acct.old')}</span><input type="password" name="old" autocomplete="current-password"></label>
      <label><span>${t('acct.new')}</span><input type="password" name="new" autocomplete="new-password" required></label>
      <label><span>${t('acct.confirm')}</span><input type="password" name="again" autocomplete="new-password" required></label>
      <p class="muted" style="font-size:12.5px;margin:0">${esc(t('acct.rule'))}</p>
      <div><button class="primary" type="submit">${t('acct.save')}</button> <span id="pwMsg"></span></div>
    </form>`;
  let users = null;
  if (m.admin && !m.fixed && !m.must_change) {
    const r = await fetch('/api/users');
    if (r.ok) users = await r.json();
  }
  const userRows = users ? users.users.map(u => `<tr><td><b>${esc(u)}</b>${u === users.admin ? ` <span class="tag">${t('acct.admin_tag')}</span>` : ''}</td>
      <td class="nw"><button class="btn" data-reset="${esc(u)}">${t('acct.reset')}</button>${u === users.admin ? '' : ` <button class="btn" data-udel="${esc(u)}">${t('acct.delete')}</button>`}</td></tr>
      <tr data-resetrow="${esc(u)}" hidden><td colspan="2"><form class="acctinline" data-resetform="${esc(u)}"><input type="password" name="pw" placeholder="${esc(t('acct.new'))}" autocomplete="new-password" required> <button class="primary" type="submit">${t('acct.save')}</button></form></td></tr>`).join('') : '';
  el.innerHTML = `<div class="grid">
    ${panel(users ? 'c6' : 'c12', t('acct.title_pw'), esc(m.user || ''), pwForm)}
    ${users ? panel('c6', t('acct.users'), t('acct.users_sub'), `<table class="accttable">${userRows}</table>
      <form class="acctinline" id="addUser" autocomplete="off"><input name="user" placeholder="${esc(t('acct.user'))}" required> <input type="password" name="pw" placeholder="${esc(t('acct.new'))}" autocomplete="new-password" required> <button class="primary" type="submit">${t('acct.add')}</button></form>
      ${users.limit > 0 ? `<p class="muted" style="font-size:13px;margin:8px 0 0">${esc(t('acct.users_count', {used: users.users.length, n: users.limit}))}</p>` : ''}
      <p id="userMsg" style="font-size:13px;margin:8px 0 0"></p>`) : ''}
    ${users ? panel('c12', t('acct.ldap'), t('acct.dev'), `<p class="muted" style="margin-top:0">${esc(t('acct.ldap_dev'))}</p>
      <div class="acctform ldap" aria-disabled="true"><label><span>${t('acct.ldap_server')}</span><input disabled placeholder="ldaps://dc.example.com:636"></label>
      <label><span>${t('acct.ldap_base')}</span><input disabled placeholder="DC=example,DC=com"></label></div>`) : ''}
  </div>`;
  const errText = c => t('acct.err_' + c) !== 'acct.err_' + c ? t('acct.err_' + c) : c;
  const say = (id, ok, text) => { const n = $(id); if (n) { n.style.color = ok ? 'var(--good)' : 'var(--crit)'; n.textContent = text; } };
  const pf = $('#pwForm');
  if (pf) pf.onsubmit = async e => {
    e.preventDefault();
    const old = m.must_change ? (sessionStorage.getItem('t66.old') || 'traffic66') : pf.old.value;
    if (pf.new.value !== pf.again.value) return say('#pwMsg', false, t('acct.err_mismatch'));
    const r = await fetch('/api/password', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({old, new: pf.new.value})});
    const j = await r.json().catch(() => ({}));
    if (!r.ok) return say('#pwMsg', false, errText(j.error));
    sessionStorage.removeItem('t66.old');
    toast(t('acct.saved'));
    if (m.must_change) { me.must_change = false; go('overview'); } else render();
  };
  const putUser = async (user, password) => {
    const r = await fetch('/api/users', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({user, password})});
    const j = await r.json().catch(() => ({}));
    if (!r.ok) { say('#userMsg', false, errText(j.error)); return false; }
    toast(t('acct.added')); render(); return true;
  };
  const au = $('#addUser');
  if (au) au.onsubmit = e => { e.preventDefault(); putUser(au.user.value.trim(), au.pw.value); };
  el.querySelectorAll('[data-reset]').forEach(b => b.onclick = () => { const row = el.querySelector(`[data-resetrow="${CSS.escape(b.dataset.reset)}"]`); row.hidden = !row.hidden; });
  el.querySelectorAll('[data-resetform]').forEach(f => f.onsubmit = e => { e.preventDefault(); putUser(f.dataset.resetform, f.pw.value); });
  el.querySelectorAll('[data-udel]').forEach(b => b.onclick = async () => {
    if (!confirm(t('acct.confirm_del', {u: b.dataset.udel}))) return;
    const r = await fetch('/api/users?user=' + encodeURIComponent(b.dataset.udel), {method: 'DELETE'});
    const j = await r.json().catch(() => ({}));
    if (!r.ok) return say('#userMsg', false, errText(j.error));
    render();
  });
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
  const key = state.r + JSON.stringify(state.f);
  if (state.rk !== key) { state.rk = key; state.rp = 0; }
  const per = state.rps || 50, page = state.rp || 0;
  const live = state.rl && !inSB();
  const d = await api('records', {limit: per, offset: page * per, stats: 1, ...(live ? {range: '15m'} : {})});
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
  el.innerHTML = `<div class="panel" style="margin-bottom:16px"><div class="rechead"><div class="kpi"><div class="lab">${t('rec.title')}</div><div class="n big">${nf(total)}</div><div class="d muted">${esc(t('rec.in', {r: live ? t('range.15m') : rangeLabel()}))}${live ? ' · ' + esc(t('rec.live')) : ''}</div>${!live && !inSB() ? `<button class="btn" id="recLive" style="margin-top:8px">${t('rec.go_live')}</button>` : ''}</div>
      <div class="chart" id="chRec" style="height:150px" aria-label="${esc(t('rec.title'))}"></div></div></div>
    <div class="panel"><div class="ph"><h2>${t('rec.title')}</h2><span class="sub">${esc(t('rec.sorted'))}</span>
      <div class="cols"><button class="btn" id="colBtn">${t('rec.columns')}</button><div class="menu" id="colMenu" hidden>${REC_COLS.map(c => `<label><input type="checkbox" value="${c[0]}" ${on.has(c[0]) ? 'checked' : ''}>${t('rc.' + c[0])}</label>`).join('')}</div></div></div>
    <div style="overflow-x:auto"><table><tr>${cols.map(c => `<th class="${num.has(c) ? 'num' : ''}">${t('rc.' + c)}</th>`).join('')}</tr>
    ${rows.map(r => `<tr>${cols.map(c => `<td class="${num.has(c) ? 'num' : ''}">${cell(c, r)}</td>`).join('')}</tr>`).join('') || `<tr><td colspan="${cols.length}" class="empty">${emptyText()}</td></tr>`}</table></div>${pager}</div>`;
  if (d.hist) barChart($('#chRec'), d.hist.times, d.hist.values[0], v => t('rec.n', {n: nf(v)}));
  $('#recPer').onchange = e => { state.rps = +e.target.value; state.rp = 0; render(); };
  if ($('#recLive')) $('#recLive').onclick = () => { state.rl = true; state.rp = 0; render(); };
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
// ifDev is an interface's worst deviation over the directions that have
// flows to compare, or null when no direction has any.
function ifDev(f) {
  const d = [f.in_no_flow ? null : Math.abs(f.in_dev), f.out_no_flow ? null : Math.abs(f.out_dev)].filter(v => v != null);
  return d.length ? Math.max(...d) : null;
}
// ok, warn, noflow (counters but no flows: an export problem, not a
// deviation, and not ranked as one), or null without counters
function devKind(f) {
  if (!f.has_counters) return null;
  const dv = ifDev(f);
  if (dv != null && dv > Math.max(0.02, 2 * (f.stat_err || 0))) return 'warn';
  if (f.in_no_flow || f.out_no_flow) return 'noflow';
  return 'ok';
}
// Interface check: the selected interface's traffic in both directions
// (ingress green, egress blue) in bits/s and packets/s, the list of
// interfaces, and flow numbers next to the interface counters.
const IF_IN = 'var(--c3)', IF_OUT = 'var(--c1)';
function ifVerdict(r, rc, dirIn) {
  const dev = dirIn ? r.in_dev : r.out_dev, err = dirIn ? r.in_stat_err : r.out_stat_err;
  if (!r.has_counters) return {text: t('if.no_counters'), kind: 'none', dev, err};
  if (dirIn ? r.in_no_flow : r.out_no_flow) return {text: t('if.no_flow'), kind: 'noflow', dev, err};
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
  const list = (d.ifaces || []).slice().sort((a, b) => (!!a.peer - !!b.peer) || (!!b.has_counters - !!a.has_counters) || ((devKind(b) === 'warn') - (devKind(a) === 'warn')));
  if (!list.length) {
    // local capture has no device interfaces or counters to compare with
    const src = await api('sources').catch(() => ({}));
    const capOnly = (src.captures || []).length && !(src.sources || []).length;
    el.innerHTML = `<div class="panel"><div class="empty">${esc(t(capOnly ? 'if.capture_only' : 'empty.nodata'))}</div></div>`;
    return;
  }
  if (!state.ifc || !list.some(f => f.exporter === state.ifc.exporter && f.ifindex === state.ifc.ifindex)) { const d = list.find(f => f.default) || list.find(f => !f.peer) || list[0]; state.ifc = {exporter: d.exporter, ifindex: d.ifindex}; }
  const key = state.ifc.exporter + '/' + state.ifc.ifindex, sel = list.find(f => f.exporter === state.ifc.exporter && f.ifindex === state.ifc.ifindex);
  // one device's interface: count everything it saw, also traffic another device reported too
  const fil = [...state.f, {f: 'iface', v: key, neg: false}];
  const [ri, ro, pi, po, rc] = await Promise.all([['if_in', 'wire'], ['if_out', 'wire'], ['if_in', 'pkts'], ['if_out', 'pkts']].map(([by, measure]) => api('series', {by, measure, top: 20, dup: 1}, fil))
    .concat([api('recon', {exporter: state.ifc.exporter, ifindex: state.ifc.ifindex})]));
  const pick = s => { const i = (s.names || []).indexOf(key); return i < 0 ? (s.times || []).map(() => 0) : s.values[i]; };
  const name = `${ifName(sel)} · ${sel.device || sel.exporter}`;
  const r = rc.recon || {}, ctr = !!r.has_counters;
  const leg = withCtr => `<div class="legend"><span><i style="background:${IF_IN}"></i>${t('if.in')}</span><span><i style="background:${IF_OUT}"></i>${t('if.out')}</span>${withCtr ? `<span><i class="line dash" style="border-top:2px dashed var(--ink-2);background:none"></i>${t('if.counter')}</span>` : ''}</div>`;
  el.innerHTML = `<div class="grid" style="margin-bottom:16px">
    ${panel('c12', `${esc(name)} · ${t('ch.bps')}`, sel.peer ? esc(t('if.peer_note')) : '', `<div class="chart" id="ifB" style="height:270px" aria-label="${esc(t('ch.bps'))}"></div>${leg(ctr)}`)}
    ${panel('c12', `${esc(name)} · ${t('ch.pps')}`, '', `<div class="chart" id="ifP" style="height:270px" aria-label="${esc(t('ch.pps'))}"></div>${leg(false)}`)}</div>
    <div class="grid">
    ${panel('c4', t('if.title'), t('if.sub'), `<div class="iflist">${list.map((f, i) => {
      const k = devKind(f), cur = f.exporter === state.ifc.exporter && f.ifindex === state.ifc.ifindex, id = esc(f.exporter + '/' + f.ifindex);
      const head = f.peer && !list[i - 1]?.peer ? `<div class="ifpeers">${esc(t('if.peers'))}</div>` : '';
      return `${head}<div class="ifrow${f.peer ? ' peer' : ''}"><button data-e="${esc(f.exporter)}" data-i="${f.ifindex}" aria-current="${cur}"><span>${esc(ifName(f))}${f.tag ? `<span class="iftag">${esc(f.tag)}</span>` : ''}<br><span class="muted" style="font-size:12.5px">${esc(f.device || f.exporter)} · ${f.ifindex}</span></span>${k === 'noflow' && ifDev(f) == null ? `<span class="muted" style="font-size:12.5px">${t('if.no_flow_short')}</span>` : k ? status(k === 'noflow' ? 'ok' : k, nf(ifDev(f) * 100, 1) + '%') : `<span class="muted" style="font-size:12.5px">${t('if.no_ctr_short')}</span>`}</button>
        ${f.peer && !f.default || !adm() ? '' : `<button class="ifstar ${f.default ? 'on' : ''}" data-star="${id}" title="${esc(t(f.default ? 'if.is_default' : 'if.set_default'))}" aria-label="${esc(t('if.set_default'))}">${f.default ? '★' : '☆'}</button>`}${admOnly(`<button class="ifedit" data-edit="${id}" title="${esc(t('if.rename'))}" aria-label="${esc(t('if.rename'))}">✎</button>`)}</div>
        ${admOnly(`<form class="ifform" data-form="${id}" hidden><input name="name" value="${esc(f.name || '')}" placeholder="${esc(t('if.name_ph'))}"><input name="tag" value="${esc(f.tag || '')}" placeholder="${esc(t('if.tag_ph'))}"><button class="btn primary" type="submit">${t('if.save')}</button><button class="btn" type="button" data-cancel>${t('nm.cancel')}</button></form>`)}`;
    }).join('')}</div>`)}
    <div class="panel c8" id="recon"></div></div>`;
  // the device counters as dashed lines over the flow estimate: each counter
  // interval holds until the next one
  const onTimes = (times, vals) => { let j = -1; return times.map(x => { while (j + 1 < r.times.length && r.times[j + 1] <= x) j++; return j < 0 || vals == null ? null : vals[j]; }); };
  const lb = [{name: t('if.in'), color: IF_IN, data: pick(ri), fill: true}, {name: t('if.out'), color: IF_OUT, data: pick(ro), fill: true}];
  if (ctr && r.times && r.times.length) lb.push({name: t('if.in') + ' · ' + t('if.counter'), color: IF_IN, dash: true, keep: true, data: onTimes(ri.times, r.in_counter)},
    {name: t('if.out') + ' · ' + t('if.counter'), color: IF_OUT, dash: true, keep: true, data: onTimes(ri.times, r.out_counter)});
  tsChart($('#ifB'), {times: ri.times, settledTo: ri.settled_to, dataFrom: ri.data_from, lines: lb});
  tsChart($('#ifP'), {times: pi.times, settledTo: pi.settled_to, dataFrom: pi.data_from, fmtY: fmtAxisBps, fmtV: fmtPps, lines: [{name: t('if.in'), color: IF_IN, data: pick(pi), fill: true}, {name: t('if.out'), color: IF_OUT, data: pick(po), fill: true}]});
  el.querySelectorAll('.iflist button[data-e]').forEach(b => b.onclick = () => { state.ifc = {exporter: b.dataset.e, ifindex: +b.dataset.i}; render(); });
  // name, tag and default: written to the names (iface line) at once
  const saveIf = async (id, body) => {
    const [exporter, idx] = [id.slice(0, id.lastIndexOf('/')), +id.slice(id.lastIndexOf('/') + 1)];
    const res = await fetch('/api/iface', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({exporter, ifindex: idx, ...body})});
    if (!res.ok) { const j = await res.json().catch(() => ({})); toast(j.error || res.statusText); return; }
    ifList = null; render();
  };
  const ifOf = id => list.find(f => f.exporter + '/' + f.ifindex === id) || {};
  el.querySelectorAll('[data-star]').forEach(b => b.onclick = () => { const f = ifOf(b.dataset.star); saveIf(b.dataset.star, {name: f.name || '', tag: f.tag || '', default: !f.default}); });
  el.querySelectorAll('[data-edit]').forEach(b => b.onclick = () => { const fm = el.querySelector(`[data-form="${b.dataset.edit}"]`); fm.hidden = !fm.hidden; if (!fm.hidden) fm.name.focus(); });
  el.querySelectorAll('[data-form]').forEach(fm => {
    fm.querySelector('[data-cancel]').onclick = () => { fm.hidden = true; };
    fm.onsubmit = e => { e.preventDefault(); const f = ifOf(fm.dataset.form); saveIf(fm.dataset.form, {name: fm.name.value, tag: fm.tag.value, default: !!f.default}); };
  });

  const vin = ifVerdict(r, rc, true), vout = ifVerdict(r, rc, false);
  const head = v => ctr && v.kind !== 'noflow' ? t('if.dev', {v: fmtPct(v.dev)}) : '—';
  const vk = [vin, vout].some(v => v.kind === 'warn') ? 'warn' : [vin, vout].some(v => v.kind === 'noflow') ? 'noflow' : [vin, vout].every(v => v.kind === 'none') ? 'none' : '';
  $('#recon').innerHTML = `<div class="ph"><div><div class="sub">${esc(name)} · ${t('if.flow')} / ${t('if.counter')}</div>
      <div style="font-size:20px;font-weight:650"><span style="color:${IF_IN}">${t('if.in')}</span> ${head(vin)} · <span style="color:${IF_OUT}">${t('if.out')}</span> ${head(vout)}</div></div></div>
    <div class="verdict ${vk}"><b>${t('if.in')}</b>: ${vin.text}<br><b>${t('if.out')}</b>: ${vout.text}</div>`;
};

// ------------------------------------------------------ 95th percentile
// The bill a carrier would send: sort the five-minute samples of a period,
// discard the top five per cent, charge the highest that is left. The
// period is worked out here rather than on the server, because this is
// where the user's time zone is known — a calendar month means the
// customer's midnights, not UTC's.
// last month by default, this month while last month has no counter
// samples yet (a new installation)
let billSel = {p: 'last', from: '', to: '', auto: true};
const billPeriod = () => {
  const now = new Date(), mStart = off => new Date(now.getFullYear(), now.getMonth() + off, 1);
  if (billSel.p === 'this') return [mStart(0), now];
  if (billSel.p === 'last') return [mStart(-1), mStart(0)];
  if (billSel.p === '30d') return [new Date(now - 30 * 864e5), now];
  // Both dates the user typed are days they mean to include, so the end is
  // the midnight after the one they picked.
  const f = new Date(billSel.from + 'T00:00'), t = new Date(billSel.to + 'T00:00');
  if (isNaN(f) || isNaN(t) || t < f) return [mStart(-1), mStart(0)];
  return [f, new Date(t.getTime() + 864e5)];
};
views.billing = async (el) => {
  const [from, to] = billPeriod();
  const [d, ifs] = await Promise.all([
    api('billing', {from: +from, to: +to}).catch(e => ({error: e.message || String(e)})),
    api('ifaces').catch(() => ({ifaces: []})),
  ]);
  const rows = d.bills || [];
  if (billSel.auto) {
    billSel = {...billSel, auto: false};
    if (billSel.p === 'last' && !d.error && !rows.some(b => b.buckets)) { billSel = {...billSel, p: 'this'}; return views.billing(el); }
  }
  const ikey = f => f.exporter + '/' + f.ifindex;
  // A device reports every interface it has. Which of them somebody pays a
  // carrier for is not something that can be worked out from the traffic,
  // so it is marked by hand and kept on the interface's line in the names
  // file, beside its name and its speed.
  const cand = (ifs.ifaces || []).filter(f => f.ifindex && !f.billed);
  const markIf = async (f, billed) => {
    const res = await fetch('/api/iface', {method: 'POST', headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({exporter: f.exporter, ifindex: f.ifindex, name: f.name || '', tag: f.tag || '', default: !!f.default, billed})});
    if (!res.ok) { const j = await res.json().catch(() => ({})); toast(j.error || res.statusText); return; }
    ifList = null; render();
  };
  const zone = Intl.DateTimeFormat().resolvedOptions().timeZone || '';
  const day = x => new Date(x).toLocaleDateString(LANG);
  const iso = x => new Date(x.getTime() - x.getTimezoneOffset() * 60000).toISOString().slice(0, 10);
  const opt = (v, k) => `<option value="${v}" ${billSel.p === v ? 'selected' : ''}>${esc(t(k))}</option>`;
  const picker = `<div class="billbar">
    <label>${esc(t('bill.period'))} <select id="billP">${opt('this', 'bill.this_month')}${opt('last', 'bill.last_month')}${opt('30d', 'bill.last30')}${opt('custom', 'bill.custom')}</select></label>
    <span id="billC" ${billSel.p === 'custom' ? '' : 'hidden'}><input type="date" id="billF" value="${esc(billSel.from || iso(from))}"> – <input type="date" id="billT" value="${esc(billSel.to || iso(new Date(to.getTime() - 864e5)))}"></span>
    <span class="muted">${esc(day(from))} – ${esc(day(to.getTime() - 1))} · ${esc(zone)}</span>
    ${admOnly(cand.length ? `<span style="margin-inline-start:auto"><select id="billAdd"><option value="">${esc(t('bill.add'))}</option>${
      cand.map(f => `<option value="${esc(ikey(f))}">${esc((f.name || t('bill.ifindex', {n: f.ifindex})) + ' · ' + (f.device || f.exporter))}</option>`).join('')}</select></span>` : '')}</div>
    <p class="muted" style="margin:0 0 10px;font-size:13px">${esc(d.marked ? t('bill.marked', {n: nf(d.marked)}) : t('bill.all_shown'))}</p>`;
  const body = d.error ? `<div class="empty">${esc(d.error)}</div>`
    : !rows.length ? `<div class="empty">${esc(t('bill.empty'))}</div>`
    : `<table><tr><th>${t('bill.iface')}</th><th class="num">${t('bill.billed')}</th><th class="num">${t('if.in')} 95</th><th class="num">${t('if.out')} 95</th><th class="num">${t('bill.peak')}</th><th class="num">${t('bill.avg')}</th><th class="num">${t('bill.coverage')}</th><th></th></tr>
      ${rows.map(b => {
        const nm = b.name || t('bill.ifindex', {n: b.ifindex}), dev = b.device || b.exporter;
        const cov = b.coverage, low = cov < 0.95;
        const head = `<td>${esc(nm)}<br><span class="muted" style="font-size:12.5px">${esc(dev)} · ${b.ifindex}</span></td>`;
        if (!b.buckets) return `<tr>${head}<td colspan="6" class="muted">${esc(t('bill.nodata'))}</td>
          <td class="nw" style="text-align:end">${admOnly(`<button class="btn" data-unbill="${esc(b.exporter + '/' + b.ifindex)}">${t('bill.remove')}</button>`)}</td></tr>`;
        return `<tr><td>${esc(nm)}<br><span class="muted" style="font-size:12.5px">${esc(dev)} · ${b.ifindex}</span></td>
          <td class="num"><b>${esc(fmtBps(b.billed))}</b><br><span class="muted" style="font-size:12.5px">${esc(t(b.dir === 'out' ? 'if.out' : 'if.in'))}</span></td>
          <td class="num">${esc(fmtBps(b.in95))}</td><td class="num">${esc(fmtBps(b.out95))}</td>
          <td class="num">${esc(fmtBps(Math.max(b.in_max, b.out_max)))}</td>
          <td class="num">${esc(fmtBps(Math.max(b.in_avg, b.out_avg)))}</td>
          <td class="num"${low ? ' style="color:var(--warn)"' : ''}>${esc(nf(cov * 100, 1))}%<br><span class="muted" style="font-size:12.5px">${esc(nf(b.buckets))}/${esc(nf(b.possible))}</span></td>
          <td class="nw" style="text-align:end">${b.marked ? admOnly(`<button class="btn" data-unbill="${esc(b.exporter + '/' + b.ifindex)}">${t('bill.remove')}</button>`) : ''}</td></tr>`;
      }).join('')}</table>
      <p class="muted" style="margin:12px 0 0;font-size:13px">${esc(t('bill.gap_note'))}</p>`;
  el.innerHTML = `<div class="grid">${panel('c12', t('bill.title'), t('bill.sub'), picker + body)}
    ${panel('c12', t('bill.how'), '', `<p class="muted" style="margin:0;font-size:13px">${esc(t('bill.how_note'))}<br>${esc(t('bill.retention', {n: nf(d.retention_days || 0)}))}</p>`)}</div>`;
  const sel = $('#billP');
  sel.onchange = () => { billSel = {...billSel, p: sel.value}; render(); };
  const f = $('#billF'), tt = $('#billT');
  if (f) f.onchange = tt.onchange = () => { billSel = {p: 'custom', from: f.value, to: tt.value}; render(); };
  const add = $('#billAdd');
  if (add) add.onchange = () => { const f = (ifs.ifaces || []).find(x => ikey(x) === add.value); if (f) markIf(f, true); };
  el.querySelectorAll('[data-unbill]').forEach(b => b.onclick = () => {
    const k = b.dataset.unbill, i = k.lastIndexOf('/');
    const f = (ifs.ifaces || []).find(x => ikey(x) === k) || {exporter: k.slice(0, i), ifindex: +k.slice(i + 1)};
    markIf(f, false);
  });
};

views.sources = async (el) => {
  const [d, gd] = await Promise.all([api('sources'), api('geo').catch(() => ({sources: []}))]);
  const blocked = d.blocked || [];
  const dfmt = d => d && !d.startsWith('0001') ? new Intl.DateTimeFormat(LANG, {dateStyle: 'medium'}).format(new Date(d)) : '';
  const VENDOR = {dbip: 'DB-IP Lite', maxmind: 'MaxMind GeoLite2', ipinfo: 'IPinfo Lite', iptoasn: 'IPtoASN'};
  const geoRows = (gd.sources || []).map(g => `<tr><td>${status('ok', t('geo.kind_' + g.kind))}</td>
      <td>${esc(VENDOR[g.vendor] || g.type)} <span class="muted">${esc(g.file === 'built-in' ? t('geo.builtin') : g.file)}${g.entries ? ' · ' + esc(t('geo.entries', {n: nf(g.entries)})) : ''}</span></td>
      <td class="nw">${esc(dfmt(g.built))}</td><td class="muted">${t(g.fallback ? 'geo.role_fallback' : 'geo.role_main')}</td>
      <td>${adm() && g.file !== 'built-in' ? `<button class="btn" data-geodel="${esc(g.file)}">${t('geo.remove')}</button>` : ''}</td></tr>`).join('');
  const FREE = [
    ['DB-IP Lite', 'geo.f_dbip', 'CC BY 4.0', 'https://db-ip.com/db/lite.php'],
    ['MaxMind GeoLite2', 'geo.f_maxmind', 'GeoLite2 EULA', 'https://www.maxmind.com/en/geolite2/signup'],
    ['IPinfo Lite', 'geo.f_ipinfo', 'CC BY-SA 4.0', 'https://ipinfo.io/lite'],
    ['IPtoASN', 'geo.f_iptoasn', 'PDDL 1.0', 'https://iptoasn.com']];
  const freeRows = FREE.map(([n, k, lic, url]) => `<tr><td class="nw"><a href="${url}" target="_blank" rel="noopener">${n}</a></td><td>${esc(t(k))}</td><td class="nw muted">${lic}</td></tr>`).join('');
  const srcs = d.sources || [];
  const ifd = await loadIfaces().catch(() => ({})), sampledIf = new Map();
  (ifd.ifaces || []).filter(f => !f.peer && (f.in_wire || f.out_wire)).forEach(f => { if (!sampledIf.has(f.exporter)) sampledIf.set(f.exporter, []); sampledIf.get(f.exporter).push(ifName(f)); });
  const obsLine = s => { const l = sampledIf.get(s.exporter), dir = s.dir_field ? t('src.dir_' + s.dir_field) : ''; return l || dir ? `<br><span class="muted" style="font-size:12px">${l ? esc(t('src.sampled_if', {v: l.join(', ')})) : ''}${l && dir ? ' · ' : ''}${esc(dir)}</span>` : ''; };
  const issueText = (code, s) => t('issue.' + code, {a: s.exporter, n: nf(s.pending), p: nf(code === 'agent_drops' ? s.drop_pct : code === 'loss' ? (s.transit_pct ?? s.lost_pct) : s.lost_pct, 1) + '%', s: Math.round(Math.abs(s.clock_skew_ns) / 1e9) + ' s'});
  const rows = srcs.map(s => {
    const kind = s.status;
    return `<tr class="${s.issues?.length ? 'has-fix' : ''}"><td>${s.name ? esc(s.name) + ' ' : ''}<span class="muted" style="font-size:12.5px">${esc(s.exporter)}${s.domain ? ' / ' + s.domain : ''}</span>${obsLine(s)}</td>
      <td class="nw">${esc(s.proto)}</td><td class="num">${nf(s.rec_per_sec, 1)}</td>
      <td class="nw">${s.sampling ? esc(s.sampling) : (s.sampling_state === 'waiting' ? '<span class="muted">?</span>' : '1:1')}${s.sampling_state === 'manual' ? ` <span class="muted">(${esc(t('src.manual'))})</span>` : ''}${s.effective ? ` <span class="muted">(${esc(t('src.effective', {v: '1:' + nf(s.effective)}))})</span>` : ''}${s.wait_step ? `<br><span class="muted" style="font-size:12px">${esc(t('src.wait_' + s.wait_step))}${s.first_seen ? ' · ' + esc(t('wait.elapsed', {m: Math.max(1, Math.floor((Date.now() - Date.parse(s.first_seen)) / 60000))})) : ''}</span>` : ''}</td>
      <td class="num">${nf(s.lost_pct, 2)}%</td><td class="nw muted">${s.last_seen ? ago(Date.parse(s.last_seen)) : ''}</td>
      <td>${status(kind, t('st.' + kind))}</td>
      ${admOnly(`<td class="nw" style="text-align:end">${s.blocked
        ? `<span class="tag">${esc(t('blk.on'))}</span>`
        : `<button class="btn sm danger" data-block="${esc(s.exporter)}">${esc(t('blk.block'))}</button>`}</td>`)}</tr>
      ${s.issues?.length ? `<tr><td colspan="${adm() ? 8 : 7}" style="padding-top:0">${s.issues.map(c => `<div class="issue">${esc(issueText(c, s))}</div>`).join('')}${['assumed1', 'waiting', 'manual'].includes(s.sampling_state) && s.template_info?.length ? `<div class="issue muted" style="font-family:ui-monospace,monospace;font-size:12px">${esc(t('src.templates'))} ${s.template_info.map(esc).join(' · ')}</div>` : ''}</td></tr>` : ''}`;
  }).join('');
  const lis = (d.listeners || []).map(l => `<span class="tag">UDP ${esc(l.addr)} · ${esc(l.proto)} · ${nf(l.packets)}</span>`).join(' ');
  const caps = (d.captures || []).map(c => `<span class="tag">${esc(c.iface)} · ${esc(c.method)} · ${nf(c.packets)}${c.error ? ' · ' + esc(c.error) : ''}</span>`).join(' ');
  const snmp = (d.snmp || []).map(x => `<span class="tag" title="${esc(x.error || '')}">${esc(x.name ? x.name + " " + x.exporter : x.exporter)} · ${x.ok
    ? '<span style="color:var(--good)">✔</span> ' + esc(t('src.snmp_ok', {n: nf(x.interfaces)}))
    : '<span style="color:var(--crit)">✖</span> ' + esc(t('src.snmp_fail'))}</span>`).join(' ');
  el.innerHTML = `<div class="grid">
    ${panel('c12', t('src.title'), t('src.sub'), `<table><tr><th>${t('col.device')}</th><th>${t('col.proto')}</th><th class="num">${t('col.rate')}</th><th>${t('col.sampling')}</th><th class="num">${t('col.lost')}</th><th>${t('col.last')}</th><th>${t('col.status')}</th>${admOnly('<th></th>')}</tr>
      ${rows || `<tr><td colspan="${adm() ? 8 : 7}" class="empty">${waitHint(firstText())}</td></tr>`}</table>
      <p style="margin:12px 0 0;font-size:12.5px"><span class="muted">${t('src.listeners')}</span> ${lis || '—'} ${caps}</p>
      ${(d.stale || []).length ? `<p style="margin:6px 0 0;font-size:12.5px"><span class="muted">${t('src.stale')}</span> ${d.stale.map(x => `<span class="tag">${esc(x.name || x.exporter)} / ${x.domain} · ${esc(x.proto)} · ${esc(ago(Date.parse(x.last_seen)))}</span>`).join(' ')}</p>` : ''}
      ${snmp ? `<p style="margin:6px 0 0;font-size:12.5px"><span class="muted">${t('src.snmp')}</span> ${snmp}</p>` : ''}`)}
    ${admOnly(panel('c12', t('lic.install'), t('lic.install_sub'), `<div class="licbox">
      <p class="muted" style="margin:0 0 8px;font-size:13px">${esc(t('lic.id', {id: lic?.installation_id || '—'}))}</p>
      <textarea class="inv" id="licText" spellcheck="false" placeholder="${esc(t('lic.paste'))}"></textarea>
      <div style="display:flex;gap:10px;align-items:center;margin-top:8px;flex-wrap:wrap">
        <label class="btn" style="cursor:pointer"><input type="file" id="licFile" accept=".json,application/json" hidden>${t('lic.choose')}</label>
        <button class="primary" id="licSave">${t('lic.install_btn')}</button>
        <span id="licMsg" style="font-size:13px"></span></div></div>`))}
    ${admOnly(blocked.length ? panel('c12', t('blk.title'), t('blk.sub'), `<table><tr><th>${t('blk.col_addr')}</th><th>${t('blk.col_since')}</th><th class="num">${t('blk.col_dropped')}</th><th>${t('blk.col_note')}</th><th></th></tr>
      ${blocked.map(b => `<tr><td class="nw"><code>${esc(b.address)}</code></td>
        <td class="nw muted">${b.since ? esc(new Intl.DateTimeFormat(LANG, {dateStyle: 'short', timeStyle: 'short'}).format(new Date(b.since))) : ''}</td>
        <td class="num">${nf(b.dropped || 0)}</td><td>${esc(b.note || '')}</td>
        <td class="nw" style="text-align:end"><button class="btn sm" data-unblock="${esc(b.address)}">${esc(t('blk.unblock'))}</button></td></tr>`).join('')}</table>
      <p class="muted" style="font-size:12.5px;margin:10px 0 0">${esc(t('blk.firewall'))}</p>`) : '')}
    ${panel('c12', t('geo.title'), t('geo.sub'), `<table><tr><th>${t('geo.col_holds')}</th><th>${t('geo.col_db')}</th><th>${t('geo.col_built')}</th><th>${t('geo.col_use')}</th><th></th></tr>${geoRows}</table>
      ${admOnly(`<div style="display:flex;gap:10px;align-items:center;flex-wrap:wrap;margin-top:12px"><button class="primary" id="geoDbip">${t('geo.update')}</button><label class="btn" style="cursor:pointer">${t('geo.upload')}<input type="file" id="geoFile" accept=".mmdb,.tsv,.gz,.txt,.csv" hidden></label><span id="geoMsg" class="muted" style="font-size:13px"></span></div>`)}
      <h3 class="subhead">${t('geo.free')}</h3><p class="muted" style="font-size:12.5px;margin:0 0 6px">${t('geo.free_sub')}</p>
      <table><tr><th>${t('geo.col_db')}</th><th>${t('geo.col_holds')}</th><th>${t('geo.col_lic')}</th></tr>${freeRows}</table>
      <p class="muted" style="font-size:12.5px;margin:10px 0 0">${t('geo.where')}</p>`)}
    ${panel('c12', t('logo.title'), t('logo.sub'), `<div class="logoprev"><img id="logoPrev" src="/logo?v=${Date.now()}" alt="logo">
      <div style="display:grid;gap:8px">${admOnly(`<div style="display:flex;gap:10px;align-items:center;flex-wrap:wrap"><label class="primary" style="cursor:pointer;display:inline-block;padding:6px 14px;border-radius:6px;background:var(--accent);color:#fff;font-weight:600">${t('logo.upload')}<input type="file" id="logoFile" accept=".png,.svg,.jpg,.jpeg,.webp,.gif,image/*" hidden></label><button class="btn" id="logoReset">${t('logo.reset')}</button></div>`)}
      ${admOnly(`<span class="muted" style="font-size:12.5px">${t('logo.hint')}</span>`)}<span id="logoMsg" style="font-size:13px"></span></div></div>`)}
  </div>`;
  // Installing a licence from the page. Until now the only way to register
  // was to put license.json into the data directory by hand, which means a
  // shell on the machine — a strange thing to ask of the one step that
  // turns a paying customer into a registered installation.
  if (adm()) {
    const licMsg = (ok, text) => { const m = $('#licMsg'); m.style.color = ok ? 'var(--good)' : 'var(--crit)'; m.textContent = text; };
    $('#licFile').onchange = async e => {
      const f = e.target.files[0];
      if (f) $('#licText').value = await f.text();
      e.target.value = '';
    };
    $('#licSave').onclick = async () => {
      const text = $('#licText').value.trim();
      if (!text) { licMsg(false, t('lic.paste')); return; }
      const res = await fetch('/api/license', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({text})});
      const j = await res.json().catch(() => ({}));
      if (!res.ok) { licMsg(false, j.error || res.statusText); return; }
      licMsg(true, t('src.saved'));
      $('#licText').value = '';
      loadStatus(); // the badge, the footer and the watermark all follow from it
    };
  }
  const geoMsg = (ok, text) => { const m = $('#geoMsg'); m.style.color = ok === null ? '' : ok ? 'var(--good)' : 'var(--crit)'; m.textContent = text; };
  if (adm()) {
  const reblock = async (method, path, okMsg) => {
    const res = await fetch(path, {method});
    const j = await res.json().catch(() => ({}));
    if (!res.ok) { toast(j.error === 'no_collector' ? t('blk.no_collector') : (j.error || res.statusText)); return; }
    toast(okMsg);
    render();
  };
  el.querySelectorAll('[data-block]').forEach(b => b.onclick = async () => {
    const a = b.dataset.block;
    if (!confirm(t('blk.confirm', {a}))) return;
    const res = await fetch('/api/sources/block', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({exporter: a})});
    const j = await res.json().catch(() => ({}));
    if (!res.ok) { toast(j.error === 'no_collector' ? t('blk.no_collector') : (j.error || res.statusText)); return; }
    // The honest part: this stops a known nuisance, not somebody who does
    // not want to be stopped, so the message says where the real fix is.
    toast(t('blk.done_firewall', {a}));
    render();
  });
  el.querySelectorAll('[data-unblock]').forEach(b => b.onclick = () =>
    reblock('DELETE', '/api/sources/block?exporter=' + encodeURIComponent(b.dataset.unblock), t('blk.unblocked', {a: b.dataset.unblock})));
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
    if (ok) { const v = Date.now(), dk = document.documentElement.dataset.theme === 'black' ? 1 : 0; $('#logoPrev').src = '/logo?v=' + v; document.querySelectorAll('img.logo').forEach(i => i.src = `/logo?dark=${dk}&v=${v}`); }
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
  }
};

// Names: which addresses are what. It lives under asset management rather
// than in the settings, because naming a host is the same kind of work as
// recording a rack, and it is the one page there that an unregistered
// installation keeps: a flow analyser that cannot tell you which address is
// which is of little use, and that is not a feature to sell.
views.names = async (el) => {
  const inv = await api('inventory').catch(() => ({text: ''}));
  el.innerHTML = `<div class="grid">
    ${panel('c12', t('src.names'), t('src.names_sub'), `${admOnly(`<div class="nmform" id="nmForm">
        <select id="nmK">${NM_KINDS.map(k => `<option value="${k}">${t('nm.k_' + k)}</option>`).join('')}</select>
        <input id="nmA" autocomplete="off" spellcheck="false"><input id="nmI" autocomplete="off" inputmode="numeric" placeholder="${esc(t('nm.ifindex'))}" hidden>
        <input id="nmN" autocomplete="off" spellcheck="false"><select id="nmC" hidden></select>
        <button class="primary" id="nmAdd">${t('nm.add')}</button><button class="btn" id="nmCancel" hidden>${t('nm.cancel')}</button></div>
      <div id="nmMsg" style="font-size:13px;min-height:1.2em;margin:4px 0 6px"></div>`)}
      <table class="nmtab" id="nmTab"></table>
      <details class="nmadv"><summary>${t('nm.advanced')}</summary><div class="helpbox">${esc(t('src.names_help'))}<pre>host   192.168.3.28    ${esc(t('src.ex_host'))}
net    192.168.3.0/24  ${esc(t('src.ex_net'))}
device 192.168.1.1     ${esc(t('src.ex_device'))}
iface  192.168.1.1 3   ${esc(t('src.ex_iface'))}
snmp   192.168.1.1     public</pre></div><textarea class="inv" id="inv" spellcheck="false" ${adm() ? '' : 'readonly'}>${esc(inv.text || '')}</textarea>
      ${admOnly(`<div style="display:flex;gap:10px;align-items:center;margin-top:8px"><button class="primary" id="invSave">${t('src.save')}</button><span id="invMsg" class="muted" style="font-size:13px"></span></div>`)}</details>`)}
  </div>`;
  if (adm()) {
    $('#invSave').onclick = async () => {
      const res = await fetch('/api/inventory', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({text: $('#inv').value})});
      const j = await res.json().catch(() => ({}));
      $('#invMsg').textContent = res.ok ? t('src.saved') : (j.error || res.statusText);
      $('#invMsg').style.color = res.ok ? 'var(--good)' : 'var(--crit)';
      if (res.ok) { loadStatus(); namesForm(); }
    };
  }
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
  const form = !!$('#nmForm');
  const rows = nmParse($('#inv').value);
  const sync = () => {
    const k = $('#nmK').value;
    $('#nmI').hidden = k !== 'iface';
    $('#nmA').placeholder = t(k === 'net' ? 'nm.ph_net' : 'nm.ph_addr');
    $('#nmN').placeholder = t(k === 'snmp' ? 'nm.community' : 'nm.name');
    $('#nmC').hidden = k !== 'net';
  };
  // countries for networks, named in the reader's language
  if (form) worldMap().then(w => {
    const cur = $('#nmC').value;
    $('#nmC').innerHTML = `<option value="">${esc(t('nm.country_none'))}</option>` + Object.keys(w.c).map(cc => [cc, country(cc)]).sort((x, y) => x[1].localeCompare(y[1], LANG)).map(([cc, n]) => `<option value="${cc}">${esc(n)}</option>`).join('');
    $('#nmC').value = cur;
  }).catch(() => {});
  tab.innerHTML = `<tr><th>${t('nm.type')}</th><th>${t('nm.address')}</th><th>${t('nm.name')}</th><th></th></tr>` + (rows.map(r => `<tr><td class="nw">${t('nm.k_' + r.k)}</td>
      <td class="nw"><code>${esc(r.a)}${r.k === 'iface' ? ' #' + esc(r.x) : ''}</code></td><td>${r.c ? `<span class="tag">${esc(country(r.c))}</span> ` : ''}${r.k === 'snmp' ? `<span class="muted">${esc(t('nm.community'))}:</span> ` : ''}${esc(r.n.replace(/\s*\bspeed=\d+/, ''))}${/\bspeed=(\d+)/.test(r.n) ? ` <span class="muted">${fmtBps(+r.n.match(/\bspeed=(\d+)/)[1])}</span>` : ''}</td>
      <td class="nw" style="text-align:end">${admOnly(`<button class="btn" data-nme="${r.i}">${t('nm.edit')}</button> <button class="btn" data-nmd="${r.i}">${t('nm.delete')}</button>`)}</td></tr>`).join('') ||
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
  // The table above is for everyone; the rest of this wires the form,
  // which is only on the page for an administrator.
  if (!form) return;
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
  const lines = d.baseline && d.basis ? [{name: baseText(d, 'line'), color: 'var(--base)', dash: true, keep: true, data: d.baseline}] : [];
  const table = (rows, heads, cell) => { const m = Math.max(1, ...rows.map(r => r.wire)); return `<table><tr><th></th>${heads.map(h => `<th>${t(h)}</th>`).join('')}<th class="num">${t('col.traffic')}</th><th style="width:26%">${t('col.share')}</th></tr>
    ${rows.map((r, i) => `<tr><td class="rank">${i + 1}</td>${cell(r).map(c => `<td>${c}</td>`).join('')}<td class="num">${fmtBytes(r.wire)}</td><td>${bar(r.wire, m)}</td></tr>`).join('') || `<tr><td colspan="${heads.length + 3}" class="empty">${emptyText()}</td></tr>`}</table>`; };
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
    <div class="panel c12"><div class="ph"><h2>${t('ov.bw_title')}</h2><span class="sub">${lines.length ? esc(baseText(d, 'sub')) : ''}</span></div>
      <div class="chart" id="chDet" style="height:220px" aria-label="${esc(t('ov.bw_title'))}"></div>
      <div class="legend">${areas.map((a, i) => `<span><i style="background:${a.color}"></i>${series.names[i] === '__other__' ? esc(a.name) : V('app', series.names[i], a.name)}</span>`).join('')}</div></div>
    ${panel('c12', t('det.conv'), '', table(cv, ['col.client', 'col.server', 'col.service', 'col.country'], r => [ipCell(r.key), ipCell(r.key2), V('port', r.key3, r.key3), esc(country(r.extra))]))}
    ${isIP
      ? panel('c6', t('det.services'), '', table(s1, ['col.port', 'col.app'], r => [V('port', r.key, r.key), esc(r.extra)])) + panel('c6', t('ov.country'), '', table(s2, ['col.country'], r => [V('country', r.key, country(r.key))]))
      : panel('c6', t('det.clients'), '', table(s1, ['col.client'], r => [ipCell(r.key)])) + panel('c6', t('det.servers'), '', table(s2, ['col.server', 'col.country'], r => [ipCell(r.key), r.extra ? esc(country(r.extra)) : '']))}
    ${panel('c12', t('det.flows'), esc(t('rec.latest', {n: rr.length})), `<div style="overflow-x:auto"><table><tr><th>${t('rc.time')}</th><th>${t('rc.client')}</th><th>${t('rc.server')}</th><th class="num">${t('rc.port')}</th><th>${t('rc.app')}</th><th>${t('rc.country')}</th><th class="num">${t('rc.traffic')}</th></tr>
      ${rr.map(r => `<tr><td class="muted nw">${new Intl.DateTimeFormat(LANG, {hour: '2-digit', minute: '2-digit', second: '2-digit'}).format(new Date(r.ts))}</td><td>${ipCell(r.client)}</td><td>${ipCell(r.server)}</td><td class="num">${r.port ? V('port', r.port + '/' + (r.proto === 17 ? 'udp' : 'tcp'), r.port) : '—'}</td><td>${V('app', r.app, r.app)}</td><td>${r.dir === 3 ? `<span class="muted">${t('internal')}</span>` : (r.cc ? V('country', r.cc, country(r.cc)) : '—')}</td><td class="num">${fmtBytes(r.wire)}</td></tr>`).join('') || `<tr><td colspan="7" class="empty">${emptyText()}</td></tr>`}</table></div>`)}
  </div>`;
  tsChart($('#chDet'), {times: series.times, areas, lines, settledTo: series.settled_to, dataFrom: series.data_from});
  bindFindings(el);
};

// ------------------------------------------------------------ shell
// Settings, data cleanup, offline analysis and live flow records have no use
// for the time range, refresh and copy link.
const NOBAR_VIEWS = new Set(['sources', 'cleanup', 'sandbox', 'account', 'racks', 'devices', 'ipam', 'billing', 'names']);
function renderBar() {
  const off = NOBAR_VIEWS.has(state.v) || (state.v === 'records' && state.rl);
  if (off) $('#range').hidden = true;
  $('#refresh').hidden = off || $('#range').hidden;
  $('#share').hidden = off;
}
// The sandbox bar replaces the time range while capture files are shown.
function renderSB() {
  const on = inSB() && !['ifaces', 'sources', 'racks', 'devices', 'ipam'].includes(state.v);
  $('#sbbar').hidden = !on;
  $('#range').hidden = on;
  $('#refresh').hidden = on;
  document.querySelectorAll('#nav [data-v=ifaces], #nav [data-v=sources], #nav [data-v=cleanup], #nav .navgrp').forEach(b => b.hidden = inSB() || offlineMode);
  if (!on) return;
  const r = sbRange(), span = r.to - r.from;
  const df = new Intl.DateTimeFormat(LANG, {dateStyle: 'medium', timeStyle: 'short'}), tf = new Intl.DateTimeFormat(LANG, {timeStyle: 'short'});
  const sameDay = new Date(r.from).toDateString() === new Date(r.to).toDateString();
  // one file at a time; the others are a click away
  const done = sbInfo.files.filter(f => f.status === 'done');
  const pick = done.length > 1 ? `<select id="sbPick" aria-label="${esc(t('sb.col_file'))}">${done.map(f => `<option${f.name === sbInfo.active ? ' selected' : ''}>${esc(f.name)}</option>`).join('')}</select>` : `<span class="sbfiles">${esc(sbInfo.active)}</span>`;
  $('#sbbar').innerHTML = `<span class="sbtag">${t('sb.banner')}</span>${pick}
    <span class="muted nw">${esc(df.format(r.from))} – ${esc(sameDay ? tf.format(r.to) : df.format(r.to))} (${esc(fmtDur(span))})</span>
    ${offlineMode ? '' : `<button class="btn" id="sbBack">${t('sb.back')}</button>`}`;
  if ($('#sbBack')) $('#sbBack').onclick = () => { state.ds = ''; render(); };
  if ($('#sbPick')) $('#sbPick').onchange = async e => { if (await sbSelect(e.target.value)) render(); };
}
// sbSelect makes one capture file the one every page shows.
async function sbSelect(name) {
  if (name === sbInfo.active && sbInfo.ready) return true;
  const res = await fetch('/api/sandbox/active?name=' + encodeURIComponent(name), {method: 'POST'});
  const j = await res.json().catch(() => ({}));
  if (!res.ok) { toast(j.error || res.statusText); return false; }
  sbInfo = j;
  return true;
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
  // the default password of a new installation is changed before anything else
  if (me?.must_change) state.v = 'account';
  document.body.classList.toggle('mustchange', !!me?.must_change);
  renderSB();
  renderBar();
  renderFilters();
  renderFbar();
  writeHash(push);
  $('#title').textContent = t('nav.' + state.v);  // the detail view refines it
  document.title = t('nav.' + state.v) + ' · traffic66';
  document.querySelectorAll('#nav button').forEach(b => b.dataset.v === state.v ? b.setAttribute('aria-current', 'page') : b.removeAttribute('aria-current'));
  // a group whose page is open says so even when it is folded away
  document.querySelectorAll('#nav .navgrp').forEach(g => {
    const inside = [...g.querySelectorAll('[data-v]')].some(b => b.dataset.v === state.v) || (g.id === 'grpAssets' && ASSET_VIEWS.has(state.v));
    g.querySelector('.grphead').dataset.in = inside ? '1' : '0';
  });
  renderRange();
  await renderIfsel();
  writeHash(false);
  document.querySelectorAll('.view').forEach(s => s.classList.toggle('on', s.id === 'v-' + state.v));
  applyMark();
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
    waiting = s.waiting || null;
    setup = s;
    const L = lic = s.license || null;
    if (L) {
      const id = `<span class="muted">${esc(t('lic.id', {id: L.installation_id}))}</span>`;
      let head;
      if (L.registered) head = t(L.days < 0 ? 'lic.perp' : 'lic.ok', {c: L.customer, n: L.days});
      else if (L.kind === 'expired') head = t('lic.exp', {c: L.customer, d: L.expires});
      else head = t('lic.unreg');
      const tail = L.registered ? '' : `<br>${esc(t('lic.free'))} ${esc(t('lic.mark'))} `;
      $('#lic').innerHTML = `<b>${esc(head)}</b>${tail ? tail : ' '}${id}`;
      $('#lic').className = 'lic ' + L.kind; $('#lic').hidden = false;

      marked = !!L.watermark;
      markNotice = L.notice || '';
      applyMark();
      const b = $('#licBadge');
      b.textContent = L.registered ? (L.days < 0 ? t('lic.badge_ok') : t('lic.badge_days', {n: L.days})) : t('lic.badge_get');
      b.className = 'licbadge ' + L.kind;
      b.title = head;
      b.hidden = false;
    }
    $('#verNo').textContent = 'v' + s.version + (s.build && s.build !== 'official' ? ' · ' + s.build : '');
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
  // the default password, while a new installation (or the demo) still has it
  fetch('/api/loginhint').then(r => r.json()).then(h => {
    const el = $('#loginHint');
    el.hidden = !h.default;
    // The demo hands out its credentials on purpose and says them here. A
    // real installation does not send them, so the hint says where to read
    // them instead: on the console, where the first start printed them.
    if (h.default) {
      el.textContent = h.demo ? t('login.hint_demo', {u: h.user, p: h.password}) : t('login.hint');
      if (h.user && !$('#loginForm [name=user]').value) $('#loginForm [name=user]').value = h.user;
    }
  }).catch(() => {});
}
// me: the signed-in user (see /api/me)
let me = null;
async function loadMe() {
  try { const r = await fetch('/api/me'); if (r.ok) me = await r.json(); } catch (e) {}
  $('#acctName').textContent = me?.user ? me.user : t('nav.account');
}
$('#loginForm').addEventListener('submit', async e => {
  e.preventDefault();
  const f = new FormData(e.target);
  const res = await fetch('/api/login', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({user: f.get('user'), password: f.get('password')})});
  if (!res.ok) { $('#loginErr').hidden = false; $('#loginErr').textContent = t('login.bad'); return; }
  const j = await res.json().catch(() => ({}));
  $('#login').hidden = true; $('#app').hidden = false;
  if (j.must_change) sessionStorage.setItem('t66.old', f.get('password'));
  await Promise.all([loadStatus(), loadSB(), loadMe()]);
  // A new installation has minutes of data, a sliver at the edge of a day's
  // chart: it opens on the last hour until it has two hours of history.
  // A range in the link still wins.
  if (!new URLSearchParams(location.hash.slice(1)).get('r') && setup && !setup.demo && !setup.offline && !(setup.oldest > 0 && Date.now() - setup.oldest > 2 * 3600e3)) state.r = '1h';
  render();
  startTimers();
});
// The refresh timers start once, whether the page opened signed in or after
// signing in (before, signing in left the page without automatic refresh).
let timers = false;
function startTimers() {
  if (timers) return;
  timers = true;
  setInterval(loadStatus, 10000);
  // live flow records: every 5 seconds
  setInterval(() => {
    if (state.v !== 'records' || !state.rl || document.hidden || pop.style.display === 'block' || (state.rp || 0) > 0) return;
    if (document.activeElement?.closest?.('.cols, #recPer')) return;
    render();
  }, 5000);
  setInterval(() => {
    // not while something is being typed or picked (the names form, filters)
    if (document.hidden || pop.style.display === 'block' || ['TEXTAREA', 'INPUT', 'SELECT'].includes(document.activeElement?.tagName)) return;
    if (state.v === 'records' && state.rl) return;
    if (state.v === 'sources' || state.v === 'records' || state.v === 'overview' || state.v === 'findings' || state.r === '15m' || state.r === '1h') render();
  }, 30000);
}
$('#logout').onclick = async () => { await fetch('/api/logout', {method: 'POST'}); me = null; showLogin(); };
$('#acct').onclick = () => go('account');

async function init() {
  $('#lang').innerHTML = LANGS.map(([c, n]) => `<option value="${c}">${n}</option>`).join('');
  const lang = pickLang();
  await loadLang(lang);
  $('#lang').value = lang;
  $('#ifsel').onchange = () => { state.ifv = $('#ifsel').value; render(); };
  $('#theme').onclick = () => { if (window.t66Theme) t66Theme.next(); themeLabel(); render(); };
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
  $('#nav').onclick = e => {
    const g = e.target.closest('[data-grp]');
    if (g) { const open = g.getAttribute('aria-expanded') !== 'true'; g.setAttribute('aria-expanded', open); try { localStorage.setItem('grp.' + g.dataset.grp, open ? '1' : '0'); } catch (_) {} return; }
    const b = e.target.closest('[data-v]');
    if (!b) return;
    if (b.dataset.v === 'records') state.rl = true;
    go(b.dataset.v);
  };
  document.querySelectorAll('#nav [data-grp]').forEach(g => {
    let open = '1';
    try { open = localStorage.getItem('grp.' + g.dataset.grp) ?? '1'; } catch (_) {}
    g.setAttribute('aria-expanded', open === '1');
  });
  $('#share').onclick = () => { navigator.clipboard?.writeText(location.href).then(() => toast(t('top.copied')), () => toast(location.href)); };
  document.addEventListener('click', e => { if (!e.target.closest('.cols')) { const m = $('#colMenu'); if (m) m.hidden = true; } });
  readHash();
  window.addEventListener('hashchange', () => { const before = location.hash; readHash(); if (before) render(); });
  // Back and Forward return to the previous page, e.g. out of a drill-down
  window.addEventListener('popstate', () => { state.f = []; state.det = null; readHash(); render(); });
  const res = await fetch('/api/status');
  if (res.status === 401) { showLogin(); return; }
  $('#app').hidden = false;
  await Promise.all([loadStatus(), loadSB(), loadMe()]);
  render();
  startTimers();
}
init();
})();
