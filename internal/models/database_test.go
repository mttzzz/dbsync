package models

import (
	"testing"
	"time"
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
