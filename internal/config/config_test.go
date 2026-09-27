package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dwrth/knaller/internal/config"
)

func TestConfig_Validate(t *testing.T) {
	tmp := t.TempDir()

	rootfs := filepath.Join(tmp, "rootfs")
	if err := os.MkdirAll(rootfs, 0755); err != nil {
		t.Fatalf("failed to create temp rootfs: %v", err)
	}

	kernel := filepath.Join(tmp, "kernel")
	if err := os.MkdirAll(kernel, 0755); err != nil {
		t.Fatalf("failed to create temp kernel: %v", err)
	}

	validCfg := config.Config{
		Storage: config.Storage{
			BaseRootfs: rootfs,
			Kernel:     kernel,
		},
		Scheduler: config.Scheduler{
			HostMemoryReserveMiB:    256,
			CPUOvercommitRatio:      2.0,
			MinimumSandboxMemoryMiB: 128,
			MaximumSandboxMemoryMiB: 512,
			MaximumSandboxVCPUs:     4,
		},
		Network: config.Network{
			GuestCidr:   "10.0.0.0/24",
			TransitCidr: "192.168.2.0/24",
		},
		Jailer: config.Jailer{
			BaseDir:  "/var/jailer",
			UidStart: 1001,
			GidStart: 1001,
		},
		State: config.State{
			Directory: "/var/state",
		},
		Firecracker: config.Firecracker{
			SandboxDirectory: "/var/sandboxes",
		},
	}

	tests := []struct {
		name    string
		wantErr bool
		config  config.Config
	}{
		{
			name:    "valid config",
			wantErr: false,
			config:  validCfg,
		},
		{
			name:    "missing base_rootfs",
			wantErr: true,
			config: func() config.Config {
				cfg := validCfg
				cfg.Storage.BaseRootfs = ""
				return cfg
			}(),
		},
		{
			name:    "missing kernel",
			wantErr: true,
			config: func() config.Config {
				cfg := validCfg
				cfg.Storage.Kernel = ""
				return cfg
			}(),
		},
		{
			name:    "base_rootfs does not exist",
			wantErr: true,
			config: func() config.Config {
				cfg := validCfg
				cfg.Storage.BaseRootfs = "/path/to/non/existent/rootfs"
				return cfg
			}(),
		},
		{
			name:    "kernel does not exist",
			wantErr: true,
			config: func() config.Config {
				cfg := validCfg
				cfg.Storage.Kernel = "/path/to/non/existent/kernel"
				return cfg
			}(),
		},
		{
			name:    "cpu_overcommit_ratio too low",
			wantErr: true,
			config: func() config.Config {
				cfg := validCfg
				cfg.Scheduler.CPUOvercommitRatio = 0
				return cfg
			}(),
		},
		{
			name:    "host_memory_reserve_mib too low",
			wantErr: true,
			config: func() config.Config {
				cfg := validCfg
				cfg.Scheduler.HostMemoryReserveMiB = 0
				return cfg
			}(),
		},
		{
			name:    "minimum_sandbox_memory_mib too low",
			wantErr: true,
			config: func() config.Config {
				cfg := validCfg
				cfg.Scheduler.MinimumSandboxMemoryMiB = 0
				return cfg
			}(),
		},
		{
			name:    "maximum_sandbox_memory_mib too low",
			wantErr: true,
			config: func() config.Config {
				cfg := validCfg
				cfg.Scheduler.MaximumSandboxMemoryMiB = 0
				return cfg
			}(),
		},
		{
			name:    "maximum_sandbox_memory_mib less than minimum_sandbox_memory_mib",
			wantErr: true,
			config: func() config.Config {
				cfg := validCfg
				cfg.Scheduler.MaximumSandboxMemoryMiB = 64
				cfg.Scheduler.MinimumSandboxMemoryMiB = 128
				return cfg
			}(),
		},
		{
			name:    "invalid guest_cidr",
			wantErr: true,
			config: func() config.Config {
				cfg := validCfg
				cfg.Network.GuestCidr = "invalid_cidr"
				return cfg
			}(),
		},
		{
			name:    "invalid transit_cidr",
			wantErr: true,
			config: func() config.Config {
				cfg := validCfg
				cfg.Network.TransitCidr = "not_a_cidr"
				return cfg
			}(),
		},
		{
			name:    "missing jailer basedir",
			wantErr: true,
			config: func() config.Config {
				cfg := validCfg
				cfg.Jailer.BaseDir = ""
				return cfg
			}(),
		},
		{
			name:    "jailer uid_start too low",
			wantErr: true,
			config: func() config.Config {
				cfg := validCfg
				cfg.Jailer.UidStart = 1000
				return cfg
			}(),
		},
		{
			name:    "jailer gid_start too low",
			wantErr: true,
			config: func() config.Config {
				cfg := validCfg
				cfg.Jailer.GidStart = 1000
				return cfg
			}(),
		},
		{
			name:    "missing state.directory",
			wantErr: true,
			config: func() config.Config {
				cfg := validCfg
				cfg.State.Directory = ""
				return cfg
			}(),
		},
		{
			name:    "missing firecracker.sandbox_directory",
			wantErr: true,
			config: func() config.Config {
				cfg := validCfg
				cfg.Firecracker.SandboxDirectory = ""
				return cfg
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := tt.config.Validate()
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("Validate() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("Validate() succeeded unexpectedly")
			}
		})
	}
}
