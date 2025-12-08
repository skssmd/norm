package norm

import (
	"fmt"
	"strings"

	"github.com/skssmd/norm/core/registry"
)

func Norm() {
	// ------------------------
	// Print Registry Summary
	// ------------------------
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("REGISTRY SUMMARY")
	fmt.Println(strings.Repeat("=", 60))

	// DB Registry Info
	dbInfo := registry.GetRegistryInfo()
	fmt.Println("\n📊 Database Connection Registry:")
	fmt.Printf("  Mode: %v\n", dbInfo["mode"])
	fmt.Printf("  Total Connection Pools: %d\n", registry.GetPoolCount())

	if globalPools, ok := dbInfo["global_pools"].([]string); ok && len(globalPools) > 0 {
		fmt.Println("\n  Global Pools:")
		for _, poolName := range globalPools {
			fmt.Printf("    • %s\n", poolName)
		}
	}

	if shards, ok := dbInfo["shards"].(map[string]map[string]interface{}); ok && len(shards) > 0 {
		fmt.Println("\n  Shards:")
		for shardName, shardInfo := range shards {
			fmt.Printf("    • %s:\n", shardName)
			if hasPrimary, ok := shardInfo["has_primary"].(bool); ok && hasPrimary {
				fmt.Println("        - Primary: ✓")
			}
			if standalonePools, ok := shardInfo["standalone_pools"].([]string); ok && len(standalonePools) > 0 {
				fmt.Printf("        - Standalone pools: %v\n", standalonePools)
			}
		}
	}

	// Table Registry Info
	fmt.Println("\n📋 Table Registry:")
	allTables := registry.ListTables()
	fmt.Printf("  Total Tables Registered: %d\n", len(allTables))

	if len(allTables) > 0 {
		fmt.Println("\n  Table Mappings:")
		for _, tableName := range allTables {
			mapping, err := registry.GetTableMapping(tableName)
			if err == nil {
				if mapping.IsGlobal() {
					fmt.Printf("    • %s → Global (mode: %s)\n", tableName, dbInfo["mode"])
				} else {
					fmt.Printf("    • %s → Shard: %s, Role: %s\n",
						tableName, mapping.ShardName(), mapping.Role())
				}
			}
		}
	}

}
