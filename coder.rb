class Coder < Formula
  desc "A command-line tool for the Coder remote development platform"
  homepage "https://github.com/cdr/coder-cli"
  url "https://github.com/cdr/coder-cli/releases/download/v1.14.2/coder-cli-darwin-amd64-v1.14.2.zip"
  sha256 "69b69497a75ce19851681974aa7561af7c7061357f8616e24446e58795bf6e1f"
  bottle :unneeded
  def install
    bin.install "coder"
  end
  test do
    system "#{bin}/coder", "--version"
  end
end
