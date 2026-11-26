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
KERNEL_TMP="${KERNEL_DEST}.tmp"
HEADERS_TMP="/tmp/fc-kernel-headers.$$"

cleanup() {
    rm -f "${KERNEL_TMP}" "${HEADERS_TMP}"
}
trap cleanup EXIT

compute_sha256() {
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$1" | awk '{print $1}'
    else
        shasum -a 256 "$1" | awk '{print $1}'
    fi
}

compute_md5() {
    if command -v md5sum >/dev/null 2>&1; then
        md5sum "$1" | awk '{print $1}'
    else
        md5 -q "$1"
    fi
}

echo "==> Finding official Firecracker kernel for ${FC_ARCH}..."

# Try official S3 CI bucket v1.10 build 5.10.223
PRIMARY_URL="https://s3.amazonaws.com/spec.ccfc.min/firecracker-ci/v1.10/${FC_ARCH}/vmlinux-5.10.223"
BACKUP_URL="https://s3.amazonaws.com/spec.ccfc.min/firecracker-ci/v1.10/${FC_ARCH}/vmlinux-6.1.102"

if curl --head --silent --fail --proto '=https' --tlsv1.2 "${PRIMARY_URL}" >/dev/null; then
    KERNEL_URL="${PRIMARY_URL}"
elif curl --head --silent --fail --proto '=https' --tlsv1.2 "${BACKUP_URL}" >/dev/null; then
    KERNEL_URL="${BACKUP_URL}"
else
    # Auto-query latest kernel from S3 XML listing
    LATEST_KEY=$(curl -fsSL --proto '=https' --tlsv1.2 "https://s3.amazonaws.com/spec.ccfc.min?list-type=2&prefix=firecracker-ci/v1.10/${FC_ARCH}/" \
      | grep -o "firecracker-ci/v1.10/${FC_ARCH}/vmlinux-[0-9][^\<]*" \
      | grep -v "\.config" \
      | grep -v "\-no-acpi" \
      | head -n 1 || true)
    if [ -z "${LATEST_KEY}" ]; then
        echo "ERROR: Could not locate a valid vmlinux binary in S3 bucket" >&2
        exit 1
    fi
    KERNEL_URL="https://s3.amazonaws.com/spec.ccfc.min/${LATEST_KEY}"
fi

echo "==> Downloading kernel from ${KERNEL_URL}..."
curl -fSL --proto '=https' --tlsv1.2 -D "${HEADERS_TMP}" -o "${KERNEL_TMP}" "${KERNEL_URL}"

# 1. Verify ELF binary header (prevents saving HTML/XML error pages as executable kernels)
ELF_MAGIC="$(head -c 4 "${KERNEL_TMP}" | od -An -tx1 | tr -d ' \n')"
if [ "${ELF_MAGIC}" != "7f454c46" ]; then
    echo "ERROR: Downloaded kernel is not a valid ELF binary (magic: ${ELF_MAGIC})" >&2
    exit 1
fi

# 2. Verify pinned SHA-256 checksum if provided
ACTUAL_SHA256="$(compute_sha256 "${KERNEL_TMP}")"
if [ -n "${KERNEL_SHA256:-}" ]; then
    if [ "${ACTUAL_SHA256}" != "${KERNEL_SHA256}" ]; then
        echo "ERROR: Kernel SHA-256 mismatch!" >&2
        echo "  Expected: ${KERNEL_SHA256}" >&2
        echo "  Actual:   ${ACTUAL_SHA256}" >&2
        exit 1
    fi
    echo "==> Pinned SHA-256 verified (${ACTUAL_SHA256})"
else
    # 3. Verify S3 single-part ETag MD5 digest when available
    ETAG="$(grep -i '^etag:' "${HEADERS_TMP}" | tail -n 1 | sed -E 's/^[^:]+:[[:space:]]*"?([^"\r\n]+)"?.*/\1/' | tr -d '\r' || true)"
    if echo "${ETAG}" | grep -Eq '^[0-9a-fA-F]{32}$'; then
        ACTUAL_MD5="$(compute_md5 "${KERNEL_TMP}")"
        if [ "${ACTUAL_MD5}" != "$(echo "${ETAG}" | tr '[:upper:]' '[:lower:]')" ]; then
            echo "ERROR: Kernel ETag MD5 integrity mismatch!" >&2
            echo "  Expected ETag: ${ETAG}" >&2
            echo "  Actual MD5:    ${ACTUAL_MD5}" >&2
            exit 1
        fi
        echo "==> Kernel ETag MD5 integrity verified (${ACTUAL_MD5})"
    fi
    echo "==> Kernel SHA-256: ${ACTUAL_SHA256} (set KERNEL_SHA256 to pin in CI)"
fi

mv "${KERNEL_TMP}" "${KERNEL_DEST}"
chmod 0755 "${KERNEL_DEST}"

echo "==> Kernel successfully downloaded to ${KERNEL_DEST}"
