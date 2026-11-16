#!/usr/bin/env bash

set -euo pipefail

function usage() {
    echo "Usage: bash install.sh [ -d directory ] [ -v version ] [ --no-checksums ]"
    exit 2
}

INSTALL_DIR="${HOME}/bin"
VERSION=""
VERIFY_CHECKSUMS="true"
NAME=totpcli
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m | tr '[:upper:]' '[:lower:]')

case "${OS}" in
linux) OS="linux" ;;
darwin) OS="darwin" ;;
*)
    echo "Unsupported operating system: ${OS}"
    exit 1
    ;;
esac

case "${ARCH}" in
x86_64) ARCH="amd64" ;;
aarch64 | arm64) ARCH="arm64" ;;
*)
    echo "Unsupported architecture: ${ARCH}"
    exit 1
    ;;
esac

if which go &>/dev/null; then
    gobin=$(go env GOBIN)
    gopath="$(go env GOPATH)"
    gopathbin="${gopath}/bin"
    if [ -n "${gobin}" ] && [ -d "${gobin}" ]; then
        INSTALL_DIR="${gobin}"
    elif [ -n "${gopath}" ] && [ -d "${gopathbin}" ]; then
        INSTALL_DIR="${gopathbin}"
    fi
fi

while [ $# -gt 0 ]; do
    case "$1" in
    -d)
        if [ $# -lt 2 ]; then
            usage
        fi
        INSTALL_DIR="$2"
        shift 2
        ;;
    -v)
        if [ $# -lt 2 ]; then
            usage
        fi
        VERSION="$2"
        shift 2
        ;;
    --no-checksums)
        VERIFY_CHECKSUMS="false"
        shift
        ;;
    -h | --help)
        usage
        ;;
    *)
        usage
        ;;
    esac
done

if [ ! -d "${INSTALL_DIR}" ]; then
    echo "Directory ${INSTALL_DIR} does not exist"
    exit 1
fi
if [ ! -w "${INSTALL_DIR}" ]; then
    echo "Directory ${INSTALL_DIR} is not writable"
    exit 1
fi

APP_PATH="${INSTALL_DIR}/${NAME}"
ARTIFACT_NAME="${NAME}-${OS}-${ARCH}"
# Download and verify a temp file next to the target so a failed run never
# touches an existing installation, and the final move is a same-filesystem rename.
TMP_FILE=$(mktemp "${INSTALL_DIR}/.${NAME}.XXXXXX")
CHECKSUMS_FILE=""

function cleanup() {
    rm -f "${TMP_FILE}"
    if [ -n "${CHECKSUMS_FILE}" ] && [ -f "${CHECKSUMS_FILE}" ]; then
        rm -f "${CHECKSUMS_FILE}"
    fi
}
trap cleanup EXIT

function verify_checksum() {
    local app_path="$1"
    local checksums_url="$2"
    local artifact_name="$3"
    local checksum_bin=""
    local expected_checksum=""
    local actual_checksum=""

    if command -v sha256sum &>/dev/null; then
        checksum_bin="sha256sum"
    elif command -v shasum &>/dev/null; then
        checksum_bin="shasum -a 256"
    else
        echo "Error: no SHA-256 checksum tool found (sha256sum/shasum); rerun with --no-checksums to skip verification" >&2
        return 1
    fi

    CHECKSUMS_FILE=$(mktemp)
    echo "Downloading ${checksums_url}..."
    if ! curl -sSL --fail-with-body "${checksums_url}" -o "${CHECKSUMS_FILE}"; then
        echo "Error: failed to download ${checksums_url}" >&2
        rm -f "${CHECKSUMS_FILE}"
        CHECKSUMS_FILE=""
        return 1
    fi

    expected_checksum=$(awk -v target="${artifact_name}" '$2 == target { print $1 }' "${CHECKSUMS_FILE}")
    if [ -z "${expected_checksum}" ]; then
        echo "Error: checksum entry for ${artifact_name} not found" >&2
        return 1
    fi

    actual_checksum=$(${checksum_bin} "${app_path}" | awk '{ print $1 }')
    if [ "${expected_checksum}" != "${actual_checksum}" ]; then
        echo "Checksum verification failed for ${app_path}" >&2
        echo "Expected: ${expected_checksum}" >&2
        echo "Actual:   ${actual_checksum}" >&2
        return 1
    fi

    echo "Checksum verified for ${app_path}"
}

echo "Installing into ${APP_PATH}..."

if [ -z "${VERSION}" ]; then
    VERSION=$(
        curl -sSL --fail-with-body https://api.github.com/repos/trezorg/${NAME}/releases/latest |
            awk -F '"' '/tag_name/ { print $4 }'
    )
    if [ -z "${VERSION}" ]; then
        echo "Failed to detect latest version"
        exit 1
    fi
fi
DOWNLOAD_URL="https://github.com/trezorg/${NAME}/releases/download/${VERSION}/${ARTIFACT_NAME}"
CHECKSUMS_URL="https://github.com/trezorg/${NAME}/releases/download/${VERSION}/checksums.txt"
echo "Downloading ${DOWNLOAD_URL}..."

if ! curl -sSL --fail-with-body "${DOWNLOAD_URL}" -o "${TMP_FILE}"; then
    echo "Error: failed to download ${DOWNLOAD_URL}" >&2
    exit 1
fi

if [ "${VERIFY_CHECKSUMS}" = "true" ]; then
    verify_checksum "${TMP_FILE}" "${CHECKSUMS_URL}" "${ARTIFACT_NAME}"
fi

chmod +x "${TMP_FILE}"
"${TMP_FILE}" --help
mv -f "${TMP_FILE}" "${APP_PATH}"
echo "Installed ${APP_PATH}"
