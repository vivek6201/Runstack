#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# build-rootfs.sh
# Builds the Docker rootfs image and exports it into an ext4 disk image for Firecracker
# Uses a privileged container to format ext4 so it runs on both macOS and Linux!
# ==============================================================================

OUTPUT_DIR="$(pwd)/build/firecracker"
ROOTFS_IMG="${OUTPUT_DIR}/rootfs.ext4"
# Allocate 2500MB to comfortably fit OpenJDK 17, Go compiler, GCC/G++, Python, Node, and Bun
DISK_SIZE_MB=2500

mkdir -p "${OUTPUT_DIR}"
rm -f "${ROOTFS_IMG}"

echo "==> [1/3] Building Rootfs Docker image with all 7 language runtimes..."
docker build -f infra/docker/Dockerfile.rootfs -t coderunner-rootfs:latest .

echo "==> [2/3] Exporting rootfs tar archive..."
CONTAINER_ID=$(docker create coderunner-rootfs:latest)
docker export "${CONTAINER_ID}" -o "${OUTPUT_DIR}/rootfs.tar"
docker rm "${CONTAINER_ID}"

echo "==> [3/3] Creating and formatting ${DISK_SIZE_MB}MB ext4 disk image..."
# Use a privileged Alpine container to format and extract the ext4 disk
docker run --rm --privileged \
  -v "${OUTPUT_DIR}:/out" \
  alpine:3.20 sh -c "
    apk add --no-cache e2fsprogs tar &&
    dd if=/dev/zero of=/out/rootfs.ext4 bs=1M count=${DISK_SIZE_MB} status=none &&
    mkfs.ext4 -F -q /out/rootfs.ext4 &&
    mkdir -p /mnt/rootfs &&
    mount -o loop /out/rootfs.ext4 /mnt/rootfs &&
    tar -xf /out/rootfs.tar -C /mnt/rootfs &&
    umount /mnt/rootfs &&
    rm -f /out/rootfs.tar
  "

echo "==> Successfully generated ${ROOTFS_IMG}!"
