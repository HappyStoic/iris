package connmgr

import (
	logging "github.com/ipfs/go-log/v2"
	libp2pConnMngr "github.com/libp2p/go-libp2p-connmgr"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"time"

	"happystoic/p2pnetwork/pkg/config"
	"happystoic/p2pnetwork/pkg/messaging/protocols"
	"happystoic/p2pnetwork/pkg/messaging/utils"
	"happystoic/p2pnetwork/pkg/reliability"
)

// TODO: when connected:
// 		* OK log
//		* OK tell it to TL
//		* exchange and verify signatures
//
// TODO: when disconnected:
//		* OK log
//		* OK tell it to TL
//
// TODO:
//		* OK use reliability as peer's tag
//
// TODO:
//		* goroutine for making new connections when there is a space?
//		or does libp2p does it somewhere and I should change it?
//		cuz my heuristics to choose peers will be different I guess
//
// TODO when trimming:
//		* maybe consider the final sum of reliability I will be left with. Always above some threshold?
//		* I should still have some connection with my org so metadata can get to me
//
// TODO:
//		* move initialisaion of connections here? that would make sense?

var log = logging.Logger("iris")

type RedisNotifyChange struct {
	Peers []utils.PeerMetadata `json:"peers"`
}

type Manager struct {
	*libp2pConnMngr.BasicConnMgr
	*utils.ProtoUtils

	connecter *Connecter

	orgSigProtocol *protocols.OrgSigProtocol
	cfg            *config.Connections

	ready bool
}

func NewManager(cfg *config.Connections) (*Manager, error) {
	cm, err := libp2pConnMngr.NewConnManager(
		cfg.Low,  // Lowwater
		cfg.High, // HighWater
		libp2pConnMngr.WithGracePeriod(time.Second*5),    // TODO remove
		libp2pConnMngr.WithSilencePeriod(time.Second*40), // TODO remove
	)
	if err != nil {
		return nil, err
	}
	m := &Manager{
		BasicConnMgr: cm,
		cfg:          cfg,
		ready:        false,
	}
	return m, nil
}

func (m *Manager) SetDeps(pu *utils.ProtoUtils, os *protocols.OrgSigProtocol, c *Connecter) {
	m.ProtoUtils = pu
	m.orgSigProtocol = os
	m.connecter = c
	m.ready = true
}

func (m *Manager) SetReliabilityTagCallback() reliability.Callback {
	return func(p peer.ID, r reliability.Reliability) {
		// TODO [?] is this conversion precise enough?
		m.TagPeer(p, "reliability", int(r*1e10))
	}
}

func (m *Manager) notifyTL() {
	conns := m.ConnectedPeers()
	peers := make([]utils.PeerMetadata, 0, len(conns))
	for _, p := range conns {
		peers = append(peers, m.MetadataOfPeer(p))
	}

	msg := RedisNotifyChange{Peers: peers}
	err := m.RedisClient.PublishMessage("nl2tl_peers_list", msg)
	if err != nil {
		log.Errorf("error sending TL peer connections: %s", err)
	}
}

func (m *Manager) connected(_ network.Network, c network.Conn) {
	log.Debugf("connected to '%s' via %s", c.RemotePeer(), c.RemoteMultiaddr())

	// exchange organisation signatures
	m.orgSigProtocol.AskForOrgSignatures(c.RemotePeer())

	// notify TL about a change
	m.notifyTL()
}

func (m *Manager) disconnected(_ network.Network, c network.Conn) {
	log.Debugf("disconnected from '%s'", c.RemotePeer())

	// notify TL about a change
	m.notifyTL()

	// check if we should try to add new connections cuz we have too few
	m.connecter.notify()
}

// Notifee returns a sink through which Notifiers can inform the Manager when
// events occur
func (m *Manager) Notifee() network.Notifiee {
	return (network.Notifiee)(m)
}

// Connected is called by notifiers to inform that a new connection has
// been established.
func (m *Manager) Connected(n network.Network, c network.Conn) {
	for !m.ready {
		time.Sleep(100 * time.Millisecond)
	}

	// call parent libp2p notifee
	m.BasicConnMgr.Notifee().Connected(n, c)

	// call my own procedure (in a goroutine not to block the network callee)
	go m.connected(n, c)
}

// Disconnected is called by notifiers to inform that an existing connection
// has been closed or terminated.
func (m *Manager) Disconnected(n network.Network, c network.Conn) {
	for !m.ready {
		time.Sleep(100 * time.Millisecond)
	}

	// call parent libp2p notifee
	m.BasicConnMgr.Notifee().Disconnected(n, c)

	// call my own procedure (in a goroutine not to block the network callee)
	go m.disconnected(n, c)
}

// Listen ListenClose OpenedStream and ClosedStream are not implemented
func (m *Manager) Listen(network.Network, ma.Multiaddr)         {}
func (m *Manager) ListenClose(network.Network, ma.Multiaddr)    {}
func (m *Manager) OpenedStream(network.Network, network.Stream) {}
func (m *Manager) ClosedStream(network.Network, network.Stream) {}
