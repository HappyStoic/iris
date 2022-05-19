package protocols

import (
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"

	"happystoic/p2pnetwork/pkg/messaging/pb"
	"happystoic/p2pnetwork/pkg/messaging/utils"
)

// p2p protocol definition
const p2pPeerQueryProtocol = "/peer-query/0.0.1"

const ResponsePeers = 5 // TODO maybe useful in yaml configuration?

// PeerQueryProtocol type
type PeerQueryProtocol struct {
	*utils.ProtoUtils
}

func NewPeerQuery(pu *utils.ProtoUtils) *PeerQueryProtocol {
	pq := &PeerQueryProtocol{pu}

	pq.Host.SetStreamHandler(p2pPeerQueryProtocol, pq.onPeerQueryRequest)
	return pq
}

func (pq *PeerQueryProtocol) onPeerQueryRequest(s network.Stream) {
	defer s.Close()
	log.Infof("received p2p peer query request")

	// prepare response metadata
	msgMetaData, err := pq.NewProtoMetaData()
	if err != nil {
		log.Errorf("error generating new proto metadata: %s", err)
		return
	}

	// get list of peers to return
	peers, err := pq.GetNPeersExpProbAllAllow(pq.ConnectedPeers(), ResponsePeers)
	if err != nil {
		log.Errorf("error getting n peers from connected peers %s", err)
		return
	}
	strPeers := make([]string, 0, len(peers))
	for _, p := range peers {
		strPeers = append(strPeers, p.String())
	}

	// construct the response and sign it
	protoMsg := &pb.PeerQueryResponse{
		Metadata: msgMetaData,
		PeerIds:  strPeers,
	}
	signature, err := pq.SignProtoMessage(protoMsg)
	if err != nil {
		log.Errorf("error generating signature for new alert message: %s", err)
	}
	protoMsg.Metadata.Signature = signature

	// send the response
	err = pq.WriteProtoMsg(protoMsg, s)
	if err != nil {
		log.Errorf("error responding with peer query response: %s", err)
		return
	}
	log.Debugf("successfully replied with peer query response")
}

func (pq *PeerQueryProtocol) SendPeerQuery(p peer.ID) ([]peer.ID, error) {
	log.Infof("sending p2p peer query")

	// open stream
	s, err := pq.OpenStream(p, p2pPeerQueryProtocol)
	if err != nil {
		return nil, err
	}

	// get response and validate signature
	response := &pb.PeerQueryResponse{}
	err = pq.DeserializeMessageFromStream(s, response, true)
	if err != nil {
		return nil, err
	}
	err = pq.AuthenticateMessage(response, response.Metadata)
	if err != nil {
		return nil, err
	}

	// parse the response from strings into peer IDs
	peers := make([]peer.ID, 0, len(response.PeerIds))
	for _, p := range response.PeerIds {
		pDecoded, err := peer.Decode(p)
		if err != nil {
			return nil, err
		}
		peers = append(peers, pDecoded)
	}

	log.Debugf("received %d peers from %s", len(peers), p)
	return peers, nil
}
