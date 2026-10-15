#!/usr/bin/env bash
# Build a self-contained traffic66 binary for the current platform.
# Usage: scripts/build.sh [version] [output]
set -euo pipefail
# No version given: the series under development, from the newest heading in
# CHANGELOG.md, which is also what cmd/traffic66 falls back to. Hardcoding it
# here is how the default came to say 0.1.0 long after the program was at 2.x.
default_version="$(sed -n 's/^## \([0-9][0-9.]*\)[[:space:]]*$/\1/p' \
  "$(dirname "$0")/../CHANGELOG.md" | head -1)-dev"
version="${1:-$default_version}"
out="${2:-traffic66}"
export CGO_ENABLED=1
# The release workflow sets TRAFFIC66_BUILD_KIND=official; a plain build of
# this script says community, which is what it is.
ldflags="-s -w -X main.version=${version} -X main.buildKind=${TRAFFIC66_BUILD_KIND:-community}"
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
