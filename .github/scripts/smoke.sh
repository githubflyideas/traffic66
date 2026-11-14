#!/usr/bin/env bash
# Smoke test of a built program: it starts, answers /healthz, collects
# simulated exports, reports them in /metrics, and refuses a bad TLS
# certificate before opening anything.
set -uo pipefail
exe="$1"
base="${RUNNER_TEMP:-$(mktemp -d)}"; command -v cygpath >/dev/null && base="$(cygpath -u "$base")"
tmp="$base/smoke"
rm -rf "$tmp"; mkdir -p "$tmp"
fail() { echo "::error::smoke: $*"; [[ -f "$tmp/serve.log" ]] && tail -40 "$tmp/serve.log"; kill $(jobs -p) 2>/dev/null; exit 1; }
port=18066

"$exe" serve -data "$tmp/data" -password ci -addr 127.0.0.1:$port -listen "sflow=127.0.0.1:6343,netflow=127.0.0.1:2055,ipfix=127.0.0.1:4739" > "$tmp/serve.log" 2>&1 &
srv=$!
for i in $(seq 1 90); do curl -sf "http://127.0.0.1:$port/healthz" -o /dev/null && break; sleep 1; done
curl -sf "http://127.0.0.1:$port/healthz" -o /dev/null || fail "/healthz does not answer"
echo "healthz ok"

"$exe" simulate -to 127.0.0.1 > "$tmp/sim.log" 2>&1 &
sim=$!
n=0
for i in $(seq 1 60); do
  curl -sf -u admin:ci "http://127.0.0.1:$port/metrics" -o "$tmp/metrics.txt" || fail "/metrics refused"
  n=$(sed -n 's/^traffic66_records_total \([0-9.e+]*\)$/\1/p' "$tmp/metrics.txt" | head -1)
  [[ -n "$n" && "$n" != 0 ]] && break
  sleep 1
done
kill $sim 2>/dev/null
[[ -n "$n" && "$n" != 0 ]] || fail "no records collected"
echo "records collected: $n"
grep -c '^traffic66_udp_datagrams_total' "$tmp/metrics.txt"
code=$(curl -s -o /dev/null -w '%{http_code}' "http://127.0.0.1:$port/metrics")
[[ "$code" == 401 ]] || fail "/metrics without sign-in answered $code, want 401"
curl -sf -u admin:ci "http://127.0.0.1:$port/api/sources" -o "$tmp/sources.json" || fail "/api/sources"
head -c 300 "$tmp/sources.json"; echo
kill $srv; wait $srv 2>/dev/null

# a bad certificate stops the start, before any port is opened
"$exe" serve -data "$tmp/data2" -password ci -addr 127.0.0.1:18067 -listen "sflow=127.0.0.1:16343" -tls-cert "$tmp/none.pem" -tls-key "$tmp/none.key" > "$tmp/tls.log" 2>&1 &
bad=$!
for i in $(seq 1 30); do kill -0 $bad 2>/dev/null || break; sleep 1; done
if kill -0 $bad 2>/dev/null; then kill $bad; fail "a missing certificate did not stop the start"; fi
wait $bad; rc=$?
[[ $rc != 0 ]] || fail "a missing certificate exited with 0"
curl -s -o /dev/null "http://127.0.0.1:18067/healthz" && fail "port opened despite the bad certificate"
cat "$tmp/tls.log" | tail -3
echo "smoke ok"
