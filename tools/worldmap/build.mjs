// Builds internal/web/static/world.json, the outline of every country for
// the world map on the Geo page, from Natural Earth (public domain) as
// packaged by world-atlas (ISC):
//
//   npm i d3-geo topojson-client topojson-simplify world-atlas i18n-iso-countries
//   node build.mjs > ../../internal/web/static/world.json
//
// Countries are keyed by ISO 3166 alpha-2 code, projected with the Natural
// Earth projection into a 960-wide box; Antarctica is left out.
import {createRequire} from 'module';
import {geoNaturalEarth1, geoPath} from 'd3-geo';
import {feature} from 'topojson-client';
import {presimplify, simplify, quantile} from 'topojson-simplify';
import countries from 'i18n-iso-countries';

const require = createRequire(import.meta.url);
let topo = require('world-atlas/countries-50m.json');
topo = presimplify(topo);
topo = simplify(topo, quantile(topo, 0.12));

const byName = {'Kosovo': 'XK', 'Somaliland': 'SO', 'N. Cyprus': 'CY'};
const fc = feature(topo, topo.objects.countries);
const W = 960;
const keep = fc.features.filter(f => {
  const id = f.id ? countries.numericToAlpha2(f.id) : byName[f.properties.name];
  f.cc = id;
  return id && id !== 'AQ';
});
const proj = geoNaturalEarth1().fitWidth(W, {type: 'FeatureCollection', features: keep});
const path = geoPath(proj).digits(1);
const [[, y0], [, y1]] = path.bounds({type: 'FeatureCollection', features: keep});
proj.translate([proj.translate()[0], proj.translate()[1] - y0 + 2]);
const out = {};
for (const f of keep) {
  const d = path(f);
  if (d) out[f.cc] = (out[f.cc] || '') + d;
}
// centres, for countries too small to click
const ctr = {};
for (const f of keep) {
  const [x, y] = path.centroid(f);
  const a = path.area(f);
  if (!ctr[f.cc] || a > ctr[f.cc][2]) ctr[f.cc] = [Math.round(x), Math.round(y), a];
}
const small = {};
for (const [cc, [x, y, a]] of Object.entries(ctr)) if (a < 12) small[cc] = [x, y];
process.stdout.write(JSON.stringify({w: W, h: Math.ceil(y1 - y0 + 4), src: 'Natural Earth (public domain) via world-atlas', c: out, s: small}));
