package cache

import (
	"context"
	"os"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/quantmind-br/repodocs-go/internal/domain"
)

// BadgerCache is a cache implementation using BadgerDB
type BadgerCache struct {
	db *badger.DB
}

// NewBadgerCache creates a new BadgerDB cache
func NewBadgerCache(opts Options) (*BadgerCache, error) {
	var badgerOpts badger.Options

	if opts.InMemory {
		badgerOpts = badger.DefaultOptions("").WithInMemory(true)
	} else {
		if opts.Directory == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return nil, err
			}
			opts.Directory = homeDir + "/.repodocs/cache"
		}

		// Ensure directory exists
		if err := os.MkdirAll(opts.Directory, 0755); err != nil {
			return nil, err
		}

		badgerOpts = badger.DefaultOptions(opts.Directory)
	}

	// Disable logging unless explicitly enabled
	if !opts.Logger {
		badgerOpts = badgerOpts.WithLogger(nil)
	}

	db, err := badger.Open(badgerOpts)
	if err != nil {
		return nil, err
	}

	// Start background garbage collection
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			_ = db.RunValueLogGC(0.5)
		}
	}()

	return &BadgerCache{db: db}, nil
}

// Get retrieves a value from cache
func (c *BadgerCache) Get(ctx context.Context, key string) ([]byte, error) {
	// Generate cache key from URL
	cacheKey := GenerateKey(key)

	var value []byte
	err := c.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(cacheKey))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return domain.ErrCacheMiss
			}
			return err
		}

		value, err = item.ValueCopy(nil)
		return err
	})

	if err != nil {
		return nil, err
	}

	return value, nil
}

// Set stores a value in cache with TTL
func (c *BadgerCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	// Generate cache key from URL
	cacheKey := GenerateKey(key)

	return c.db.Update(func(txn *badger.Txn) error {
		e := badger.NewEntry([]byte(cacheKey), value)
		if ttl > 0 {
			e = e.WithTTL(ttl)
		}
		return txn.SetEntry(e)
	})
}

// Has checks if a key exists in cache
func (c *BadgerCache) Has(ctx context.Context, key string) bool {
	cacheKey := GenerateKey(key)

	err := c.db.View(func(txn *badger.Txn) error {
		_, err := txn.Get([]byte(cacheKey))
		return err
	})

	return err == nil
}

// Delete removes a key from cache
func (c *BadgerCache) Delete(ctx context.Context, key string) error {
	cacheKey := GenerateKey(key)

	return c.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(cacheKey))
	})
}

// Close releases cache resources
func (c *BadgerCache) Close() error {
	return c.db.Close()
}

// Clear removes all entries from the cache
func (c *BadgerCache) Clear() error {
	return c.db.DropAll()
}

// Size returns the number of entries in the cache
func (c *BadgerCache) Size() int64 {
	var count int64
	_ = c.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			count++
		}
		return nil
	})
	return count
}

// Stats returns cache statistics
func (c *BadgerCache) Stats() map[string]interface{} {
	lsm, vlog := c.db.Size()
	return map[string]interface{}{
		"entries":   c.Size(),
		"lsm_size":  lsm,
		"vlog_size": vlog,
	}
}
