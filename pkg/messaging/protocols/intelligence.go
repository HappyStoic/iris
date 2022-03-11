package protocols

import (
	"context"
	"encoding/json"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"

	"happystoic/p2pnetwork/pkg/config"
	"happystoic/p2pnetwork/pkg/messaging/pb"
	"happystoic/p2pnetwork/pkg/messaging/utils"
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

type RedisTl2NlIntelResponse struct {
	RequestId string      `json:"request_id"`
	Payload   interface{} `json:"payload"`
}

type RedisNl2TlIntelligenceResponse []*IntelligenceResponse
type IntelligenceResponse struct {
	Sender  utils.PeerMetadata `json:"sender"`
	Payload interface{}        `json:"payload"`
}

// IntelligenceProtocol type
type IntelligenceProtocol struct {
	*utils.ProtoUtils

	ctx                  context.Context
	respStorage          *utils.ResponseAggregator
	settings             *config.IntelligenceSettings
	cacheRequestToSender map[string]peer.ID
}

func NewIntelligenceProtocol(ctx context.Context,
	pu *utils.ProtoUtils,
	c *config.IntelligenceSettings) *IntelligenceProtocol {
	ip := &IntelligenceProtocol{
		ProtoUtils:           pu,
		ctx:                  ctx,
		settings:             c,
		cacheRequestToSender: make(map[string]peer.ID),
	}
	ip.respStorage = utils.NewResponseAggregator(ip.onAggregatedP2PResponses)
	//
	_ = ip.RedisClient.SubscribeCallback("tl2nl_intelligence_request", ip.onRedisIntelligenceRequest)
	_ = ip.RedisClient.SubscribeCallback("tl2nl_intelligence_response", ip.onRedisIntelligenceResponse)
	ip.Host.SetStreamHandler(p2pIntelRequestProtocol, ip.onP2PRequest)
	ip.Host.SetStreamHandler(p2pIntelResponseProtocol, ip.onP2PResponse)
	return ip
}

// ###################################################
// ### TL sends through Redis Intelligence request ###
// ###################################################
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
	ip.SeenMessagesCache.NewMsgSeen(p2pRequest.IntelligenceRequest.Metadata.Id, ip.Host.ID())

	// TODO change how many and the way I choose recipient peers, currently I send the request to everyone.
	pids := ip.ConnectedPeers()

	// start waiter, who will process all responses when they are aggregated or timeout elapses
	reqId := p2pRequest.IntelligenceRequest.Metadata.Id
	err = ip.respStorage.StartWaiting(ip.ctx, reqId, nil, len(pids), ip.settings.RootTimeout)
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

// ####################################################################
// ### We have all intelligence responses (from other peers and TL) ###
// ####################################################################
// Two scenarios possible:
// 		a) This peer is the original requester. In that case I need to decrypt all the single responses and forward
// 		   it to TL through Redis
//
//      b) This peer is just a middleman, in that case just wrap all the aggregated responses and forward this p2P
//         response to the peer that asked
//
func (ip *IntelligenceProtocol) onAggregatedP2PResponses(requestId string, responses []proto.Message, meta *utils.StorageMetadata) {
	log.Debugf("all intelligence responses were aggregated, starting to collect them")

	listOfSingleResponses := make([][]byte, 0, len(responses))
	for i := range responses {
		resp := pb.IntelligenceResponse{}
		bytes, _ := proto.Marshal(responses[i])
		_ = proto.Unmarshal(bytes, &resp)

		if !resp.Processed {
			log.Debugf("p2p responses from %s was flagged as no processed, skipping", resp.Metadata.Id)
			continue
		}

		listOfSingleResponses = append(listOfSingleResponses, resp.Responses...)
	}

	if len(listOfSingleResponses) == 0 {
		log.Errorf("aggregaed zero responses, ending handler")
		return
	}

	// I should send collected responses back to the sender
	if meta != nil && meta.ResponsesReceiver != ip.Host.ID() {
		resp, err := ip.createP2PIntelligenceResponse(requestId, listOfSingleResponses)
		if err != nil {
			log.Errorf("error creating p2p intelligence response: %s", err)
			return
		}
		err = ip.SendProtoMessage(meta.ResponsesReceiver, p2pIntelResponseProtocol, resp)
		if err != nil {
			log.Errorf("error sending p2p intelligence response: %s", err)
			return
		}

		// I am actually the one who initiated the request, I need to send response back to my TL through Redis
	} else {
		err := ip.sendIntelligenceResponseToRedis(listOfSingleResponses)
		if err != nil {
			log.Errorf("error sending intelligence response to TL through Redis: %s", err)
			return
		}
	}
	log.Debugf("successfully ended onAggregatedP2PResponses")
}

func (ip *IntelligenceProtocol) sendIntelligenceResponseToRedis(responses [][]byte) error {
	log.Debugf("sending intelligence data back to TL through redis")

	//responses might need to be decrypted
	recomRedisResp := make(RedisNl2TlIntelligenceResponse, 0, len(responses))
	for i := range responses {
		//TODO decrypt the messages here first (so far it's only marshalled)
		// ...

		// decode the response
		singleResp := &pb.SingleEntityResponse{}
		err := proto.Unmarshal(responses[i], singleResp)
		if err != nil {
			log.Errorf("error unmarshalling singleEntityResponse: %s", err)
			continue
		}

		// verify signature
		err = ip.AuthenticateMessage(singleResp, singleResp.Metadata)
		if err != nil {
			log.Errorf("error authenticating singleEntityResponse: %s", err)
			continue
		}

		var v interface{}
		err = json.Unmarshal(singleResp.Payload, &v)
		if err != nil {
			log.Errorf("error unmarshalling data from one peer before sending them to TL: %s", err)
			continue
		}

		senderPeerId, err := peer.Decode(singleResp.Metadata.OriginalSender.NodeId)
		if err != nil {
			log.Errorf("error decoding peer ID: %s", err)
		}

		recomRedisResp = append(recomRedisResp, &IntelligenceResponse{
			Sender:  ip.MetadataOfPeer(senderPeerId),
			Payload: v,
		})
	}
	err := ip.RedisClient.PublishMessage("nl2tl_intelligence_response", recomRedisResp)
	if err != nil {
		return errors.WithMessage(err, "error publishing intelligence response to TL: ")
	}
	return nil
}

func (ip *IntelligenceProtocol) createP2PIntelligenceResponse(requestId string, responses [][]byte) (*pb.IntelligenceResponse, error) {
	msgMetaData, err := ip.NewProtoMetaData()
	if err != nil {
		return nil, errors.WithMessage(err, "error generating new proto metadata: ")
	}

	resp := &pb.IntelligenceResponse{
		Metadata:  msgMetaData,
		RequestId: requestId,
		Processed: true,
		Responses: responses,
	}

	signature, err := ip.SignProtoMessage(resp)
	if err != nil {
		return nil, errors.WithMessage(err, "error generating signature for new p2p intelligence response: ")
	}
	resp.Metadata.Signature = signature
	return resp, nil
}

// ########################################################
// ### Some peer sends as intelligence response         ###
// ########################################################
func (ip *IntelligenceProtocol) onP2PResponse(s network.Stream) {
	log.Infof("received p2p intelligence response")
	intelResp := &pb.IntelligenceResponse{}

	err := ip.DeserializeMessageFromStream(s, intelResp, true)
	if err != nil {
		log.Errorf("error deserilising p2P intelligence response from stream: %s", err)
		return
	}
	err = ip.AuthenticateMessage(intelResp, intelResp.Metadata)
	if err != nil {
		log.Errorf("error authenticating p2P intelligence response: %s", err)
		return
	}
	err = ip.respStorage.AddResponse(intelResp.RequestId, intelResp)
	if err != nil {
		log.Errorf("error adding intel response to respStorage with id '%s': '%s'", intelResp.RequestId, err)
		return
	}
	log.Debug("p2p intelligence response was successfully put into response storage")
}

// ########################################################
// ### TL sends us intelligence response through redis  ###
// ########################################################
func (ip *IntelligenceProtocol) onRedisIntelligenceResponse(data []byte) {
	redisResponse := RedisTl2NlIntelResponse{}
	err := json.Unmarshal(data, &redisResponse)
	if err != nil {
		log.Errorf("error unmarshalling RedisTl2NlIntelResponse from redis: %s", err)
		return
	}
	log.Debug("received intelligence response from TL")

	fakeResp, err := ip.createFakeIntelResponse(&redisResponse)
	if err != nil {
		log.Errorf("error creating fake intelligence response: %s", err)
		return
	}

	err = ip.respStorage.AddResponse(fakeResp.RequestId, fakeResp)
	if err != nil {
		log.Errorf("error fake intelligence response to response storage: %s", err)
		return
	}
	log.Debugf("successfuly ended onRedisIntelligenceResponse handler")
}

func (ip *IntelligenceProtocol) createFakeIntelResponse(redisResp *RedisTl2NlIntelResponse) (*pb.IntelligenceResponse, error) {
	payloadBytes, err := json.Marshal(redisResp.Payload)
	if err != nil {
		return nil, err
	}
	msgMetaData, err := ip.NewProtoMetaData()
	if err != nil {
		return nil, errors.WithMessage(err, "error generating new proto metadata: ")
	}

	protoMsg := &pb.SingleEntityResponse{
		Metadata: msgMetaData,
		Payload:  payloadBytes,
	}
	signature, err := ip.SignProtoMessage(protoMsg)
	if err != nil {
		return nil, errors.WithMessage(err, "error generating signature for new p2p intelligence request: ")
	}
	protoMsg.Metadata.Signature = signature

	// TODO implement encryption with original sender public key (I need to first store the key in the map when I received the request)
	// encrypt protoMsg
	encrypted, err := proto.Marshal(protoMsg) // change this to encryption method
	if err != nil {
		return nil, errors.WithMessage(err, "error encrypting the message (TODO for now it's just marshal: ")
	}

	resp := &pb.IntelligenceResponse{
		Metadata:  nil,
		RequestId: redisResp.RequestId,
		Processed: true,
		Responses: [][]byte{encrypted},
	}
	return resp, nil
}

// ##########################################################
// ### We receive intelligence request from another peer  ###
// ##########################################################
func (ip *IntelligenceProtocol) onP2PRequest(s network.Stream) {
	log.Infof("received p2p intelligence request")

	// Deserialize the proto message
	intelReqEnvelope := &pb.IntelligenceReqEnvelope{}
	err := ip.DeserializeMessageFromStream(s, intelReqEnvelope, true)
	if err != nil {
		log.Errorf("error deserilising p2p intelligence request from stream: %s", err)
		return
	}

	// Authenticate the message
	intelReq := intelReqEnvelope.IntelligenceRequest
	err = ip.AuthenticateMessage(intelReq, intelReq.Metadata)
	if err != nil {
		log.Errorf("error authenticating p2P intelligence request: %s", err)
		return
	}

	// Check if this message was already seen (maybe from another peer)
	if ip.SeenMessagesCache.WasMsgSeen(intelReq.Metadata.Id) {
		log.Debugf("received already seen intelligence request with id %s. No processing", intelReq.Metadata.Id)
		if err = ip.respondNoProcessing(s.Conn().RemotePeer(), intelReq.Metadata.Id); err != nil {
			log.Errorf("error while trying to respond with no processing response: %s", err)
		}
		return
	}
	ip.SeenMessagesCache.NewMsgSeen(intelReq.Metadata.Id, s.Conn().RemotePeer())

	// Process the request
	err = ip.processP2PRequest(intelReqEnvelope, s.Conn().RemotePeer())
	if err != nil {
		log.Errorf("error processing p2p intelligence request: %s", err)
		return
	}
	log.Debug("handler onP2PRequest successfully ended")
}

func (ip *IntelligenceProtocol) respondNoProcessing(receiver peer.ID, requestId string) error {
	metadata, err := ip.NewProtoMetaData()
	if err != nil {
		return err
	}
	response := &pb.IntelligenceResponse{
		Metadata:  metadata,
		RequestId: requestId,
		Processed: false,
		Responses: nil,
	}
	err = ip.SendProtoMessage(receiver, p2pIntelResponseProtocol, response)
	return err
}

func (ip *IntelligenceProtocol) processP2PRequest(e *pb.IntelligenceReqEnvelope, sender peer.ID) error {
	var v interface{}
	if err := json.Unmarshal(e.IntelligenceRequest.Payload, &v); err != nil {
		return err
	}

	senderPeerId, err := peer.Decode(e.IntelligenceRequest.Metadata.OriginalSender.NodeId)
	if err != nil {
		log.Errorf("error decoding peer ID: %s", err)
	}
	// send request to redis
	requestToRedis := RedisNl2TlIntelRequest{
		RequestId: e.IntelligenceRequest.Metadata.Id,
		Sender:    ip.MetadataOfPeer(senderPeerId),
		Payload:   v,
	}
	err = ip.RedisClient.PublishMessage("nl2tl_intelligence_request", requestToRedis)
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
	err = ip.respStorage.StartWaiting(ip.ctx, reqId, &utils.StorageMetadata{ResponsesReceiver: sender}, waitForResponses, waitTimeout)
	return err
}

func (ip *IntelligenceProtocol) updateEnvelope(e *pb.IntelligenceReqEnvelope) (*pb.IntelligenceReqEnvelope, error) {
	// TODO change the way how ttl and timeout is decreased/updated and use max timeout settings field
	// TODO maybe decide not to forward the message if parent timeout is too small
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

	receivedFrom, _ := ip.SeenMessagesCache.SenderOf(intelReqEnv.IntelligenceRequest.Metadata.Id)

	// send intelligence request to receivers
	sent := 0
	for _, pid := range pids {
		if pid == receivedFrom {
			continue
		}

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
