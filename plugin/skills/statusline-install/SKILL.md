---
name: statusline-install
description: Install the claude-code-statusline binary and wire it into Claude Code's statusLine setting. Use when the user wants to set up, enable, or configure the claude-code-statusline plugin as their status line.
---

# Install claude-code-statusline

This skill installs the native `claude-code-statusline` binary that ships with
this plugin and points Claude Code's `statusLine` setting directly at it.

Claude Code plugins cannot set the top-level `statusLine` field from their own
bundled settings, so this one-time skill does it. After this runs, the status
line renders by executing the binary directly — no wrapper, no per-render
network calls. A `SessionStart` hook keeps the binary up to date automatically
when the plugin is later updated, so you should not need to run this again.

Do the following steps in order. Report a short summary at the end.

## 1. Download the binary

Run the plugin's sync script. It downloads the release binary matching this
plugin's version and this machine's OS/architecture into the plugin's
persistent data directory, verifying the checksum:

```bash
CLAUDE_PLUGIN_ROOT="${CLAUDE_PLUGIN_ROOT}" \
CLAUDE_PLUGIN_DATA="${CLAUDE_PLUGIN_DATA}" \
"${CLAUDE_PLUGIN_ROOT}/scripts/sync-binary.sh"
```

If it exits non-zero, the download failed (usually no network access or the
release asset is missing). Report the error from stderr and stop — do not edit
settings.json, because there is no binary to point at.

The binary path is:

```
${CLAUDE_PLUGIN_DATA}/bin/claude-code-statusline
```

Confirm it exists and is executable before continuing:

```bash
test -x "${CLAUDE_PLUGIN_DATA}/bin/claude-code-statusline" && echo OK
```

## 2. Wire it into settings.json

Point the user's Claude Code settings at that binary. Use their **user**
settings at `~/.claude/settings.json` unless they explicitly ask for project
settings (`.claude/settings.json` in the project).

Preserve every other setting already in the file — only add or replace the
`statusLine` key. If the file does not exist, create it as `{}` first.

Prefer `jq` when it is available:

```bash
SETTINGS="$HOME/.claude/settings.json"
BIN="${CLAUDE_PLUGIN_DATA}/bin/claude-code-statusline"
mkdir -p "$(dirname "$SETTINGS")"
[ -f "$SETTINGS" ] || echo '{}' > "$SETTINGS"
tmp="$(mktemp)"
jq --arg cmd "$BIN" '.statusLine = {type: "command", command: $cmd}' "$SETTINGS" > "$tmp" \
  && mv "$tmp" "$SETTINGS"
```

If `jq` is not installed, read the file, parse the JSON yourself, set the
`statusLine` key to `{"type": "command", "command": "<absolute binary path>"}`
using the resolved absolute path (not the `${CLAUDE_PLUGIN_DATA}` variable —
Claude Code does not expand plugin variables inside the user's settings.json),
and write it back with the rest of the file intact.

## 3. Confirm

Verify the resulting `statusLine` block and its `command` path, then tell the
user:

- which settings file you edited,
- the binary path it now points to,
- that the status line appears on their next render (they can start a new
  session or send a message to see it), and
- that updates are automatic — the plugin re-syncs the binary on session start
  after a version bump, so they will not need to run this again.

If they want to customize the segments, layout, or colors, point them to the
configuration section of the project README
(`https://github.com/timrcase/claude-code-statusline`); config lives at
`~/.config/claude-code-statusline/config.toml`.
