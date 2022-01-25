package peer_discovery

import (
	"strings"

	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"

	"happystoic/p2pnetwork/pkg/config"
)

var log = logging.Logger("p2pnetwork")

func GetInitPeers(pd config.PeerDiscovery) ([]*peer.AddrInfo, error) {
	multiAddrs := make([]*peer.AddrInfo, 0)

	// firstly, use static list found in config file
	for _, s := range pd.ListOfMultiAddresses {
		ai, err := addrInfoFromConnectionString(s)
		if err != nil {
			log.Errorf("error creating AddrInfo from config line %s: %s", s, err)
			continue
		}

		multiAddrs = append(multiAddrs, ai)
	}

	if pd.UseRedisCache {
		cacheResult, err := GetInitCachePeers()
		if err != nil {
			return nil, err
		}
		multiAddrs = append(multiAddrs, cacheResult...)
	}

	if pd.UseDns {
		dnsResult, err := GetInitDnsPeers()
		if err != nil {
			return nil, err
		}
		multiAddrs = append(multiAddrs, dnsResult...)
	}
	return multiAddrs, nil
}

// Connection string is multiaddr and peer ID separated by space
// For now I cannot use any standard multiaddr parsing, cuz quic protocol does not have defined argument (which libp2p
// needs to also provide peer's ID, so I do it myself like this. TODO: try to find standard approach
func addrInfoFromConnectionString(s string) (*peer.AddrInfo, error) {
	split := strings.Split(s, " ")
	if len(split) != 2 {
		return nil, errors.New("wrong format of connection string")
	}

	ma, err := multiaddr.NewMultiaddr(split[0])
	if err != nil {
		return nil, err
	}

	id, err := peer.Decode(split[1])
	if err != nil {
		return nil, err
	}

	ai := &peer.AddrInfo{
		ID:    id,
		Addrs: []multiaddr.Multiaddr{ma},
	}
	return ai, nil
}
