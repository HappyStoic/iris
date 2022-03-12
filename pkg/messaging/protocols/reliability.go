package protocols

import (
	"encoding/json"

	"github.com/libp2p/go-libp2p-core/peer"

	"happystoic/p2pnetwork/pkg/messaging/utils"
	"happystoic/p2pnetwork/pkg/reliability"
)

// ReliabilityReceiver type
type ReliabilityReceiver struct {
	*utils.ProtoUtils

	relBook *reliability.Book
}

type RedisRelUpdate struct {
	PeerId      string                  `json:"peer_id"`
	Reliability reliability.Reliability `json:"reliability"`
}

func NewReliabilityReceiver(pu *utils.ProtoUtils, rb *reliability.Book) *ReliabilityReceiver {
	rc := &ReliabilityReceiver{pu, rb}

	_ = rc.RedisClient.SubscribeCallback("tl2nl_peers_reliability", rc.onRedisReliabilityUpdate)
	return rc
}

func (rc *ReliabilityReceiver) onRedisReliabilityUpdate(data []byte) {
	relUpdate := make([]RedisRelUpdate, 0)
	err := json.Unmarshal(data, &relUpdate)
	if err != nil {
		log.Errorf("error unmarshalling RedisRelUpdate from redis: %s", err)
		return
	}
	log.Debug("received new reliability update from Redis")
	updateCount := 0
	for _, item := range relUpdate {
		decodedPeer, err := peer.Decode(item.PeerId)
		if err != nil {
			log.Errorf("error decoding peer ID: %s", err)
			continue
		}
		rc.relBook.UpdatePeerRel(decodedPeer, item.Reliability)
		log.Debugf("updated reliability of peer: '%s' to %f", item.PeerId, item.Reliability)
		updateCount++
	}
	log.Infof("successfuly updated reliability of %d peers", updateCount)
}
