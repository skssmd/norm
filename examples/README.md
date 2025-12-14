# Examples

This directory contains practical examples demonstrating various features of Norm ORM.

## Available Examples

### 1. Basic CRUD Operations
- **File**: `basic_crud/main.go`
- **Description**: Simple INSERT, SELECT, UPDATE, DELETE operations
- **Run**: `cd basic_crud && go run main.go`

### 2. JOIN Queries
- **File**: `joins/main.go`
- **Description**: Native, App-Side, and Distributed JOINs with struct scanning
- **Run**: `cd joins && go run main.go`

### 3. Bulk Operations
- **File**: `bulk_operations/main.go`
- **Description**: Bulk inserts and upserts
- **Run**: `cd bulk_operations && go run main.go`

### 4. Sharding
- **File**: `sharding/main.go`
- **Description**: Multi-shard setup with distributed queries
- **Run**: `cd sharding && go run main.go`

### 5. Complete Application
- **File**: `complete_app/main.go`
- **Description**: Full-featured application demonstrating all Norm capabilities
- **Run**: `cd complete_app && go run main.go`

## Prerequisites

All examples require:
- PostgreSQL database running
- Environment variables set (see `.env.example` in root)

## Setup

1. Copy `.env.example` to `.env` in the root directory
2. Update database connection strings
3. Run any example:

```bash
cd examples/basic_crud
go run main.go
```

## Notes

- Examples automatically create and drop tables
- Each example is self-contained
- Check comments in code for detailed explanations
