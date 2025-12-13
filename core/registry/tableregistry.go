package registry

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/skssmd/norm/core/utils"
)

// tableRegistry holds table-to-shard mappings
type tableRegistry struct {
	tables map[string]*TableShardMapping // tableName => shardName & role
	models map[string]*TableModel        // tableName => model struct
	mu     sync.RWMutex
}

// TableShardMapping defines which shard and role a table uses
type TableShardMapping struct {
	shardName string
	role      string // primary/read/write/standalone or empty for global
}

// global table registry singleton
var tableReg = &tableRegistry{
	tables: make(map[string]*TableShardMapping),
	models: make(map[string]*TableModel),
}
type TableModel struct {
	Fields []Field
}
// TableBuilder for fluent API
type TableBuilder struct {
	tableName string
}
type Field struct{
	Fieldname string
	Pk bool
	Fkey string
	Skey string
	Serial bool
	Indexed bool
	Unique bool
	OnDelete string
	Fieldtype string
	Max string
}
// Table registers a table with the ORM for migrations and routing
// Usage:
//
//	Table(User{}, "users")  // With custom table name
//	Table(User{})           // Auto-generate from struct name
func Table(model interface{}, tableName ...string) *TableBuilder {
	var name string
	if len(tableName) > 0 && tableName[0] != "" {
		name = tableName[0]
	} else {
		name = getTableName(model)
	}

	// Store model in registry
	tableReg.mu.Lock()
	table := registerTable(model,name)
	tableReg.models[name] = &table

	// Auto-register as global if in global mode
	dbMode := GetMode()
	if dbMode == "" || dbMode == "global" {
		if _, exists := tableReg.tables[name]; !exists {
			tableReg.tables[name] = &TableShardMapping{
				shardName: "",
				role:      "",
			}
		}
	}
	tableReg.mu.Unlock()
	
	// Register model for auto-migration callback
	registerModelForMigration(table)

	return &TableBuilder{
		tableName: name,
	}
}

func registerTable(model interface{}, tableName string) TableModel {
	t := reflect.TypeOf(model)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		panic("model must be struct or pointer to struct")
	}

	var table TableModel
	table.Fields = make([]Field, 0, t.NumField())

	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)

		// skip unexported fields
		if sf.PkgPath != "" {
			continue
		}

		normTag := sf.Tag.Get("norm")
		tags := utils.ParseNormTags(normTag)

		f := Field{
	Fieldname: utils.ToSnakeCase(sf.Name),
	Fieldtype: utils.GetPostgresType(sf),
}

		if _, ok := tags["index"]; ok {
			f.Indexed = true
		}
		if _, ok := tags["pk"]; ok {
			f.Pk = true
		}
		if name, ok := tags["name"]; ok {
			f.Fieldname = name.(string)
		}
		if _, ok := tags["unique"]; ok {
			f.Unique = true
		}
		if maxLen, ok := tags["max"]; ok {
			f.Max = maxLen.(string)
		}
		if skey, ok := tags["skey"]; ok {
			skParts := strings.Split(skey.(string), ".")
			if len(skParts) != 2 {
				fmt.Println("error while registering table", tableName, "at col", f.Fieldname)
				panic("skey")
			}
			f.Skey = skey.(string)
			f.Indexed = true
		}
		if fkey, ok := tags["fkey"]; ok {
			fkParts := strings.Split(fkey.(string), ".")
			if len(fkParts) != 2 {
				fmt.Println("error while registering table", tableName, "at col", f.Fieldname)
				panic("fkey")
			}
			f.Fkey = fkey.(string)
			f.OnDelete = "NO ACTION"

			if od, ok := tags["ondelete"]; ok {
				f.OnDelete = strings.ToUpper(od.(string))
			}
		}

		table.Fields = append(table.Fields, f)
	}

	return table
}



// registerModelForMigration is a placeholder that will be set by norm package
var registerModelForMigration = func(model interface{}) {
	// This will be overridden by norm package init
}

// SetModelRegistrationCallback sets the callback for model registration
func SetModelRegistrationCallback(callback func(interface{})) {
	registerModelForMigration = callback
}

// getTableName extracts table name from struct
func getTableName(model interface{}) string {
	t := reflect.TypeOf(model)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Name()
}

// GetRegisteredTableName looks up the registered table name for a model
// Returns the registered table name if found, otherwise returns empty string
func GetRegisteredTableName(model interface{}) string {
	t := reflect.TypeOf(model)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	modelType := t

	tableReg.mu.RLock()
	defer tableReg.mu.RUnlock()

	// Search for matching model type in registry
	for tableName, registeredModel := range tableReg.models {
		registeredType := reflect.TypeOf(registeredModel)
		if registeredType.Kind() == reflect.Ptr {
			registeredType = registeredType.Elem()
		}

		// Compare types
		if registeredType == modelType {
			return tableName
		}
	}

	return ""
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


// ListTables returns a slice of all registered table names
func ListTables() []string {
	tableReg.mu.RLock()
	defer tableReg.mu.RUnlock()

	tables := make([]string, 0, len(tableReg.models))
	for name := range tableReg.models {
		tables = append(tables, name)
	}
	return tables
}


func GetTable(name string) (*TableModel,bool) {
	tableReg.mu.RLock()
	defer tableReg.mu.RUnlock()

	table, exists := tableReg.models[name]
	if !exists {
		return nil,false
	}
	return table,true
}
// UnregisterTable removes a table from the registry (useful for testing)
func UnregisterTable(tableName string) error {
	tableReg.mu.Lock()
	defer tableReg.mu.Unlock()

	if _, exists := tableReg.tables[tableName]; !exists {
		return errors.New("table not registered")
	}

	delete(tableReg.tables, tableName)
	delete(tableReg.models, tableName)
	return nil
}

// GetAllModels returns all registered model structs
func GetAllModels() []interface{} {
	tableReg.mu.RLock()
	defer tableReg.mu.RUnlock()

	models := make([]interface{}, 0, len(tableReg.models))
	for _, model := range tableReg.models {
		models = append(models, model)
	}
	return models
}
