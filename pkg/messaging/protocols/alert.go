package protocols

import (
	"encoding/json"
	"github.com/golang/protobuf/proto"
	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"

	"happystoic/p2pnetwork/pkg/messaging/pb"
	"happystoic/p2pnetwork/pkg/messaging/utils"
)

var log = logging.Logger("p2pnetwork")

// p2p protocol definition
const p2pAlertProtocol = "/alert/0.0.1"

// AlertProtocol type
type AlertProtocol struct {
	*utils.ProtoUtils
}

type RedisAlertRequestData struct {
	Payload interface{} `json:"payload"`
}

type RedisAlertResponseData struct {
	Sender  utils.PeerMetadata `json:"sender"`
	Payload interface{}        `json:"payload"`
}

func NewAlertProtocol(pu *utils.ProtoUtils) *AlertProtocol {
	ap := &AlertProtocol{pu}

	ap.Host.SetStreamHandler(p2pAlertProtocol, ap.onP2PAlertMessage)
	_ = ap.RedisClient.SubscribeCallback("tl2nl_alert", ap.onRedisAlertMessage)
	return ap
}

func (ap *AlertProtocol) onRedisAlertMessage(data []byte) {
	alertData := RedisAlertRequestData{}
	err := json.Unmarshal(data, &alertData)
	if err != nil {
		log.Errorf("error unmarshalling RedisAlertRequestData from redis: %s", err)
		return
	}
	log.Debug("received alert message from TL")
	ap.InitiateP2PAlert(alertData.Payload)
}

// InitiateP2PAlert initiates an alert message and sends it to all connected peers
func (ap *AlertProtocol) InitiateP2PAlert(payload interface{}) {
	alert, err := ap.createP2PAlert(payload)
	if err != nil {
		log.Error(err)
		return
	}

	// send alert to all connected peers
	for _, pid := range ap.ConnectedPeers() {
		log.Debugf("sending alert message to peer %s", pid)
		err = ap.SendProtoMessage(pid, p2pAlertProtocol, alert)
		if err != nil {
			log.Errorf("error sending alert message to node %s: %s", pid, err)
		}
	}
}

func (ap *AlertProtocol) createP2PAlert(payload interface{}) (*pb.Alert, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	msgMetaData, err := ap.NewProtoMetaData()
	if err != nil {
		return nil, errors.WithMessage(err, "error generating new proto metadata: ")
	}

	// store this msg as seen in case it comes back from another peer
	ap.NewMsgSeen(msgMetaData.Id, ap.Host.ID())

	protoMsg := &pb.Alert{
		Metadata: msgMetaData,
		Payload:  payloadBytes,
	}
	signature, err := ap.SignProtoMessage(protoMsg)
	if err != nil {
		return nil, errors.WithMessage(err, "error generating signature for new alert message: ")
	}
	protoMsg.Metadata.Signature = signature
	return protoMsg, err
}

func (ap *AlertProtocol) createRedisAlert(s network.Stream, payload []byte) (*RedisAlertResponseData, error) {
	var v interface{}
	err := json.Unmarshal(payload, &v)
	if err != nil {
		return nil, err
	}

	return &RedisAlertResponseData{
		Sender:  ap.MetadataOfPeer(s.Conn().RemotePeer()),
		Payload: v,
	}, nil
}

// onP2PAlertMessage receives an alert, sends it to local TL and forwards it further into the p2p network
func (ap *AlertProtocol) onP2PAlertMessage(s network.Stream) {
	log.Infof("received p2p alert message")
	alert := &pb.Alert{}

	err := ap.DeserializeMessageFromStream(s, alert, true)
	if err != nil {
		log.Errorf("error deserilising alert proto message from stream: %s", err)
		return
	}

	if ap.WasMsgSeen(alert.Metadata.Id) {
		log.Debugf("received already seen alert message, forwarded by %s", s.Conn().RemotePeer())
		return
	}
	ap.NewMsgSeen(alert.Metadata.Id, s.Conn().RemotePeer())

	err = ap.AuthenticateMessage(alert, alert.Metadata)
	if err != nil {
		log.Errorf("error authenticating alert message: %s", err)
		return
	}

	log.Debugf("Received Alert message authored by %s and forwarded by %s",
		alert.Metadata.OriginalSender.NodeId, s.Conn().RemotePeer())

	resp, err := ap.createRedisAlert(s, alert.Payload)
	if err != nil {
		log.Errorf("error creating alert message for redis: %s", err)
		return
	}

	err = ap.RedisClient.PublishMessage("nl2tl_alert", resp)
	if err != nil {
		log.Errorf("Error passing alert to trust layer: %s", err)
		return
	}

	// Forward alert msg to other connected peers
	ap.ForwardP2PAlert(alert, s.Conn().RemotePeer())

	log.Debugf("onP2PAlertMessage handler successfully ended")
}

func (ap *AlertProtocol) ForwardP2PAlert(protoMsg proto.Message, senderID peer.ID) {
	for _, pid := range ap.ConnectedPeers() {
		if pid == senderID {
			continue // do not send it back
		}

		log.Debugf("Forwarding alert message to peer %s", pid)
		err := ap.SendProtoMessage(pid, p2pAlertProtocol, protoMsg)
		if err != nil {
			log.Errorf("error forwarding alert message to node %s: %s", pid, err)
		}
	}
}
