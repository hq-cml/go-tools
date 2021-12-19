package bigcache

import (
    "github.com/allegro/bigcache/v3"
    "time"
)

type Cache interface {
    Set()
    Get()
}

type cache struct {
    cache *bigcache.BigCache
}

func NewCache() (Cache, error) {
    c, err := bigcache.NewBigCache(bigcache.Config{
        Shards:             128,
        LifeWindow:         5 * time.Minute,
        CleanWindow:        5 * time.Minute,
        MaxEntriesInWindow: 1000 * 10 * 60,
        MaxEntrySize:       4096,
        StatsEnabled:       false,
        // Verbose:            true,
    })
    if err != nil {
        return nil, err
    }
    return &cache{cache:c}, nil
}

func (c *cache)Get() {

}

func (c *cache)Set() {

}