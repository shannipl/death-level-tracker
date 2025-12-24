package discord

import (
	"fmt"
	"sync"
)

type channelCache struct {
	mu    sync.RWMutex
	items map[string]string
}

func newChannelCache() *channelCache {
	return &channelCache{
		items: make(map[string]string),
	}
}

func (c *channelCache) Get(guildID, channelName string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	id, ok := c.items[c.key(guildID, channelName)]
	return id, ok
}

func (c *channelCache) Set(guildID, channelName, id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[c.key(guildID, channelName)] = id
}

func (c *channelCache) Invalidate(guildID, channelName string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, c.key(guildID, channelName))
}

func (c *channelCache) key(guildID, channelName string) string {
	return fmt.Sprintf("%s:%s", guildID, channelName)
}
