# claude-code-statusline tasks. Run `just` (or `just --list`) to see them.
# Requires: go, git, jq. Install just: https://github.com/casey/just
#
# This is the single source of truth for build/release tasks: the GitHub
# release workflow installs just and calls `just build-all` / `just
# render-formula`, and developers use the same recipes locally.

bin      := "claude-code-statusline"
manifest := "plugin/.claude-plugin/plugin.json"

# List available recipes.
default:
    @just --list

# Build a local dev binary, stamping the version from `git describe`.
build:
    go build -trimpath \
        -ldflags="-X main.version=$(git describe --tags --always --dirty)" \
        -o {{bin}} .

# Run the test suite.
test:
    go test ./...

# Run go vet.
vet:
    go vet ./...

# Everything CI gates on: tests + vet.
check: test vet

# Build, then render a sample payload through the binary.
# Usage: just run                       (uses testdata/payload.json)
#        just run testdata/minimal.json
run payload="testdata/payload.json": build
    ./{{bin}} < {{payload}}

# Show the current plugin-manifest version and git description.
version:
    @jq -r '"plugin.json: " + .version' {{manifest}}
    @printf 'git:         %s\n' "$(git describe --tags --always --dirty)"

# Cross-compile every release tarball into dist/ (pure Go, no extra toolchains).
# VERSION overrides the stamped version — the release workflow passes the git
# tag; otherwise it derives from `git describe`, falling back to "dev".
build-all:
    #!/usr/bin/env bash
    set -euo pipefail
    bin="{{bin}}"
    version="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
    mkdir -p dist
    for t in darwin/arm64 darwin/amd64 linux/amd64 linux/arm64; do
        goos="${t%/*}"; goarch="${t#*/}"
        out="dist/${bin}-${goos}-${goarch}"
        mkdir -p "$out"
        GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 \
            go build -trimpath -ldflags="-s -w -X main.version=${version}" -o "${out}/${bin}" .
        tar -czf "${out}.tar.gz" -C "$out" "$bin"
        rm -r "$out"
        echo "${out}.tar.gz"
    done
    echo
    ls -la dist/

# Render the Homebrew formula to stdout, for the timrcase/homebrew-tap tap.
# Used by the release workflow's homebrew job.
# Usage: just render-formula <version> <checksums.txt>
#   <checksums.txt> is the `sha256sum *.tar.gz` output from a release.
render-formula version checksums:
    #!/usr/bin/env bash
    set -euo pipefail
    version="{{version}}"; version="${version#v}"
    checksums="{{checksums}}"
    bin="{{bin}}"

    # Pull the sha256 for a given tarball name out of the checksums file.
    sha_for() {
        local name=$1 sha
        sha=$(awk -v n="$name" '$2 == n { print $1 }' "$checksums")
        if [ -z "$sha" ]; then
            echo "error: no checksum for $name in $checksums" >&2
            exit 1
        fi
        printf '%s' "$sha"
    }

    darwin_arm=$(sha_for "${bin}-darwin-arm64.tar.gz")
    darwin_amd=$(sha_for "${bin}-darwin-amd64.tar.gz")
    linux_arm=$(sha_for "${bin}-linux-arm64.tar.gz")
    linux_amd=$(sha_for "${bin}-linux-amd64.tar.gz")

    base="https://github.com/timrcase/claude-code-statusline/releases/download/v#{version}"

    cat <<EOF
    class ClaudeCodeStatusline < Formula
      desc "Fast, config-driven statusline for Claude Code"
      homepage "https://github.com/timrcase/claude-code-statusline"
      version "${version}"
      license "MIT"

      on_macos do
        on_arm do
          url "${base}/${bin}-darwin-arm64.tar.gz"
          sha256 "${darwin_arm}"
        end
        on_intel do
          url "${base}/${bin}-darwin-amd64.tar.gz"
          sha256 "${darwin_amd}"
        end
      end

      on_linux do
        on_arm do
          url "${base}/${bin}-linux-arm64.tar.gz"
          sha256 "${linux_arm}"
        end
        on_intel do
          url "${base}/${bin}-linux-amd64.tar.gz"
          sha256 "${linux_amd}"
        end
      end

      def install
        bin.install "${bin}"
      end

      def caveats
        <<~EOS
          To enable the statusline in Claude Code, add this to ~/.claude/settings.json:

              "statusLine": {
                "type": "command",
                "command": "#{HOMEBREW_PREFIX}/bin/${bin}"
              }

          Then restart Claude Code (or start a new session).
        EOS
      end

      test do
        # Empty stdin prints "Claude" (see main.go).
        assert_equal "Claude", shell_output("#{bin}/${bin} < /dev/null").strip
      end
    end
    EOF

# Cut a release: bump plugin.json, commit if it changed, tag, push together.
# Pushing the bump WITH the tag keeps origin/master in sync, so the release
# workflow's plugin-version job finds the manifest already correct and no-ops
# instead of pushing a commit behind your back. Usage: just release 1.5.0
release version:
    #!/usr/bin/env bash
    set -euo pipefail
    ver="{{version}}"; ver="${ver#v}"
    tag="v${ver}"
    [[ "$ver" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] \
        || { echo "error: version must be X.Y.Z, got '$ver'" >&2; exit 1; }
    [ "$(git rev-parse --abbrev-ref HEAD)" = "master" ] \
        || { echo "error: releases are cut from master" >&2; exit 1; }
    [ -z "$(git status --porcelain)" ] \
        || { echo "error: working tree not clean; commit or stash first" >&2; exit 1; }
    git pull --ff-only origin master
    just check
    git rev-parse -q --verify "refs/tags/${tag}" >/dev/null \
        && { echo "error: tag ${tag} already exists" >&2; exit 1; }
    tmp="$(mktemp)"
    jq --arg v "$ver" '.version = $v' "{{manifest}}" > "$tmp" && mv "$tmp" "{{manifest}}"
    if ! git diff --quiet -- "{{manifest}}"; then
        git add "{{manifest}}"
        git commit -m "chore: bump plugin version to ${ver}"
    fi
    git tag "$tag"
    git push --atomic origin master "$tag"
    echo "released ${tag}"
