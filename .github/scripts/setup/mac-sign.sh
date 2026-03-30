#!/usr/bin/env bash

set -euo pipefail

# Import an Apple Developer certificate into a temporary keychain and sign
# binaries using gon.
#
# Required environment variables:
#   MACOS_CERTIFICATE             Base64-encoded .p12 developer certificate
#   MACOS_CERTIFICATE_PASSWORD    Password for the .p12 file
#
# Usage: mac-sign.sh [--macos-skip-root-certificate] <gon-config.hcl> ...

readonly APPLE_ROOT_CERTIFICATE="http://certs.apple.com/devidg2.der"

function main {
  local mac_skip_root_certificate=""
  local assets=()

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --macos-skip-root-certificate)
        mac_skip_root_certificate=true
        shift
        ;;
      --help)
        echo "Usage: $0 [--macos-skip-root-certificate] <gon-config.hcl> ..."
        exit 0
        ;;
      -*)
        echo "ERROR: Unrecognized argument: $1" >&2
        exit 1
        ;;
      *)
        assets=("$@")
        break
        ;;
    esac
  done

  if [[ ${#assets[@]} -eq 0 ]]; then
    echo "ERROR: No gon config files specified" >&2
    exit 1
  fi

  ensure_macos
  import_certificate_mac "${mac_skip_root_certificate}"
  sign_mac "${assets[@]}"
}

function ensure_macos {
  if [[ $OSTYPE != 'darwin'* ]]; then
    echo "ERROR: Signing of Mac binaries is supported only on macOS" >&2
    exit 1
  fi
}

function sign_mac {
  local -r assets=("$@")
  for filepath in "${assets[@]}"; do
    echo "Signing ${filepath}"
    gon -log-level=info "${filepath}"
  done
}

function import_certificate_mac {
  local -r mac_skip_root_certificate="$1"
  assert_env_var_not_empty "MACOS_CERTIFICATE"
  assert_env_var_not_empty "MACOS_CERTIFICATE_PASSWORD"

  trap "rm -rf /tmp/*-keychain" EXIT

  local mac_certificate_pwd="${MACOS_CERTIFICATE_PASSWORD}"
  local keystore_pw="${RANDOM}"

  local db_file
  db_file=$(mktemp "/tmp/XXXXXX-keychain")
  rm -rf "${db_file}"

  echo "Creating temporary keychain for certificate"
  security create-keychain -p "${keystore_pw}" "${db_file}"
  security default-keychain -s "${db_file}"
  security unlock-keychain -p "${keystore_pw}" "${db_file}"

  echo "${MACOS_CERTIFICATE}" | base64 -d | security import /dev/stdin \
    -f pkcs12 -k "${db_file}" -P "${mac_certificate_pwd}" -T /usr/bin/codesign

  if [[ "${mac_skip_root_certificate}" == "" ]]; then
    curl -v "${APPLE_ROOT_CERTIFICATE}" --output certificate.der
    sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain certificate.der
  fi

  security set-key-partition-list -S apple-tool:,apple:,codesign: \
    -s -k "${keystore_pw}" "${db_file}"
}

function assert_env_var_not_empty {
  local -r var_name="$1"
  local -r var_value="${!var_name}"

  if [[ -z "$var_value" ]]; then
    echo "ERROR: Required environment variable $var_name not set." >&2
    exit 1
  fi
}

main "$@"
