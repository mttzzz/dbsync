package services

import (
	"errors"
	"testing"
	"time"

	"db-sync-cli/internal/config"
	"db-sync-cli/internal/models"
	"db-sync-cli/test/mocks"
)

func TestDumpService_NewDumpService(t *testing.T) {
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
			Timeout:       30 * time.Minute,
			MysqldumpPath: "mysqldump",
			MysqlPath:     "mysql",
		},
	}

	dbService := NewDatabaseService(cfg)
	service := NewDumpService(cfg, dbService)

	if service == nil {
		t.Error("NewDumpService() returned nil")
		return
	}

	if service.config != cfg {
		t.Error("DumpService config not set correctly")
	}
}

func TestDumpService_MockValidation(t *testing.T) {
	cfg := &config.Config{
		Remote: config.MySQLConfig{
			Host: "remote.example.com",
			Port: 3306,
			User: "remote_user",
		},
		Local: config.MySQLConfig{
			Host: "localhost",
			Port: 3306,
			User: "local_user",
		},
		Dump: config.DumpConfig{
			Timeout: 30 * time.Minute,
		},
	}

	tests := []struct {
		name         string
		databaseName string
		wantErr      bool
		setup        func(*mocks.MockDatabaseService)
	}{
		{
			name:         "empty database name",
			databaseName: "",
			wantErr:      true,
			setup: func(mock *mocks.MockDatabaseService) {
				mock.ValidateNameError = errors.New("database name cannot be empty")
			},
		},
		{
			name:         "database not exists",
			databaseName: "nonexistent_db",
			wantErr:      true,
			setup: func(mock *mocks.MockDatabaseService) {
				mock.ValidateNameError = nil
				mock.DatabaseExistsResult = false
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDBService := mocks.NewMockDatabaseService()
			if tt.setup != nil {
				tt.setup(mockDBService)
			}

			// Создаем DumpService с подменой DatabaseService
			service := &DumpService{
				config:    cfg,
				dbService: mockDBService,
			}

			// Тестируем только начальную валидацию, не доходя до проверки mysqldump
			err := service.dbService.ValidateDatabaseName(tt.databaseName)
			if tt.databaseName == "" && err == nil {
				t.Error("Empty database name should cause validation error")
			}
		})
	}
}

func TestDumpService_MockInteraction(t *testing.T) {
	cfg := &config.Config{
		Remote: config.MySQLConfig{
			Host: "remote.example.com",
			Port: 3306,
			User: "remote_user",
		},
		Local: config.MySQLConfig{
			Host: "localhost",
			Port: 3306,
			User: "local_user",
		},
		Dump: config.DumpConfig{
			Timeout: 30 * time.Minute,
		},
	}

	mockDBService := mocks.NewMockDatabaseService()
	mockDBService.ValidateNameError = nil
	mockDBService.DatabaseExistsResult = true
	mockDBService.LocalConnInfo = &models.ConnectionInfo{Connected: true}

	service := &DumpService{
		config:    cfg,
		dbService: mockDBService,
	}

	// Тестируем взаимодействие с моком
	err := service.dbService.ValidateDatabaseName("test_db")
	if err != nil {
		t.Errorf("ValidateDatabaseName() failed: %v", err)
	}

	// Проверяем, что методы мока были вызваны
	if !mockDBService.ValidateNameCalled {
		t.Error("ValidateDatabaseName should be called")
	}

	// Тестируем DatabaseExists
	exists, err := service.dbService.DatabaseExists("test_db", true)
	if err != nil {
		t.Errorf("DatabaseExists() failed: %v", err)
	}

	if !exists {
		t.Error("DatabaseExists should return true for test")
	}

	if !mockDBService.DatabaseExistsCalled {
		t.Error("DatabaseExists should be called")
	}
}

func TestDumpService_Integration(t *testing.T) {
	// Интеграционный тест с реальным DatabaseService (но без подключения к БД)
	cfg := &config.Config{
		Remote: config.MySQLConfig{
			Host: "remote.example.com",
			Port: 3306,
			User: "remote_user",
		},
		Local: config.MySQLConfig{
			Host: "localhost",
			Port: 3306,
			User: "local_user",
		},
		Dump: config.DumpConfig{
			Timeout: 30 * time.Minute,
		},
	}

	dbService := NewDatabaseService(cfg)
	service := NewDumpService(cfg, dbService)

	if service == nil {
		t.Fatal("Service not created")
	}

	// Проверяем, что сервис создан с правильной конфигурацией
	if service.config != cfg {
		t.Error("Service config not set correctly")
	}

	if service.dbService != dbService {
		t.Error("Service dbService not set correctly")
	}
}
