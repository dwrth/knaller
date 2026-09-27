package network

import (
	"fmt"
	"net/netip"
)

// addrWithPrefix joins an IP with the prefix length of subnetCIDR (e.g. 10.200.1.1 + 10.200.1.0/30 → 10.200.1.1/30).
func addrWithPrefix(ip, subnetCIDR string) (string, error) {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return "", fmt.Errorf("network: invalid address %q: %w", ip, err)
	}
	prefix, err := netip.ParsePrefix(subnetCIDR)
	if err != nil {
		return "", fmt.Errorf("network: invalid subnet %q: %w", subnetCIDR, err)
	}
	if !prefix.Contains(addr) {
		return "", fmt.Errorf("network: address %s is not in subnet %s", ip, subnetCIDR)
	}
	return netip.PrefixFrom(addr, prefix.Bits()).String(), nil
}
