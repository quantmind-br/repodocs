// Package cache provides persistent content caching using BadgerDB.
//
// It defines the cache contract used by fetchers and strategies, plus a Badger
// implementation for durable storage between runs. The package also centralizes
// cache key generation so equivalent requests map to stable entries.
package cache
