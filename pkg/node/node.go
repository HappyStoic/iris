package node

import (
	"context"
	"fmt"

	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/routing"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	libp2pquic "github.com/libp2p/go-libp2p-quic-transport"
	"github.com/pkg/errors"

	"happystoic/p2pnetwork/pkg/config"
	"happystoic/p2pnetwork/pkg/cryptotools"
	"happystoic/p2pnetwork/pkg/messaging/clients"
	"happystoic/p2pnetwork/pkg/messaging/protocols"
	"happystoic/p2pnetwork/pkg/messaging/utils"
	"happystoic/p2pnetwork/pkg/peer-discovery"
)

var log = logging.Logger("p2pnetwork")

type Node struct {
	host.Host                // TODO is host really important here?
	*protocols.AlertProtocol // TODO are protocols really important here?
	*protocols.RecommendationProtocol

	conf *config.Config
	ctx  context.Context
}

func NewNode(conf *config.Config, ctx context.Context) (*Node, error) {
	key, err := cryptotools.GetPrivateKey(&conf.Identity)
	if err != nil {
		return nil, err
	}

	var idht *dht.IpfsDHT

	// Let's prevent our node from having too many
	// connections by attaching a connection manager.
	cm, err := connmgr.NewConnManager( // TODO create my own Connection manager
		100, // Lowwater
		400, // HighWater
	)
	if err != nil {
		return nil, err
	}

	p2phost, err := libp2p.New(
		// Use the keypair we generated
		libp2p.Identity(key),
		// Multiple listen addresses
		libp2p.ListenAddrStrings(
			fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic", conf.Server.Port), // a UDP endpoint for the QUIC transport
		),
		// support QUIC
		libp2p.Transport(libp2pquic.NewTransport),
		libp2p.ConnectionManager(cm),
		// Attempt to open ports using uPNP for NATed hosts.
		libp2p.NATPortMap(),
		// Let this host use the DHT to find other hosts
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			idht, err = dht.New(ctx, h)
			return idht, err
		}),
		// Let this host use relays and advertise itself on relays if
		// it finds it is behind NAT. Use libp2p.Relay(options...) to
		// enable active relays and more.
		libp2p.EnableAutoRelay(),
		// If you want to help other peers to figure out if they are behind
		// NATs, you can launch the server-side of AutoNAT too (AutoRelay
		// already runs the client)
		//
		// This service is highly rate-limited and should not cause any
		// performance issues.
		libp2p.EnableNATService(),
	)
	n := &Node{
		Host: p2phost,
		conf: conf,
		ctx:  ctx,
	}
	// create redis client
	redisClient, err := clients.NewRedisClient(&conf.Redis, ctx)
	if err != nil {
		return nil, errors.Errorf("error creating redis client: %s", err)
	}

	// setup all protocols
	cryptoKit := cryptotools.NewCryptoKit(p2phost)
	messageUtils := utils.NewMessageUtils(cryptoKit, p2phost, redisClient)
	n.AlertProtocol = protocols.NewAlertProtocol(messageUtils)
	n.RecommendationProtocol = protocols.NewRecommendationProtocol(messageUtils)

	return n, nil
}

func (n *Node) ConnectToInitPeers() {
	initPeers, err := peer_discovery.GetInitPeers(n.conf.PeerDiscovery)
	if len(initPeers) == 0 {
		log.Warnf("got 0 init peers, cannot make initial contact with the network")
	}

	for _, addr := range initPeers {
		err = n.Connect(n.ctx, *addr)
		if err != nil {
			log.Errorf("error connecting to peer %s: %s\n", addr.ID, err)
			continue
		}
		log.Infof("successfuly connected to peer: %s", addr.ID)
	}
}

func (n *Node) Start(doSomething bool) {
	if doSomething {
		log.Info("Doing something (not really)")
		//n.InitiateP2PAlert([]byte("prdel!!! zachovejte paniku, cusasaan Milan"))
	}
	<-make(chan struct{})
}
