package protocols

import (
	"context"
	"encoding/json"
	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/pkg/errors"
	"happystoic/p2pnetwork/pkg/config"
	"happystoic/p2pnetwork/pkg/messaging/pb"
	"happystoic/p2pnetwork/pkg/messaging/utils"
	"time"
)

// p2p protocol definition
const p2pIntelRequestProtocol = "/intelligence-request/0.0.1"
const p2pIntelResponseProtocol = "/intelligence-response/0.0.1"

type RedisTl2NlIntelRequest struct {
	Payload interface{} `json:"payload"`
}

type RedisNl2TlIntelRequest struct {
	RequestId string             `json:"request_id"`
	Sender    utils.PeerMetadata `json:"sender"`
	Payload   interface{}        `json:"payload"`
}

// IntelligenceProtocol type
type IntelligenceProtocol struct {
	*utils.ProtoUtils

	ctx         context.Context
	respStorage *utils.RespStorageManager
	settings    *config.IntelligenceSettings
}

func NewIntelligenceProtocol(ctx context.Context,
	pu *utils.ProtoUtils,
	c *config.IntelligenceSettings) *IntelligenceProtocol {
	ip := &IntelligenceProtocol{
		ProtoUtils: pu,
		ctx:        ctx,
		settings:   c,
	}
	ip.respStorage = utils.NewResponseStorage(ip.onAggregatedP2PResponses)
	//
	_ = ip.RedisClient.SubscribeCallback("tl2nl_intelligence_request", ip.onRedisIntelligenceRequest)
	_ = ip.RedisClient.SubscribeCallback("nl2tl_intelligence_response", ip.onRedisIntelligenceResponse)
	ip.Host.SetStreamHandler(p2pIntelRequestProtocol, ip.onP2PRequest)
	ip.Host.SetStreamHandler(p2pIntelResponseProtocol, ip.onP2PResponse)
	return ip
}

func (ip *IntelligenceProtocol) onRedisIntelligenceRequest(data []byte) {
	req := RedisTl2NlIntelRequest{}
	err := json.Unmarshal(data, &req)
	if err != nil {
		log.Errorf("error unmarshalling RedisTl2NlIntelRequest from redis: %s", err)
		return
	}
	log.Debug("received intelligence request from TL")
	ip.initiateP2PIntelligenceRequest(&req)
}

func (ip *IntelligenceProtocol) initiateP2PIntelligenceRequest(req *RedisTl2NlIntelRequest) {
	p2pRequest, err := ip.createP2PIntelRequest(req.Payload)
	if err != nil {
		log.Errorf("error creating p2p intelligence request: %s", err)
		return
	}
	// TODO change how many and the way I choose recipient peers, currently I send the request to everyone.
	pids := ip.ConnectedPeers()

	// start waiter, who will process all responses when they are aggregated or timeout elapses
	reqId := p2pRequest.IntelligenceRequest.Metadata.Id
	err = ip.respStorage.StartWaiting(ip.ctx, reqId, len(pids), ip.settings.RootTimeout)
	if err != nil {
		log.Errorf("error when starting to wait for intelligence responses: %s", err)
		return
	}

	// send intelligence request to receivers
	for _, pid := range pids {
		log.Debugf("sending intelligence request to peer %s", pid)
		err = ip.SendProtoMessage(pid, p2pIntelRequestProtocol, p2pRequest)
		if err != nil {
			log.Errorf("error sending intelligence request to node %s: %s", pid, err)
			continue
		}
	}
}

func (ip *IntelligenceProtocol) createP2PIntelRequest(payload interface{}) (*pb.IntelligenceReqEnvelope, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	msgMetaData, err := ip.NewProtoMetaData()
	if err != nil {
		return nil, errors.WithMessage(err, "error generating new proto metadata: ")
	}
	ip.SeenMessagesCache.NewMsgSeen(msgMetaData.Id, ip.Host.ID())

	protoMsg := &pb.IntelligenceRequest{
		Metadata: msgMetaData,
		Payload:  payloadBytes,
	}
	signature, err := ip.SignProtoMessage(protoMsg)
	if err != nil {
		return nil, errors.WithMessage(err, "error generating signature for new p2p intelligence request: ")
	}
	protoMsg.Metadata.Signature = signature

	envelope := &pb.IntelligenceReqEnvelope{
		Ttl:                 ip.settings.Ttl,
		ParentTimeout:       ip.settings.RootTimeout.String(),
		IntelligenceRequest: protoMsg,
	}
	return envelope, nil
}

func (ip *IntelligenceProtocol) onAggregatedP2PResponses(_ string, responses []proto.Message) {
	// TODO code this
}

func (ip *IntelligenceProtocol) onP2PResponse(s network.Stream) {
	// TODO code this
}

func (ip *IntelligenceProtocol) onRedisIntelligenceResponse(data []byte) {
	// TODO code this (don't forget to encrypt the response with receiver's public key)
}

func (ip *IntelligenceProtocol) onP2PRequest(s network.Stream) {
	log.Infof("received p2p intelligence request")
	intelReqEnvelope := &pb.IntelligenceReqEnvelope{}
	err := ip.DeserializeMessageFromStream(s, intelReqEnvelope)
	if err != nil {
		log.Errorf("error deserilising p2p intelligence request from stream: %s", err)
		return
	}

	intelReq := intelReqEnvelope.IntelligenceRequest
	err = ip.AuthenticateMessage(intelReq, intelReq.Metadata)
	if err != nil {
		log.Errorf("error authenticating p2P intelligence request: %s", err)
		return
	}
	if ip.SeenMessagesCache.WasMsgSeen(intelReq.Metadata.Id) {
		log.Debugf("received already seen intelligence request with id %s. No processing", intelReq.Metadata.Id)
		// TODO reply telling with response that this request won't be processed so the asker is not waiting
		return
	}
	ip.SeenMessagesCache.NewMsgSeen(intelReq.Metadata.Id, s.Conn().RemotePeer())

	err = ip.processP2PRequest(intelReqEnvelope)
	if err != nil {
		log.Errorf("error processing p2p intelligence request: %s", err)
		return
	}
	log.Debug("handler onP2PRequest successfully ended")
}

func (ip *IntelligenceProtocol) processP2PRequest(e *pb.IntelligenceReqEnvelope) error {
	var v interface{}
	if err := json.Unmarshal(e.IntelligenceRequest.Payload, &v); err != nil {
		return err
	}

	// send request to redis
	requestToRedis := RedisNl2TlIntelRequest{
		RequestId: e.IntelligenceRequest.Metadata.Id,
		Sender:    ip.MetadataOfPeer(e.IntelligenceRequest.Metadata.OriginalSender.NodeId),
		Payload:   v,
	}
	err := ip.RedisClient.PublishMessage("nl2tl_intelligence_request", requestToRedis)
	if err != nil {
		return err
	}
	waitForResponses := 1 // for now just wait for response from redis

	// update envelope (timeout and ttl) and send further into the network
	if e.Ttl != 0 {
		updatedMsg, err := ip.updateEnvelope(e)
		if err != nil {
			log.Errorf("error updating envelope in intelligence request: %s", err)
		} else {
			waitForResponses += ip.forwardP2PRequest(updatedMsg)
		}
	}

	// start waiter, who will process all responses when they are aggregated or timeout elapses
	waitTimeout, err := time.ParseDuration(e.ParentTimeout)
	if err != nil {
		log.Errorf("using default timeout, error parsing waiting timeout after update: %s", err)
		waitTimeout = time.Second * 3
	}

	reqId := e.IntelligenceRequest.Metadata.Id
	err = ip.respStorage.StartWaiting(ip.ctx, reqId, waitForResponses, waitTimeout)
	return err
}

func (ip *IntelligenceProtocol) updateEnvelope(e *pb.IntelligenceReqEnvelope) (*pb.IntelligenceReqEnvelope, error) {
	// TODO change the way how ttl and timeout is decreased/updated and use max timeout settings field
	// decrease timeout if possible
	parentTimeout, err := time.ParseDuration(e.ParentTimeout)
	if err != nil {
		return nil, err
	}
	secondLess := parentTimeout - time.Second
	if secondLess > 0 {
		e.ParentTimeout = secondLess.String()
	} else {
		e.ParentTimeout = time.Second.String()
	}
	// decrease TTL if possible
	ttlLess := e.Ttl - 1
	if ttlLess > ip.settings.MaxTtl {
		e.Ttl = ip.settings.MaxTtl
	} else {
		e.Ttl = ttlLess
	}
	return e, nil
}

func (ip *IntelligenceProtocol) forwardP2PRequest(intelReqEnv *pb.IntelligenceReqEnvelope) int {
	// TODO change how many and the way I choose recipient peers, currently I send the request to everyone.
	pids := ip.ConnectedPeers()

	// send intelligence request to receivers
	sent := 0
	for _, pid := range pids {
		log.Debugf("sending intelligence request to peer %s", pid)
		err := ip.SendProtoMessage(pid, p2pIntelRequestProtocol, intelReqEnv)
		if err != nil {
			log.Errorf("error sending intelligence request to node %s: %s", pid, err)
			continue
		}
		sent++
	}
	return sent
}
