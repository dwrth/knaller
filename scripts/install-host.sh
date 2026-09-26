#!/usr/bin/env bash
set -euo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if [[ "$(id -u)" -ne 0 ]]; then
	echo "This installer must run as root" >&2
	exit 1
fi

# shellcheck disable=SC1091
source "$REPO/versions.env"

# shellcheck disable=SC1091
source "$REPO/artifacts/kernel/manifest.env"

FC_BIN="$REPO/build/artifacts/firecracker"
JAILER_BIN="$REPO/build/artifacts/jailer"
KERNEL="$REPO/artifacts/kernel/$KERNEL_FILENAME"

VM1_IMAGE="$REPO/build/guests/vm1.ext4"
VM2_IMAGE="$REPO/build/guests/vm2.ext4"

VM1_CONFIG="$REPO/guests/vm1/config.json"
VM2_CONFIG="$REPO/guests/vm2/config.json"

die() {
	echo "ERROR: $*" >&2
	exit 1
}

verify_file() {
	[[ -f "$1" ]] || die "Required file missing: $1"
}

verify_artifacts() {
	echo "==> Verifying required artifacts"

	verify_file "$FC_BIN"
	verify_file "$JAILER_BIN"
	verify_file "$KERNEL"

	verify_file "$VM1_IMAGE"
	verify_file "$VM2_IMAGE"

	verify_file "$VM1_CONFIG"
	verify_file "$VM2_CONFIG"

	"$REPO/scripts/verify-kernel.sh"

	EXPECTED_FC="$(
		awk '$1 == "firecracker" {print $2}' \
			"$REPO/artifacts/artifacts.sha256"
	)"

	EXPECTED_JAILER="$(
		awk '$1 == "jailer" {print $2}' \
			"$REPO/artifacts/artifacts.sha256"
	)"

	ACTUAL_FC="$(sha256sum "$FC_BIN" | awk '{print $1}')"
	ACTUAL_JAILER="$(sha256sum "$JAILER_BIN" | awk '{print $1}')"

	[[ "$ACTUAL_FC" == "$EXPECTED_FC" ]] ||
		die "Firecracker checksum mismatch"

	[[ "$ACTUAL_JAILER" == "$EXPECTED_JAILER" ]] ||
		die "Jailer checksum mismatch"

	(
		cd "$REPO/build/guests"
		sha256sum -c "$REPO/artifacts/guest-images.sha256"
	)

	jq empty "$VM1_CONFIG"
	jq empty "$VM2_CONFIG"

	echo "==> Artifact verification passed"
}

install_packages() {
	echo "==> Installing host packages"

	apt-get update

	DEBIAN_FRONTEND=noninteractive apt-get install -y \
		ca-certificates \
		curl \
		jq \
		iproute2 \
		nftables \
		e2fsprogs \
		util-linux \
		kmod
}

ensure_account() {
	NAME="$1"
	UID_NUM="$2"
	GID_NUM="$3"

	if getent group "$NAME" >/dev/null; then
		CURRENT_GID="$(getent group "$NAME" | cut -d: -f3)"

		[[ "$CURRENT_GID" == "$GID_NUM" ]] ||
			die "$NAME group exists with GID $CURRENT_GID, expected $GID_NUM"
	elif getent group "$GID_NUM" >/dev/null; then
		die "GID $GID_NUM already belongs to another group"
	else
		groupadd \
			--system \
			--gid "$GID_NUM" \
			"$NAME"
	fi

	if getent passwd "$NAME" >/dev/null; then
		CURRENT_UID="$(id -u "$NAME")"

		[[ "$CURRENT_UID" == "$UID_NUM" ]] ||
			die "$NAME user exists with UID $CURRENT_UID, expected $UID_NUM"
	elif getent passwd "$UID_NUM" >/dev/null; then
		die "UID $UID_NUM already belongs to another user"
	else
		useradd \
			--system \
			--uid "$UID_NUM" \
			--gid "$GID_NUM" \
			--no-create-home \
			--shell /usr/sbin/nologin \
			"$NAME"
	fi
}

install_accounts() {
	echo "==> Creating knaller identities"

	ensure_account kn-vm1 "$VM1_UID" "$VM1_GID"
	ensure_account kn-vm2 "$VM2_UID" "$VM2_GID"
}

install_binaries() {
	echo "==> Installing Firecracker binaries"

	install \
		-o root \
		-g root \
		-m 0755 \
		"$FC_BIN" \
		/usr/local/bin/firecracker

	install \
		-o root \
		-g root \
		-m 0755 \
		"$JAILER_BIN" \
		/usr/local/bin/jailer

	/usr/local/bin/firecracker --version
	/usr/local/bin/jailer --version
}

install_host_helpers() {
	echo "==> Installing host helpers"

	for SCRIPT in \
		knaller-kvm-setup \
		knaller-host-network \
		knaller-vm-network \
		knaller-prepare \
		knaller-start \
		knaller-stop; do
		install \
			-o root \
			-g root \
			-m 0755 \
			"$REPO/host/$SCRIPT" \
			"/usr/local/sbin/$SCRIPT"
	done
}

install_host_configuration() {
	echo "==> Installing kernel/module/sysctl configuration"

	install \
		-o root \
		-g root \
		-m 0644 \
		"$REPO/host/modules-load/knaller-kvm.conf" \
		/etc/modules-load.d/knaller-kvm.conf

	install \
		-o root \
		-g root \
		-m 0644 \
		"$REPO/host/sysctl/99-knaller.conf" \
		/etc/sysctl.d/99-knaller.conf

	sysctl --system >/dev/null

	/usr/local/sbin/knaller-kvm-setup

	[[ -c /dev/kvm ]] ||
		die "/dev/kvm is unavailable"
}

install_vm() {
	VM="$1"
	UID_NUM="$2"
	GID_NUM="$3"
	IMAGE="$4"
	CONFIG="$5"
	SLOT="$6"
	NAMESPACE="$7"

	DEST="/var/lib/knaller/vms/$VM"
	STATE_DIR="/var/lib/knaller/state"
	STATE="$STATE_DIR/$VM.json"

	echo "==> Installing $VM"

	install -d \
		-o root \
		-g root \
		-m 0755 \
		/var/lib/knaller/vms

	install -d \
		-o root \
		-g root \
		-m 0755 \
		"$STATE_DIR"

	install -d \
		-o root \
		-g root \
		-m 0750 \
		"$DEST"

	cp --reflink=auto --sparse=always \
		"$IMAGE" \
		"$DEST/rootfs.ext4"

	install \
		-o root \
		-g "$GID_NUM" \
		-m 0440 \
		"$KERNEL" \
		"$DEST/vmlinux"

	install \
		-o root \
		-g "$GID_NUM" \
		-m 0440 \
		"$CONFIG" \
		"$DEST/config.json"

	chown "$UID_NUM:$GID_NUM" \
		"$DEST/rootfs.ext4"

	chmod 0600 \
		"$DEST/rootfs.ext4"

	# State consumed by generic knaller-prepare/start (no vm1/vm2 branches).
	jq -n \
		--arg id "$VM" \
		--arg name "$VM" \
		--argjson slot "$SLOT" \
		--argjson uid "$UID_NUM" \
		--argjson gid "$GID_NUM" \
		--arg namespace "$NAMESPACE" \
		'{
			id: $id,
			name: $name,
			slot: $slot,
			uid: $uid,
			gid: $gid,
			namespace: $namespace,
			desired_state: "stopped",
			observed_state: "requested"
		}' >"$STATE"
	chmod 0640 "$STATE"
}

install_vm_artifacts() {
	echo "==> Installing VM artifacts"

	install -d \
		-o root \
		-g root \
		-m 0755 \
		/var/lib/knaller

	install -d \
		-o root \
		-g root \
		-m 0755 \
		/srv/jailer

	install -d \
		-o root \
		-g root \
		-m 0755 \
		/srv/jailer/firecracker

	# knaller-prepare uses hard links.
	VAR_DEV="$(stat -c %d /var/lib/knaller)"
	JAIL_DEV="$(stat -c %d /srv/jailer/firecracker)"

	[[ "$VAR_DEV" == "$JAIL_DEV" ]] ||
		die "/var/lib/knaller and /srv/jailer must be on the same filesystem"

	install_vm \
		vm1 \
		"$VM1_UID" \
		"$VM1_GID" \
		"$VM1_IMAGE" \
		"$VM1_CONFIG" \
		1 \
		kn-vm1

	install_vm \
		vm2 \
		"$VM2_UID" \
		"$VM2_GID" \
		"$VM2_IMAGE" \
		"$VM2_CONFIG" \
		2 \
		kn-vm2
}

install_systemd() {
	echo "==> Installing systemd units"

	install \
		-o root \
		-g root \
		-m 0644 \
		"$REPO/host/systemd/knaller-host-network.service" \
		/etc/systemd/system/knaller-host-network.service

	install \
		-o root \
		-g root \
		-m 0644 \
		"$REPO/host/systemd/knaller-network@.service" \
		'/etc/systemd/system/knaller-network@.service'

	install \
		-o root \
		-g root \
		-m 0644 \
		"$REPO/host/systemd/knaller@.service" \
		'/etc/systemd/system/knaller@.service'

	# Remove obsolete pre-rebrand / pre-template units if present.
	systemctl disable firecracker-network.service \
		>/dev/null 2>&1 || true
	systemctl disable firecracker-host-network.service \
		>/dev/null 2>&1 || true
	systemctl disable firecracker@vm1.service \
		>/dev/null 2>&1 || true
	systemctl disable firecracker@vm2.service \
		>/dev/null 2>&1 || true

	rm -f /etc/systemd/system/firecracker-network.service
	rm -f /etc/systemd/system/firecracker-host-network.service
	rm -f '/etc/systemd/system/firecracker-network@.service'
	rm -f '/etc/systemd/system/firecracker@.service'
	rm -f /usr/local/sbin/firecracker-{kvm-setup,host-network,vm-network,prepare,start,stop}
	rm -f /etc/modules-load.d/firecracker-kvm.conf
	rm -f /etc/sysctl.d/99-firecracker.conf

	systemctl daemon-reload

	systemctl enable knaller-host-network.service
	systemctl enable knaller@vm1.service
	systemctl enable knaller@vm2.service
}

validate_installation() {
	echo "==> Validating installation"

	test -x /usr/local/bin/firecracker
	test -x /usr/local/bin/jailer

	test -x /usr/local/sbin/knaller-host-network
	test -x /usr/local/sbin/knaller-vm-network
	test -x /usr/local/sbin/knaller-prepare
	test -x /usr/local/sbin/knaller-start
	test -x /usr/local/sbin/knaller-stop

	test -c /dev/kvm

	test -f /var/lib/knaller/vms/vm1/vmlinux
	test -f /var/lib/knaller/vms/vm1/rootfs.ext4
	test -f /var/lib/knaller/vms/vm1/config.json
	test -f /var/lib/knaller/state/vm1.json

	test -f /var/lib/knaller/vms/vm2/vmlinux
	test -f /var/lib/knaller/vms/vm2/rootfs.ext4
	test -f /var/lib/knaller/vms/vm2/config.json
	test -f /var/lib/knaller/state/vm2.json

	jq empty /var/lib/knaller/vms/vm1/config.json
	jq empty /var/lib/knaller/vms/vm2/config.json
	jq -e '.uid and .gid and .namespace' /var/lib/knaller/state/vm1.json >/dev/null
	jq -e '.uid and .gid and .namespace' /var/lib/knaller/state/vm2.json >/dev/null

	systemctl is-enabled knaller-host-network.service
	systemctl is-enabled knaller@vm1.service
	systemctl is-enabled knaller@vm2.service

	echo "==> Installation validation passed"
}

main() {
	verify_artifacts
	install_packages
	install_accounts
	install_binaries
	install_host_helpers
	install_host_configuration
	install_vm_artifacts
	install_systemd
	validate_installation

	echo
	echo "Host installation complete."
	echo
	echo "The knaller VMs have been installed and enabled,"
	echo "but this installer intentionally does not start them."
	echo
	echo "To test:"
	echo "  systemctl start knaller-host-network.service"
	echo "  systemctl start knaller@vm1.service"
	echo "  systemctl start knaller@vm2.service"
}

main "$@"
