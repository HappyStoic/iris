package cryptotools

import (
	"github.com/golang/protobuf/proto"
	libp2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"

	"happystoic/p2pnetwork/pkg/messaging/pb"
)

type CryptoKit struct {
	host host.Host
}

func NewCryptoKit(host host.Host) *CryptoKit {
	return &CryptoKit{host: host}
}

// AuthenticateMessage authenticates p2p message.
// message: a protobufs go data object
// metadata: common p2p metadata
func (ck *CryptoKit) AuthenticateMessage(message proto.Message, metadata *pb.MetaData) error {
	// store a temp ref to signature and remove it from message we can verify the message
	sign := metadata.Signature
	metadata.Signature = nil

	// marshall data without the signature to protobufs3 binary format
	bin, err := proto.Marshal(message)
	if err != nil {
		return err
	}

	// restore sig in message data (for possible future use)
	metadata.Signature = sign

	// restore node id binary format from base58 encoded node id data
	peerId, err := peer.Decode(metadata.OriginalSender.NodeId)
	if err != nil {
		return err
	}

	// extract node id from the provided public key
	key, err := libp2pcrypto.UnmarshalPublicKey(metadata.OriginalSender.NodePubKey)
	if err != nil {
		return err
	}
	idFromKey, err := peer.IDFromPublicKey(key)
	if err != nil {
		return err
	}

	// verify that message author node id matches the provided node public key
	if idFromKey != peerId {
		return nil
	}

	valid, err := key.Verify(bin, sign) // TODO maybe use []byte(sign)
	if err != nil {
		return err
	}
	if !valid {
		return errors.New("signature does not match")
	}
	return nil
}

// SignProtoMessage signs an outgoing p2p message payload
func (ck *CryptoKit) SignProtoMessage(message proto.Message) ([]byte, error) {
	data, err := proto.Marshal(message)
	if err != nil {
		return nil, err
	}
	privateKey := ck.host.Peerstore().PrivKey(ck.host.ID())
	return privateKey.Sign(data)
}
