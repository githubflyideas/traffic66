#!/bin/sh
# traffic66 one-line install for Linux and macOS.
#
#   curl -fsSL https://raw.githubusercontent.com/githubflyideas/traffic66/main/scripts/install.sh | sh
#
# Downloads the latest release for this system into ./traffic66 (or
# $TRAFFIC66_DIR) and starts the collector, which prints the address and
# UDP ports to point your devices at. Arguments replace "serve", for example
# "sh -s -- serve -listen netflow=:9995" or "sh -s -- version".
# TRAFFIC66_VERSION=v2.3.3 picks a release; TRAFFIC66_NO_START=1 only
# downloads. Nothing is installed system-wide.
set -eu

repo=githubflyideas/traffic66
dir=${TRAFFIC66_DIR:-traffic66}

case $(uname -s) in
Linux) os=linux ;;
Darwin) os=darwin ;;
*) echo "traffic66: unsupported system $(uname -s); see https://github.com/$repo/releases" >&2; exit 1 ;;
esac
case $(uname -m) in
x86_64 | amd64) arch=amd64 ;;
aarch64 | arm64) arch=arm64 ;;
*) echo "traffic66: unsupported CPU $(uname -m); see https://github.com/$repo/releases" >&2; exit 1 ;;
esac

if [ -n "${TRAFFIC66_VERSION:-}" ]; then
	url=https://github.com/$repo/releases/download/$TRAFFIC66_VERSION/traffic66-$os-$arch.tar.gz
else
	url=https://github.com/$repo/releases/latest/download/traffic66-$os-$arch.tar.gz
fi

if command -v curl >/dev/null 2>&1; then
	get() { curl -fsSL "$1"; }
elif command -v wget >/dev/null 2>&1; then
	get() { wget -qO- "$1"; }
else
	echo "traffic66: needs curl or wget" >&2
	exit 1
fi

echo "traffic66: downloading $url" >&2
mkdir -p "$dir"
get "$url" | tar xzf - -C "$dir" --strip-components=1
[ "$os" = darwin ] && xattr -dr com.apple.quarantine "$dir" 2>/dev/null || true
chmod +x "$dir/traffic66"
"$dir/traffic66" version >&2

if [ -n "${TRAFFIC66_NO_START:-}" ]; then
	echo "traffic66: installed in $dir; start it with: cd $dir && ./traffic66" >&2
	exit 0
fi
[ $# -gt 0 ] || set -- serve
cd "$dir"
exec ./traffic66 "$@"
