package protocols

import (
	"happystoic/p2pnetwork/pkg/messaging/utils"
)

// p2p protocol definition
const recommendationRequest = "/recommendation-request/0.0.1"
const recommendationResponse = "/recommendation-response/0.0.1"

// RecommendationProtocol type
type RecommendationProtocol struct {
	*utils.ProtoUtils
}

func NewRecommendationProtocol(mu *utils.ProtoUtils) *RecommendationProtocol {
	rp := &RecommendationProtocol{mu}

	//rp.host.SetStreamHandler(recommendationRequest, rp.onP2PRecommendationRequest)
	//rp.redisClient.subscribeCallback(rp.redisClient.channels.Tl2nlRecommendation, rp.onRedisRecommendationRequest)
	return rp
}

//
//func (rp *RecommendationProtocol) onP2PRecommendationMessage(s network.Stream) {
//	recomRequest := &pb.RecommendationRequest{}
//
//	err := rp.deserializeMessageFromStream(s, recomRequest)
//	if err != nil {
//		log.Errorf("error deserilising recommendationRequest proto message from stream: %s", err)
//		return
//	}
//
//	// TODO
//
//	return
//}
//
//func (rp *RecommendationProtocol) onRedisRecommendationRequest(message string) {
//	req := RedisTl2NlRecommendationRequest{}
//	err := json.Unmarshal([]byte(message), &req)
//	if err != nil {
//		log.Errorf("error unmarshalling redis recommendation request: %s", err)
//		return
//	}
//	log.Debugf("received alarm message from TL asking %s peers about peer %s", req.ReceiverIds, req.TargetId)
//
//	msgMetaData, err := rp.NewProtoMetaData()
//	if err != nil {
//		log.Errorf("Error generating new proto metadata: %s", err)
//		return
//	}
//	protoMsg := &pb.RecommendationRequest{
//		Metadata: msgMetaData,
//		TargetId: req.TargetId,
//	}
//	signature, err := rp.SignProtoMessage(protoMsg)
//	if err != nil {
//		log.Errorf("Error generating signature for new recommendation request message: %s", err)
//		return
//	}
//	protoMsg.Metadata.Signature = signature
//
//	// send request to all desired peers
//	for _, pidString := range req.ReceiverIds {
//		pid, err := peer.Decode(pidString)
//		if err != nil {
//			log.Errorf("error decoding receiver peer's id '%s' received by NL: %s", pidString, err)
//			continue
//		}
//		err = rp.SendProtoMessage(pid, recommendationRequest, protoMsg)
//		if err != nil {
//			log.Errorf("Error sending recommendation request message to node %s: %s", pid, err)
//			continue
//		}
//		log.Debugf("successfully sent recommendation request to peer: %s", pid)
//	}
//}
//
//func (rp *RecommendationProtocol) onP2PRecommendationRequest(s network.Stream) {
//	log.Infof("received p2p recommendation request message")
//	req := &pb.RecommendationRequest{}
//
//	err := rp.deserializeMessageFromStream(s, req)
//	if err != nil {
//		log.Errorf("error deserilising p2p recommendation request from stream: %s", err)
//		return
//	}
//	valid, err := rp.AuthenticateMessage(req, req.Metadata)
//	if err != nil {
//		log.Errorf("Error authenticating p2p recommendation request message: %s", err)
//		return
//	}
//	if !valid {
//		log.Warnf("Alarm message failed authentication verification")
//		return
//	}
//
//	log.Debugf("received p2p recommendation req from %s asking about %s", s.Conn().RemotePeer(), req.TargetId)
//
//}
