package process

import (
	"fmt"

	"github.com/suman9054/supersand/healper"
	"github.com/vishvananda/netlink"
)

// SetupNetwork creates a veth pair, moves one end into the container's network
// namespace, and configures IP addresses + NAT on the host side.
func (s *Process) SetupNetwork() error {
	pid := s.cmd.Process.Pid
	id := healper.GenrateNetworkid()
	host := fmt.Sprintf("veth-host-%s", id)
	peername := fmt.Sprintf("eth-%s", id)

	veth := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{
			Name: host,
		},
		PeerName: peername,
	}

	if err := netlink.LinkAdd(veth); err != nil {
		return fmt.Errorf("error in veth setup %w", err)
	}

	link, erro := netlink.LinkByName(peername)

	if erro != nil {
		return fmt.Errorf("error in conect veth by name %w", erro)
	}

	if err := netlink.LinkSetNsPid(link, pid); err != nil {
		return fmt.Errorf("eror in namespace veth setup for %s error is %w ", id, err)
	}
	hostl, errh := netlink.LinkByName(host)
	if errh != nil {
		return fmt.Errorf("error in seting up vet on host %w", errh)
	}

	netlink.LinkSetUp(hostl)
	adder, erroradder := netlink.ParseAddr("10.0.0.1/24")
	if erroradder != nil {
		return fmt.Errorf("eeror in netlink setup %w", erroradder)
	}
	netlink.AddrAdd(hostl, adder)
	s.mu.Lock()
	s.veth = host
	s.peername = peername
	s.mu.Unlock()
	return nil
}
