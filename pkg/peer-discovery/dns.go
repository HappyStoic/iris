package peer_discovery

import (
	"github.com/libp2p/go-libp2p-core/peer"
)

var DefaultDnsDomains = []string{
	"p2p.stratosphere.info",
}

func GetInitDnsPeers() ([]*peer.AddrInfo, error) {
	multiaddrs := make([]*peer.AddrInfo, 0)

	// TODO

	return multiaddrs, nil
}
