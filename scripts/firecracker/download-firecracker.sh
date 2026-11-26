#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# download-firecracker.sh
# Downloads the official Firecracker VMM binary for the current architecture
# ==============================================================================

FC_VERSION="v1.10.0"
OUTPUT_DIR="$(pwd)/build/firecracker"
mkdir -p "${OUTPUT_DIR}"

if [ -f "${OUTPUT_DIR}/firecracker" ]; then
    echo "==> Firecracker binary already exists at ${OUTPUT_DIR}/firecracker (skipping download)"
    exit 0
fi

ARCH="$(uname -m)"
if [ "${ARCH}" = "arm64" ] || [ "${ARCH}" = "aarch64" ]; then
    FC_ARCH="aarch64"
else
    FC_ARCH="x86_64"
fi

TAR_NAME="firecracker-${FC_VERSION}-${FC_ARCH}.tgz"
URL="https://github.com/firecracker-microvm/firecracker/releases/download/${FC_VERSION}/${TAR_NAME}"
SHA_URL="${URL}.sha256.txt"
TMP_TAR="/tmp/${TAR_NAME}"
TMP_SHA="/tmp/${TAR_NAME}.sha256.txt"

cleanup() {
    rm -rf "${TMP_TAR}" "${TMP_SHA}" "/tmp/release-${FC_VERSION}-${FC_ARCH}"
}
trap cleanup EXIT

compute_sha256() {
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$1" | awk '{print $1}'
    else
        shasum -a 256 "$1" | awk '{print $1}'
    fi
}

echo "==> Downloading Firecracker ${FC_VERSION} for ${FC_ARCH}..."
curl -fSL --proto '=https' --tlsv1.2 -o "${TMP_TAR}" "${URL}"

if [ -n "${FIRECRACKER_SHA256:-}" ]; then
    EXPECTED_HASH="${FIRECRACKER_SHA256}"
else
    echo "==> Fetching official SHA-256 checksum (${SHA_URL})..."
    curl -fSL --proto '=https' --tlsv1.2 -o "${TMP_SHA}" "${SHA_URL}"
    EXPECTED_HASH="$(awk '{print $1}' "${TMP_SHA}" | head -n 1)"
fi

if [ -z "${EXPECTED_HASH}" ]; then
    echo "ERROR: Could not determine expected SHA-256 checksum for ${TAR_NAME}" >&2
    exit 1
fi

ACTUAL_HASH="$(compute_sha256 "${TMP_TAR}")"
if [ "${ACTUAL_HASH}" != "${EXPECTED_HASH}" ]; then
    echo "ERROR: SHA-256 checksum mismatch for ${TAR_NAME}!" >&2
    echo "  Expected: ${EXPECTED_HASH}" >&2
    echo "  Actual:   ${ACTUAL_HASH}" >&2
    exit 1
fi
echo "==> Checksum verified (${ACTUAL_HASH})"

echo "==> Extracting binary..."
tar -xzf "${TMP_TAR}" -C /tmp
mv "/tmp/release-${FC_VERSION}-${FC_ARCH}/firecracker-${FC_VERSION}-${FC_ARCH}" "${OUTPUT_DIR}/firecracker"
chmod 0755 "${OUTPUT_DIR}/firecracker"

echo "==> Firecracker installed at ${OUTPUT_DIR}/firecracker"
