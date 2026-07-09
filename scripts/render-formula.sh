#!/usr/bin/env bash
# Render the Homebrew formula for the timrcase/homebrew-tap tap.
#
# Usage: scripts/render-formula.sh <version> <checksums.txt>
#   <version>       release version without the leading 'v', e.g. 1.0.0
#   <checksums.txt> the `sha256sum *.tar.gz` output from a release
#
# Prints the rendered Formula/claude-code-statusline.rb to stdout.
set -euo pipefail

version=${1:?usage: render-formula.sh <version> <checksums.txt>}
checksums=${2:?usage: render-formula.sh <version> <checksums.txt>}
version=${version#v}

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

bin=claude-code-statusline
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

  test do
    # Empty stdin prints "Claude" (see main.go).
    assert_equal "Claude", shell_output("#{bin}/${bin} < /dev/null").strip
  end
end
EOF
