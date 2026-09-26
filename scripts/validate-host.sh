#!/usr/bin/env bash
set -euo pipefail

fail=0

check() {
	DESCRIPTION="$1"
	shift

	printf '%-45s' "$DESCRIPTION"

	if "$@" >/dev/null 2>&1; then
		echo "OK"
	else
		echo "FAIL"
		fail=1
	fi
}

echo "=== KNALLER HOST VALIDATION ==="
echo

check \
	"/dev/kvm available" \
	test -c /dev/kvm

check \
	"Firecracker installed" \
	test -x /usr/local/bin/firecracker

check \
	"Jailer installed" \
	test -x /usr/local/bin/jailer

check \
	"VM1 rootfs installed" \
	test -f /var/lib/knaller/vms/vm1/rootfs.ext4

check \
	"VM2 rootfs installed" \
	test -f /var/lib/knaller/vms/vm2/rootfs.ext4

check \
	"VM1 kernel installed" \
	test -f /var/lib/knaller/vms/vm1/vmlinux

check \
	"VM2 kernel installed" \
	test -f /var/lib/knaller/vms/vm2/vmlinux

check \
	"VM1 config valid" \
	jq empty /var/lib/knaller/vms/vm1/config.json

check \
	"VM2 config valid" \
	jq empty /var/lib/knaller/vms/vm2/config.json

check \
	"VM1 state present" \
	jq -e '.uid and .gid and .namespace' /var/lib/knaller/state/vm1.json

check \
	"VM2 state present" \
	jq -e '.uid and .gid and .namespace' /var/lib/knaller/state/vm2.json

check \
	"Host network enabled" \
	systemctl is-enabled knaller-host-network.service

check \
	"VM1 enabled" \
	systemctl is-enabled knaller@vm1.service

check \
	"VM2 enabled" \
	systemctl is-enabled knaller@vm2.service

echo

EXPECTED_KERNEL="$(
	awk -F= \
		'$1 == "KERNEL_SHA256" {print $2}' \
		"$(dirname "$0")/../artifacts/kernel/manifest.env"
)"

VM1_KERNEL="$(
	sha256sum /var/lib/knaller/vms/vm1/vmlinux |
		awk '{print $1}'
)"

VM2_KERNEL="$(
	sha256sum /var/lib/knaller/vms/vm2/vmlinux |
		awk '{print $1}'
)"

printf '%-45s' "VM1 kernel checksum"
if [[ "$VM1_KERNEL" == "$EXPECTED_KERNEL" ]]; then
	echo "OK"
else
	echo "FAIL"
	fail=1
fi

printf '%-45s' "VM2 kernel checksum"
if [[ "$VM2_KERNEL" == "$EXPECTED_KERNEL" ]]; then
	echo "OK"
else
	echo "FAIL"
	fail=1
fi

echo

if [[ "$fail" -ne 0 ]]; then
	echo "Host validation FAILED"
	exit 1
fi

echo "Host validation PASSED"
