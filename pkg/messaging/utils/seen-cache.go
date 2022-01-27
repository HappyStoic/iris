package utils

import (
	"github.com/libp2p/go-libp2p-core/peer"
	"time"
)

type SeenMessagesCache struct {
	cache map[string]peer.ID
	ttl   time.Duration
}

func newMessageCache() *SeenMessagesCache {
	return &SeenMessagesCache{
		cache: make(map[string]peer.ID),
		ttl:   0, // TODO proper ttl and make goroutine ticker to remove old ones
	}
}

func (c *SeenMessagesCache) WasMsgSeen(keyId string) bool {
	_, ok := c.cache[keyId]
	return ok
}

func (c *SeenMessagesCache) NewMsgSeen(keyId string, peerId peer.ID) {
	c.cache[keyId] = peerId
}

func (c *SeenMessagesCache) senderOf(keyId string) (peer.ID, bool) {
	peerId, ok := c.cache[keyId]
	return peerId, ok
}
