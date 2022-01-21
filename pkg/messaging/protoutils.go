package messaging

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
	"happystoic/p2pnetwork/pkg/messaging/pb"
	"io/ioutil"
	"time"
)

type MessageUtils struct {
	*cryptotools.CryptoKit
	*SeenMessagesCache

	host        host.Host
	redisClient *RedisClient
}

func NewProtoUtils(ck *cryptotools.CryptoKit, host host.Host, client *RedisClient) *MessageUtils {
	msgCache := newMessageCache()
	return &MessageUtils{ck, msgCache, host, client}
}

func (mu *MessageUtils) ConnectedPeers() []peer.ID {
	return mu.host.Network().Peers()
}

func (mu *MessageUtils) SendProtoMessage(id peer.ID, protocol protocol.ID, data proto.Message) error {
	s, err := mu.host.NewStream(context.Background(), id, protocol)
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

func (mu *MessageUtils) NewProtoMetaData() (*pb.MetaData, error) {
	// Add protobufs bin data for message author public key
	// this is useful for authenticating  messages forwarded by a node authored by another node
	nodePubKey, err := crypto.MarshalPublicKey(mu.host.Peerstore().PubKey(mu.host.ID()))
	if err != nil {
		return nil, errors.New("Failed to get public key for sender from local node store.")
	}

	sender := &pb.PeerIdentity{
		NodeId:        mu.host.ID().String(),
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

func (mu *MessageUtils) deserializeMessageFromStream(s network.Stream, msg proto.Message) error {
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
