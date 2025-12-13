package registry

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/skssmd/norm/core/utils"
)

// tableRegistry holds table-to-shard mappings
type tableRegistry struct {
	models map[string]*TableModel // tableName => model
	mu     sync.RWMutex
}

var tableReg = &tableRegistry{
	models: make(map[string]*TableModel),
}

type TableModel struct {
	TableName string
	Fields    []Field

	// role -> set of shards
	Roles map[string]map[string]struct{}
}

// TableBuilder for fluent API

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
func Table(model interface{}, tableName ...string) *TableModel {
	var name string
	if len(tableName) > 0 && tableName[0] != "" {
		name = tableName[0]
	} else {
		name = getTableName(model)
	}

	tableReg.mu.Lock()
	defer tableReg.mu.Unlock()

	table := registerTable(model, name)

	// initialize roles
	table.Roles = make(map[string]map[string]struct{})

	// store pointer
	tableReg.models[name] = &table

	// migration bookkeeping
	registerModelForMigration(table)

	// RETURN THE SAME POINTER
	return tableReg.models[name]
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
    Fieldname: utils.ResolveColumnName(sf),
    Fieldtype: utils.GetPostgresType(sf),
}

		if _, ok := tags["index"]; ok {
			f.Indexed = true
		}
		if _, ok := tags["pk"]; ok {
			f.Pk = true
		}
	
		if _, ok := tags["auto"]; ok {
			f.Serial = true
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
		
			f.Indexed = true
			
		}
		f.OnDelete = "NO ACTION"

			if od, ok := tags["ondelete"]; ok {
				f.OnDelete = strings.ToUpper(od.(string))
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

// Shard specifies which shard the table belongs to


	// TableModel fluent API for shard assignment
func (tm *TableModel) Primary(shard string) error {
	if tm.Roles == nil {
		tm.Roles = make(map[string]map[string]struct{})
	}
	if tm.Roles["primary"] == nil {
		tm.Roles["primary"] = make(map[string]struct{})
	}

	if _, exists := tm.Roles["primary"][shard]; exists {
		return fmt.Errorf("table %s already has primary for shard %s", tm.TableName, shard)
	}

	tm.Roles["primary"][shard] = struct{}{}
	return nil
}

func (tm *TableModel) Read(shard string) error {
	if tm.Roles == nil {
		tm.Roles = make(map[string]map[string]struct{})
	}
	if tm.Roles["read"] == nil {
		tm.Roles["read"] = make(map[string]struct{})
	}
	tm.Roles["read"][shard] = struct{}{}
	return nil
}

func (tm *TableModel) Write(shard string) error {
	if tm.Roles == nil {
		tm.Roles = make(map[string]map[string]struct{})
	}
	if tm.Roles["write"] == nil {
		tm.Roles["write"] = make(map[string]struct{})
	}
	tm.Roles["write"][shard] = struct{}{}
	return nil
}

func (tm *TableModel) Standalone(shard string) error {
	if tm.Roles == nil {
		tm.Roles = make(map[string]map[string]struct{})
	}
	if tm.Roles["standalone"] == nil {
		tm.Roles["standalone"] = make(map[string]struct{})
	}
	tm.Roles["standalone"][shard] = struct{}{}
	return nil
}
// Roles returns a slice of role names assigned to this table
func (tm *TableModel) RoleNames() []string {
	tmRoles := make([]string, 0, len(tm.Roles))
	for role := range tm.Roles {
		tmRoles = append(tmRoles, role)
	}
	return tmRoles
}

// IsGlobal returns true if the table has no shard-specific roles
func (tm *TableModel) IsGlobal() bool {
	// global table has no shard-specific roles
	return len(tm.Roles) == 0
}



// GetModel returns a registered TableModel by table name
func GetModel(tableName string) (*TableModel, bool) {
	tableReg.mu.RLock()
	defer tableReg.mu.RUnlock()
	t, exists := tableReg.models[tableName]
	return t, exists
}



// GetTableMapping retrieves the shard mapping for a table
func GetTableMapping(tableName string) (*TableModel, error) {
	tableReg.mu.RLock()
	defer tableReg.mu.RUnlock()

	mapping, exists := tableReg.models[tableName]
	if !exists {
		return nil, fmt.Errorf("table %s not registered", tableName)
	}
	return mapping, nil
}

// GetTableMappingByModel retrieves the shard mapping using a model struct
func GetTableMappingByModel(model interface{}) (*TableModel, error) {
	tableName := getTableName(model)
	return GetTableMapping(tableName)
}

// ShardName returns the shard name for this mapping

// Role returns the role for this mapping



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

	table, exists := tableReg.models[tableName]
	if !exists {
		return fmt.Errorf("table %s not registered", tableName)
	}

	// Clear roles and fields for safety
	table.Roles = nil
	table.Fields = nil

	// Remove from registry
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

// resetTables clears the table registry
func resetTables() {
	tableReg.mu.Lock()
	defer tableReg.mu.Unlock()
	tableReg.models = make(map[string]*TableModel)
}
