package p2p

import (
	"context"
	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/pkg/errors"
	"happystoic/p2pnetwork/pkg/cryptotools"
	"happystoic/p2pnetwork/pkg/messaging/p2p/pb"
	"io/ioutil"
	"time"
)

type ProtoUtils struct {
	*cryptotools.CryptoKit
	*SeenMessagesCache

	host host.Host
}

func NewProtoUtils(ck *cryptotools.CryptoKit, host host.Host) *ProtoUtils {
	msgCache := newMessageCache()
	return &ProtoUtils{ck, msgCache, host}
}

func (pt *ProtoUtils) ConnectedPeers() []peer.ID {
	return pt.host.Network().Peers()
}

func (pt *ProtoUtils) SendProtoMessage(id peer.ID, protocol protocol.ID, data proto.Message) error {
	s, err := pt.host.NewStream(context.Background(), id, protocol)
	if err != nil {
		return err
	}
	defer s.Close()

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

func (pt *ProtoUtils) NewProtoMetaData() (*pb.MetaData, error) {
	// Add protobufs bin data for message author public key
	// this is useful for authenticating  messages forwarded by a node authored by another node
	nodePubKey, err := crypto.MarshalPublicKey(pt.host.Peerstore().PubKey(pt.host.ID()))
	if err != nil {
		return nil, errors.New("Failed to get public key for sender from local node store.")
	}

	sender := &pb.PeerIdentity{
		NodeId:        pt.host.ID().String(),
		NodePubKey:    nodePubKey,
		Organisations: nil,
	}
	metadata := &pb.MetaData{
		OriginalSender: sender,
		Timestamp:      time.Now().Unix(),
		Id:             cryptotools.GenerateUUID(),
	}

	return metadata, nil
}

func (pt *ProtoUtils) deserializeMessageFromStream(s network.Stream, msg proto.Message) error {
	// read received bytes
	buf, err := ioutil.ReadAll(s)
	if err != nil {
		_ = s.Reset()
		return err
	}
	_ = s.Close()

	// unmarshal it
	err = proto.Unmarshal(buf, msg)
	if err != nil {
		return err
	}
	return nil
}

type SeenMessagesCache struct {
	cache map[string]bool
	ttl   time.Duration
}

func newMessageCache() *SeenMessagesCache {
	return &SeenMessagesCache{
		cache: make(map[string]bool),
		ttl:   0,
	}
}

func (c *SeenMessagesCache) wasMsgSeen(s string) bool {
	_, ok := c.cache[s]
	return ok
}

func (c *SeenMessagesCache) newMsgSeen(s string) {
	c.cache[s] = true
}
