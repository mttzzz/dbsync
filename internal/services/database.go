package services

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"db-sync-cli/internal/config"
	"db-sync-cli/internal/models"

	_ "github.com/go-sql-driver/mysql"
)

// DatabaseService предоставляет функции для работы с MySQL
type DatabaseService struct {
	config *config.Config
}

const (
	exactRowCountConcurrency = 4
	exactRowCountTimeout     = 1500 * time.Millisecond
)

// NewDatabaseService создает новый экземпляр DatabaseService
func NewDatabaseService(cfg *config.Config) *DatabaseService {
	return &DatabaseService{
		config: cfg,
	}
}

func (ds *DatabaseService) mysqlConfig(isRemote bool) config.MySQLConfig {
	if isRemote {
		return ds.config.Remote
	}

	return ds.config.Local
}

func (ds *DatabaseService) openConnection(isRemote bool, database string) (*sql.DB, func(), error) {
	mysqlConfig := ds.mysqlConfig(isRemote)
	host := mysqlConfig.Host
	port := mysqlConfig.Port
	cleanup := func() {}

	if isRemote && mysqlConfig.HasProxy() {
		tunnel, err := newProxyTunnel(mysqlConfig)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to start proxy tunnel: %w", err)
		}

		host = tunnel.Host()
		port = tunnel.Port()
		cleanup = func() {
			_ = tunnel.Close()
		}
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&timeout=10s",
		mysqlConfig.User, mysqlConfig.Password, host, port, database)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		cleanup()
		return nil, nil, err
	}

	return db, cleanup, nil
}

// TestConnection проверяет подключение к серверу MySQL
func (ds *DatabaseService) TestConnection(isRemote bool) (*models.ConnectionInfo, error) {
	var label string
	mysqlConfig := ds.mysqlConfig(isRemote)

	if isRemote {
		label = "remote"
	} else {
		label = "local"
	}

	connInfo := &models.ConnectionInfo{
		Host: mysqlConfig.Host,
		Port: mysqlConfig.Port,
		User: mysqlConfig.User,
	}

	db, cleanup, err := ds.openConnection(isRemote, "")
	if err != nil {
		connInfo.Error = fmt.Sprintf("failed to open connection: %v", err)
		return connInfo, err
	}
	defer cleanup()
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
	db, cleanup, err := ds.openConnection(isRemote, "")
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}
	defer cleanup()
	defer db.Close()

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping server: %w", err)
	}

	// Получаем список БД, исключая системные
	query := `
		SELECT 
			SCHEMA_NAME as db_name,
			COALESCE(
				(SELECT ROUND(SUM(data_length)) 
				 FROM information_schema.TABLES 
				 WHERE TABLE_SCHEMA = SCHEMA_NAME), 0
			) as data_size,
			COALESCE(
				(SELECT ROUND(SUM(index_length)) 
				 FROM information_schema.TABLES 
				 WHERE TABLE_SCHEMA = SCHEMA_NAME), 0
			) as index_size,
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
		var dataSize sql.NullInt64
		var indexSize sql.NullInt64
		var dbSize sql.NullInt64
		var tablesCount sql.NullInt32

		if err := rows.Scan(&dbName, &dataSize, &indexSize, &dbSize, &tablesCount); err != nil {
			return nil, fmt.Errorf("failed to scan database row: %w", err)
		}

		database := models.Database{
			Name:      dbName,
			Size:      dbSize.Int64,
			DataSize:  dataSize.Int64,
			IndexSize: indexSize.Int64,
			Tables:    int(tablesCount.Int32),
		}

		databases = append(databases, database)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating database rows: %w", err)
	}

	return databases, nil
}

// ListTables возвращает список таблиц для указанной базы данных.
func (ds *DatabaseService) ListTables(databaseName string, isRemote bool) ([]models.Table, error) {
	db, cleanup, err := ds.openConnection(isRemote, "")
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}
	defer cleanup()
	defer db.Close()

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping server: %w", err)
	}

	query := `
		SELECT
			TABLE_SCHEMA,
			TABLE_NAME,
			COALESCE(ROUND(DATA_LENGTH), 0) AS data_size,
			COALESCE(ROUND(INDEX_LENGTH), 0) AS index_size,
			COALESCE(ROUND(DATA_LENGTH + INDEX_LENGTH), 0) AS table_size,
			COALESCE(TABLE_ROWS, 0) AS table_rows,
			COALESCE(ENGINE, ''),
			COALESCE(TABLE_COLLATION, ''),
			COALESCE(DATA_FREE, 0)
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = ?
		ORDER BY data_size DESC, TABLE_NAME ASC
	`

	rows, err := db.Query(query, databaseName)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	tables := make([]models.Table, 0)
	for rows.Next() {
		var table models.Table
		if err := rows.Scan(
			&table.DatabaseName,
			&table.Name,
			&table.DataSize,
			&table.IndexSize,
			&table.Size,
			&table.Rows,
			&table.Engine,
			&table.Collation,
			&table.DataFree,
		); err != nil {
			return nil, fmt.Errorf("failed to scan table row: %w", err)
		}
		table.RowsApprox = true
		tables = append(tables, table)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating table rows: %w", err)
	}

	ds.enrichExactTableRows(db, databaseName, tables)

	return tables, nil
}

func (ds *DatabaseService) enrichExactTableRows(db *sql.DB, databaseName string, tables []models.Table) {
	if len(tables) == 0 {
		return
	}

	workerCount := exactRowCountConcurrency
	if len(tables) < workerCount {
		workerCount = len(tables)
	}

	indices := make(chan int)
	var waitGroup sync.WaitGroup
	for worker := 0; worker < workerCount; worker++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			for index := range indices {
				count, ok := ds.exactTableRowCount(db, databaseName, tables[index].Name)
				if !ok {
					continue
				}
				tables[index].Rows = count
				tables[index].RowsApprox = false
			}
		}()
	}

	for index := range tables {
		indices <- index
	}
	close(indices)
	waitGroup.Wait()
}

func (ds *DatabaseService) exactTableRowCount(db *sql.DB, databaseName string, tableName string) (int64, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), exactRowCountTimeout)
	defer cancel()

	query := fmt.Sprintf("SELECT COUNT(*) FROM %s.%s", quoteIdentifier(databaseName), quoteIdentifier(tableName))
	var count int64
	if err := db.QueryRowContext(ctx, query).Scan(&count); err != nil {
		return 0, false
	}

	return count, true
}

func quoteIdentifier(value string) string {
	return "`" + strings.ReplaceAll(value, "`", "``") + "`"
}

// ListTableDependencies возвращает внешние зависимости для выбранных таблиц.
func (ds *DatabaseService) ListTableDependencies(databaseName string, tableNames []string, isRemote bool) ([]models.TableDependency, error) {
	if len(tableNames) == 0 {
		return nil, nil
	}

	db, cleanup, err := ds.openConnection(isRemote, "")
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}
	defer cleanup()
	defer db.Close()

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping server: %w", err)
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(tableNames)), ",")
	query := fmt.Sprintf(`
		SELECT
			TABLE_SCHEMA,
			TABLE_NAME,
			REFERENCED_TABLE_NAME,
			CONSTRAINT_NAME
		FROM information_schema.KEY_COLUMN_USAGE
		WHERE TABLE_SCHEMA = ?
		  AND REFERENCED_TABLE_NAME IS NOT NULL
		  AND TABLE_NAME IN (%s)
		ORDER BY TABLE_NAME, REFERENCED_TABLE_NAME, CONSTRAINT_NAME
	`, placeholders)

	args := make([]any, 0, len(tableNames)+1)
	args = append(args, databaseName)
	for _, tableName := range tableNames {
		args = append(args, tableName)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query table dependencies: %w", err)
	}
	defer rows.Close()

	dependencies := make([]models.TableDependency, 0)
	for rows.Next() {
		var dependency models.TableDependency
		if err := rows.Scan(
			&dependency.DatabaseName,
			&dependency.TableName,
			&dependency.ReferencedTable,
			&dependency.ConstraintName,
		); err != nil {
			return nil, fmt.Errorf("failed to scan dependency row: %w", err)
		}
		dependencies = append(dependencies, dependency)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating dependency rows: %w", err)
	}

	sort.Slice(dependencies, func(i, j int) bool {
		if dependencies[i].TableName == dependencies[j].TableName {
			if dependencies[i].ReferencedTable == dependencies[j].ReferencedTable {
				return dependencies[i].ConstraintName < dependencies[j].ConstraintName
			}
			return dependencies[i].ReferencedTable < dependencies[j].ReferencedTable
		}
		return dependencies[i].TableName < dependencies[j].TableName
	})

	return dependencies, nil
}

// DatabaseExists проверяет существование базы данных
func (ds *DatabaseService) DatabaseExists(databaseName string, isRemote bool) (bool, error) {
	db, cleanup, err := ds.openConnection(isRemote, "")
	if err != nil {
		return false, fmt.Errorf("failed to open connection: %w", err)
	}
	defer cleanup()
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
	db, cleanup, err := ds.openConnection(isRemote, "")
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}
	defer cleanup()
	defer db.Close()

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping server: %w", err)
	}
	// Получаем информацию о БД
	query := `
		SELECT 
			SCHEMA_NAME as db_name,
			COALESCE(
				(SELECT ROUND(SUM(data_length)) 
				 FROM information_schema.TABLES 
				 WHERE TABLE_SCHEMA = ?), 0
			) as data_size,
			COALESCE(
				(SELECT ROUND(SUM(index_length)) 
				 FROM information_schema.TABLES 
				 WHERE TABLE_SCHEMA = ?), 0
			) as index_size,
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
	var dataSize sql.NullInt64
	var indexSize sql.NullInt64
	var dbSize sql.NullInt64
	var tablesCount sql.NullInt32

	if err := db.QueryRow(query, databaseName, databaseName, databaseName, databaseName, databaseName).Scan(&dbName, &dataSize, &indexSize, &dbSize, &tablesCount); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("database '%s' not found", databaseName)
		}
		return nil, fmt.Errorf("failed to get database info: %w", err)
	}

	database := &models.Database{
		Name:      dbName,
		Size:      dbSize.Int64,
		DataSize:  dataSize.Int64,
		IndexSize: indexSize.Int64,
		Tables:    int(tablesCount.Int32),
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
