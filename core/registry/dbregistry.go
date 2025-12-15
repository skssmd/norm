package registry

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/skssmd/norm/core/driver"
)

// Registry holds the DB and shard pools
type Registry struct {
	pools  map[string]*driver.PGPool // global primary/replica/read/write
	shards map[string]*ShardPools    // shardName => shard pools
	mu     sync.RWMutex
	mode   string // "" | "global" | "shard"
	cacher Cacher
}
type Cacher interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, pattern string) error // Delete by glob pattern (e.g., *user*)
}
// ShardPools holds primary and standalone pools
type ShardPools struct {
	primary    *driver.PGPool
	standalone map[string]*driver.PGPool // tableName => pool
}

// global singleton
var norm = &Registry{
	pools:  make(map[string]*driver.PGPool),
	shards: make(map[string]*ShardPools),
}

// --- Connection builder ---
type ConnBuilder struct {
	reg *Registry
	dsn string
}

// Register a new DSN
func Register(dsn string) *ConnBuilder {
	return &ConnBuilder{
		reg: norm,
		dsn: dsn,
	}
}

// --- Global roles ---

func (c *ConnBuilder) Primary() error {
	c.reg.mu.Lock()
	defer c.reg.mu.Unlock()

	if c.reg.mode == "shard" {
		return errors.New("cannot register global primary when shards exist")
	}
	if len(c.reg.pools) > 0 {
		if _, ok := c.reg.pools["read"]; ok {
			return errors.New("primary cannot coexist with read/write pools")
		}
		if _, ok := c.reg.pools["write"]; ok {
			return errors.New("primary cannot coexist with read/write pools")
		}
	}
	if _, exists := c.reg.pools["primary"]; exists {
		return errors.New("primary already registered")
	}

	pool, err := driver.Connect(c.dsn)
	if err != nil {
		return err
	}

	c.reg.pools["primary"] = pool
	c.reg.mode = "global"
	return nil
}

func (c *ConnBuilder) Replica() error {
	c.reg.mu.Lock()
	defer c.reg.mu.Unlock()

	if c.reg.mode == "shard" {
		return errors.New("cannot register global replica when shards exist")
	}
	if _, ok := c.reg.pools["read"]; ok {
		return errors.New("replica cannot coexist with read/write pools")
	}
	if _, ok := c.reg.pools["write"]; ok {
		return errors.New("replica cannot coexist with read/write pools")
	}

	pool, err := driver.Connect(c.dsn)
	if err != nil {
		return err
	}

	// store replicas as "replica1", "replica2", ...
	count := 1
	for {
		key := fmt.Sprintf("replica%d", count)
		if _, exists := c.reg.pools[key]; !exists {
			c.reg.pools[key] = pool
			break
		}
		count++
	}

	c.reg.mode = "global"
	return nil
}

func (c *ConnBuilder) Read() error {
	c.reg.mu.Lock()
	defer c.reg.mu.Unlock()

	if c.reg.mode == "shard" {
		return errors.New("cannot register global read when shards exist")
	}
	if _, ok := c.reg.pools["primary"]; ok {
		return errors.New("read pools cannot coexist with primary")
	}

	pool, err := driver.Connect(c.dsn)
	if err != nil {
		return err
	}

	// store read pools as read1, read2...
	count := 1
	for {
		key := fmt.Sprintf("read%d", count)
		if _, exists := c.reg.pools[key]; !exists {
			c.reg.pools[key] = pool
			break
		}
		count++
	}

	c.reg.mode = "global"
	return nil
}

func (c *ConnBuilder) Write() error {
	c.reg.mu.Lock()
	defer c.reg.mu.Unlock()

	if c.reg.mode == "shard" {
		return errors.New("cannot register global write when shards exist")
	}
	if _, ok := c.reg.pools["primary"]; ok {
		return errors.New("write pool cannot coexist with primary")
	}
	if _, exists := c.reg.pools["write"]; exists {
		return errors.New("write pool already registered")
	}

	pool, err := driver.Connect(c.dsn)
	if err != nil {
		return err
	}

	c.reg.pools["write"] = pool
	c.reg.mode = "global"
	return nil
}

// --- Shard builder ---

type ShardBuilder struct {
	reg       *Registry
	dsn       string
	shardName string
}

func (c *ConnBuilder) Shard(name string) *ShardBuilder {
	return &ShardBuilder{
		reg:       c.reg,
		dsn:       c.dsn,
		shardName: name,
	}
}

func (s *ShardBuilder) Primary() error {
	s.reg.mu.Lock()
	defer s.reg.mu.Unlock()

	if s.reg.mode == "global" && len(s.reg.pools) > 0 {
		return errors.New("cannot register shard when global pools exist")
	}

	if s.reg.shards[s.shardName] == nil {
		s.reg.shards[s.shardName] = &ShardPools{
			standalone: make(map[string]*driver.PGPool),
		}
	}

	if s.reg.shards[s.shardName].primary != nil {
		return errors.New("primary pool for shard already exists")
	}

	// Check if standalone pools already exist for this shard
	if len(s.reg.shards[s.shardName].standalone) > 0 {
		return fmt.Errorf("cannot register primary for shard '%s': standalone pools already exist (shard cannot be both primary and standalone)", s.shardName)
	}

	pool, err := driver.Connect(s.dsn)
	if err != nil {
		return err
	}

	s.reg.shards[s.shardName].primary = pool
	s.reg.mode = "shard"
	return nil
}

func (s *ShardBuilder) Standalone(tables ...string) error {
	s.reg.mu.Lock()
	defer s.reg.mu.Unlock()

	if s.reg.mode == "global" && len(s.reg.pools) > 0 {
		return errors.New("cannot register shard standalone when global pools exist")
	}

	if s.reg.shards[s.shardName] == nil {
		s.reg.shards[s.shardName] = &ShardPools{
			standalone: make(map[string]*driver.PGPool),
		}
	}

	// Check if primary pool already exists for this shard
	if s.reg.shards[s.shardName].primary != nil {
		return fmt.Errorf("cannot register standalone for shard '%s': primary pool already exists (shard cannot be both primary and standalone)", s.shardName)
	}

	pool, err := driver.Connect(s.dsn)
	if err != nil {
		return err
	}

	if len(tables) > 0 {
		// Register pool for specific tables
		for _, table := range tables {
			if _, exists := s.reg.shards[s.shardName].standalone[table]; exists {
				return fmt.Errorf("standalone pool for table '%s' already exists in shard '%s'", table, s.shardName)
			}
			s.reg.shards[s.shardName].standalone[table] = pool
		}
	} else {
		// Fallback: register with generic key (legacy/default behavior)
		// WARNING: Router looks up by table name, so this might not be reachable for specific queries logic!
		key := fmt.Sprintf("standalone%d", len(s.reg.shards[s.shardName].standalone)+1)
		s.reg.shards[s.shardName].standalone[key] = pool
	}

	s.reg.mode = "shard"
	return nil
}

// SetCacher sets the global cacher
func SetCacher(c Cacher) {
	norm.mu.Lock()
	defer norm.mu.Unlock()
	norm.cacher = c
}

// GetCacher returns the global cacher
func GetCacher() Cacher {
	norm.mu.RLock()
	defer norm.mu.RUnlock()
	return norm.cacher
}

// --- Registry Info Functions ---

// GetRegistryInfo returns information about the current registry state
func GetRegistryInfo() map[string]interface{} {
	norm.mu.RLock()
	defer norm.mu.RUnlock()

	info := make(map[string]interface{})
	info["mode"] = norm.mode

	// Global pools (with actual pool references)
	globalPools := make(map[string]interface{})
	for poolName, pool := range norm.pools {
		globalPools[poolName] = pool
	}
	info["pools"] = globalPools

	// Shard info (with actual pool references)
	shardInfo := make(map[string]interface{})
	for shardName, shardPools := range norm.shards {
		sInfo := make(map[string]interface{})
		sInfo["has_primary"] = shardPools.primary != nil
		sInfo["primary_pool"] = shardPools.primary

		// Return actual pool references for standalone pools
		standalonePools := make(map[string]*driver.PGPool)
		for key, pool := range shardPools.standalone {
			standalonePools[key] = pool
		}
		sInfo["standalone_pools"] = standalonePools

		shardInfo[shardName] = sInfo
	}
	info["shards"] = shardInfo

	return info
}

// GetPoolCount returns the total number of connection pools
func GetPoolCount() int {
	norm.mu.RLock()
	defer norm.mu.RUnlock()

	count := len(norm.pools)
	for _, shardPools := range norm.shards {
		if shardPools.primary != nil {
			count++
		}
		count += len(shardPools.standalone)
	}
	return count
}

// GetMode returns the current registry mode
func GetMode() string {
	norm.mu.RLock()
	defer norm.mu.RUnlock()
	return norm.mode
}

// Reset closes all connections and clears the registry
// Useful for testing scenarios
func Reset() {
	norm.mu.Lock()
	defer norm.mu.Unlock()

	// Close global pools
	for _, pool := range norm.pools {
		pool.Close()
	}
	norm.pools = make(map[string]*driver.PGPool)

	// Close shard pools
	for _, shard := range norm.shards {
		if shard.primary != nil {
			shard.primary.Close()
		}
		for _, pool := range shard.standalone {
			pool.Close()
		}
	}
	norm.shards = make(map[string]*ShardPools)

	// Reset mode
	norm.mode = ""

	// Reset table registry
	resetTables()

	// Reset cacher
	norm.cacher = nil
}
