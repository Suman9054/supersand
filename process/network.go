package process

import (
	"fmt"

	"github.com/vishvananda/netlink"
)

// SetupNetwork creates a veth pair, moves one end into the container's network
// namespace, and configures IP addresses + NAT on the host side.
func (s *Process) SetupNetwork() error {
	host := fmt.Sprintf("veth-host-%s", s.id.String())
	peername := fmt.Sprintf("eth-%s", s.id.String())

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

	if err := netlink.LinkSetNsPid(link, s.pid); err != nil {
		return fmt.Errorf("eror in namespace veth setup for %s error is %w ", s.id, err)
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
