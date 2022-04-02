package node

import (
	"context"
	"fmt"
	"os"

	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/routing"
	libp2pquic "github.com/libp2p/go-libp2p-quic-transport"
	"github.com/pkg/errors"

	"happystoic/p2pnetwork/pkg/config"
	connmgr "happystoic/p2pnetwork/pkg/connections"
	"happystoic/p2pnetwork/pkg/cryptotools"
	ldht "happystoic/p2pnetwork/pkg/dht"
	"happystoic/p2pnetwork/pkg/files"
	"happystoic/p2pnetwork/pkg/messaging/clients"
	"happystoic/p2pnetwork/pkg/messaging/protocols"
	"happystoic/p2pnetwork/pkg/messaging/utils"
	"happystoic/p2pnetwork/pkg/org"
	"happystoic/p2pnetwork/pkg/peer-discovery"
	"happystoic/p2pnetwork/pkg/reliability"
)

var log = logging.Logger("p2pnetwork")

type Node struct {
	host.Host                // TODO is host really important here?
	*protocols.AlertProtocol // TODO are protocols really important here?
	*protocols.RecommendationProtocol
	*protocols.IntelligenceProtocol
	*protocols.FileShareProtocol
	*protocols.OrgSigProtocol

	dht     *ldht.Dht
	relBook *reliability.Book
	orgBook *org.Book
	conf    *config.Config
	ctx     context.Context
}

func NewNode(conf *config.Config, ctx context.Context) (*Node, error) {
	key, err := cryptotools.GetPrivateKey(&conf.Identity)
	if err != nil {
		return nil, err
	}

	var dht *ldht.Dht

	cm, err := connmgr.NewManager(&conf.Connections)
	if err != nil {
		return nil, err
	}

	//metricsTest := metrics.NewBandwidthCounter() // TODO use metrics somehow?
	//go func() {
	//	for {
	//		time.Sleep(10 * time.Second)
	//		stats := metricsTest.GetBandwidthTotals()
	//		fmt.Printf("total in: %d\n", stats.TotalIn)
	//		fmt.Printf("total out: %d\n", stats.TotalOut)
	//		fmt.Printf("rate in: %f\n", stats.RateIn)
	//		fmt.Printf("rate out: %f\n", stats.RateOut)
	//	}
	//}()

	p2phost, err := libp2p.New(
		// Use the keypair we generated
		libp2p.Identity(key),
		// Multiple listen addresses
		libp2p.ListenAddrStrings(
			fmt.Sprintf("/ip4/%s/udp/%d/quic", conf.Server.Host, conf.Server.Port), // a UDP endpoint for the QUIC transport
		),
		// support QUIC
		libp2p.Transport(libp2pquic.NewTransport),
		libp2p.ConnectionManager(cm),
		// Attempt to open ports using uPNP for NATed hosts.
		libp2p.NATPortMap(),
		// Let this host use the DHT to find other hosts
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			dht, err = ldht.New(ctx, h)
			return dht, err
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
		//libp2p.BandwidthReporter(metricsTest),
	)
	if err != nil {
		return nil, err
	}

	// create redis client
	redisClient, err := clients.NewRedisClient(&conf.Redis, ctx)
	if err != nil {
		return nil, errors.Errorf("error creating redis client: %s", err)
	}

	// setup books
	relBook := reliability.NewBook()
	fileBook := files.NewFileBook()
	orgBook, err := org.NewBook(&conf.Organisations, dht, p2phost.ID())
	if err != nil {
		return nil, errors.Errorf("error creating org book: %s", err)
	}
	orgBook.RunUpdater(ctx)

	n := &Node{
		Host:    p2phost,
		dht:     dht,
		relBook: relBook,
		orgBook: orgBook,
		conf:    conf,
		ctx:     ctx,
	}

	// setup kits
	cryptoKit := cryptotools.NewCryptoKit(p2phost)
	protoUtils := utils.NewProtoUtils(cryptoKit, p2phost, redisClient, orgBook, relBook, dht)

	// setup all protocols
	n.OrgSigProtocol = protocols.NewOrgSigProtocol(protoUtils)
	n.AlertProtocol = protocols.NewAlertProtocol(protoUtils)
	n.RecommendationProtocol = protocols.NewRecommendationProtocol(ctx, protoUtils, &conf.ProtocolSettings.Recommendation)
	n.IntelligenceProtocol = protocols.NewIntelligenceProtocol(ctx, protoUtils, &conf.ProtocolSettings.Intelligence)
	n.FileShareProtocol = protocols.NewFileShareProtocol(ctx, protoUtils, fileBook, dht, &conf.ProtocolSettings.FileShare)
	_ = protocols.NewReliabilityReceiver(protoUtils, relBook)

	connecter := connmgr.NewConnecter(&conf.Connections, protoUtils)
	connecter.Start(ctx)

	// inject missing dependencies
	cm.SetDeps(protoUtils, n.OrgSigProtocol, connecter)

	// setup callbacks
	relBook.SubscribeForChange(cm.SetReliabilityTagCallback())

	return n, nil
}

func (n *Node) connectToInitPeers() {
	// TODO use this as example how to find init peers of my orgs
	//go func() {
	//	for {
	//		//key := "prdelOrganizace"
	//		//valBytes, err := idht.GetValue(context.Background(), key)
	//		//if err != nil {
	//		//	log.Errorf("error in GetValue: %s", err)
	//		//} else {
	//		//	log.Debugf("value for '%s' is '%s'", key, valBytes)
	//		//}
	//		c, err := cid.Decode("bafzbeigai3eoy2ccc7ybwjfz5r3rdxqrinwi4rwytly24tdbh6yk7zslrm")
	//		if err != nil {
	//			log.Errorf("%s", err)
	//		}
	//
	//		//err = idht.Provide(context.Background(), c, false)
	//		//if err != nil {
	//		//	log.Errorf("%s", err)
	//		//}
	//		//break
	//
	//		providers, err := idht.FindProviders(context.Background(), c)
	//		if err != nil {
	//			log.Errorf("error in FindProviders: %s", err)
	//		} else {
	//			log.Debugf("providers for '%s'", c.String())
	//			for _, p := range providers {
	//				log.Debugf("\t%s", p.String())
	//			}
	//		}
	//		time.Sleep(time.Second * 5)
	//	}
	//}()
	//p2phost.Peerstore().PubKey()

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
	}
}

// advertiseMyOrgs tells the network that I am member of my organizations
func (n *Node) advertiseMyOrgs(_ context.Context) {
	for _, o := range n.orgBook.MyOrgs {
		c, err := o.Cid()
		if err != nil {
			log.Errorf("error converting org to cid: %s", err)
			continue
		}
		err = n.dht.StartProviding(c)
		if err != nil {
			log.Errorf("error putting myself as member of org %s to a DHT: %s ", o.String(), err)
		}
	}
}

func (n *Node) Start(ctx context.Context) {
	// connect node to the network
	n.connectToInitPeers()

	// tell the network which organisations I am member of
	n.advertiseMyOrgs(ctx)

	// tmp
	if len(os.Getenv("DO_SOMETHING")) > 0 {
		log.Info("Doing something (not really)")
		//n.InitiateP2PAlert([]byte("prdel!!! zachovejte paniku, cusasaan Milan"))
	}

	// block running
	<-make(chan struct{})
}
