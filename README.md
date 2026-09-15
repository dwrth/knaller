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
knaller capacity          # host resources and what is still free
knaller create            # provision a new sandbox
knaller list              # running and stopped sandboxes
knaller inspect <name>    # full sandbox state and paths
knaller start <name>      # start via systemd
knaller stop <name>       # graceful shutdown
knaller delete <name>     # tear down sandbox and storage
knaller config validate   # check /etc/knaller/config.yaml
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

Higher-level sizing and fleet placement are intentionally outside Knaller.

## Repository layout

- `cmd/knaller/` - CLI entrypoint
- `internal/` - config, capacity, scheduler, state, allocate, storage, ...
- `config/` - example host configuration
- `host/` - jailer helpers, host networking, systemd units
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

systemd: knaller@.service, knaller-network@.service
```

Sandbox identity is a permanent ULID. Node-local slots (reused after
state deletion) drive UID/GID, network namespaces, and `/30` guest/transit
networks. Desired vs observed lifecycle state is persisted for recovery.
A node-local flock guards mutating operations (CLI wiring comes with create).

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
- [ ] Generic jailer / supervision
- [ ] Generic networking / default isolation
- [ ] create / delete with rollback
- [ ] list / inspect / start / stop
- [ ] Resource enforcement
- [ ] doctor / reconcile
- [ ] Fault / reboot testing
- [ ] Golden image / clean-room test
- [ ] Zunder integration
