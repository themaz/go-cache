package cache

import (
	"fmt"
	"sync"
	"time"
)

type unexportedInterface interface {
	Set(string, string, time.Duration)
	Add(string, string, time.Duration) error
	Replace(string, string, time.Duration) error
	Get(string) (string, bool)
	Delete(string)
	DeleteExpired()
	ItemCount() int
}

type item struct {
	Object     string
	Expiration *time.Time
}

// Returns true if the item has expired.
func (i *item) Expired() bool {
	if i.Expiration == nil {
		return false
	}
	return i.Expiration.Before(time.Now())
}

type Cache struct {
	*cache
	// If this is confusing, see the comment at the bottom of New()
}

type cache struct {
	sync.RWMutex
	defaultExpiration time.Duration
	items             map[string]*item
	consumerChannel   chan []string
}

// Add an item to the cache, replacing any existing item. If the duration is 0,
// the cache's default expiration time is used. If it is -1, the item never
// expires.
func (c *cache) Set(k string, x string, d time.Duration) {
	c.consumerChannel <- []string{"set", k, x}
}

func (c *cache) set(k string, x string, d time.Duration) {
	var e *time.Time
	if d == 0 {
		d = c.defaultExpiration
	}
	if d > 0 {
		t := time.Now().Add(d)
		e = &t
	}
	c.items[k] = &item{
		Object:     x,
		Expiration: e,
	}
}

// Add an item to the cache only if an item doesn't already exist for the given
// key, or if the existing item has expired. Returns an error otherwise.
func (c *cache) Add(k string, x string, d time.Duration) error {
	_, found := c.get(k)
	if found {
		return fmt.Errorf("Item %s already exists", k)
	}
	c.Set(k, x, d)
	return nil
}

// Set a new value for the cache key only if it already exists, and the existing
// item hasn't expired. Returns an error otherwise.
func (c *cache) Replace(k string, x string, d time.Duration) error {
	_, found := c.get(k)
	if !found {
		return fmt.Errorf("Item %s doesn't exist", k)
	}
	c.Set(k, x, d)
	return nil
}

// Get an item from the cache. Returns the item or nil, and a bool indicating
// whether the key was found.
func (c *cache) Get(k string) (string, bool) {
	x, found := c.get(k)
	return x, found
}

func (c *cache) get(k string) (string, bool) {
	item, found := c.items[k]
	if !found || item.Expired() {
		c.Delete(k)
		return "", false
	}
	return item.Object, true
}

// Delete an item from the cache. Does nothing if the key is not in the cache.
func (c *cache) Delete(k string) {
	c.consumerChannel <- []string{"delete", k, ""}
}

func (c *cache) delete(k string) {
	delete(c.items, k)
}

// Delete all expired items from the cache.
func (c *cache) DeleteExpired() {
	for k, v := range c.items {
		if v.Expired() {
			c.consumerChannel <- []string{"delete", k, ""}
		}
	}
}

// Returns the number of items in the cache. This may include items that have
// expired, but have not yet been cleaned up.
func (c *cache) ItemCount() int {
	n := len(c.items)
	return n
}

func newCache(de time.Duration) *cache {
	if de == 0 {
		de = -1
	}
	c := &cache{
		defaultExpiration: de,
		items:             map[string]*item{},
	}
	return c
}

// Return a new cache with a given default expiration duration.
// If the expiration duration is less than 1, the items in the cache
// never expire (by default), and must be deleted manually.
func New(defaultExpiration time.Duration) *Cache {
	c := newCache(defaultExpiration)
	// This trick ensures that the consumer goroutine (which--granted it
	// was enabled--is running DeleteExpired on c forever) does not keep
	// the returned C object from being garbage collected. When it is
	// garbage collected, the finalizer stops the consumer goroutine, after
	// which c can be collected.
	C := &Cache{c}
	c.consumerChannel = make(chan []string)

	go runConsumer(c)
	return C
}

func runConsumer(c *cache) {
	for {
		select {
		case v := <- c.consumerChannel:
			operation := v[0]
			key := v[1]
			val := v[2]
			if operation == "set" {
				c.set(key, val, c.defaultExpiration)
			} else if operation == "delete" {
				c.delete(key)
			}
		}
	}
}
