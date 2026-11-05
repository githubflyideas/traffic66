// Applies the colour theme before the page is drawn, so it does not flash.
// The choice is kept per browser; clicking the theme button goes to the next.
(function () {
  var THEMES = ['light', 'black', 'teal', 'orange', 'midnight'];
  // themes with a dark background get the logo with light lettering
  var DARK = {black: 1, midnight: 1};
  var OLD = {bright: 'light', dim: 'black', dark: 'black', auto: 'light', gray: 'light'};
  function get() {
    var v = 'light';
    try { v = localStorage.getItem('t66.theme') || 'light'; } catch (e) {}
    if (OLD[v]) { v = OLD[v]; try { localStorage.setItem('t66.theme', v); } catch (e) {} }
    return THEMES.indexOf(v) < 0 ? 'light' : v;
  }
  function apply(x) {
    if (x === 'light') document.documentElement.removeAttribute('data-theme');
    else document.documentElement.setAttribute('data-theme', x);
    // the built-in logo has light lettering on black
    var q = '/logo?dark=' + (DARK[x] ? 1 : 0);
    var set = function () { document.querySelectorAll('img.logo').forEach(function (i) { if (i.getAttribute('src').split('&')[0] !== q) i.src = q; }); };
    if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', set); else set();
  }
  apply(get());
  window.t66Theme = {
    list: THEMES, get: get,
    isDark: function (v) { return !!DARK[v === undefined ? get() : v]; },
    set: function (v) { try { localStorage.setItem('t66.theme', v); } catch (e) {} apply(v); },
    next: function () { var v = THEMES[(THEMES.indexOf(get()) + 1) % THEMES.length]; this.set(v); return v; }
  };
})();
