package services

import (
	"database/sql"
	"fmt"
	"strings"

	"db-sync-cli/internal/config"
	"db-sync-cli/internal/models"

	_ "github.com/go-sql-driver/mysql"
)

// DatabaseService предоставляет функции для работы с MySQL
type DatabaseService struct {
	config *config.Config
}

// NewDatabaseService создает новый экземпляр DatabaseService
func NewDatabaseService(cfg *config.Config) *DatabaseService {
	return &DatabaseService{
		config: cfg,
	}
}

// TestConnection проверяет подключение к серверу MySQL
func (ds *DatabaseService) TestConnection(isRemote bool) (*models.ConnectionInfo, error) {
	var mysqlConfig config.MySQLConfig
	var label string

	if isRemote {
		mysqlConfig = ds.config.Remote
		label = "remote"
	} else {
		mysqlConfig = ds.config.Local
		label = "local"
	}

	connInfo := &models.ConnectionInfo{
		Host: mysqlConfig.Host,
		Port: mysqlConfig.Port,
		User: mysqlConfig.User,
	}

	// Подключаемся к серверу (без указания конкретной БД)
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=utf8mb4&parseTime=True&loc=Local&timeout=10s",
		mysqlConfig.User, mysqlConfig.Password, mysqlConfig.Host, mysqlConfig.Port)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		connInfo.Error = fmt.Sprintf("failed to open connection: %v", err)
		return connInfo, err
	}
	defer db.Close()

	// Проверяем подключение
	if err := db.Ping(); err != nil {
		connInfo.Error = fmt.Sprintf("failed to ping %s server: %v", label, err)
		return connInfo, err
	}

	// Получаем версию MySQL
	var version string
	if err := db.QueryRow("SELECT VERSION()").Scan(&version); err != nil {
		connInfo.Error = fmt.Sprintf("failed to get version: %v", err)
		return connInfo, err
	}

	connInfo.Connected = true
	connInfo.Version = version

	return connInfo, nil
}

// ListDatabases возвращает список баз данных на сервере
func (ds *DatabaseService) ListDatabases(isRemote bool) (models.DatabaseList, error) {
	var mysqlConfig config.MySQLConfig

	if isRemote {
		mysqlConfig = ds.config.Remote
	} else {
		mysqlConfig = ds.config.Local
	}

	// Подключаемся к серверу
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=utf8mb4&parseTime=True&loc=Local&timeout=10s",
		mysqlConfig.User, mysqlConfig.Password, mysqlConfig.Host, mysqlConfig.Port)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping server: %w", err)
	}

	// Получаем список БД, исключая системные
	query := `
		SELECT 
			SCHEMA_NAME as db_name,
			COALESCE(
				(SELECT ROUND(SUM(data_length + index_length)) 
				 FROM information_schema.TABLES 
				 WHERE TABLE_SCHEMA = SCHEMA_NAME), 0
			) as db_size,
			COALESCE(
				(SELECT COUNT(*) 
				 FROM information_schema.TABLES 
				 WHERE TABLE_SCHEMA = SCHEMA_NAME), 0
			) as tables_count
		FROM information_schema.SCHEMATA 
		WHERE SCHEMA_NAME NOT IN ('information_schema', 'performance_schema', 'mysql', 'sys')
		ORDER BY SCHEMA_NAME
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query databases: %w", err)
	}
	defer rows.Close()

	var databases models.DatabaseList

	for rows.Next() {
		var dbName string
		var dbSize sql.NullInt64
		var tablesCount sql.NullInt32

		if err := rows.Scan(&dbName, &dbSize, &tablesCount); err != nil {
			return nil, fmt.Errorf("failed to scan database row: %w", err)
		}

		database := models.Database{
			Name:   dbName,
			Size:   dbSize.Int64,
			Tables: int(tablesCount.Int32),
		}

		databases = append(databases, database)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating database rows: %w", err)
	}

	return databases, nil
}

// DatabaseExists проверяет существование базы данных
func (ds *DatabaseService) DatabaseExists(databaseName string, isRemote bool) (bool, error) {
	var mysqlConfig config.MySQLConfig

	if isRemote {
		mysqlConfig = ds.config.Remote
	} else {
		mysqlConfig = ds.config.Local
	}

	// Подключаемся к серверу
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=utf8mb4&parseTime=True&loc=Local&timeout=10s",
		mysqlConfig.User, mysqlConfig.Password, mysqlConfig.Host, mysqlConfig.Port)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return false, fmt.Errorf("failed to open connection: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return false, fmt.Errorf("failed to ping server: %w", err)
	}

	var count int
	query := "SELECT COUNT(*) FROM information_schema.SCHEMATA WHERE SCHEMA_NAME = ?"
	if err := db.QueryRow(query, databaseName).Scan(&count); err != nil {
		return false, fmt.Errorf("failed to check database existence: %w", err)
	}

	return count > 0, nil
}

// GetDatabaseInfo возвращает детальную информацию о базе данных
func (ds *DatabaseService) GetDatabaseInfo(databaseName string, isRemote bool) (*models.Database, error) {
	var mysqlConfig config.MySQLConfig

	if isRemote {
		mysqlConfig = ds.config.Remote
	} else {
		mysqlConfig = ds.config.Local
	}

	// Подключаемся к серверу
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=utf8mb4&parseTime=True&loc=Local&timeout=10s",
		mysqlConfig.User, mysqlConfig.Password, mysqlConfig.Host, mysqlConfig.Port)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping server: %w", err)
	}
	// Получаем информацию о БД
	query := `
		SELECT 
			SCHEMA_NAME as db_name,
			COALESCE(
				(SELECT ROUND(SUM(data_length + index_length)) 
				 FROM information_schema.TABLES 
				 WHERE TABLE_SCHEMA = ?), 0
			) as db_size,
			COALESCE(
				(SELECT COUNT(*) 
				 FROM information_schema.TABLES 
				 WHERE TABLE_SCHEMA = ?), 0
			) as tables_count
		FROM information_schema.SCHEMATA 
		WHERE SCHEMA_NAME = ?
	`

	var dbName string
	var dbSize sql.NullInt64
	var tablesCount sql.NullInt32

	if err := db.QueryRow(query, databaseName, databaseName, databaseName).Scan(&dbName, &dbSize, &tablesCount); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("database '%s' not found", databaseName)
		}
		return nil, fmt.Errorf("failed to get database info: %w", err)
	}

	database := &models.Database{
		Name:   dbName,
		Size:   dbSize.Int64,
		Tables: int(tablesCount.Int32),
	}

	return database, nil
}

// ValidateDatabaseName проверяет корректность имени базы данных
func (ds *DatabaseService) ValidateDatabaseName(name string) error {
	if name == "" {
		return fmt.Errorf("database name cannot be empty")
	}

	if len(name) > 64 {
		return fmt.Errorf("database name too long (max 64 characters)")
	}

	// Проверяем на запрещенные символы
	if strings.ContainsAny(name, " \t\n\r/\\") {
		return fmt.Errorf("database name contains invalid characters")
	}

	// Проверяем что имя не является системной БД
	systemDatabases := []string{"information_schema", "performance_schema", "mysql", "sys"}
	for _, sysDB := range systemDatabases {
		if strings.EqualFold(name, sysDB) {
			return fmt.Errorf("cannot sync system database '%s'", name)
		}
	}

	return nil
}

// FormatSize форматирует размер в байтах в человекочитаемый формат
func FormatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
