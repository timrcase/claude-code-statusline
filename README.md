# claude-code-statusline

A fast, config-driven statusline for [Claude Code](https://claude.com/claude-code),
written in Go. Single static binary, ~1–3 ms per render, zero runtime
dependencies.

```
Opus 4.8 | myrepo@main [wt-review] (+12 -4) | ██▊░░ 112k/200k (56%) | effort: high | 5h ██░░░ 39% @02:10 | 7d ▏░░░░ 3% @Thu Jul 9, 00:00
```

It reads the JSON payload Claude Code pipes to the statusline command on
stdin, renders truecolor ANSI to stdout, and exits.

**Pure stdin → stdout, by design.** No network calls, no OAuth/credential
reading. The 5h/7d rate-limit display comes straight from the `rate_limits`
data Claude Code provides on stdin. The only disk write is a local copy of the
last payload for [`--print-payload`](#discovering-fields); the only subprocesses
are `git` (branch, diff stats, worktree name) and any [custom
segment](#custom-segments) you opt into.

## Install

### Homebrew

```sh
brew install timrcase/tap/claude-code-statusline
```

### From source

```sh
go build -trimpath -ldflags="-s -w" -o claude-code-statusline .
cp claude-code-statusline ~/.local/bin/
```

or straight from the module:

```sh
go install github.com/timrcase/claude-code-statusline@latest
```

### Wire up Claude Code

In `~/.claude/settings.json`:

```json
"statusLine": { "type": "command", "command": "~/.local/bin/claude-code-statusline" }
```

## Configuration

Optional. With no config file you get the default line shown above.

Location: `$XDG_CONFIG_HOME/claude-code-statusline/config.toml`, falling back
to `~/.config/claude-code-statusline/config.toml`. A parse error falls back to
the defaults with a warning on stderr; a malformed section or an unknown
layout entry is skipped, not fatal.

The schema is starship-style: each segment is a named section that only
controls how that segment *looks*; the `[layout]` stanza controls which
segments appear, in what order, on which line, and with what separator. **A
segment renders only if it appears in a layout line.**

The full default configuration — also available as
[`config.example.toml`](config.example.toml), ready to copy into place:

```toml
[layout]
separator = " | "        # rendered dim between segments
line1 = ["model", "directory", "context", "effort", "limit_5h", "limit_7d"]
# line2 = ["custom.example"]   # add lineN keys for a multi-line statusline

[model]
color = "0099ff"         # colors are hex, no leading '#'

[directory]
git = true               # branch after dir as @branch
diff = true              # (+adds -dels) from git diff --numstat
worktree = true          # [name] when in a linked git worktree
color          = "2e9599"
branch_color   = "00a000"
worktree_color = "ffb055"
adds_color     = "00a000"
dels_color     = "ff5555"

[context]
bar = "blocks"           # blocks | dots | none
width = 5                # cells per bar
counts = true            # "112k/200k"
percent = true           # "(56%)"
counts_color   = "ffb055"
ok_color       = "00a000"   # usage < 50% (bar fill + percent)
warn_color     = "e6c800"   # >= 50%
hot_color      = "ffb055"   # >= 70%
critical_color = "ff5555"   # >= 90%

[effort]
label_color  = "dcdcdc"  # the "effort:" prefix
medium_color = "ffb055"
high_color   = "00a000"
xhigh_color  = "a78bfa"
max_color    = "ff5555"  # ("low" always renders dim)

[limit_5h]
bar = "blocks"
width = 5
reset = true             # "@13:00" (local time)
label_color    = "dcdcdc"
ok_color       = "00a000"
warn_color     = "e6c800"
hot_color      = "ffb055"
critical_color = "ff5555"

[limit_7d]               # same options as limit_5h
bar = "blocks"
width = 5
reset = true             # "@Mon Jul 6, 13:00"
```

A segment with nothing to show (no cwd, no git repo, missing rate-limit data)
is skipped or rendered as a dim `5h -` placeholder — never a fabricated `0%`.
A layout line whose segments all come up empty is dropped entirely.

### Segments

| name        | renders                                              | options |
|-------------|------------------------------------------------------|---------|
| `model`     | model display name (`"Opus 4.6 (1M context)"` → `Opus 4.6 1M`) | `color` |
| `directory` | cwd basename, `@branch`, `[worktree]`, `(+adds -dels)` | `git`, `diff`, `worktree` (all default `true`); colors |
| `context`   | usage bar, token counts, percent of context window   | `bar`, `width`, `counts`, `percent`; colors |
| `effort`    | effort level from stdin (`low`/`med`/`high`/`xhigh`/`max`) | per-level colors |
| `limit_5h`  | 5-hour rate limit: bar, used %, reset time           | `bar`, `width`, `reset`; colors |
| `limit_7d`  | 7-day rate limit: bar, used %, reset time            | `bar`, `width`, `reset`; colors |
| `custom.*`  | first line of your command's stdout, verbatim        | `command` (required), `timeout_ms` (default `300`) |

Set `diff = false` in huge repos where `git diff --numstat` is slow.

### Bar styles

Every bar-bearing section takes its own `bar` and `width`:

- `blocks` (default): `██▊░░` — partial-block glyphs give a 5-cell bar 40
  gradations
- `dots`: `●●●○○`
- `none`: no graphic bar, numbers only — `bar = "none"` on the limits
  reproduces the classic `5h 39% @02:10` look

All glyphs are plain Unicode; no Nerd Font or emoji required. Bars are colored
by usage: ok < 50% ≤ warn < 70% ≤ hot < 90% ≤ critical, each threshold color
overridable per section.

### Custom segments

The escape hatch for anything that would otherwise bloat the core (network
calls, credentials, version checks…). Declare a named `[custom.<name>]`
section and reference it in the layout as `"custom.<name>"` — as many as you
like. Your script's first stdout line is spliced in as a segment, ANSI colors
and all. It is killed and the segment skipped if it exceeds `timeout_ms`.

```toml
[layout]
line1 = ["model", "directory", "context"]
line2 = ["custom.usage"]

[custom.usage]
command = "my-usage-fetcher"   # run via `sh -c`
timeout_ms = 300
```

### Field segments

The built-in segments cover the common cases, but Claude Code's stdin payload
carries much more — `cost.*`, `pr.*`, `vim.mode`, `agent.name`, `workspace.repo.*`,
and [more](https://code.claude.com/docs/en/statusline#available-data), with new
fields added over time. A **field segment** surfaces *any* of them by JSON path,
no per-field code required. Declare `[field.<name>]` and reference it in the
layout as `"field.<name>"`.

```toml
[layout]
line1 = ["model", "directory", "field.cost", "field.rl5h"]

[field.cost]
path   = "cost.total_cost_usd"   # dotted path into the stdin JSON (required)
symbol = "$"                     # any string: $, a Nerd Font glyph, or an emoji
format = "usd"                   # optional; see below
color  = "00a000"                # optional; hex. An explicit color always wins.

[field.rl5h]
path    = "rate_limits.five_hour.used_percentage"
format  = "pie"                  # compact one-glyph gauge
percent = true                   # also append "39%"
```

`format` options:

| format | example | notes |
| --- | --- | --- |
| *(unset)* | `NORMAL`, `156` | raw value; whole numbers as ints |
| `usd` | `0.06` | 2 decimals (`symbol` supplies the `$`) |
| `tokens` | `15k` | same scaling as the context counts |
| `duration` | `1m30s` | from a millisecond count |
| `percent` | `24%` | rounds to a whole number |
| `epoch` | `Jul 6 13:00` | Unix seconds → local time |
| `bar` / `dots` / `pie` | `██▎░░` · `●●○○○` · `◑` | gauges for a 0–100 number |

Notes:

- A missing path, `null`, or a **false boolean** renders nothing (no stray
  symbol). Booleans (`thinking.enabled`, `exceeds_200k_tokens`) show `symbol`
  only when true — a flag.
- `bar`/`dots`/`pie` color by the usage gradient (like the context bar) unless
  you set `color`; `width` sets bar/dots cells (default 5); `pie` needs no font
  beyond standard Unicode geometric shapes.

Don't know the exact path for a field? See [Discovering fields](#discovering-fields).

## Discovering fields

Run:

```sh
claude-code-statusline --print-payload
```

Every render caches the raw stdin JSON to
`${XDG_STATE_HOME:-~/.local/state}/claude-code-statusline/last-payload.json`, and
`--print-payload` pretty-prints the last one (with its capture time), so you can
copy exact paths into `[field.*]` sections. Because the payload only arrives on a
Claude Code refresh, trigger one first — or pipe a saved copy:

```sh
claude-code-statusline --print-payload < payload.json
```

stdout stays valid JSON (the capture note goes to stderr), so you can pipe it to
`jq`. This is also the quickest way to see why a segment is blank on your setup —
e.g. Claude Enterprise omits `rate_limits`.

## Checking your config

Config parse warnings (`unknown segment`, `no [field.x] section`, `bad config …
using defaults`) are written to stderr during a render — which Claude Code
discards, so a mistyped segment or a config that silently fell back to defaults
is invisible. To see them:

```sh
claude-code-statusline --check-config
```

It reports the resolved config path (and whether it loaded or fell back to
defaults), every parse warning, the **effective** layout after normalization, and
the declared field/custom sections — then exits non-zero if there were warnings,
so it works as a lint. For example, a layout referencing a non-existent `cost`
segment shows:

```
config: ~/.config/claude-code-statusline/config.toml (loaded)

warnings (1):
  - unknown segment "cost" in layout of …/config.toml; skipping

layout (separator " | "):
  line1: model, directory, context, effort

field sections:  rl5h
```

## Behavior on odd input

- Empty stdin → prints `Claude`, exits 0
- Unparseable JSON, missing fields, unknown fields → still renders; missing
  data shows as blank/`-` segments, never a crash
- `resets_at` accepts a Unix epoch or an ISO 8601 string
- Context window size defaults to 200k when absent

## Building releases

```sh
just build-all
```

Cross-compiles `darwin/arm64`, `darwin/amd64`, `linux/amd64`, and
`linux/arm64` (pure Go, no extra toolchains) and drops tarballs in `dist/`.

## Development

Tasks live in the [`justfile`](justfile) (`just --list` to see them); they need
[`just`](https://github.com/casey/just), plus `go`, `git`, and `jq`.

```sh
just check                                        # go test ./... + go vet ./...
just run                                          # build, then render testdata/payload.json
just run testdata/minimal.json                    # render a specific payload
just release 1.5.0                                # bump plugin.json, tag, push
```

The GitHub release workflow uses the same recipes (`just build-all`,
`just render-formula`), so there is one source of truth for build/release logic.

Test fixtures in `testdata/`: `payload.json` (real captured payload),
`legacy.json` (no rate_limits/effort), `minimal.json` (model only). To capture a fresh payload,
temporarily wrap your statusline command:

```json
"statusLine": { "type": "command",
  "command": "tee /tmp/statusline-payload.json | ~/.local/bin/claude-code-statusline" }
```
