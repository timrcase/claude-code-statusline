#!/usr/bin/env bash
# Build release binaries for all supported targets and tarball them in dist/.
# Go cross-compiles everything natively — no extra toolchains needed.
set -euo pipefail
cd "$(dirname "$0")/.."

BIN=claude-code-statusline
TARGETS=(darwin/arm64 darwin/amd64 linux/amd64 linux/arm64)

mkdir -p dist

for t in "${TARGETS[@]}"; do
    GOOS=${t%/*}
    GOARCH=${t#*/}
    out="dist/${BIN}-${GOOS}-${GOARCH}"
    mkdir -p "$out"
    GOOS="$GOOS" GOARCH="$GOARCH" CGO_ENABLED=0 \
        go build -trimpath -ldflags="-s -w" -o "${out}/${BIN}" .
    tar -czf "${out}.tar.gz" -C "$out" "$BIN"
    rm -r "$out"
    echo "${out}.tar.gz"
done

echo
ls -la dist/
