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

echo "==> Downloading Firecracker ${FC_VERSION} for ${FC_ARCH}..."
curl -fSL -o "/tmp/${TAR_NAME}" "${URL}"

echo "==> Extracting binary..."
tar -xzf "/tmp/${TAR_NAME}" -C /tmp
mv "/tmp/release-${FC_VERSION}-${FC_ARCH}/firecracker-${FC_VERSION}-${FC_ARCH}" "${OUTPUT_DIR}/firecracker"
chmod +x "${OUTPUT_DIR}/firecracker"
rm -rf "/tmp/${TAR_NAME}" "/tmp/release-${FC_VERSION}-${FC_ARCH}"

echo "==> Firecracker installed at ${OUTPUT_DIR}/firecracker"
