package utils

import (
	"context"
	"happystoic/p2pnetwork/pkg/dht"
	"io/ioutil"
	"sort"
	"time"

	"github.com/golang/protobuf/proto"
	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	wr "github.com/mroth/weightedrand"
	"github.com/pkg/errors"

	"happystoic/p2pnetwork/pkg/cryptotools"
	"happystoic/p2pnetwork/pkg/messaging/clients"
	"happystoic/p2pnetwork/pkg/messaging/pb"
	"happystoic/p2pnetwork/pkg/org"
	"happystoic/p2pnetwork/pkg/reliability"
)

var log = logging.Logger("iris")

type PeerMetadata struct {
	Id            string   `json:"id"`
	Organisations []string `json:"organisations"`
}

type ProtoUtils struct {
	*cryptotools.CryptoKit
	*SeenMessagesCache

	Host        host.Host
	RedisClient *clients.RedisClient
	OrgBook     *org.Book
	RelBook     *reliability.Book
	Dht         *dht.Dht
}

func NewProtoUtils(ck *cryptotools.CryptoKit, host host.Host, client *clients.RedisClient, ob *org.Book, rb *reliability.Book, dht *dht.Dht) *ProtoUtils {
	msgCache := newMessageCache()
	return &ProtoUtils{ck, msgCache, host, client, ob, rb, dht}
}

func (pu *ProtoUtils) ConnectedPeers() []peer.ID {
	return pu.Host.Network().Peers()
}

func (pu *ProtoUtils) NumberOfConnections() int {
	return len(pu.Host.Network().Peers())
}

func (pu *ProtoUtils) OpenStream(id peer.ID, protocol protocol.ID) (network.Stream, error) {
	return pu.Host.NewStream(context.Background(), id, protocol)
}

func (pu *ProtoUtils) SendProtoMessage(id peer.ID, protocol protocol.ID, data proto.Message) error {
	s, err := pu.OpenStream(id, protocol)
	if err != nil {
		return err
	}
	defer s.Close()

	return pu.WriteProtoMsg(data, s)
}

func (pu *ProtoUtils) WriteProtoMsg(data proto.Message, s network.Stream) error {
	bytes, err := proto.Marshal(data)
	if err != nil {
		return err
	}

	_, err = s.Write(bytes)
	if err != nil {
		_ = s.Reset()
		return err
	}
	return nil
}

func (pu *ProtoUtils) InitiateStream(id peer.ID, protocol protocol.ID, data proto.Message) (network.Stream, error) {
	var s network.Stream
	s, err := pu.OpenStream(id, protocol)
	if err != nil {
		return s, err
	}

	bytes, err := proto.Marshal(data)
	if err != nil {
		_ = s.Close()
		return s, err
	}

	_, err = s.Write(bytes)
	if err != nil {
		_ = s.Reset()
		_ = s.Close()
		return s, err
	}
	return s, nil
}

// NewProtoMetaData creates new protobuf metadata
func (pu *ProtoUtils) NewProtoMetaData() (*pb.MetaData, error) {
	// Add protobufs bin data for message author public key
	// this is useful for authenticating  messages forwarded by a node authored by another node
	nodePubKey, err := crypto.MarshalPublicKey(pu.Host.Peerstore().PubKey(pu.Host.ID()))
	if err != nil {
		return nil, errors.New("Failed to get public key for sender from local node store.")
	}

	sender := &pb.PeerIdentity{
		NodeId:     pu.Host.ID().String(),
		NodePubKey: nodePubKey,
	}
	metadata := &pb.MetaData{
		OriginalSender: sender,
		Timestamp:      time.Now().Unix(),
		Id:             cryptotools.GenerateUUID(),
	}

	return metadata, nil
}

// MetadataOfPeer creates given peer's Metadata structure
func (pu *ProtoUtils) MetadataOfPeer(p peer.ID) PeerMetadata {
	return PeerMetadata{
		Id:            p.String(),
		Organisations: pu.OrgBook.StringOrgsOfPeer(p),
	}
}

// DeserializeMessageFromStream deserializes protobuf message from stream
func (pu *ProtoUtils) DeserializeMessageFromStream(s network.Stream, msg proto.Message, closeStream bool) error {
	// read received bytes
	buf, err := ioutil.ReadAll(s)
	if err != nil {
		_ = s.Reset()
		return err
	}
	if closeStream {
		_ = s.Close()
	}

	// unmarshal it
	err = proto.Unmarshal(buf, msg)
	if err != nil {
		return err
	}
	return nil
}

// ReportPeer sends a report to TL via Redis
func (pu *ProtoUtils) ReportPeer(p peer.ID, reason string) error {
	log.Debugf("reporting to TL peer '%s' with reason '%s'", p, reason)
	type RedisPeerReport struct {
		Peer   PeerMetadata `json:"peer"`
		Reason string       `json:"reason"`
	}
	report := &RedisPeerReport{
		Peer:   pu.MetadataOfPeer(p),
		Reason: reason,
	}
	return pu.RedisClient.PublishMessage("nl2tl_peer_report", report)
}

// GetNPeersExpProbAllAllow selects n peers from given list.
// Each peer has a weighted exponential probability based on its reliability
func (pu *ProtoUtils) GetNPeersExpProbAllAllow(from []peer.ID, n int) ([]peer.ID, error) {
	var noRights []*org.Org
	var blacklist map[peer.ID]struct{}
	return pu.GetNPeersExpProb(from, n, noRights, blacklist)
}

// GetNPeersExpProb selects n peers from given list.
// Each peer has a weighted exponential probability based on its reliability
// Function does not select peers provided in blacklist argument
// Function does not select peers without authorization. Rights are provided in rights list
func (pu *ProtoUtils) GetNPeersExpProb(from []peer.ID, n int, rights []*org.Org, blacklist map[peer.ID]struct{}) ([]peer.ID, error) {
	selected := make(map[peer.ID]struct{})
	for i := 0; i < n; i++ {
		candidates := make([]wr.Choice, 0)
		for _, p := range from {
			// don't take already selected peers
			if _, exists := selected[p]; exists {
				continue
			}
			// don't take peers who are in the blacklist
			if _, exists := blacklist[p]; exists {
				continue
			}
			// check authorization
			if len(rights) != 0 && !pu.OrgBook.HasPeerRight(p, rights) {
				continue
			}
			candidates = append(candidates, wr.Choice{
				Item:   p,
				Weight: pu.RelBook.ExpTransformedPeerRel(p),
			})
		}
		if len(candidates) == 0 {
			// no more viable candidates exist, we can break
			break
		}
		chooser, err := wr.NewChooser(candidates...)
		if err != nil {
			return nil, err
		}
		selected[chooser.Pick().(peer.ID)] = struct{}{}
	}
	// take those unique peers and put them in a slice
	peers := make([]peer.ID, 0, n)
	for p := range selected {
		peers = append(peers, p)
	}
	return peers, nil
}

// ReliabilitySort sorts the addrs in descending based on reliability of given
// peers
func (pu *ProtoUtils) ReliabilitySort(addrs []peer.AddrInfo) {
	sort.Slice(addrs, func(i, j int) bool {
		iRel := float64(pu.RelBook.PeerRel(addrs[i].ID))
		jRel := float64(pu.RelBook.PeerRel(addrs[j].ID))
		return iRel > jRel
	})
}
