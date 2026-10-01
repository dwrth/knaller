package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/dwrth/knaller/internal/config"
	"github.com/dwrth/knaller/internal/network"
	"github.com/dwrth/knaller/internal/state"
)

func runNetwork(args []string) int {
	if len(args) == 0 {
		printNetworkUsage(os.Stderr)
		return 2
	}

	switch args[0] {
	case "ensure-host":
		return runNetworkEnsureHost(args[1:])
	case "teardown-host":
		return runNetworkTeardownHost(args[1:])
	case "setup":
		return runNetworkSandbox(args[1:], "setup", network.Setup)
	case "teardown":
		return runNetworkSandbox(args[1:], "teardown", network.Teardown)
	case "status":
		return runNetworkStatus(args[1:])
	case "help", "-h", "--help":
		printNetworkUsage(os.Stdout)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", args[0])
		printNetworkUsage(os.Stderr)
		return 2
	}
}

func runNetworkEnsureHost(args []string) int {
	cfg, code := loadNetworkConfig("network ensure-host", args)
	if code != 0 || cfg == nil {
		return code
	}
	if err := network.EnsureHost(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "network ensure-host: %v\n", err)
		return 1
	}
	return 0
}

func runNetworkTeardownHost(args []string) int {
	cfg, code := loadNetworkConfig("network teardown-host", args)
	if code != 0 || cfg == nil {
		return code
	}
	if err := network.TeardownHost(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "network teardown-host: %v\n", err)
		return 1
	}
	return 0
}

func runNetworkSandbox(args []string, name string, fn func(*config.Config, state.Sandbox) error) int {
	cfg, id, code := loadNetworkConfigAndID("network "+name, args)
	if code != 0 || cfg == nil {
		return code
	}
	sandbox, err := state.New(cfg.State.Directory).Load(id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "network %s: load sandbox %s: %v\n", name, id, err)
		return 1
	}
	if err := fn(cfg, sandbox); err != nil {
		fmt.Fprintf(os.Stderr, "network %s: %v\n", name, err)
		return 1
	}
	return 0
}

func runNetworkStatus(args []string) int {
	cfg, id, code := loadNetworkConfigAndID("network status", args)
	if code != 0 || cfg == nil {
		return code
	}
	sandbox, err := state.New(cfg.State.Directory).Load(id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "network status: load sandbox %s: %v\n", id, err)
		return 1
	}
	report, err := network.Status(sandbox)
	if err != nil {
		fmt.Fprintf(os.Stderr, "network status: %v\n", err)
		return 1
	}
	fmt.Fprintf(os.Stdout, "sandbox %s\n", sandbox.ID)
	fmt.Fprintf(os.Stdout, "  netns       %s\n", present(report.NetNS))
	fmt.Fprintf(os.Stdout, "  host_veth   %s\n", present(report.HostVeth))
	fmt.Fprintf(os.Stdout, "  guest_route %s\n", present(report.GuestRoute))
	return 0
}

func present(ok bool) string {
	if ok {
		return "present"
	}
	return "absent"
}

func loadNetworkConfig(name string, args []string) (*config.Config, int) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: knaller %s [flags]\nFlags:\n", name)
		fs.PrintDefaults()
	}
	configPath := fs.String("config", defaultConfigPath, "path to knaller config file")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil, 0
		}
		return nil, 2
	}
	var cfg config.Config
	if err := cfg.Load(*configPath); err != nil {
		fmt.Fprintf(os.Stderr, "failed to load configuration: %v\n", err)
		return nil, 1
	}
	return &cfg, 0
}

func loadNetworkConfigAndID(name string, args []string) (*config.Config, string, int) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: knaller %s [flags] <sandbox-id>\nFlags:\n", name)
		fs.PrintDefaults()
	}
	configPath := fs.String("config", defaultConfigPath, "path to knaller config file")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil, "", 0
		}
		return nil, "", 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "sandbox id is required")
		fs.Usage()
		return nil, "", 2
	}
	var cfg config.Config
	if err := cfg.Load(*configPath); err != nil {
		fmt.Fprintf(os.Stderr, "failed to load configuration: %v\n", err)
		return nil, "", 1
	}
	return &cfg, fs.Arg(0), 0
}

func printNetworkUsage(w io.Writer) {
	fmt.Fprint(w, `Usage: knaller network <command> [flags]
Commands:
  ensure-host     Apply host forwarding and nft isolation/NAT
  teardown-host   Remove Knaller nft tables
  setup <id>      Create per-sandbox dataplane from state
  teardown <id>   Remove per-sandbox dataplane
  status <id>     Show whether netns/veth/route are present
`)
}
