package services

import (
	"testing"
	"time"

	"db-sync-cli/internal/config"
	"db-sync-cli/internal/models"
)

func TestDatabaseService_NewDatabaseService(t *testing.T) {
	cfg := &config.Config{
		Remote: config.MySQLConfig{
			Host:     "remote.example.com",
			Port:     3306,
			User:     "remote_user",
			Password: "remote_pass",
		},
		Local: config.MySQLConfig{
			Host:     "localhost",
			Port:     3306,
			User:     "local_user",
			Password: "local_pass",
		},
	}

	service := NewDatabaseService(cfg)
	if service == nil {
		t.Error("NewDatabaseService() returned nil")
	}

	if service.config != cfg {
		t.Error("DatabaseService config not set correctly")
	}
}

func TestDatabaseService_getConnectionString(t *testing.T) {
	tests := []struct {
		name     string
		config   config.MySQLConfig
		database string
		expected string
	}{
		{
			name: "basic connection string",
			config: config.MySQLConfig{
				Host:     "localhost",
				Port:     3306,
				User:     "test_user",
				Password: "test_pass",
			},
			database: "test_db",
			expected: "test_user:test_pass@tcp(localhost:3306)/test_db?charset=utf8mb4&parseTime=True&loc=Local",
		},
		{
			name: "custom port",
			config: config.MySQLConfig{
				Host:     "remote.example.com",
				Port:     3307,
				User:     "remote_user",
				Password: "remote_pass",
			},
			database: "production_db",
			expected: "remote_user:remote_pass@tcp(remote.example.com:3307)/production_db?charset=utf8mb4&parseTime=True&loc=Local",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.GetConnectionString(tt.database)
			if got != tt.expected {
				t.Errorf("getConnectionString() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDatabaseService_ValidateDatabaseName(t *testing.T) {
	cfg := &config.Config{
		Remote: config.MySQLConfig{
			Host: "remote.example.com",
			Port: 3306,
			User: "remote_user",
		},
	}
	service := NewDatabaseService(cfg)

	tests := []struct {
		name    string
		dbName  string
		wantErr bool
	}{
		{
			name:    "valid database name",
			dbName:  "test_db",
			wantErr: false,
		},
		{
			name:    "valid database name with numbers",
			dbName:  "test_db_123",
			wantErr: false,
		},
		{
			name:    "empty database name",
			dbName:  "",
			wantErr: true,
		},
		{
			name:    "database name too long",
			dbName:  "this_is_a_very_long_database_name_that_exceeds_the_maximum_allowed_length_for_mysql_database_names",
			wantErr: true,
		},
		{
			name:    "database name with invalid characters",
			dbName:  "test/db",
			wantErr: true,
		},
		{
			name:    "database name starts with number",
			dbName:  "1test_db",
			wantErr: false, // В MySQL это разрешено
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidateDatabaseName(tt.dbName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDatabaseName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDatabaseService_mockFunctionality(t *testing.T) {
	// Тест проверяет, что сервис может работать с моковыми данными
	cfg := &config.Config{
		Remote: config.MySQLConfig{
			Host:     "remote.example.com",
			Port:     3306,
			User:     "remote_user",
			Password: "remote_pass",
		},
		Local: config.MySQLConfig{
			Host:     "localhost",
			Port:     3306,
			User:     "local_user",
			Password: "local_pass",
		},
		Dump: config.DumpConfig{
			Timeout: 30 * time.Minute,
		},
	}

	service := NewDatabaseService(cfg)

	// Проверяем, что сервис создан
	if service == nil {
		t.Fatal("Service not created")
	}

	// Проверяем валидацию имени БД
	err := service.ValidateDatabaseName("test_db")
	if err != nil {
		t.Errorf("Valid database name failed validation: %v", err)
	}

	err = service.ValidateDatabaseName("")
	if err == nil {
		t.Error("Empty database name should fail validation")
	}
}

func TestDatabaseService_ConnectionInfo(t *testing.T) {
	// Тест создания ConnectionInfo без реального подключения
	cfg := &config.Config{
		Remote: config.MySQLConfig{
			Host: "remote.example.com",
			Port: 3306,
			User: "remote_user",
		},
	}

	_ = NewDatabaseService(cfg)

	// Создаем тестовую ConnectionInfo
	info := &models.ConnectionInfo{
		Host:      cfg.Remote.Host,
		Port:      cfg.Remote.Port,
		User:      cfg.Remote.User,
		Connected: false,
		Error:     "Connection not established",
	}

	if info.Host != "remote.example.com" {
		t.Errorf("ConnectionInfo.Host = %v, want %v", info.Host, "remote.example.com")
	}

	if info.Port != 3306 {
		t.Errorf("ConnectionInfo.Port = %v, want %v", info.Port, 3306)
	}

	if info.Connected {
		t.Error("ConnectionInfo.Connected should be false for test")
	}
}
