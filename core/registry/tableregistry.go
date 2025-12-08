package registry

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
)

// TableRegistry holds table-to-shard mappings
type TableRegistry struct {
	tables map[string]*TableShardMapping // tableName => shardName & role
	mu     sync.RWMutex
}

// TableShardMapping defines which shard and role a table uses
type TableShardMapping struct {
	shardName string
	role      string // primary/read/write/standalone or empty for global
}

// global table registry singleton
var tableReg = &TableRegistry{
	tables: make(map[string]*TableShardMapping),
}

// TableBuilder for fluent API
type TableBuilder struct {
	tableName string
}

// Table registers a table and returns a builder for configuration
func Table(model interface{}) *TableBuilder {
	tableName := getTableName(model)
	return &TableBuilder{
		tableName: tableName,
	}
}

// getTableName extracts table name from struct
func getTableName(model interface{}) string {
	t := reflect.TypeOf(model)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Name()
}

// Global registers table to use global pools
func (tb *TableBuilder) Global() error {
	tableReg.mu.Lock()
	defer tableReg.mu.Unlock()

	// Check if DB registry mode is shard
	dbMode := GetMode()
	if dbMode == "shard" {
		return fmt.Errorf("cannot register table as global when DB registry mode is 'shard'")
	}

	if _, exists := tableReg.tables[tb.tableName]; exists {
		return fmt.Errorf("table %s already registered", tb.tableName)
	}

	tableReg.tables[tb.tableName] = &TableShardMapping{
		shardName: "",
		role:      "",
	}
	return nil
}

// TableShardBuilder for shard-specific configuration
type TableShardBuilder struct {
	tableName string
	shardName string
}

// Shard specifies which shard the table belongs to
func (tb *TableBuilder) Shard(shardName string) *TableShardBuilder {
	return &TableShardBuilder{
		tableName: tb.tableName,
		shardName: shardName,
	}
}

// Primary registers table to use shard's primary pool
func (tsb *TableShardBuilder) Primary() error {
	return tsb.register("primary")
}

// Read registers table to use shard's read pool
func (tsb *TableShardBuilder) Read() error {
	return tsb.register("read")
}

// Write registers table to use shard's write pool
func (tsb *TableShardBuilder) Write() error {
	return tsb.register("write")
}

// Standalone registers table to use shard's standalone pool
func (tsb *TableShardBuilder) Standalone() error {
	return tsb.register("standalone")
}

// register is the internal method to register table with shard and role
func (tsb *TableShardBuilder) register(role string) error {
	tableReg.mu.Lock()
	defer tableReg.mu.Unlock()

	// Check if DB registry mode is global
	dbMode := GetMode()
	if dbMode == "global" {
		return fmt.Errorf("cannot register table to shard '%s' when DB registry mode is 'global'", tsb.shardName)
	}

	if _, exists := tableReg.tables[tsb.tableName]; exists {
		return fmt.Errorf("table %s already registered", tsb.tableName)
	}

	tableReg.tables[tsb.tableName] = &TableShardMapping{
		shardName: tsb.shardName,
		role:      role,
	}
	return nil
}

// GetTableMapping retrieves the shard mapping for a table
func GetTableMapping(tableName string) (*TableShardMapping, error) {
	tableReg.mu.RLock()
	defer tableReg.mu.RUnlock()

	mapping, exists := tableReg.tables[tableName]
	if !exists {
		return nil, fmt.Errorf("table %s not registered", tableName)
	}
	return mapping, nil
}

// GetTableMappingByModel retrieves the shard mapping using a model struct
func GetTableMappingByModel(model interface{}) (*TableShardMapping, error) {
	tableName := getTableName(model)
	return GetTableMapping(tableName)
}

// ShardName returns the shard name for this mapping
func (tsm *TableShardMapping) ShardName() string {
	return tsm.shardName
}

// Role returns the role for this mapping
func (tsm *TableShardMapping) Role() string {
	return tsm.role
}

// IsGlobal returns true if the table uses global pools
func (tsm *TableShardMapping) IsGlobal() bool {
	return tsm.shardName == ""
}

// ListTables returns all registered table names
func ListTables() []string {
	tableReg.mu.RLock()
	defer tableReg.mu.RUnlock()

	tables := make([]string, 0, len(tableReg.tables))
	for name := range tableReg.tables {
		tables = append(tables, name)
	}
	return tables
}

// UnregisterTable removes a table from the registry (useful for testing)
func UnregisterTable(tableName string) error {
	tableReg.mu.Lock()
	defer tableReg.mu.Unlock()

	if _, exists := tableReg.tables[tableName]; !exists {
		return errors.New("table not registered")
	}

	delete(tableReg.tables, tableName)
	return nil
}
