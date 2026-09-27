// Package network manages per-sandbox dataplane and host-wide isolation policy.
//
// Setup/Teardown/Status operate on one sandbox from persisted state.
// Status probes netns, host veth, and guest route presence.
// EnsureHost configures node-wide forwarding and nft policy from config.
//
// Setup and Teardown refuse if the jail pidfile names a live process
// (via jailer.EnsureStopped); callers must Stop first.
package network
