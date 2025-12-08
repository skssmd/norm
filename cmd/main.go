package main

import (
	"fmt"
	"log"

	"github.com/skssmd/norm/core/registry"
)

func main() {
	// ------------------------
	// Global Monolith Scenario
	// ------------------------
	fmt.Println("=== Global Monolith Scenario ===")
	err := registry.Register("postgres://user:pass@primary-db:5432/dbname").Primary()
	if err != nil {
		log.Fatal(err)
	}

	// Add a replica
	err = registry.Register("postgres://user:pass@replica-db:5432/dbname").Replica()
	if err != nil {
		log.Fatal(err)
	}

	// Trying to add Read pool alongside Primary should fail
	err = registry.Register("postgres://user:pass@read-db:5432/dbname").Read()
	if err != nil {
		fmt.Println("Expected error:", err)
	}

	// ------------------------
	// Read/Write Scenario
	// ------------------------
	fmt.Println("=== Read/Write Scenario ===")
	// Create a fresh registry for this example
	regRW := registry.Register("postgres://user:pass@readwrite-db:5432/dbname")
	err = regRW.Write()
	if err != nil {
		log.Fatal(err)
	}

	// Multiple reads allowed
	err = registry.Register("postgres://user:pass@read1-db:5432/dbname").Read()
	if err != nil {
		log.Fatal(err)
	}
	err = registry.Register("postgres://user:pass@read2-db:5432/dbname").Read()
	if err != nil {
		log.Fatal(err)
	}

	// Trying to add another Write should fail
	err = registry.Register("postgres://user:pass@write2-db:5432/dbname").Write()
	if err != nil {
		fmt.Println("Expected error:", err)
	}

	// ------------------------
	// Shard Scenario
	// ------------------------
	fmt.Println("=== Shard Scenario ===")
	// Shard 1 primary
	err = registry.Register("postgres://user:pass@shard1-primary:5432/dbname").Shard("shard1").Primary()
	if err != nil {
		log.Fatal(err)
	}

	// Shard 1 standalone table
	err = registry.Register("postgres://user:pass@shard1-standalone:5432/dbname").Shard("shard1").Standalone()
	if err != nil {
		log.Fatal(err)
	}

	// Shard 2 primary
	err = registry.Register("postgres://user:pass@shard2-primary:5432/dbname").Shard("shard2").Primary()
	if err != nil {
		log.Fatal(err)
	}

	// Trying to register global Primary when shards exist should fail
	err = registry.Register("postgres://user:pass@another-primary:5432/dbname").Primary()
	if err != nil {
		fmt.Println("Expected error:", err)
	}

	// ------------------------
	// Table-level registration
	// ------------------------
	fmt.Println("=== Table-level registration ===")

	// // Example table structs
	// type User struct{}
	// type Order struct{}

	// // Register User table to Shard1
	// norm.Table(User{}).Shard("shard1")
	// norm.Table(User{}).Shard("shard1").Read()
	// norm.Table(User{}).Shard("shard1").Write()
	// norm.Table(User{}).Shard("shard1").Standalone()

	// // Register Order table to Shard2
	// norm.Table(Order{}).Shard("shard2")
	// norm.Table(Order{}).Shard("shard2").Read()
	// norm.Table(Order{}).Shard("shard2").Write()

	// fmt.Println("All registrations completed successfully!")
}
