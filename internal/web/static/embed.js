// The page other platforms open for one address: its bandwidth in and out
// over a period, and nothing else. It stands alone: no session, no menu,
// the key from its own link, redrawn every minute.
(function () {
  'use strict';
  const LANGS = ['en', 'zh', 'ja', 'ko', 'de', 'fr', 'es', 'pt', 'ru', 'ar', 'hi', 'bn', 'ur', 'id', 'vi'];
  const RANGES = ['1h', '24h', '7d', '30d'];
  const qs = new URLSearchParams(location.search);
  const root = document.getElementById('em'), ip = root.dataset.ip, key = qs.get('k') || '';
  // dark unless the platform asks for light, to sit in a light page
  document.documentElement.dataset.theme = qs.get('theme') === 'light' ? 'light' : 'dark';
  let range = RANGES.includes(qs.get('range')) ? qs.get('range') : '24h';
  let L = {}, lang = 'en', data = null, timer = 0;
  const t = (k, p) => { let s = L[k] || k; for (const [a, b] of Object.entries(p || {})) s = s.split('{' + a + '}').join(b); return s; };
  const esc = s => String(s).replace(/[&<>"']/g, c => ({'&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'}[c]));
  const nf = (v, d = 0) => new Intl.NumberFormat(lang, {maximumFractionDigits: d, minimumFractionDigits: d}).format(v);
  function bps(v) {
    if (v == null || !isFinite(v)) return '—';
    for (const [n, m] of [['Tbps', 1e12], ['Gbps', 1e9], ['Mbps', 1e6], ['Kbps', 1e3]]) if (v >= m) return nf(v / m, v / m < 10 ? 2 : v / m < 100 ? 1 : 0) + ' ' + n;
    return nf(v) + ' bps';
  }
  function axis(v) {
    if (v === 0) return '0';
    for (const [n, m] of [['T', 1e12], ['G', 1e9], ['M', 1e6], ['K', 1e3]]) if (v >= m) return nf(v / m, v / m < 10 && v % m ? 1 : 0) + n;
    return nf(v);
  }
  function pick() {
    const want = (qs.get('lang') || '').toLowerCase();
    if (LANGS.includes(want)) return want;
    for (const l of navigator.languages || [navigator.language || 'en']) {
      const c = l.toLowerCase().split('-')[0];
      if (LANGS.includes(c)) return c;
    }
    return 'en';
  }

  async function load() {
    clearTimeout(timer);
    const p = new URLSearchParams({ip, k: key, range, tz: String(-new Date().getTimezoneOffset())});
    try {
      const r = await fetch('/embed/vip/data?' + p, {cache: 'no-store'});
      const j = await r.json().catch(() => ({}));
      if (!r.ok) throw new Error(j.error || r.statusText);
      data = j;
      draw();
    } catch (e) {
      data = null;
      root.innerHTML = head() + `<div class="emerr">${esc(e.message)}</div>`;
      bind();
    }
    timer = setTimeout(load, 60000);
  }

  function head() {
    return `<header><div><h1>${esc(ip)}</h1><span>${esc(t('em.sub'))}</span></div>
      <nav>${RANGES.map(r => `<button data-r="${r}" aria-pressed="${r === range}">${esc(t('range.' + r))}</button>`).join('')}</nav></header>`;
  }
  function bind() {
    root.querySelectorAll('[data-r]').forEach(b => b.onclick = () => { range = b.dataset.r; load(); });
  }

  // the minutes still arriving are left off the end rather than drawn as
  // a fall to nothing
  function settled(d, vals) {
    return vals.map((v, i) => (d.settled_to && d.times[i] >= d.settled_to) || (d.data_from && d.times[i] + d.step * 1000 <= d.data_from) ? null : v);
  }
  function stats(v) {
    const xs = v.filter(x => x != null);
    if (!xs.length) return {now: null, peak: null, avg: null};
    return {now: xs[xs.length - 1], peak: Math.max(...xs), avg: xs.reduce((a, b) => a + b, 0) / xs.length};
  }

  function draw() {
    const d = data, vin = settled(d, d.in || []), vout = settled(d, d.out || []);
    const si = stats(vin), so = stats(vout);
    const card = (cls, name, s) => `<div class="emk ${cls}"><b>${esc(name)}</b>
      <dl><div><dt>${esc(t('em.now'))}</dt><dd>${bps(s.now)}</dd></div><div><dt>${esc(t('em.peak'))}</dt><dd>${bps(s.peak)}</dd></div><div><dt>${esc(t('em.avg'))}</dt><dd>${bps(s.avg)}</dd></div></dl></div>`;
    root.innerHTML = head() + `<div class="emks">${card('in', t('if.in'), si)}${card('out', t('if.out'), so)}</div>
      <div class="emch" id="ch"></div><footer>traffic66</footer>`;
    bind();
    chart(document.getElementById('ch'), d.times || [], vin, vout);
  }

  function chart(el, times, a, b) {
    const W = el.clientWidth, H = el.clientHeight, n = times.length;
    const has = v => v.some(x => x != null && x > 0);
    if (n < 2 || (!has(a) && !has(b))) { el.innerHTML = `<div class="emerr">${esc(t('em.nodata'))}</div>`; return; }
    const rtl = document.documentElement.dir === 'rtl';
    const Lp = rtl ? 10 : 52, Rp = rtl ? 52 : 10, T = 10, B = 24;
    const top = Math.max(1, ...a.filter(x => x != null), ...b.filter(x => x != null));
    const mag = Math.pow(10, Math.floor(Math.log10(top))), nice = [1, 2, 2.5, 5, 10].map(m => m * mag).find(m => m * 4 >= top) || top;
    const ymax = nice * 4;
    const x = i => rtl ? W - Rp - (W - Lp - Rp) * i / (n - 1) : Lp + (W - Lp - Rp) * i / (n - 1);
    const y = v => T + (H - T - B) * (1 - v / ymax);
    const path = v => { let s = '', on = false; v.forEach((p, i) => { if (p == null) { on = false; return; } s += (on ? 'L' : 'M') + x(i).toFixed(1) + ' ' + y(p).toFixed(1); on = true; }); return s; };
    const area = v => { let s = '', run = []; const flush = () => { if (run.length) s += 'M' + x(run[0]).toFixed(1) + ' ' + y(0) + run.map(i => 'L' + x(i).toFixed(1) + ' ' + y(v[i]).toFixed(1)).join('') + 'L' + x(run[run.length - 1]).toFixed(1) + ' ' + y(0) + 'Z'; run = []; }; v.forEach((p, i) => p == null ? flush() : run.push(i)); flush(); return s; };
    const span = times[n - 1] - times[0];
    const fmt = new Intl.DateTimeFormat(lang, span > 2 * 864e5 ? {month: 'numeric', day: 'numeric'} : {hour: '2-digit', minute: '2-digit'});
    const tip = new Intl.DateTimeFormat(lang, {month: 'numeric', day: 'numeric', hour: '2-digit', minute: '2-digit'});
    let grid = '';
    for (let k = 0; k <= 4; k++) {
      const v = nice * k, yy = y(v).toFixed(1);
      grid += `<line x1="${Lp}" x2="${W - Rp}" y1="${yy}" y2="${yy}" class="g"/><text x="${rtl ? W - Rp + 6 : Lp - 6}" y="${yy}" class="ya" text-anchor="${rtl ? 'start' : 'end'}" dominant-baseline="middle">${axis(v)}</text>`;
    }
    const ticks = Math.max(2, Math.min(7, Math.floor((W - Lp - Rp) / 90)));
    for (let k = 0; k < ticks; k++) {
      const i = Math.round(k * (n - 1) / (ticks - 1));
      // the first and last labels keep inside the chart
      const anchor = k === 0 ? (rtl ? 'end' : 'start') : k === ticks - 1 ? (rtl ? 'start' : 'end') : 'middle';
      grid += `<text x="${x(i).toFixed(1)}" y="${H - 6}" class="xa" text-anchor="${anchor}">${esc(fmt.format(new Date(times[i])))}</text>`;
    }
    el.innerHTML = `<svg width="${W}" height="${H}" role="img" aria-label="${esc(ip)}">${grid}
      <path d="${area(a)}" class="fi"/><path d="${area(b)}" class="fo"/><path d="${path(a)}" class="li"/><path d="${path(b)}" class="lo"/>
      <line class="cur" y1="${T}" y2="${H - B}" visibility="hidden"/></svg><div class="tip" hidden></div>`;
    const svg = el.querySelector('svg'), cur = el.querySelector('.cur'), tp = el.querySelector('.tip');
    svg.addEventListener('pointermove', e => {
      const r = svg.getBoundingClientRect(), px = e.clientX - r.left;
      let i = Math.round((rtl ? W - Rp - px : px - Lp) / (W - Lp - Rp) * (n - 1));
      i = Math.max(0, Math.min(n - 1, i));
      cur.setAttribute('x1', x(i)); cur.setAttribute('x2', x(i)); cur.setAttribute('visibility', 'visible');
      tp.hidden = false;
      tp.innerHTML = `<b>${esc(tip.format(new Date(times[i])))}</b><span class="in">${esc(t('if.in'))} ${bps(a[i])}</span><span class="out">${esc(t('if.out'))} ${bps(b[i])}</span>`;
      tp.style.left = Math.min(W - tp.offsetWidth - 4, Math.max(4, x(i) + 10)) + 'px';
    });
    svg.addEventListener('pointerleave', () => { cur.setAttribute('visibility', 'hidden'); tp.hidden = true; });
  }

  async function start() {
    lang = pick();
    document.documentElement.lang = lang;
    if (['ar', 'ur'].includes(lang)) document.documentElement.dir = 'rtl';
    try { L = await (await fetch('/i18n/' + lang + '.json')).json(); } catch (e) { L = {}; }
    await load();
    let w = innerWidth;
    addEventListener('resize', () => { if (data && innerWidth !== w) { w = innerWidth; draw(); } });
  }
  start();
})();
