#!/usr/bin/env bash
# Build a self-contained traffic66 binary for the current platform.
# Usage: scripts/build.sh [version] [output]
set -euo pipefail
version="${1:-0.1.0-dev}"
out="${2:-traffic66}"
export CGO_ENABLED=1
ldflags="-s -w -X main.version=${version}"
tags=""
case "$(uname -s)" in
  Linux)
    # link libstdc++ statically so only libc is needed at run time
    tmp="$(mktemp -d)"
    ln -sf "$(g++ -print-file-name=libstdc++.a)" "$tmp/libstdc++.a"
    export CGO_LDFLAGS="-L$tmp ${CGO_LDFLAGS:-}"
    # fully static: no dependency on the system's glibc, so it runs on any
    # distribution with kernel 3.2 or later (CentOS 7 included); getentropy
    # gets a fallback for kernels without getrandom (entropy_linux.go)
    ldflags="$ldflags -linkmode=external -extldflags '-static -Wl,--wrap=getentropy'"
    tags="netgo,osusergo"
    ;;
  MINGW*|MSYS*|CYGWIN*)
    # link the MinGW runtime statically: no libstdc++/libgcc/libwinpthread
    # DLLs are needed next to the program
    ldflags="$ldflags -extldflags=-static"
    ;;
  Darwin)
    # run on macOS 11 and later, not only on the version it was built on
    export MACOSX_DEPLOYMENT_TARGET="${MACOSX_DEPLOYMENT_TARGET:-11.0}"
    ;;
esac
go build -trimpath ${tags:+-tags "$tags"} -ldflags "$ldflags" -o "$out" ./cmd/traffic66
echo "built $out"
