// Remote sponsor banner for unregistered installations.
//
// The slides are listed in a manifest in
// https://github.com/githubflyideas/ads-sponsor (folder traffic66/), so they
// can change without a release. Only the browser fetches it: the traffic66
// server never contacts GitHub for it, keeps no copy and serves none, and no
// user or installation data is sent. A browser that cannot reach GitHub
// shows a built-in text box instead; a broken image is never shown.
// Registered installations never call this.
//
// Usage: t66Sponsor.load()   t66Sponsor.remove()
(function () {
  'use strict';
  var URL_ = 'https://raw.githubusercontent.com/githubflyideas/ads-sponsor/main/traffic66/sponsors.json';
  // MAX is how many advertisers are fetched and rotated through.
  var TIMEOUT_MS = 3000, ROTATE_MS = 5500, MAX = 6;
  // A cell narrower than this is not worth drawing a 728x90 banner into, so
  // it decides how many fit across the window.
  var MIN_CELL = 420;
  // A small banner is 728x90 (1456x180 at 2x) and shown SMALL_COPIES times
  // side by side; an image LONG_RATIO times as wide as it is high or wider
  // is a long banner and fills the strip alone.
  var SMALL_W = 728, SMALL_H = 90, SMALL_COPIES = 3, LONG_RATIO = 12;
  var CACHE_KEY = 't66.sponsor', CACHE_MS = 30 * 60 * 1000;
  var started = false, timerId = null, roId = null;

  function httpURL(v, base, httpsOnly) {
    if (typeof v !== 'string' || !v.trim()) return null;
    try {
      var u = new URL(v.trim(), base);
      if (u.protocol === 'https:' || (!httpsOnly && u.protocol === 'http:')) return u;
    } catch (e) {}
    return null;
  }
  function str(v, n) { return typeof v === 'string' ? v.trim().slice(0, n) : ''; }
  function slug(v) {
    return String(v || '').toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '').slice(0, 40);
  }
  // keeps the sponsor's own parameters (ref, affiliate ids, utm_*) and only
  // adds the utm_* ones that are not there yet
  function tag(u, campaign) {
    var p = u.searchParams, add = {utm_source: 'traffic66', utm_medium: 'sponsor', utm_campaign: campaign};
    Object.keys(add).forEach(function (k) { if (add[k] && !p.has(k)) p.set(k, add[k]); });
    return u.href;
  }

  function loadImg(src) {
    return new Promise(function (ok, bad) {
      var img = new Image();
      img.decoding = 'async';
      img.referrerPolicy = 'no-referrer';
      img.onload = function () { img.naturalWidth > 0 ? ok(img) : bad(); };
      img.onerror = bad;
      img.src = src;
    });
  }
  function settle(ps) {
    return Promise.all(ps.map(function (p) { return p.then(function (v) { return v; }, function () { return null; }); }));
  }

  function cached() {
    try {
      var c = JSON.parse(sessionStorage.getItem(CACHE_KEY) || 'null');
      if (c && c.at && Date.now() - c.at < CACHE_MS && c.m) return c.m;
    } catch (e) {}
    return null;
  }
  function getJSON(url, opt) {
    if (!window.fetch) return Promise.reject();
    var ctl = window.AbortController ? new AbortController() : null;
    var t = setTimeout(function () { if (ctl) ctl.abort(); }, TIMEOUT_MS);
    opt.signal = ctl ? ctl.signal : undefined;
    return fetch(url, opt)
      .then(function (r) { if (!r.ok) throw 0; return r.json(); })
      .then(function (m) { clearTimeout(t); return m; }, function (e) { clearTimeout(t); throw e; });
  }
  function remote() {
    var c = cached();
    if (c) return Promise.resolve(c);
    return getJSON(URL_, {credentials: 'omit', referrerPolicy: 'no-referrer', cache: 'no-cache', mode: 'cors'})
      .then(function (m) {
        try { sessionStorage.setItem(CACHE_KEY, JSON.stringify({at: Date.now(), m: m})); } catch (e) {}
        return m;
      });
  }
  // resolves with the slides whose images loaded, rejects when none did
  function ready(m, base) {
    var want = slidesOf(m, base);
    if (!want.length) return Promise.reject();
    return settle(want.map(function (s) { return loadImg(s.src); })).then(function (imgs) {
      var ok = [];
      want.forEach(function (s, i) { if (imgs[i]) { s.img = imgs[i]; ok.push(s); } });
      if (!ok.length) throw 0;
      return ok;
    });
  }

  function slidesOf(m, base) {
    var list = m && Array.isArray(m.slides) ? m.slides : [], out = [];
    for (var i = 0; i < list.length && out.length < MAX; i++) {
      var s = list[i] || {}, img = httpURL(s.image, base, true);
      if (!img) continue;
      var alt = str(s.alt, 120), id = slug(str(s.id, 60)) || slug(alt) || ('slide-' + (i + 1));
      var link = httpURL(s.link, base, false);
      out.push({src: img.href, alt: alt, title: str(s.title, 80), link: link ? tag(link, id) : null});
    }
    return out;
  }

  // The strip shows one advertiser at a time and rotates between them.
  //   - a long banner (LONG_RATIO wide to 1 high or wider, 2880x120 is the
  //     intended size) has the strip to itself;
  //   - a small banner (the 728x90 kind) is shown three times side by side,
  //     the same ad in every cell, so the strip never mixes two advertisers.
  // A narrow window has room for fewer cells (one per MIN_CELL pixels).
  // The strip keeps the height of a row of small banners whichever ad is
  // up, so rotating never moves the page underneath.
  //
  // One image element cannot be in two cells at once, so each cell gets a
  // clone of an already-loaded image; nothing is fetched twice.
  function build(slot, slides) {
    var car = slot.querySelector('.spc');
    car.hidden = false;
    var strip = document.createElement('div'); strip.className = 'spc-strip';
    var dots = document.createElement('div'); dots.className = 'spc-dots';
    var ds = [], cur = 0, paused = false;

    function slideEl(s) {
      var a = document.createElement(s.link ? 'a' : 'div');
      if (s.link) { a.href = s.link; a.target = '_blank'; a.rel = 'noopener noreferrer sponsored'; }
      if (s.title || s.alt) a.title = s.title || s.alt;
      var img = s.img.cloneNode(false);
      img.alt = s.alt;
      a.appendChild(img);
      return a;
    }
    function isLong(s) { return s.img.naturalWidth >= LONG_RATIO * s.img.naturalHeight; }
    function draw() {
      var s = slides[cur];
      var w = car.clientWidth || window.innerWidth || MIN_CELL;
      var k = Math.max(1, Math.min(SMALL_COPIES, Math.floor(w / MIN_CELL)));
      var rowH = Math.round((w - (k - 1)) / k * SMALL_H / SMALL_W);
      var n = isLong(s) ? 1 : k;
      strip.textContent = '';
      strip.style.gridTemplateColumns = 'repeat(' + n + ', minmax(0, 1fr))';
      for (var i = 0; i < n; i++) {
        var box = document.createElement('div'); box.className = 'spc-cell';
        box.style.height = rowH + 'px';
        box.appendChild(slideEl(s));
        strip.appendChild(box);
      }
      ds.forEach(function (d, i) { d.classList.toggle('on', i === cur); });
      measure();
    }
    function show(i) { cur = ((i % slides.length) + slides.length) % slides.length; draw(); }
    // The sidebar's height is the window less this strip (see app.css).
    function measure() {
      try { document.documentElement.style.setProperty('--sp-h', slot.offsetHeight + 'px'); } catch (e) {}
    }

    car.textContent = '';
    car.appendChild(strip);
    if (slides.length > 1) {
      slides.forEach(function (s, i) {
        var d = document.createElement('button');
        d.type = 'button'; d.className = 'spc-dot';
        d.setAttribute('aria-label', String(i + 1));
        d.onclick = function () { show(i); };
        dots.appendChild(d); ds.push(d);
      });
      car.appendChild(dots);
      ['mouseenter', 'focusin'].forEach(function (e) { car.addEventListener(e, function () { paused = true; }); });
      ['mouseleave', 'focusout'].forEach(function (e) { car.addEventListener(e, function () { paused = false; }); });
      timerId = setInterval(function () {
        if (!paused && !document.hidden) show(cur + 1);
      }, ROTATE_MS);
    }
    slot.hidden = false;
    show(0);
    if (window.ResizeObserver) { roId = new ResizeObserver(draw); roId.observe(car); }
    else window.addEventListener('resize', draw);
  }

  window.t66Sponsor = {
    load: function () {
      var slot = document.getElementById('sponsor');
      if (started || !slot || !window.URL || !window.Promise) return;
      started = true;
      remote().then(function (m) { return ready(m, URL_); })
        .then(function (ok) { if (started) build(slot, ok); },
              function () {
                if (!started) return;
                slot.querySelector('.sp-text').hidden = false;
                slot.hidden = false;
                try { document.documentElement.style.setProperty('--sp-h', slot.offsetHeight + 'px'); } catch (e) {}
              });
    },
    remove: function () {
      started = false;
      if (timerId) clearInterval(timerId);
      if (roId) { roId.disconnect(); roId = null; }
      try { document.documentElement.style.setProperty('--sp-h', '0px'); } catch (e) {}
      var slot = document.getElementById('sponsor');
      if (slot) slot.remove();
    }
  };
})();
