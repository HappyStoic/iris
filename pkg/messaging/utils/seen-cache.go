package utils

import (
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
)

// SeenMessagesCache represents cache to know if we have already seen given P2P message or not.
// It implements a map with request IDs as keys and sender peer ID (who we actually received the message from) as values
type SeenMessagesCache struct {
	cache map[string]peer.ID
	ttl   time.Duration
}

func newMessageCache() *SeenMessagesCache {
	return &SeenMessagesCache{
		cache: make(map[string]peer.ID),
		ttl:   0,
	}
}

func (c *SeenMessagesCache) WasMsgSeen(msgId string) bool {
	_, ok := c.cache[msgId]
	return ok
}

func (c *SeenMessagesCache) NewMsgSeen(msgId string, peerId peer.ID) {
	c.cache[msgId] = peerId
}

func (c *SeenMessagesCache) SenderOf(msgId string) (peer.ID, bool) {
	peerId, ok := c.cache[msgId]
	return peerId, ok
}
