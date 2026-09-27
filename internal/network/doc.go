// Package network manages per-sandbox dataplane and host-wide isolation policy.
//
// Setup/Teardown/Status operate on one sandbox from persisted state.
// EnsureHost configures node-wide forwarding and nft policy from config.
//
// Setup must eventually refuse if the sandbox Firecracker process is still
// alive; callers should Stop first. Enforcement lands with real dataplane Setup.
package network
