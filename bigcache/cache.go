/*
 * 本地缓存
 * 在高并发的系统中，缓存的设计，应该是分级的，所以本地缓存有其存在价值
 */
package bigcache

import (
	"github.com/allegro/bigcache/v3"
	"time"
)

type Cache interface {
	Set(k string, v []byte) error
	Get(k string) ([]byte, error)
}

type cache struct {
	cache *bigcache.BigCache
}

// 这些参数组合起来，会影响到整体内存占用，实际使用中，可以多尝试配置
func NewCache(shard, maxEntriesInWindow, MaxEntrySize int, lifeWindow, cleanWindow time.Duration) (Cache, error) {
	c, err := bigcache.NewBigCache(bigcache.Config{
		Shards:             shard,
		LifeWindow:         lifeWindow,         // 超过这个时间，条目可以被删除（仅仅是可以，还需要搭配CleanWindow）
		CleanWindow:        cleanWindow,        // 配合LifeWindow使用，才能起到过期的效果
		MaxEntriesInWindow: maxEntriesInWindow, // lifewindow内最大条目数
		MaxEntrySize:       MaxEntrySize,       // 单位：byte，条目最大长度

		// HardMaxCacheSize: 8192, //单位：MB，硬编码写死最大内存占用
		// CleanWindow:        5 * time.Minute, // 清理过期内存的间隔
		// Verbose:            true,
	})
	if err != nil {
		return nil, err
	}
	return &cache{cache: c}, nil
}

func (c *cache) Set(k string, v []byte) error {
	return c.cache.Set(k, v)
}

func (c *cache) Get(k string) ([]byte, error) {
	return c.cache.Get(k)
}
