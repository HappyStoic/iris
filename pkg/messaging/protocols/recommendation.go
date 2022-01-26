package protocols

import (
	"context"
	"encoding/json"
	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"

	"happystoic/p2pnetwork/pkg/config"
	"happystoic/p2pnetwork/pkg/messaging/pb"
	"happystoic/p2pnetwork/pkg/messaging/utils"
)

// p2p protocol definition
const p2pRecomRequestProtocol = "/recommendation-request/0.0.1"
const p2pRecomResponseProtocol = "/recommendation-response/0.0.1"

type RedisTl2NlRecommendationRequest struct {
	ReceiverIds []string    `json:"receiver_ids"`
	Payload     interface{} `json:"payload"`
}

type RedisNl2TlRecommendationResponse []*Recommendation
type Recommendation struct {
	Sender  utils.PeerMetadata `json:"sender"`
	Payload interface{}        `json:"payload"`
}

// RecommendationProtocol type
type RecommendationProtocol struct {
	*utils.ProtoUtils

	ctx         context.Context
	respStorage *utils.RespStorageManager
	settings    *config.RecommendationSettings
}

func NewRecommendationProtocol(ctx context.Context,
	pu *utils.ProtoUtils,
	c *config.RecommendationSettings) *RecommendationProtocol {
	rp := &RecommendationProtocol{
		ProtoUtils: pu,
		ctx:        ctx,
		settings:   c,
	}
	rp.respStorage = utils.NewResponseStorage(rp.onAggregatedP2PResponses)

	_ = rp.RedisClient.SubscribeCallback("tl2nl_recommendation_request", rp.onRedisRecommendationRequest)
	_ = rp.RedisClient.SubscribeCallback("tl2nl_recommendation_response", rp.onRedisRecommendationResponse)
	rp.Host.SetStreamHandler(p2pRecomRequestProtocol, rp.onP2PRequest)
	rp.Host.SetStreamHandler(p2pRecomResponseProtocol, rp.onP2PResponse)
	return rp
}

func (rp *RecommendationProtocol) onP2PRequest(s network.Stream) {
	// TODO
	log.Debug("### onP2PRequest handler ended ###")
}

func (rp *RecommendationProtocol) onP2PResponse(s network.Stream) {
	log.Infof("received p2p recommendation response")
	recomResp := &pb.RecommendationResponse{}

	err := rp.DeserializeMessageFromStream(s, recomResp)
	if err != nil {
		log.Errorf("error deserilising p2P recommendation response from stream: %s", err)
		return
	}

	err = rp.AuthenticateMessage(recomResp, recomResp.Metadata)
	if err != nil {
		log.Errorf("error authenticating p2P recommendation response: %s", err)
		return
	}
	err = rp.respStorage.AddResponse(recomResp.RequestId, recomResp)
	if err != nil {
		log.Errorf("error adding response to respStorage with id %s: %s", recomResp.RequestId, err)
		return
	}
	log.Debug("p2p recommendation response was sucessfully put into response storage")
}

func (rp *RecommendationProtocol) onAggregatedP2PResponses(_ string, responses []proto.Message) {
	if len(responses) == 0 {
		log.Errorf("aggregated zero responses, not sending any response to Redis")
		return
	}
	recomRedisResp := make(RedisNl2TlRecommendationResponse, 0, len(responses))
	for i := range responses {
		resp := pb.RecommendationResponse{}
		bytes, _ := proto.Marshal(responses[i])
		_ = proto.Unmarshal(bytes, &resp)

		var v interface{}
		if err := json.Unmarshal(resp.Payload, &v); err != nil {
			log.Errorf("error deserialising received payload in p2p recommendation response: %s", err)
			continue
		}

		recomRedisResp = append(recomRedisResp, &Recommendation{
			Sender:  rp.MetadataOfPeer(resp.Metadata.Id),
			Payload: v,
		})
	}
	err := rp.RedisClient.PublishMessage("nl2tl_recommendation_response", recomRedisResp)
	if err != nil {
		log.Errorf("error publishing recommendation response to TL: %s", err)
		return
	}
	log.Debug("onAggregatedP2PResponses handler successfully ended")
}

func (rp *RecommendationProtocol) onRedisRecommendationRequest(data []byte) {
	req := RedisTl2NlRecommendationRequest{}
	err := json.Unmarshal(data, &req)
	if err != nil {
		log.Errorf("error unmarshalling RedisTl2NlRecommendationRequest from redis: %s", err)
		return
	}
	log.Debug("received recommendation request from TL")
	rp.initiateP2PRecomRequest(&req)
}

func (rp *RecommendationProtocol) initiateP2PRecomRequest(req *RedisTl2NlRecommendationRequest) {
	p2pRequest, err := rp.createP2PRecomRequest(req.Payload)
	if err != nil {
		log.Errorf("error creating p2p recommendation request: %s", err)
		return
	}
	if len(req.ReceiverIds) == 0 {
		log.Warn("no receivers specified for recommendation request")
		return
	}
	// start waiter, who will process all responses when they are aggregated or timeout elapses
	err = rp.respStorage.StartWaiting(rp.ctx, p2pRequest.Metadata.Id, len(req.ReceiverIds), rp.settings.Timeout)
	if err != nil {
		log.Errorf("error when starting to wait for recommendation responses: %s", err)
		return
	}

	// send recommendation request to receivers
	for _, rawPid := range req.ReceiverIds {
		pid, err := peer.Decode(rawPid)
		if err != nil {
			log.Errorf("error decoding peer id %s: %s", rawPid, err)
			continue
		}

		log.Debugf("sending recommendation request to peer %s", pid)
		err = rp.SendProtoMessage(pid, p2pRecomRequestProtocol, p2pRequest)
		if err != nil {
			log.Errorf("error sending alert message to node %s: %s", pid, err)
			continue
		}
	}
}

func (rp *RecommendationProtocol) createP2PRecomRequest(payload interface{}) (*pb.RecommendationRequest, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	msgMetaData, err := rp.NewProtoMetaData()
	if err != nil {
		return nil, errors.WithMessage(err, "error generating new proto metadata: ")
	}

	protoMsg := &pb.RecommendationRequest{
		Metadata: msgMetaData,
		Payload:  payloadBytes,
	}
	signature, err := rp.SignProtoMessage(protoMsg)
	if err != nil {
		return nil, errors.WithMessage(err, "error generating signature for new alert message: ")
	}
	protoMsg.Metadata.Signature = signature
	return protoMsg, err
}

func (rp *RecommendationProtocol) onRedisRecommendationResponse(data []byte) {
	// TODO
}
