# Homebrew formula for tuiagger — a terminal UI for viewing and
# interacting with OpenAPI/Swagger documentation.
#
# This file lives in the app's own repo as a ready template; the actual
# tap most users install from is a separate `homebrew-tuiagger` repo (per
# Homebrew convention — a tap repo is named `homebrew-<name>` and its
# `Formula/<name>.rb` is what `brew tap`/`brew install` actually reads).
# Copy this file there verbatim after cutting a release; only the
# `url`/`sha256` need updating per release after that.
#
# Cutting a release:
#   1. git tag vX.Y.Z && git push --tags
#   2. curl -sL https://github.com/valVK/tuiagger/archive/refs/tags/vX.Y.Z.tar.gz | shasum -a 256
#   3. Update url/sha256 below (and in the homebrew-tuiagger tap repo)
#
# See the "Distribution strategy" project memory for the full flow.

class Tuiagger < Formula
  desc "Terminal UI for viewing and interacting with OpenAPI/Swagger documentation"
  homepage "https://github.com/valVK/tuiagger"
  url "https://github.com/valVK/tuiagger/archive/refs/tags/v0.1.0.tar.gz"
  sha256 "" # filled in when v0.1.0 is actually tagged — see comment above
  license "MIT"
  head "https://github.com/valVK/tuiagger.git", branch: "master"

  depends_on "go" => :build

  def install
    # main.version is a `const`, not a `var` — Go's `-X` linker flag only
    # rewrites package-level string variables, so there's nothing to inject
    # here. Just strip debug info (-s -w), standard for a released binary.
    system "go", "build", *std_go_args(ldflags: "-s -w"), "./cmd/tuiagger"
  end

  test do
    assert_match "tuiagger", shell_output("#{bin}/tuiagger --version")
  end
end
