# Generated from the SHA256SUMS of metais v1.0.0-rc.1 by
# go run ./internal/tools/release formula. Do not edit it by hand.
class Metais < Formula
  desc "Reference tool for METAIS packages, which turn input data into files"
  homepage "https://metais.org"
  license "Apache-2.0"

  on_macos do
    on_intel do
      url "https://github.com/kottera/metais/releases/download/v1.0.0-rc.1/metais_1.0.0-rc.1_darwin_amd64.tar.gz"
      sha256 "1b92b25279a3fccbce3fac3e037966a59b4373d26e0e4e2395f9e9e0a46a0796"
    end

    on_arm do
      url "https://github.com/kottera/metais/releases/download/v1.0.0-rc.1/metais_1.0.0-rc.1_darwin_arm64.tar.gz"
      sha256 "375589132f737724133bc0b762ad65905c048c8f07888f116e5a20dc381cbb38"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/kottera/metais/releases/download/v1.0.0-rc.1/metais_1.0.0-rc.1_linux_amd64.tar.gz"
      sha256 "074ac956854cc8516527e371e8476ada607ce450105e9aaf63ebdf3a4d04f0e1"
    end

    on_arm do
      url "https://github.com/kottera/metais/releases/download/v1.0.0-rc.1/metais_1.0.0-rc.1_linux_arm64.tar.gz"
      sha256 "eeedc2a6bac416c5dc9b3915bd154a17bd1b4995411a248668f575fab674f5e3"
    end
  end

  def install
    bin.install "metais"
  end

  def caveats
    <<~EOS
      To teach a coding agent to use metais, install its agent skill
      into a project by running, at the project's top level:
        metais skill
    EOS
  end

  test do
    assert_match "metais #{version}", shell_output("#{bin}/metais version")
  end
end
