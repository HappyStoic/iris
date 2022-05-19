package connmgr

import (
	"context"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"happystoic/p2pnetwork/pkg/messaging/protocols"
	myutils "happystoic/p2pnetwork/pkg/utils"
	"time"

	"happystoic/p2pnetwork/pkg/config"
	"happystoic/p2pnetwork/pkg/messaging/utils"
)

const QueryPeers = 5

type Connecter struct {
	*utils.ProtoUtils

	lastRun        time.Time
	fewConnections chan struct{}

	peerQueryProto *protocols.PeerQueryProtocol

	cfg *config.Connections
}

func NewConnecter(cfg *config.Connections, pu *utils.ProtoUtils) *Connecter {
	return &Connecter{
		ProtoUtils:     pu,
		cfg:            cfg,
		peerQueryProto: protocols.NewPeerQuery(pu),
		fewConnections: make(chan struct{}, 1),
	}
}

func (c *Connecter) notify() {
	select {
	case c.fewConnections <- struct{}{}:
		// successful
	default:
		// notification already pending
	}
}

func (c *Connecter) Start(ctx context.Context) {
	go func() {
		period := c.cfg.ReconnectInterval
		log.Infof("starting peer connecter with period %s", period)

		ticker := time.NewTicker(period)
		defer ticker.Stop()

		select {
		case <-ctx.Done():
			log.Infof("peer connecter stopped - %s", ctx.Err())
		case <-ticker.C:
			c.process()
		case <-c.fewConnections:
			c.process()
			ticker.Reset(period)
		}
	}()
}

func (c *Connecter) process() {
	if c.NumberOfConnections() >= c.cfg.Low {
		// we still are above threshold, no reconnecting
		return
	}

	fiveMinsAgo := time.Now().Add(-5 * time.Minute)
	if !c.lastRun.Before(fiveMinsAgo) {
		log.Debugf("reconnecter tried to reconnect too soon after previous reconnection")
		return
	}

	c.doUpdate()
}

func (c *Connecter) updatePeerStore() {
	// get known peers in O(1) search set
	knownPeersList := c.Host.Peerstore().PeersWithAddrs()
	knownPeersSet := make(map[peer.ID]struct{}, len(knownPeersList))
	for _, p := range knownPeersList {
		knownPeersSet[p] = struct{}{}
	}

	// sample random peers from connected peers
	receivers, err := c.GetNPeersExpProbAllAllow(c.ConnectedPeers(), QueryPeers)
	if err != nil {
		log.Errorf("error getting n peers from connected peers %s", err)
		return
	}
	// ask some of my connected peers to get new peers
	for _, receiver := range receivers {
		peers, err := c.peerQueryProto.SendPeerQuery(receiver)
		if err != nil {
			log.Errorf("error querying peer '%s' about other peers: %s", receiver, err)
			continue
		}

		for _, p := range peers {
			if _, exists := knownPeersSet[p]; exists {
				log.Debugf("we already know peer %s", p)
				continue //
			}

			res, err := c.Dht.FindPeer(context.Background(), p)
			if err != nil {
				log.Errorf("error finding peer %s in dht: %s", p, err)
			}
			c.Host.Peerstore().AddAddrs(p, res.Addrs, peerstore.TempAddrTTL)
			knownPeersSet[p] = struct{}{}
			log.Debugf("received peer %s", p)
		}
	}
}

func (c *Connecter) tryConnect(p peer.ID) error {
	peerInfo := peer.AddrInfo{
		ID:    p,
		Addrs: c.Host.Peerstore().Addrs(p),
	}

	return c.Host.Connect(context.Background(), peerInfo)
}

func (c *Connecter) doUpdate() {
	log.Debugf("trying to make new peer connections, current %d, limits (%d|%d|%d)", c.NumberOfConnections(), c.cfg.Low, c.cfg.Medium, c.cfg.High)

	// ask about some new peers
	c.updatePeerStore()

	// ** connect to some peers **
	// TODO: consider reliability when choosing new peers

	// firstly, get set of connected peers with O(1) search
	connectedPeers := c.ConnectedPeers()
	connectedPeersSet := make(map[peer.ID]struct{}, len(connectedPeers))
	for _, p := range connectedPeers {
		connectedPeersSet[p] = struct{}{}
	}
	connectedPeersSet[c.Host.ID()] = struct{}{} // prevent from connecting to self

	// how many spots can I fill?
	freeSlots := c.cfg.Medium - c.NumberOfConnections()

	// save 2/3 of free spots for org members (or atleast 1)
	myOrgs := c.OrgBook.MyOrgs
	freeSlotsPerOrg := 1
	if len(myOrgs) != 0 {
		freeSlotsPerOrg = myutils.Max((2*freeSlots)/(3*len(myOrgs)), 1)
	}

	// first try to fill spaces with peers from my organizations
	for _, o := range myOrgs {
		count := 0
		for _, p := range c.OrgBook.ClaimedMembers[*o] {
			if _, exists := connectedPeersSet[*p]; exists {
				// skipping, we are already connected to this peer
				continue
			}

			err := c.tryConnect(*p)
			if err != nil {
				log.Errorf("error trying to connect to %s: %s", p, err)
				continue
			}

			count++
			if count >= freeSlotsPerOrg {
				break
			}
		}
	}

	// use remaining spaces for whatever peer
	freeSlots = c.cfg.Medium - c.NumberOfConnections()
	count := 0
	for _, p := range c.Host.Peerstore().PeersWithAddrs() {
		if _, exists := connectedPeersSet[p]; exists {
			// skipping, we are already connected to this peer
			continue
		}

		err := c.tryConnect(p)
		if err != nil {
			log.Errorf("error trying to connect to %s: %s", p, err)
			continue
		}

		count++
		if count >= freeSlots {
			break
		}
	}

	// update last run timestamp
	c.lastRun = time.Now()
}
