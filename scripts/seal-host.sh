#!/usr/bin/env bash
set -euo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if [[ "$(id -u)" -ne 0 ]]; then
	echo "ERROR: run as root" >&2
	exit 1
fi

reset_guest() {
	VM="$1"
	ROOTFS="/var/lib/knaller/vms/$VM/rootfs.ext4"
	MNT="/mnt/knaller-seal-$VM"

	echo "==> Sealing $VM"

	mkdir -p "$MNT"

	mount -o loop "$ROOTFS" "$MNT"

	# Generate a unique systemd machine-id on the next guest boot.
	rm -f "$MNT/etc/machine-id"
	: >"$MNT/etc/machine-id"

	# dbus should refer to the system machine-id.
	rm -f "$MNT/var/lib/dbus/machine-id"
	mkdir -p "$MNT/var/lib/dbus"
	ln -s /etc/machine-id "$MNT/var/lib/dbus/machine-id"

	# Never clone an already-used systemd random seed.
	rm -f "$MNT/var/lib/systemd/random-seed"

	# Remove runtime logs from the template guest.
	rm -rf "$MNT/var/log/"*
	mkdir -p "$MNT/var/log"

	sync
	umount "$MNT"
	rmdir "$MNT"

	echo "    $VM sealed"
}

echo "==> Stopping knaller guests"

systemctl stop knaller@vm1.service || true
systemctl stop knaller@vm2.service || true

echo "==> Stopping knaller networking"

systemctl stop knaller-network@vm1.service || true
systemctl stop knaller-network@vm2.service || true
systemctl stop knaller-host-network.service || true

echo "==> Removing disposable jails"

rm -rf \
	/srv/jailer/firecracker/vm1 \
	/srv/jailer/firecracker/vm2

reset_guest vm1
reset_guest vm2

echo "==> Ensuring services remain enabled for clone boot"

systemctl enable knaller-host-network.service
systemctl enable knaller@vm1.service
systemctl enable knaller@vm2.service

echo "==> Preparing SSH host identity regeneration"

mkdir -p /etc/cloud/cloud.cfg.d

cat >/etc/cloud/cloud.cfg.d/90-knaller-golden-image.cfg <<'CLOUD'
ssh_deletekeys: true
ssh_genkeytypes:
  - rsa
  - ecdsa
  - ed25519
CLOUD

echo "==> Cleaning host cloud-init identity"

if command -v cloud-init >/dev/null 2>&1; then
	cloud-init clean --logs --machine-id
else
	echo "ERROR: cloud-init is not installed" >&2
	exit 1
fi

echo "==> Removing host random seed"

rm -f /var/lib/systemd/random-seed

echo "==> Removing temporary files"

rm -rf /tmp/*
rm -rf /var/tmp/*

apt-get clean

sync

echo
echo "===================================================="
echo "Golden image sealing completed."
echo "===================================================="
echo
echo "DO NOT reboot this machine."
echo "Power it off now and create the IONOS snapshot."
