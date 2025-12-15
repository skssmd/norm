package engine

import (
	"context"
	"fmt"
	"strings"

	"github.com/skssmd/norm/core/registry"
)

// InvalidateCache - Strict mode: invalidates a specific scope defined by table and keys.
// Pattern: *<table>*<key1>:<key2>*
func (q *Query) InvalidateCache(keys ...string) *Query {
	if len(keys) == 0 {
		panic("InvalidateCache requires at least one key")
	}

	cacher := registry.GetCacher()
	if cacher == nil {
		return q
	}

	// Join keys to form the sequence
	keySeq := strings.Join(keys, ":")

	// Pattern: *table*keySequence*
	// Use q.table which is always set for single table queries
	// For Joins, this usually invalidates based on the primary table context
	pattern := fmt.Sprintf("*%s*%s*", q.table, keySeq)

	// Use simplified Delete with glob pattern
	cacher.Delete(context.Background(), pattern)

	return q
}

// InvalidateCacheReferenced - Referenced mode: invalidates broadly by referenced key.
// Pattern: *<key>*
func (q *Query) InvalidateCacheReferenced(keys ...string) *Query {
	if len(keys) == 0 {
		panic("InvalidateCacheReferenced requires at least one key")
	}

	cacher := registry.GetCacher()
	if cacher == nil {
		return q
	}

	ctx := context.Background()

	for _, key := range keys {
		// Pattern: *key*
		// Matches any cache key containing this component
		pattern := fmt.Sprintf("*%s*", key)
		cacher.Delete(ctx, pattern)
	}

	return q
}
