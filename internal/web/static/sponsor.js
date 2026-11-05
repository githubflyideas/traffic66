// Remote sponsor banners for unregistered installations.
//
// The slides are listed in a manifest in
// https://github.com/githubflyideas/ads-sponsor (folder traffic66/), so they
// can change without a release. Only the browser fetches it: the traffic66
// server never contacts GitHub for it, keeps no copy and serves none, and no
// user or installation data is sent. A browser that cannot reach GitHub
// shows a built-in text box instead; a broken image is never shown.
// Registered installations never call this.
//
// Classification is by each image's natural size (no manifest field):
//   width/height >= WIDE_RATIO  → slim banner at the top of the main area
//   otherwise                   → square-ish ad in the sidebar
//
// Usage: t66Sponsor.load()   t66Sponsor.remove()
(function () {
  'use strict';
  var URL_ = 'https://raw.githubusercontent.com/githubflyideas/ads-sponsor/main/traffic66/sponsors.json';
  var TIMEOUT_MS = 3000, ROTATE_MS = 5500, MAX = 6;
  // Images this many times as wide as they are high (or wider) are wide
  // banners for the top strip; the rest are square-ish for the sidebar.
  var WIDE_RATIO = 3;
  // Sidebar: width fills the slot, height follows the image, capped.
  var SIDE_MAX_H = 220;
  // Top strip: classic 728x90 proportions, capped.
  var TOP_MAX_W = 728, TOP_MAX_H = 90;
  var CACHE_KEY = 't66.sponsor', CACHE_MS = 30 * 60 * 1000;
  var started = false, timers = [], ros = [];

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
      img.onload = function () { img.naturalWidth > 0 && img.naturalHeight > 0 ? ok(img) : bad(); };
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

  // Load images, drop broken ones, split into square (sidebar) and wide (top).
  function ready(m, base) {
    var want = slidesOf(m, base);
    if (!want.length) return Promise.reject();
    return settle(want.map(function (s) { return loadImg(s.src); })).then(function (imgs) {
      var square = [], wide = [];
      want.forEach(function (s, i) {
        var im = imgs[i];
        if (!im) return;
        s.img = im;
        if (im.naturalWidth >= WIDE_RATIO * im.naturalHeight) wide.push(s);
        else square.push(s);
      });
      if (!square.length && !wide.length) throw 0;
      return {square: square, wide: wide};
    });
  }

  function slideEl(s) {
    var a = document.createElement(s.link ? 'a' : 'div');
    if (s.link) { a.href = s.link; a.target = '_blank'; a.rel = 'noopener noreferrer sponsored'; }
    if (s.title || s.alt) a.title = s.title || s.alt;
    var img = s.img.cloneNode(false);
    img.alt = s.alt;
    a.appendChild(img);
    return a;
  }

  // One rotating carousel inside `car`. heightFn(carWidth, slide) → px height.
  function carousel(car, slides, heightFn) {
    var strip = document.createElement('div'); strip.className = 'spc-strip';
    var dots = document.createElement('div'); dots.className = 'spc-dots';
    var ds = [], cur = 0, paused = false;

    function draw() {
      var s = slides[cur];
      var w = car.clientWidth || TOP_MAX_W;
      strip.textContent = '';
      var box = document.createElement('div'); box.className = 'spc-cell';
      box.style.height = Math.max(1, Math.round(heightFn(w, s))) + 'px';
      box.appendChild(slideEl(s));
      strip.appendChild(box);
      ds.forEach(function (d, i) { d.classList.toggle('on', i === cur); });
    }
    function show(i) { cur = ((i % slides.length) + slides.length) % slides.length; draw(); }

    car.hidden = false;
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
      timers.push(setInterval(function () {
        if (!paused && !document.hidden) show(cur + 1);
      }, ROTATE_MS));
    }
    show(0);
    if (window.ResizeObserver) {
      var ro = new ResizeObserver(draw);
      ro.observe(car);
      ros.push(ro);
    } else {
      window.addEventListener('resize', draw);
    }
  }

  function sideHeight(w, s) {
    var ratio = s.img.naturalHeight / s.img.naturalWidth;
    return Math.min(SIDE_MAX_H, w * ratio);
  }
  function topHeight(w, s) {
    var ratio = s.img.naturalHeight / s.img.naturalWidth;
    return Math.min(TOP_MAX_H, w * ratio);
  }

  function showDefault(slot) {
    if (!slot) return;
    var t = slot.querySelector('.sp-text');
    if (t) t.hidden = false;
    var c = slot.querySelector('.spc');
    if (c) { c.hidden = true; c.textContent = ''; }
    slot.hidden = false;
  }

  function buildSidebar(slot, slides) {
    if (!slot) return;
    if (!slides.length) { showDefault(slot); return; }
    var t = slot.querySelector('.sp-text');
    if (t) t.hidden = true;
    carousel(slot.querySelector('.spc'), slides, sideHeight);
    slot.hidden = false;
  }

  function buildTop(slot, slides) {
    if (!slot) return;
    if (!slides.length) { slot.hidden = true; slot.textContent = ''; return; }
    slot.textContent = '';
    var wrap = document.createElement('div'); wrap.className = 'spc';
    slot.appendChild(wrap);
    carousel(wrap, slides, topHeight);
    slot.hidden = false;
  }

  function clearTimers() {
    timers.forEach(function (id) { clearInterval(id); });
    timers = [];
    ros.forEach(function (ro) { try { ro.disconnect(); } catch (e) {} });
    ros = [];
  }

  window.t66Sponsor = {
    load: function () {
      var side = document.getElementById('sponsor');
      var top = document.getElementById('sponsorTop');
      if (started || !side || !window.URL || !window.Promise) return;
      started = true;
      remote().then(function (m) { return ready(m, URL_); })
        .then(function (parts) {
          if (!started) return;
          buildSidebar(side, parts.square);
          buildTop(top, parts.wide);
        }, function () {
          if (!started) return;
          showDefault(side);
          if (top) top.hidden = true;
        });
    },
    remove: function () {
      started = false;
      clearTimers();
      var side = document.getElementById('sponsor');
      if (side) side.remove();
      var top = document.getElementById('sponsorTop');
      if (top) top.remove();
    }
  };
})();
