package bigcache

import (
    "testing"
)

func TestNewCache(t *testing.T) {
    got, err := NewCache()
    if err != nil {
        t.Error(err)
    }
    _ = got
}