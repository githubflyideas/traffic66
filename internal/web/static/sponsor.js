// Remote sponsor banner for unregistered installations.
//
// The slides are listed in a manifest in
// https://github.com/githubflyideas/ads-sponsor (folder traffic66/), so they
// can change without a release. The browser fetches it directly; the
// traffic66 server never contacts GitHub, and no user or installation data is
// sent. Order of fallbacks:
//   1. the manifest and images straight from GitHub (this browser);
//   2. the last set the traffic66 server downloaded from GitHub (/api/sponsor),
//      for browsers that cannot reach GitHub;
//   3. a built-in text box. A broken image is never shown.
// Registered installations never call this.
//
// Usage: t66Sponsor.load()   t66Sponsor.remove()
(function () {
  'use strict';
  var URL_ = 'https://raw.githubusercontent.com/githubflyideas/ads-sponsor/main/traffic66/sponsors.json';
  var TIMEOUT_MS = 3000, ROTATE_MS = 5500, MAX = 4;
  var CACHE_KEY = 't66.sponsor', CACHE_MS = 30 * 60 * 1000;
  var started = false, timerId = null;

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
  function local() {
    return getJSON('/api/sponsor', {credentials: 'same-origin', cache: 'no-cache'});
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
      var s = list[i] || {}, img = httpURL(s.image, base, base === URL_);
      if (!img) continue;
      var alt = str(s.alt, 120), id = slug(str(s.id, 60)) || slug(alt) || ('slide-' + (i + 1));
      var link = httpURL(s.link, base, false);
      out.push({src: img.href, alt: alt, title: str(s.title, 80), link: link ? tag(link, id) : null});
    }
    return out;
  }

  function build(slot, slides) {
    var car = slot.querySelector('.spc');
    car.hidden = false;
    var track = document.createElement('div'); track.className = 'spc-track';
    var dots = document.createElement('div'); dots.className = 'spc-dots';
    var els = [], ds = [], cur = 0, paused = false;
    function show(i) {
      cur = i;
      els.forEach(function (e, j) { e.classList.toggle('on', j === i); });
      ds.forEach(function (e, j) { e.classList.toggle('on', j === i); });
    }
    slides.forEach(function (s, i) {
      var a = document.createElement(s.link ? 'a' : 'div');
      a.className = 'spc-slide';
      if (s.link) { a.href = s.link; a.target = '_blank'; a.rel = 'noopener noreferrer sponsored'; }
      if (s.title || s.alt) a.title = s.title || s.alt;
      s.img.alt = s.alt;
      a.appendChild(s.img);
      track.appendChild(a); els.push(a);
      if (slides.length > 1) {
        var d = document.createElement('button');
        d.type = 'button'; d.className = 'spc-dot';
        d.setAttribute('aria-label', String(i + 1));
        d.onclick = function () { show(i); };
        dots.appendChild(d); ds.push(d);
      }
    });
    car.textContent = '';
    car.appendChild(track);
    if (slides.length > 1) car.appendChild(dots);
    show(0);
    if (slides.length > 1) {
      ['mouseenter', 'focusin'].forEach(function (e) { car.addEventListener(e, function () { paused = true; }); });
      ['mouseleave', 'focusout'].forEach(function (e) { car.addEventListener(e, function () { paused = false; }); });
      timerId = setInterval(function () {
        if (!paused && !document.hidden) show((cur + 1) % slides.length);
      }, ROTATE_MS);
    }
    slot.hidden = false;
  }

  window.t66Sponsor = {
    load: function () {
      var slot = document.getElementById('sponsor');
      if (started || !slot || !window.URL || !window.Promise) return;
      started = true;
      var here = location.origin + '/api/sponsor';
      remote().then(function (m) { return ready(m, URL_); })
        .catch(function () { return local().then(function (m) { return ready(m, here); }); })
        .then(function (ok) { if (started) build(slot, ok); },
              function () { if (started) { slot.querySelector('.sp-text').hidden = false; slot.hidden = false; } });
    },
    remove: function () {
      started = false;
      if (timerId) clearInterval(timerId);
      var slot = document.getElementById('sponsor');
      if (slot) slot.remove();
    }
  };
})();
