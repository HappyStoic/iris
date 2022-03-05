package utils

import (
	"context"
	"happystoic/p2pnetwork/pkg/org"
	"io/ioutil"
	"time"

	"github.com/golang/protobuf/proto"
	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/pkg/errors"

	"happystoic/p2pnetwork/pkg/cryptotools"
	"happystoic/p2pnetwork/pkg/messaging/clients"
	"happystoic/p2pnetwork/pkg/messaging/pb"
)

var log = logging.Logger("p2pnetwork")

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
}

func NewProtoUtils(ck *cryptotools.CryptoKit, host host.Host, client *clients.RedisClient, ob *org.Book) *ProtoUtils {
	msgCache := newMessageCache()
	return &ProtoUtils{ck, msgCache, host, client, ob}
}

func (pu *ProtoUtils) ConnectedPeers() []peer.ID {
	return pu.Host.Network().Peers()
}

func (pu *ProtoUtils) SendProtoMessage(id peer.ID, protocol protocol.ID, data proto.Message) error {
	s, err := pu.Host.NewStream(context.Background(), id, protocol)
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

func (pu *ProtoUtils) NewProtoMetaData() (*pb.MetaData, error) {
	// Add protobufs bin data for message author public key
	// this is useful for authenticating  messages forwarded by a node authored by another node
	nodePubKey, err := crypto.MarshalPublicKey(pu.Host.Peerstore().PubKey(pu.Host.ID()))
	if err != nil {
		return nil, errors.New("Failed to get public key for sender from local node store.")
	}

	// get list of my orgs signatures TODO do this only once
	orgs := make([]*pb.Organisation, 0, len(pu.OrgBook.MySignatures))
	for o, sig := range pu.OrgBook.MySignatures {
		orgs = append(orgs, &pb.Organisation{
			OrgId:     o.String(),
			Signature: sig,
		})
	}

	sender := &pb.PeerIdentity{
		NodeId:        pu.Host.ID().String(),
		NodePubKey:    nodePubKey,
		Organisations: orgs,
	}
	metadata := &pb.MetaData{
		OriginalSender: sender,
		Timestamp:      time.Now().Unix(),
		Id:             cryptotools.GenerateUUID(),
	}

	return metadata, nil
}

func (pu *ProtoUtils) MetadataOfPeer(p peer.ID) PeerMetadata {
	return PeerMetadata{
		Id:            p.String(),
		Organisations: pu.OrgBook.StringOrgsOfPeer(&p),
	}
}

func (pu *ProtoUtils) DeserializeMessageFromStream(s network.Stream, msg proto.Message) error {
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
