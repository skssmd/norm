package registry

import (
	"errors"
	"fmt"
	"sync"

	"github.com/skssmd/norm/core/driver"
)

// Registry holds the DB and shard pools
type Registry struct {
	pools  map[string]*driver.PGPool // global primary/replica/read/write
	shards map[string]*ShardPools    // shardName => shard pools
	mu     sync.RWMutex
	mode   string // "" | "global" | "shard"
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

	pool, err := driver.Connect(s.dsn)
	if err != nil {
		return err
	}

	s.reg.shards[s.shardName].primary = pool
	s.reg.mode = "shard"
	return nil
}

func (s *ShardBuilder) Standalone() error {
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

	pool, err := driver.Connect(s.dsn)
	if err != nil {
		return err
	}

	// use shardName + count as key to avoid collision
	key := fmt.Sprintf("standalone%d", len(s.reg.shards[s.shardName].standalone)+1)
	s.reg.shards[s.shardName].standalone[key] = pool
	s.reg.mode = "shard"
	return nil
}
