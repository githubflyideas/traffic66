#!/usr/bin/env bash
# Sign-in through real reverse proxies (nginx, Caddy, Traefik in Docker):
# the configurations from the manual must work, and a cross-site request
# must still be refused.
set -uo pipefail
exe="$1"
tmp="${RUNNER_TEMP:-$(mktemp -d)}/proxy"
rm -rf "$tmp"; mkdir -p "$tmp"
errors=0
fail() { echo "::error::proxy: $*"; errors=$((errors+1)); }

"$exe" serve -data "$tmp/a" -password ci -addr 127.0.0.1:18066 -listen "" > "$tmp/a.log" 2>&1 &
"$exe" serve -data "$tmp/b" -password ci -addr 127.0.0.1:18067 -listen "" -public-url http://localhost:18085 > "$tmp/b.log" 2>&1 &

cat > "$tmp/nginx.conf" <<'C'
server {
    listen 18081;
    location / {
        proxy_pass http://127.0.0.1:18066;
        proxy_set_header Host $http_host;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
# nginx's own default: the internal address as Host, nothing forwarded
server {
    listen 18084;
    location / { proxy_pass http://127.0.0.1:18066; }
}
server {
    listen 18085;
    location / { proxy_pass http://127.0.0.1:18067; }
}
C
cat > "$tmp/traefik.yml" <<'C'
http:
  routers:
    traffic66:
      rule: PathPrefix(`/`)
      entryPoints: [web]
      service: traffic66
  services:
    traffic66:
      loadBalancer:
        servers:
          - url: http://127.0.0.1:18066
C
docker run -d --name t66-nginx --network host -v "$tmp/nginx.conf:/etc/nginx/conf.d/default.conf:ro" nginx:stable-alpine >/dev/null
docker run -d --name t66-caddy --network host caddy:2 caddy reverse-proxy --from :18082 --to 127.0.0.1:18066 >/dev/null
docker run -d --name t66-traefik --network host -v "$tmp/traefik.yml:/dyn.yml:ro" traefik:v3 --entrypoints.web.address=:18083 --providers.file.filename=/dyn.yml >/dev/null

for p in 18066 18067 18081 18082 18083 18084 18085; do
  for i in $(seq 1 60); do curl -s -o /dev/null "http://localhost:$p/healthz" && break; sleep 1; done
done

# login PORT ORIGIN: prints the HTTP status and the error code, if any
login() {
  curl -s -o "$tmp/out.json" -w '%{http_code}' -H "Origin: $2" -H 'Content-Type: application/json' \
    -d '{"user":"admin","password":"ci"}' "http://localhost:$1/api/login"
  sed -n 's/.*"error":"\([a-z_]*\)".*/ \1/p' "$tmp/out.json"
}
want() { # name got want
  if [[ "$2" == "$3" ]]; then echo "ok   $1: $2"; else fail "$1: got '$2', want '$3'"; fi
}
want "nginx (Host \$http_host)"            "$(login 18081 http://localhost:18081)" 200
want "nginx, cross-site"                   "$(login 18081 https://evil.example)" "403 cross_origin"
want "Caddy"                               "$(login 18082 http://localhost:18082)" 200
want "Caddy, cross-site"                   "$(login 18082 https://evil.example)" "403 cross_origin"
want "Traefik"                             "$(login 18083 http://localhost:18083)" 200
want "Traefik, cross-site"                 "$(login 18083 https://evil.example)" "403 cross_origin"
want "nginx default, no -public-url"       "$(login 18084 http://localhost:18084)" "403 cross_origin"
want "nginx default, -public-url"          "$(login 18085 http://localhost:18085)" 200
want "-public-url, cross-site"             "$(login 18085 https://evil.example)" "403 cross_origin"
want "direct"                              "$(login 18066 http://localhost:18066)" 200

# a signed-in change through the proxy
curl -s -c "$tmp/jar" -o /dev/null -H 'Origin: http://localhost:18082' -H 'Content-Type: application/json' \
  -d '{"user":"admin","password":"ci"}' http://localhost:18082/api/login
want "Caddy, signed in" "$(curl -s -b "$tmp/jar" -o /dev/null -w '%{http_code}' http://localhost:18082/api/me)" 200

docker rm -f t66-nginx t66-caddy t66-traefik >/dev/null 2>&1
kill $(jobs -p) 2>/dev/null
[[ $errors == 0 ]] || { tail -20 "$tmp/a.log"; exit 1; }
echo "proxy ok"
