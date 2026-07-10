#!/usr/bin/env bash
# Ensure the claude-code-statusline binary matching this plugin's version is
# present in the persistent plugin data dir.
#
# Invoked by the /statusline-install skill (first install) and by the
# SessionStart hook (re-sync after a plugin update). It is NOT on the
# statusline render path — settings.json points straight at the binary this
# script drops, so rendering never runs a wrapper.
#
# Version is read from plugin.json (the single source of truth). The binary
# and a version marker live under ${CLAUDE_PLUGIN_DATA}, which survives plugin
# updates, so the path baked into settings.json stays valid forever.
#
# Contract: exits 0 whenever a usable binary exists at the end (freshly
# downloaded or a still-good cached copy). Exits non-zero only when no usable
# binary exists — so the SessionStart hook never breaks a working session just
# because an update download failed while offline.
set -euo pipefail

REPO="timrcase/claude-code-statusline"

ROOT="${CLAUDE_PLUGIN_ROOT:?CLAUDE_PLUGIN_ROOT is not set}"
DATA="${CLAUDE_PLUGIN_DATA:?CLAUDE_PLUGIN_DATA is not set}"

BIN_DIR="$DATA/bin"
BIN="$BIN_DIR/claude-code-statusline"
MARKER="$BIN_DIR/.version"

log() { printf 'sync-binary: %s\n' "$1" >&2; }

# --- Resolve target version from the plugin manifest -------------------------
manifest="$ROOT/.claude-plugin/plugin.json"
version="$(sed -n 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$manifest" | head -n1)"
[ -n "$version" ] || { log "could not read version from $manifest"; exit 1; }
tag="v${version}"

# --- Already current? --------------------------------------------------------
if [ -x "$BIN" ] && [ "$(cat "$MARKER" 2>/dev/null || true)" = "$tag" ]; then
    log "up to date ($tag)"
    exit 0
fi

# When we can't complete an update but a binary is already present, keep it.
keep_existing() {
    if [ -x "$BIN" ]; then
        log "$1 — keeping existing binary"
        exit 0
    fi
    log "$1 — no binary available"
    exit 1
}

# --- Map uname to the release asset (GOOS/GOARCH) ----------------------------
os="$(uname -s)"
arch="$(uname -m)"
case "$os" in
    Darwin) goos=darwin ;;
    Linux)  goos=linux ;;
    *) keep_existing "unsupported OS: $os" ;;
esac
case "$arch" in
    arm64|aarch64) goarch=arm64 ;;
    x86_64|amd64)  goarch=amd64 ;;
    *) keep_existing "unsupported architecture: $arch" ;;
esac

asset="claude-code-statusline-${goos}-${goarch}.tar.gz"
base="https://github.com/${REPO}/releases/download/${tag}"

# --- Pick a downloader -------------------------------------------------------
if command -v curl >/dev/null 2>&1; then
    fetch() { curl -fsSL "$1" -o "$2"; }
elif command -v wget >/dev/null 2>&1; then
    fetch() { wget -qO "$2" "$1"; }
else
    keep_existing "need curl or wget to download the binary"
fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

log "downloading $asset ($tag)"
fetch "$base/$asset" "$tmp/asset.tar.gz" || keep_existing "download failed"

# --- Verify checksum against the release's checksums.txt ---------------------
if fetch "$base/checksums.txt" "$tmp/checksums.txt" 2>/dev/null; then
    expected="$(awk -v f="$asset" '$2 == f {print $1}' "$tmp/checksums.txt" | head -n1)"
    if [ -n "$expected" ]; then
        if command -v sha256sum >/dev/null 2>&1; then
            actual="$(sha256sum "$tmp/asset.tar.gz" | awk '{print $1}')"
        elif command -v shasum >/dev/null 2>&1; then
            actual="$(shasum -a 256 "$tmp/asset.tar.gz" | awk '{print $1}')"
        else
            actual=""
        fi
        if [ -n "$actual" ] && [ "$actual" != "$expected" ]; then
            log "checksum mismatch for $asset (expected $expected, got $actual)"
            keep_existing "refusing to install unverified binary"
        fi
    fi
fi

# --- Extract and atomically install ------------------------------------------
tar -xzf "$tmp/asset.tar.gz" -C "$tmp" || keep_existing "failed to extract archive"
[ -f "$tmp/claude-code-statusline" ] || keep_existing "archive did not contain the binary"

mkdir -p "$BIN_DIR"
# Write to a temp name, then rename over the target so a concurrent render
# never sees a half-written or non-executable file.
mv "$tmp/claude-code-statusline" "$BIN.new"
chmod +x "$BIN.new"
mv "$BIN.new" "$BIN"
printf '%s' "$tag" > "$MARKER"

log "installed $tag -> $BIN"
