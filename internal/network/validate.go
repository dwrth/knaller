package network

import (
	"fmt"

	"github.com/dwrth/knaller/internal/state"
)

func validate(sandbox state.Sandbox) error {
	checks := []struct {
		ok  bool
		msg string
	}{
		{sandbox.ID != "", "id is required"},
		{sandbox.Namespace != "", "namespace is required"},
		{sandbox.HostVeth != "", "host_veth is required"},
		{sandbox.NSVeth != "", "ns_veth is required"},
		{sandbox.TAP != "", "tap is required"},
		{sandbox.GuestSubnet != "", "guest_subnet is required"},
		{sandbox.GatewayIP != "", "gateway_ip is required"},
		{sandbox.TransitHostIP != "", "transit_host_ip is required"},
		{sandbox.TransitNSIP != "", "transit_ns_ip is required"},
		{sandbox.TransitSubnet != "", "transit_subnet is required"},
		{sandbox.UID > 0, "uid is required"},
		{sandbox.GID > 0, "gid is required"},
	}
	for _, c := range checks {
		if !c.ok {
			return fmt.Errorf("network: %s", c.msg)
		}
	}
	return nil
}
