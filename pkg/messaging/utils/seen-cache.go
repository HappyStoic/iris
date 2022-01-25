package utils

import "time"

type SeenMessagesCache struct {
	cache map[string]bool
	ttl   time.Duration
}

func newMessageCache() *SeenMessagesCache {
	return &SeenMessagesCache{
		cache: make(map[string]bool),
		ttl:   0, // TODO proper ttl and make goroutine ticker to remove old ones
	}
}

func (c *SeenMessagesCache) WasMsgSeen(s string) bool {
	_, ok := c.cache[s]
	return ok
}

func (c *SeenMessagesCache) NewMsgSeen(s string) {
	c.cache[s] = true
}
