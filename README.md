# Knaller

Node-local Firecracker microVM runtime for an IONOS Cloud Cube.

> Knaller is the German word for "firecracker" (/ˈknalɐ/, roughly _KNALL-er_).

A single Ubuntu host runs many isolated microVM sandboxes. The `knaller` CLI
creates, starts, stops, lists, inspects, and deletes sandboxes subject to
host capacity. Each sandbox runs under Firecracker jailer with its own Linux
user, network namespace, cloned rootfs, and TAP/veth networking. systemd
supervises the lifecycle via `knaller@…` units.

The repo evolved from a static two-VM setup (`v1.0.0`) into this generic
node-local runtime. See the roadmap for current progress.

## CLI

```text
knaller capacity                 # host resources and what is still free
knaller create                   # provision a new sandbox
knaller network ensure-host      # host forwarding + nft isolation/NAT
knaller network teardown-host    # remove Knaller nft tables
knaller network setup <id>       # per-sandbox netns/veth/TAP/routes
knaller network teardown <id>    # remove per-sandbox dataplane
knaller network status <id>      # probe netns / host veth / guest route
knaller list                     # running and stopped sandboxes
knaller inspect <name>           # full sandbox state and paths
knaller start <name>             # start via systemd
knaller stop <name>              # graceful shutdown
knaller delete <name>            # tear down sandbox and storage
knaller config validate          # check /etc/knaller/config.yaml
```

Creation examples (target):

```bash
knaller create --name web-1 --cpus 2 --memory 512
knaller create --name worker-1 --cpus 1 --memory 256
```

Admission enforces strict RAM limits and configurable CPU overcommit.
Provisioning clones a shared base rootfs, assigns sandbox-local resources,
UID/GID, guest and transit subnets, generates Firecracker config, wires
networking, and persists state for reboot-safe operation.

Host networking (`EnsureHost`) applies to the whole `guest_cidr`: drop cloud
metadata, block sandbox-to-sandbox forwarding, and MASQUERADE egress on the
uplink. Per-sandbox `Setup`/`Teardown` build and remove netns, veth, TAP, and
routes from persisted state. Thin `/usr/local/sbin/knaller-*-network` wrappers
call the CLI for systemd; those scripts are temporary until a pure Go host path.

Higher-level sizing and fleet placement are intentionally outside Knaller.

## Repository layout

- `cmd/knaller/` - CLI entrypoint
- `internal/` - config, capacity, scheduler, state, allocate, storage, jailer, network, ...
- `config/` - example host configuration
- `host/` - jailer helpers, thin network wrappers, systemd units
- `guests/` - legacy static VM configs (`vm1`, `vm2`; replaced by dynamic flow)
- `scripts/` - image and host build automation
- `artifacts/` - artifact manifests and checksums
- `versions.env` - pinned software and kernel versions

## Host runtime layout

```text
/etc/knaller/config.yaml

/var/lib/knaller/images/
  base-rootfs.ext4
  vmlinux-6.18.44

/var/lib/knaller/state/
  <sandbox-id>.json

/var/lib/knaller/vms/<sandbox-id>/
  config.json
  rootfs.ext4
  vmlinux

/srv/jailer/firecracker/<sandbox-id>/   # disposable jail state

systemd: knaller-host-network.service, knaller-network@.service, knaller@.service
```

Sandbox identity is a permanent ULID. Node-local slots (reused after
state deletion) drive UID/GID, network namespaces, and `/30` guest/transit
networks. Desired vs observed lifecycle state is persisted for recovery.
A node-local flock guards mutating operations (CLI wiring comes with create).
Jailer prepare/start/stop/remove and network setup/teardown are generic over
sandbox ID. Host install expects a built `knaller` binary
(`go build -o build/artifacts/knaller ./cmd/knaller`).

## Roadmap

- [x] v1.0.0 static two-VM baseline
- [x] Go CLI skeleton
- [x] Config loading
- [x] Capacity reporting
- [x] Sandbox state model
- [x] Scheduler / node-local admission
- [x] Resource allocation
- [x] Base-rootfs workflow
- [x] Firecracker config
- [x] Runtime state / lifecycle foundations
- [x] Generic jailer / supervision
- [x] Generic networking / default isolation
- [ ] create / delete with rollback
- [ ] list / inspect / start / stop
- [ ] Resource enforcement
- [ ] doctor / reconcile
- [ ] Fault / reboot testing
- [ ] Golden image / clean-room test
- [ ] Zunder integration
