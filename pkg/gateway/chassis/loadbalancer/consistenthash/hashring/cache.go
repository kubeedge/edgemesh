package hashring

import (
	"sync"

	"github.com/buraksezer/consistent"
	"k8s.io/klog/v2"
)

// cache store hash ring in memory
var cache sync.Map

// AddOrUpdateHashRing adds or updates a hash ring in cache
func AddOrUpdateHashRing(key string, c *consistent.Consistent) {
	klog.Infof("add or update hash ring `%s` in cache, hash ring members: %+v", key, c.GetMembers())
	cache.Store(key, c)
}

// DeleteHashRing deletes a hash ring from cache
func DeleteHashRing(key string) {
	klog.Infof("delete hash ring `%s` from cache", key)
	cache.Delete(key)
}

// GetHashRing returns a hash ring from cache
func GetHashRing(key string) (*consistent.Consistent, bool) {
	h, ok := cache.Load(key)
	if !ok {
		klog.Warningf("hash ring `%s` not found", key)
		return nil, false
	}
	hr, ok := h.(*consistent.Consistent)
	if !ok {
		klog.Errorf("hash ring `%s` type invalid", key)
		return nil, false
	}
	return hr, true
}

// RangeHashRing ranges all hash rings
func RangeHashRing(fn func(key, value interface{}) bool) {
	cache.Range(fn)
}
