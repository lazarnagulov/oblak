#!/bin/bash
set -e

echo "1. Building Docker image..."
docker build -t oblak-sandbox .

echo "2. Exporting filesystem..."
docker create --name oblak_temp oblak-sandbox
docker export oblak_temp > rootfs.tar
docker rm oblak_temp

echo "3. Creating empty ext4 image..."
dd if=/dev/zero of=sandbox.rootfs.ext4 bs=1M count=150
mkfs.ext4 sandbox.rootfs.ext4

echo "4. Mounting and extracting..."
mkdir -p mnt
sudo mount sandbox.rootfs.ext4 mnt
sudo tar -xf rootfs.tar -C mnt
sudo umount mnt

# Clean up
rm rootfs.tar
rmdir mnt

echo "Completed."
mv sandbox.rootfs.ext4 ../
