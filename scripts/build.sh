#!/usr/bin/env bash
# Build a self-contained traffic66 binary for the current platform.
# Usage: scripts/build.sh [version] [output]
set -euo pipefail
version="${1:-0.1.0-dev}"
out="${2:-traffic66}"
export CGO_ENABLED=1
ldflags="-s -w -X main.version=${version}"
case "$(uname -s)" in
  Linux)
    # link libstdc++ statically so only libc is needed at run time
    tmp="$(mktemp -d)"
    ln -sf "$(g++ -print-file-name=libstdc++.a)" "$tmp/libstdc++.a"
    export CGO_LDFLAGS="-L$tmp ${CGO_LDFLAGS:-}"
    ldflags="$ldflags -extldflags=-static-libgcc"
    ;;
esac
go build -trimpath -ldflags "$ldflags" -o "$out" ./cmd/traffic66
echo "built $out"
