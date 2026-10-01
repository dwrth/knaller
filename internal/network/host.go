package network

import (
	"fmt"
	"net/netip"
	"strings"

	"github.com/dwrth/knaller/internal/config"
)

const (
	filterTable = "knaller_filter"
	natTable    = "knaller_nat"
	metadataIP  = "169.254.169.254"
)

// EnsureHost enables forwarding and installs Knaller-owned nft isolation/NAT
// for all addresses in cfg.Network.GuestCidr. Idempotent: recreates tables.
func EnsureHost(cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("network: config is required")
	}
	guest := cfg.Network.GuestCidr
	if guest == "" {
		return fmt.Errorf("network: guest_cidr is required")
	}
	if _, err := netip.ParsePrefix(guest); err != nil {
		return fmt.Errorf("network: guest_cidr is invalid: %w", err)
	}

	uplink, err := defaultUplink()
	if err != nil {
		return err
	}

	if err := run("sysctl", "-q", "-w", "net.ipv4.ip_forward=1"); err != nil {
		return err
	}

	_ = run("nft", "delete", "table", "inet", filterTable)
	_ = run("nft", "delete", "table", "ip", natTable)

	if err := run("nft", "add", "table", "inet", filterTable); err != nil {
		return err
	}
	if err := run("nft", "add", "chain", "inet", filterTable, "forward",
		"{ type filter hook forward priority filter; policy accept; }"); err != nil {
		return err
	}
	// Block cloud metadata.
	if err := run("nft", "add", "rule", "inet", filterTable, "forward",
		"ip", "saddr", guest, "ip", "daddr", metadataIP, "counter", "drop"); err != nil {
		return err
	}
	// Block sandbox-to-sandbox (any guest_cidr → guest_cidr).
	if err := run("nft", "add", "rule", "inet", filterTable, "forward",
		"ip", "saddr", guest, "ip", "daddr", guest, "counter", "drop"); err != nil {
		return err
	}

	if err := run("nft", "add", "table", "ip", natTable); err != nil {
		return err
	}
	if err := run("nft", "add", "chain", "ip", natTable, "postrouting",
		"{ type nat hook postrouting priority srcnat; policy accept; }"); err != nil {
		return err
	}
	if err := run("nft", "add", "rule", "ip", natTable, "postrouting",
		"oifname", uplink, "ip", "saddr", guest, "counter", "masquerade"); err != nil {
		return err
	}
	return nil
}

// TeardownHost removes Knaller-owned nft tables. Does not change ip_forward.
func TeardownHost(cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("network: config is required")
	}
	_ = run("nft", "delete", "table", "inet", filterTable)
	_ = run("nft", "delete", "table", "ip", natTable)
	return nil
}

func defaultUplink() (string, error) {
	out, err := runOut("ip", "route", "show", "default")
	if err != nil {
		return "", fmt.Errorf("network: determine uplink: %w", err)
	}
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || fields[0] != "default" {
			continue
		}
		for i := 0; i < len(fields)-1; i++ {
			if fields[i] == "dev" {
				return fields[i+1], nil
			}
		}
	}
	return "", fmt.Errorf("network: unable to determine uplink")
}
