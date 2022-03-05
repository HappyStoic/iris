package org

import (
	"context"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p-core/peer"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/pkg/errors"
	"time"

	"happystoic/p2pnetwork/pkg/config"
)

var log = logging.Logger("p2pnetwork")

type Org peer.ID

func (o *Org) String() string {
	return peer.ID(*o).String()
}

func (o *Org) Cid() (cid.Cid, error) {
	return cid.Decode(o.String())
}

type Book struct {
	trustworthy  []*Org
	signedPeers  map[*peer.ID][]*Org
	MySignatures map[*Org]string
	idht         *dht.IpfsDHT
}

func NewBook(cfg *config.OrgConfig, idht *dht.IpfsDHT) (*Book, error) {
	tr := make([]*Org, 0, len(cfg.Trustworthy))
	for _, rawTr := range cfg.Trustworthy {
		o, err := Decode(rawTr)
		if err != nil {
			return nil, err
		}
		tr = append(tr, o)
	}

	mySigs := make(map[*Org]string)
	for _, sig := range cfg.Signatures {
		o, err := Decode(sig.ID)
		if err != nil {
			return nil, err
		}
		mySigs[o] = sig.Signature
	}
	return &Book{
		trustworthy:  tr,
		signedPeers:  make(map[*peer.ID][]*Org, 0),
		MySignatures: mySigs,
		idht:         idht,
	}, nil
}

func (b *Book) RunUpdater(ctx context.Context) {
	updateBookTicker := time.NewTicker(time.Minute * 5) // TODO configurable durations
	go func() {
		for {
			select {
			case <-ctx.Done():
				updateBookTicker.Stop()
				log.Infof("stoping organisation updater of reason %s", ctx.Err())
				return
			case <-updateBookTicker.C:
				log.Debugf("started Organsation book update")

				// make completely new book which will replace the old one
				newRecord := make(map[*peer.ID][]*Org, 0)

				for _, tr := range b.trustworthy {
					c, err := tr.Cid()
					if err != nil {
						log.Errorf("error converting Org to cid.cid: %s", err)
						continue
					}
					res, err := b.idht.FindProviders(ctx, c)
					if err != nil {
						log.Errorf("error finding providers of %s: %s", c.String(), err)
						continue
					}

					// TODO verify the signature that they are not bullshitting us

					for _, adr := range res {
						if _, ok := newRecord[&adr.ID]; !ok {
							newRecord[&adr.ID] = make([]*Org, 0)
						}
						newRecord[&adr.ID] = append(newRecord[&adr.ID], tr)
						log.Debugf("found that peer %s is trusted by %s org", adr.ID.String(), tr.String())
					}
				}
				b.signedPeers = newRecord
				log.Debugf("ended Organisation book update")
			}
		}
	}()
	return
}

func (b *Book) AddPeerOrg(p *peer.ID, o *Org) {
	if _, ok := b.signedPeers[p]; !ok {
		b.signedPeers[p] = make([]*Org, 0)
	}
	b.signedPeers[p] = append(b.signedPeers[p], o)
}

func (b *Book) isTrustworthy(o *Org) bool {
	for _, trusted := range b.trustworthy {
		if *trusted == *o {
			return true
		}
	}
	return false
}

func (b *Book) StringOrgsOfPeer(p *peer.ID) []string {
	orgs := make([]string, 0, len(b.signedPeers[p]))
	for _, o := range b.signedPeers[p] {
		orgs = append(orgs, o.String())
	}
	return orgs
}

func Decode(s string) (*Org, error) {
	o, err := peer.Decode(s)
	if err != nil {
		return nil, errors.Errorf("error creating redis client: %s", err)
	}
	x := Org(o)
	return &x, nil
}

// TODO remove or convert to docs
// I trust some organisations, thus I need:
// * PubKey -> used as ID of the org and also as key to verify the signatures

// I am trusted by some orgs, thus I need:
// * PubKey    -> used as ID of the org
// * Signature -> used as a proof that I am trusted by this org

// Org book stores peer ID mapped to org ID (PubKey)
