package bigcache

import (
	"log"
	"testing"
	"time"
)

func TestCache(t *testing.T) {
	c, err := NewCache(
		2,
		1024*10,
		4096,
		5*time.Second,
		2*time.Second,
	)
	if err != nil {
		t.Error(err)
	}

	err = c.Set("foo", []byte("bar"))
	if err != nil {
		t.Error(err)
	}

	v1, err := c.Get("foo")
	if err != nil {
		t.Error(err)
	}
	log.Printf("Val:%v", string(v1))

	time.Sleep(6 * time.Second)
	log.Printf("Wake up!")

	// 过期
	v2, err := c.Get("foo")
	if err != nil {
		t.Error(err)
	}
	log.Printf("Val:%v", string(v2))
}
