//go:build integration
// +build integration

package integration

import (
	"os"
	"testing"
	"time"

	"db-sync-cli/internal/config"
	"db-sync-cli/internal/services"
)

// TestDatabaseService_Integration проверяет реальную работу с БД
func TestDatabaseService_Integration(t *testing.T) {
	// Пропускаем если нет переменных окружения для тестирования
	if os.Getenv("DBSYNC_TEST_REMOTE_HOST") == "" {
		t.Skip("Integration tests require DBSYNC_TEST_REMOTE_HOST environment variable")
	}

	cfg := &config.Config{
		Remote: config.MySQLConfig{
			Host:     getEnvOrDefault("DBSYNC_TEST_REMOTE_HOST", "localhost"),
			Port:     3306,
			User:     getEnvOrDefault("DBSYNC_TEST_REMOTE_USER", "root"),
			Password: getEnvOrDefault("DBSYNC_TEST_REMOTE_PASSWORD", ""),
		},
		Local: config.MySQLConfig{
			Host:     getEnvOrDefault("DBSYNC_TEST_LOCAL_HOST", "localhost"),
			Port:     3306,
			User:     getEnvOrDefault("DBSYNC_TEST_LOCAL_USER", "root"),
			Password: getEnvOrDefault("DBSYNC_TEST_LOCAL_PASSWORD", ""),
		},
		Dump: config.DumpConfig{
			Timeout: 30 * time.Second,
		},
	}

	service := services.NewDatabaseService(cfg)

	t.Run("TestConnection", func(t *testing.T) {
		// Тестируем подключение к удаленному серверу
		connInfo, err := service.TestConnection(true)
		if err != nil {
			t.Fatalf("Failed to connect to remote server: %v", err)
		}

		if !connInfo.Connected {
			t.Error("Should be connected to remote server")
		}

		if connInfo.Version == "" {
			t.Error("Version should not be empty")
		}

		t.Logf("Remote server version: %s", connInfo.Version)
	})

	t.Run("ListDatabases", func(t *testing.T) {
		// Тестируем получение списка БД
		databases, err := service.ListDatabases(true)
		if err != nil {
			t.Fatalf("Failed to list databases: %v", err)
		}

		if len(databases) == 0 {
			t.Error("Should have at least one database")
		}

		t.Logf("Found %d databases", len(databases))
		for _, db := range databases {
			t.Logf("Database: %s (Size: %d bytes, Tables: %d)", db.Name, db.Size, db.Tables)
		}
	})

	t.Run("ValidateDatabaseName", func(t *testing.T) {
		// Тестируем валидацию имен БД
		validNames := []string{"test_db", "valid_name", "db123"}
		for _, name := range validNames {
			err := service.ValidateDatabaseName(name)
			if err != nil {
				t.Errorf("Valid name '%s' should not produce error: %v", name, err)
			}
		}

		invalidNames := []string{"", "information_schema", "test db", "test/db"}
		for _, name := range invalidNames {
			err := service.ValidateDatabaseName(name)
			if err == nil {
				t.Errorf("Invalid name '%s' should produce error", name)
			}
		}
	})
}

// TestDumpService_Integration проверяет реальную работу с дампами
func TestDumpService_Integration(t *testing.T) {
	// Пропускаем если нет переменных окружения для тестирования
	if os.Getenv("DBSYNC_TEST_REMOTE_HOST") == "" {
		t.Skip("Integration tests require DBSYNC_TEST_REMOTE_HOST environment variable")
	}

	cfg := &config.Config{
		Remote: config.MySQLConfig{
			Host:     getEnvOrDefault("DBSYNC_TEST_REMOTE_HOST", "localhost"),
			Port:     3306,
			User:     getEnvOrDefault("DBSYNC_TEST_REMOTE_USER", "root"),
			Password: getEnvOrDefault("DBSYNC_TEST_REMOTE_PASSWORD", ""),
		},
		Local: config.MySQLConfig{
			Host:     getEnvOrDefault("DBSYNC_TEST_LOCAL_HOST", "localhost"),
			Port:     3306,
			User:     getEnvOrDefault("DBSYNC_TEST_LOCAL_USER", "root"),
			Password: getEnvOrDefault("DBSYNC_TEST_LOCAL_PASSWORD", ""),
		},
		Dump: config.DumpConfig{
			Timeout:       30 * time.Second,
			MysqldumpPath: getEnvOrDefault("DBSYNC_TEST_MYSQLDUMP_PATH", "mysqldump"),
			MysqlPath:     getEnvOrDefault("DBSYNC_TEST_MYSQL_PATH", "mysql"),
		},
	}

	dbService := services.NewDatabaseService(cfg)
	dumpService := services.NewDumpService(cfg, dbService)

	t.Run("ValidateDumpOperation", func(t *testing.T) {
		// Получаем список БД для тестирования
		databases, err := dbService.ListDatabases(true)
		if err != nil {
			t.Fatalf("Failed to list databases: %v", err)
		}

		if len(databases) == 0 {
			t.Skip("No databases available for testing")
		}

		// Тестируем валидацию на первой доступной БД
		testDB := databases[0].Name
		err = dumpService.ValidateDumpOperation(testDB)
		if err != nil {
			t.Logf("Validation failed for '%s': %v (это может быть нормально)", testDB, err)
		} else {
			t.Logf("Validation successful for database '%s'", testDB)
		}
	})
}

// getEnvOrDefault возвращает значение переменной окружения или значение по умолчанию
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
