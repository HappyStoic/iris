package org

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"

	"happystoic/p2pnetwork/pkg/config"
	ldht "happystoic/p2pnetwork/pkg/dht"
	"happystoic/p2pnetwork/pkg/messaging/pb"
)

type Book struct {
	updateEvery time.Duration
	dht         *ldht.Dht

	// Trustworthy defines the organizations that this peer trusts
	Trustworthy []*Org

	// MySignaturesProto is a list with org IDs and signatures of this peer
	MySignaturesProto []*pb.Organisation

	// MyOrgs is a list with orgs that this peer is a member of
	MyOrgs []*Org

	// ClaimedMembers stores potential members of organisations. This
	// information is periodically updated from a DHT and might be true or not.
	// To see verified peers, see variable VerifiedSignatures
	ClaimedMembers map[Org][]*peer.ID

	// VerifiedSignatures stores peers' orgs with successfully verified
	// signatures
	VerifiedSignatures map[peer.ID][]*Org
}

func NewBook(cfg *config.OrgConfig, dht *ldht.Dht, me peer.ID) (*Book, error) {
	trusted := make([]*Org, 0, len(cfg.Trustworthy))
	for _, rawTr := range cfg.Trustworthy {
		o, err := Decode(rawTr)
		if err != nil {
			return nil, err
		}
		trusted = append(trusted, o)
	}

	myProtoSigs := make([]*pb.Organisation, 0, len(cfg.MySignatures))
	myOrgs := make([]*Org, 0, len(cfg.MySignatures))
	for _, sig := range cfg.MySignatures {
		o, err := Decode(sig.ID)
		if err != nil {
			return nil, err
		}

		// let's also verify the signature. Other peers could lower our
		// reliability if we introduce ourselves with incorrect signatures
		ok, err := o.VerifyPeer(me, sig.Signature)
		if err != nil {
			return nil, errors.Errorf("error validating sigature %s: %s", sig.Signature, err)
		}
		if !ok {
			return nil, errors.Errorf("invalid signature '%s' of org '%s'", sig.Signature, sig.ID)
		}

		myProtoSigs = append(myProtoSigs, &pb.Organisation{
			OrgId:     sig.ID,
			Signature: sig.Signature,
		})
		myOrgs = append(myOrgs, o)
	}
	b := &Book{
		updateEvery:        cfg.DhtUpdatePeriod,
		dht:                dht,
		Trustworthy:        trusted,
		MySignaturesProto:  myProtoSigs,
		MyOrgs:             myOrgs,
		ClaimedMembers:     make(map[Org][]*peer.ID),
		VerifiedSignatures: make(map[peer.ID][]*Org),
	}

	return b, nil
}

func (b *Book) update() {
	// make completely new book which will replace the old one
	newClaim := make(map[Org][]*peer.ID)

	for _, tr := range b.Trustworthy {
		peers := make([]*peer.ID, 0)

		// convert trusted org to cid and get providers from DHT
		c, err := tr.Cid()
		if err != nil {
			log.Errorf("error converting org to cid: %s", err)
			continue
		}
		res, err := b.dht.GetProvidersOf(c)
		if err != nil {
			log.Errorf("error getting providers of %s: %s", c.String(), err)
			continue
		}

		for _, adr := range res {
			if adr.ID == b.dht.PeerID() {
				// this is local me (local peer), do not process it
				continue
			}
			peers = append(peers, &adr.ID)
			log.Debugf("DHT claims peer %s is member of %s org", adr.ID.String(), tr.String())
		}

		newClaim[*tr] = peers
	}
	b.ClaimedMembers = newClaim
}

func (b *Book) RunUpdater(ctx context.Context) {
	if len(b.Trustworthy) == 0 {
		log.Debugf("Not starting org updater - no trusted orgs")
		return
	}

	go func() {
		// sleep to until a peer connects to the network
		log.Debugf("running org book updater for the 1st time")
		time.Sleep(time.Second * 5)
		b.update()

		ticker := time.NewTicker(b.updateEvery)
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				log.Infof("stoping org updater: %s", ctx.Err())
				return
			case <-ticker.C:
				log.Debugf("starting org book update from DHT")
				b.update()
				log.Debugf("ended Organisation book update")
			}
		}
	}()
	return
}

// HasPeerRight returns true if peer p has verified signature from at least one
// organisation in orgs argument
func (b *Book) HasPeerRight(p peer.ID, orgs []*Org) bool {
	// OPTIM: some data structure? this has square complexity
	peerOrgs := b.VerifiedSignatures[p]
	for _, po := range peerOrgs {
		for _, o := range orgs {
			if *po == *o {
				return true
			}
		}
	}
	return false
}

func (b *Book) IsTrustworthy(o *Org) bool {
	for _, trusted := range b.Trustworthy {
		if *trusted == *o {
			return true
		}
	}
	return false
}

// StringOrgsOfPeer returns peer's organisations that we have verified
func (b *Book) StringOrgsOfPeer(p peer.ID) []string {
	orgs := make([]string, 0, len(b.VerifiedSignatures[p]))
	for _, o := range b.VerifiedSignatures[p] {
		orgs = append(orgs, o.String())
	}
	return orgs
}

func (b *Book) AddVerifiedSig(p peer.ID, o *Org) {
	if _, exists := b.VerifiedSignatures[p]; !exists {
		b.VerifiedSignatures[p] = make([]*Org, 0)
	}

	b.VerifiedSignatures[p] = append(b.VerifiedSignatures[p], o)
}
