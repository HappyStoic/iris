package dht

import (
	"context"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	ipfsDht "github.com/libp2p/go-libp2p-kad-dht"
)

type Dht struct {
	*ipfsDht.IpfsDHT

	ctx context.Context
}

func New(ctx context.Context, host host.Host) (*Dht, error) {
	iDht, err := ipfsDht.New(ctx, host, ipfsDht.Mode(ipfsDht.ModeServer))
	return &Dht{iDht, ctx}, err
}

func (d *Dht) StartProviding(cid cid.Cid) error {
	// TODO broadcast true or false? When true it returns error when no peers
	//      are connected...
	return d.Provide(d.ctx, cid, false)
}

func (d *Dht) GetProvidersOf(cid cid.Cid) ([]peer.AddrInfo, error) {
	return d.FindProviders(d.ctx, cid)
}
