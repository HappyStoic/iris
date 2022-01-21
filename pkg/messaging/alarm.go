package messaging

import (
	"github.com/golang/protobuf/proto"
	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"happystoic/p2pnetwork/pkg/messaging/pb"
)

var log = logging.Logger("p2pnetwork")

// protocol definition
const alarmMessage = "/alarm/0.0.1"

// AlarmProtocol type
type AlarmProtocol struct {
	*MessageUtils
}

func NewAlarmProtocol(mu *MessageUtils) *AlarmProtocol {
	ap := &AlarmProtocol{mu}

	ap.host.SetStreamHandler(alarmMessage, ap.onP2PAlarmMessage)
	ap.redisClient.subscribeCallback(ap.redisClient.channels.Tl2nlAlarmChannel, ap.onRedisAlarmMessage)
	return ap
}

func (ap *AlarmProtocol) onRedisAlarmMessage(message string) {
	// TODO deserialize first somehow the message?
	log.Debugf("received alarm message from TL: %s", message)
	ap.InitiateP2PAlarm(message)
}

// onP2PAlarmMessage receives an alarm, sends it to local TL and forwards it further into the p2p network
func (ap *AlarmProtocol) onP2PAlarmMessage(s network.Stream) {
	alarm := &pb.Alarm{}

	err := ap.deserializeMessageFromStream(s, alarm)
	if err != nil {
		log.Errorf("error deserilising alarm proto message from stream: %s", err)
	}

	if ap.wasMsgSeen(alarm.Metadata.Id) {
		log.Debugf("received already seen alarm message, forwarded by %s", s.Conn().RemotePeer())
		return
	}
	ap.newMsgSeen(alarm.Metadata.Id)

	log.Infof("Received Alarm message authored by %s and forwarded by %s", alarm.Metadata.OriginalSender.NodeId,
		s.Conn().RemotePeer())
	log.Debugf("Alarm message contains: %s", alarm.Message)

	valid, err := ap.AuthenticateMessage(alarm, alarm.Metadata)
	if err != nil {
		log.Errorf("Error authenticating alarm message: %s", err)
		return
	}
	if !valid {
		log.Warnf("Alarm message failed authentication verification")
		return
	}

	// TODO: serialize message somehow to trust layer?
	err = ap.redisClient.publishMessage(ap.redisClient.channels.Nl2tlAlarmChannel, alarm.Message)
	if err != nil {
		log.Errorf("Error passing alarm to trust layer: %s", err)
	}

	// Forward alarm msg to other connected peers
	ap.ForwardP2PAlarm(alarm, s.Conn().RemotePeer())

	log.Debugf("onP2PAlarmMessage handler successfully ended")
}

// InitiateP2PAlarm initiates an alarm message and sends it to all connected peers
func (ap *AlarmProtocol) InitiateP2PAlarm(message string) {
	msgMetaData, err := ap.NewProtoMetaData()
	if err != nil {
		log.Errorf("Error generating new proto metadata: %s", err)
		return
	}
	protoMsg := &pb.Alarm{
		Metadata: msgMetaData,
		Message:  message,
	}
	signature, err := ap.SignProtoMessage(protoMsg)
	if err != nil {
		log.Errorf("Error generating signature for new alarm message: %s", err)
		return
	}
	protoMsg.Metadata.Signature = signature

	// store this msg as seen in case it comes back from another peer
	ap.newMsgSeen(protoMsg.Metadata.Id)

	// send alarm to all connected peers
	for _, pid := range ap.ConnectedPeers() {
		err = ap.SendProtoMessage(pid, alarmMessage, protoMsg)
		if err != nil {
			log.Errorf("Error sending alarm message to node %s: %s", pid, err)
		}
	}
}
func (ap *AlarmProtocol) ForwardP2PAlarm(protoMsg proto.Message, senderID peer.ID) {
	for _, pid := range ap.ConnectedPeers() {
		if pid == senderID {
			continue // do not send it back
		}

		log.Debugf("Forwarding alarm message to peer %s", pid)
		err := ap.SendProtoMessage(pid, alarmMessage, protoMsg)
		if err != nil {
			log.Errorf("Error forwarding alarm message to node %s: %s", pid, err)
		}
	}
}
