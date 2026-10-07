// Applies the colour theme before the page is drawn, so it does not flash.
// The choice is kept per browser; "auto" follows the system's light or dark.
(function () {
  var dark = window.matchMedia && matchMedia('(prefers-color-scheme: dark)');
  function get() { try { return localStorage.getItem('t66.theme') || 'light'; } catch (e) { return 'light'; } }
  function apply(v) {
    var x = v === 'auto' ? (dark && dark.matches ? 'dark' : 'light') : v;
    if (x === 'light') document.documentElement.removeAttribute('data-theme');
    else document.documentElement.setAttribute('data-theme', x);
    // the built-in logo has light lettering for the dark themes
    var q = '/logo?dark=' + (x === 'dark' || x === 'dim' ? 1 : 0);
    var set = function () { document.querySelectorAll('img.logo').forEach(function (i) { if (i.getAttribute('src').split('&')[0] !== q) i.src = q; }); };
    if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', set); else set();
  }
  apply(get());
  window.t66Theme = {get: get, set: function (v) { try { localStorage.setItem('t66.theme', v); } catch (e) {} apply(v); }};
  if (dark && dark.addEventListener) dark.addEventListener('change', function () { apply(get()); });
})();
