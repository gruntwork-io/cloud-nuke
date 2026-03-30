#!/usr/bin/env bash

set -euo pipefail

# Sign macOS binaries using gon and Apple notarization, then extract and verify.
#
# Required environment variables:
#   AC_PASSWORD                   Apple Connect app-specific password
#   AC_PROVIDER                   Apple Connect Team ID
#   AC_USERNAME                   Apple Connect username (Apple ID email)
#   MACOS_CERTIFICATE             Base64-encoded .p12 developer certificate
#   MACOS_CERTIFICATE_PASSWORD    Password for the .p12 file
#
# Usage: sign-macos-binaries.sh [bin-dir]

function main {
  local -r bin_dir="${1:-bin}"

  : "${AC_PASSWORD:?ERROR: AC_PASSWORD is required}"
  : "${AC_PROVIDER:?ERROR: AC_PROVIDER is required}"
  : "${AC_USERNAME:?ERROR: AC_USERNAME is required}"
  : "${MACOS_CERTIFICATE:?ERROR: MACOS_CERTIFICATE is required}"
  : "${MACOS_CERTIFICATE_PASSWORD:?ERROR: MACOS_CERTIFICATE_PASSWORD is required}"

  if [[ ! -d "$bin_dir" ]]; then
    echo "ERROR: Directory $bin_dir does not exist" >&2
    exit 1
  fi

  local script_dir
  script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

  echo "Signing macOS binaries..."

  # Sign amd64
  echo "Signing amd64 binary..."
  "$script_dir/../setup/mac-sign.sh" .gon_amd64.hcl

  # Sign arm64 (skip root cert import — already done above)
  echo "Signing arm64 binary..."
  "$script_dir/../setup/mac-sign.sh" --macos-skip-root-certificate .gon_arm64.hcl

  echo "Done signing the binaries"

  # Extract signed binaries from zip files and replace unsigned ones
  local -r darwin_binaries=("cloud-nuke_darwin_amd64" "cloud-nuke_darwin_arm64")

  echo ""
  echo "Extracting and verifying signed binaries..."

  for binary in "${darwin_binaries[@]}"; do
    local zip_file="${binary}.zip"

    echo "Processing $binary..."

    if [[ ! -f "$zip_file" ]]; then
      echo "ERROR: ZIP file $zip_file not found" >&2
      exit 1
    fi

    # Remove the unsigned binary
    rm -f "$bin_dir/$binary"

    # Extract signed binary
    unzip -o "$zip_file"

    if [[ ! -f "$binary" ]]; then
      echo "ERROR: Failed to extract $binary from $zip_file" >&2
      exit 1
    fi

    # Verify signature
    echo "  Verifying signature..."
    codesign -dv --verbose=4 "$binary" 2>&1 || {
      echo "ERROR: Signature verification failed for $binary" >&2
      exit 1
    }

    mv "$binary" "$bin_dir/"
    echo "  Moved signed binary to $bin_dir/"
    echo ""
  done

  echo "All macOS binaries signed and verified successfully"
  echo ""
  echo "Contents of $bin_dir:"
  ls -lh "$bin_dir/"
}

main "$@"
