package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDatabase(t *testing.T) {
	tests := []struct {
		name     string
		database Database
		wantName string
		wantSize int64
	}{
		{
			name: "create database",
			database: Database{
				Name:    "test_db",
				Size:    1024,
				Tables:  5,
				Created: time.Now(),
			},
			wantName: "test_db",
			wantSize: 1024,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.database.Name != tt.wantName {
				t.Errorf("Database.Name = %v, want %v", tt.database.Name, tt.wantName)
			}
			if tt.database.Size != tt.wantSize {
				t.Errorf("Database.Size = %v, want %v", tt.database.Size, tt.wantSize)
			}
		})
	}
}

func TestConnectionInfo(t *testing.T) {
	tests := []struct {
		name       string
		connection ConnectionInfo
		wantHost   string
		wantPort   int
	}{
		{
			name: "create connection info",
			connection: ConnectionInfo{
				Host:      "localhost",
				Port:      3306,
				User:      "test_user",
				Connected: true,
				Version:   "8.0.0",
			},
			wantHost: "localhost",
			wantPort: 3306,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.connection.Host != tt.wantHost {
				t.Errorf("ConnectionInfo.Host = %v, want %v", tt.connection.Host, tt.wantHost)
			}
			if tt.connection.Port != tt.wantPort {
				t.Errorf("ConnectionInfo.Port = %v, want %v", tt.connection.Port, tt.wantPort)
			}
		})
	}
}

func TestSyncOptions(t *testing.T) {
	tests := []struct {
		name    string
		options SyncOptions
		wantDB  string
		wantDry bool
	}{
		{
			name: "sync options with dry run",
			options: SyncOptions{
				DatabaseName: "test_db",
				DryRun:       true,
				Force:        false,
				Verbose:      true,
				RemoteHost:   "remote.example.com",
				LocalHost:    "localhost",
			},
			wantDB:  "test_db",
			wantDry: true,
		},
		{
			name: "sync options production",
			options: SyncOptions{
				DatabaseName: "production_db",
				DryRun:       false,
				Force:        true,
				Verbose:      false,
				RemoteHost:   "prod.example.com",
				LocalHost:    "local.example.com",
			},
			wantDB:  "production_db",
			wantDry: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.options.DatabaseName != tt.wantDB {
				t.Errorf("SyncOptions.DatabaseName = %v, want %v", tt.options.DatabaseName, tt.wantDB)
			}
			if tt.options.DryRun != tt.wantDry {
				t.Errorf("SyncOptions.DryRun = %v, want %v", tt.options.DryRun, tt.wantDry)
			}
		})
	}
}

func TestSyncResult(t *testing.T) {
	startTime := time.Now()
	endTime := startTime.Add(5 * time.Minute)

	tests := []struct {
		name        string
		result      SyncResult
		wantSuccess bool
		wantDB      string
	}{
		{
			name: "successful sync",
			result: SyncResult{
				Success:      true,
				DatabaseName: "test_db",
				Duration:     5 * time.Minute,
				DumpSize:     1024000,
				TablesCount:  10,
				StartTime:    startTime,
				EndTime:      endTime,
			},
			wantSuccess: true,
			wantDB:      "test_db",
		},
		{
			name: "failed sync",
			result: SyncResult{
				Success:      false,
				DatabaseName: "failed_db",
				Duration:     1 * time.Minute,
				DumpSize:     0,
				TablesCount:  0,
				Error:        "Connection failed",
				StartTime:    startTime,
				EndTime:      startTime.Add(1 * time.Minute),
			},
			wantSuccess: false,
			wantDB:      "failed_db",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.result.Success != tt.wantSuccess {
				t.Errorf("SyncResult.Success = %v, want %v", tt.result.Success, tt.wantSuccess)
			}
			if tt.result.DatabaseName != tt.wantDB {
				t.Errorf("SyncResult.DatabaseName = %v, want %v", tt.result.DatabaseName, tt.wantDB)
			}
		})
	}
}

func TestDatabaseList(t *testing.T) {
	databases := DatabaseList{
		{Name: "db1", Size: 1024, Tables: 5},
		{Name: "db2", Size: 2048, Tables: 10},
		{Name: "db3", Size: 512, Tables: 2},
	}

	if len(databases) != 3 {
		t.Errorf("DatabaseList length = %v, want %v", len(databases), 3)
	}

	// Проверяем первую БД
	if databases[0].Name != "db1" {
		t.Errorf("databases[0].Name = %v, want %v", databases[0].Name, "db1")
	}
	if databases[0].Size != 1024 {
		t.Errorf("databases[0].Size = %v, want %v", databases[0].Size, 1024)
	}
}

func TestDatabaseList_SortBySize(t *testing.T) {
	databases := DatabaseList{
		{Name: "small_db", Size: 1000, Tables: 5},
		{Name: "large_db", Size: 5000, Tables: 50},
		{Name: "medium_db", Size: 3000, Tables: 20},
	}

	databases.SortBySize()

	// Проверяем, что базы отсортированы по размеру (сначала большие)
	assert.Equal(t, "large_db", databases[0].Name)
	assert.Equal(t, "medium_db", databases[1].Name)
	assert.Equal(t, "small_db", databases[2].Name)

	// Проверяем размеры
	assert.Equal(t, int64(5000), databases[0].Size)
	assert.Equal(t, int64(3000), databases[1].Size)
	assert.Equal(t, int64(1000), databases[2].Size)
}

func TestDatabaseList_SortBySizeAsc(t *testing.T) {
	databases := DatabaseList{
		{Name: "small_db", Size: 1000, Tables: 5},
		{Name: "large_db", Size: 5000, Tables: 50},
		{Name: "medium_db", Size: 3000, Tables: 20},
	}

	databases.SortBySizeAsc()

	// Проверяем, что базы отсортированы по размеру (сначала маленькие)
	assert.Equal(t, "small_db", databases[0].Name)
	assert.Equal(t, "medium_db", databases[1].Name)
	assert.Equal(t, "large_db", databases[2].Name)

	// Проверяем размеры
	assert.Equal(t, int64(1000), databases[0].Size)
	assert.Equal(t, int64(3000), databases[1].Size)
	assert.Equal(t, int64(5000), databases[2].Size)
}

func TestDatabaseList_SortByName(t *testing.T) {
	databases := DatabaseList{
		{Name: "zebra_db", Size: 1000, Tables: 5},
		{Name: "alpha_db", Size: 5000, Tables: 50},
		{Name: "beta_db", Size: 3000, Tables: 20},
	}

	databases.SortByName()

	// Проверяем, что базы отсортированы по имени
	assert.Equal(t, "alpha_db", databases[0].Name)
	assert.Equal(t, "beta_db", databases[1].Name)
	assert.Equal(t, "zebra_db", databases[2].Name)
}

func TestDatabaseList_EmptyList(t *testing.T) {
	var databases DatabaseList

	// Проверяем, что сортировка пустого списка не вызывает панику
	assert.NotPanics(t, func() {
		databases.SortBySize()
	})

	assert.NotPanics(t, func() {
		databases.SortBySizeAsc()
	})

	assert.NotPanics(t, func() {
		databases.SortByName()
	})
}

func TestDatabaseList_SingleItem(t *testing.T) {
	databases := DatabaseList{
		{Name: "single_db", Size: 1000, Tables: 5},
	}

	databases.SortBySize()

	assert.Len(t, databases, 1)
	assert.Equal(t, "single_db", databases[0].Name)
}
