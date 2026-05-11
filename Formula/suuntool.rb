# Hand-written template for the homebrew tap. After cutting a release with
# goreleaser, the tap repo (tajchert/homebrew-tap) will host the
# auto-generated version of this. Keep both in sync until the release flow
# is automated.
class Suuntool < Formula
  desc "Unofficial CLI for the Suunto / Sports-Tracker API"
  homepage "https://github.com/tajchert/suuntool"
  url "https://github.com/tajchert/suuntool/archive/refs/tags/v0.1.0.tar.gz"
  sha256 "b51d8f91f43e1388f04625f64794c0638e45247d274aad969dbcea8aa551372c"
  license "MIT"
  head "https://github.com/tajchert/suuntool.git", branch: "main"

  depends_on "go" => :build

  def install
    ldflags = "-s -w -X github.com/tajchert/suuntool/cmd.Version=#{version}"
    system "go", "build", *std_go_args(ldflags: ldflags), "./"
  end

  test do
    assert_match "suuntool", shell_output("#{bin}/suuntool version")
  end
end
