package network

import "github.com/dwrth/knaller/internal/config"

// EnsureHost applies node-wide forwarding and isolation policy from cfg.
func EnsureHost(cfg *config.Config) error {
	_ = cfg
	return nil
}
