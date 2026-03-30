#!/usr/bin/env bash

set -euo pipefail

# Download and install the gon binary for macOS code signing and notarization.
# Usage: install-gon.sh [gon-version]

function main {
  local gon_version="${1:-${GON_VERSION:-v0.0.37}}"

  echo "Installing gon version $gon_version..."

  local download_url="https://github.com/Bearer/gon/releases/download/${gon_version}/gon_macos.zip"

  echo "Downloading gon from: $download_url"
  curl -L -o gon.zip "$download_url"

  echo "Extracting gon binary..."
  unzip -o gon.zip -d . gon

  chmod +x ./gon
  sudo mv ./gon /usr/local/bin/gon
  sudo chmod +x /usr/local/bin/gon

  echo "Verifying gon installation..."
  gon --version

  rm -f gon.zip
  echo "gon installed successfully"
}

main "$@"
