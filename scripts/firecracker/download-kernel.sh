#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# download-kernel.sh
# Downloads the official AWS Firecracker vmlinux binary
# ==============================================================================

OUTPUT_DIR="$(pwd)/build/firecracker"
mkdir -p "${OUTPUT_DIR}"

if [ -f "${OUTPUT_DIR}/vmlinux" ]; then
    echo "==> Firecracker kernel already exists at ${OUTPUT_DIR}/vmlinux (skipping download)"
    exit 0
fi

ARCH="$(uname -m)"
if [ "${ARCH}" = "arm64" ] || [ "${ARCH}" = "aarch64" ]; then
    FC_ARCH="aarch64"
else
    FC_ARCH="x86_64"
fi

KERNEL_DEST="${OUTPUT_DIR}/vmlinux"

echo "==> Finding official Firecracker kernel for ${FC_ARCH}..."

# Try official S3 CI bucket v1.10 build 5.10.223
PRIMARY_URL="https://s3.amazonaws.com/spec.ccfc.min/firecracker-ci/v1.10/${FC_ARCH}/vmlinux-5.10.223"
BACKUP_URL="https://s3.amazonaws.com/spec.ccfc.min/firecracker-ci/v1.10/${FC_ARCH}/vmlinux-6.1.102"

if curl --head --silent --fail "${PRIMARY_URL}" >/dev/null; then
    KERNEL_URL="${PRIMARY_URL}"
elif curl --head --silent --fail "${BACKUP_URL}" >/dev/null; then
    KERNEL_URL="${BACKUP_URL}"
else
    # Auto-query latest kernel from S3 XML listing
    LATEST_KEY=$(curl -fsSL "https://s3.amazonaws.com/spec.ccfc.min?list-type=2&prefix=firecracker-ci/v1.10/${FC_ARCH}/" \
      | grep -o "firecracker-ci/v1.10/${FC_ARCH}/vmlinux-[0-9][^\<]*" \
      | grep -v "\.config" \
      | grep -v "\-no-acpi" \
      | head -n 1 || true)
    KERNEL_URL="https://s3.amazonaws.com/spec.ccfc.min/${LATEST_KEY}"
fi

echo "==> Downloading kernel from ${KERNEL_URL}..."
curl -fSL -o "${KERNEL_DEST}" "${KERNEL_URL}"
chmod +x "${KERNEL_DEST}"

echo "==> Kernel successfully downloaded to ${KERNEL_DEST}"
