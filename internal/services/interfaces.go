package services

import (
	"db-sync-cli/internal/models"
)

// DatabaseServiceInterface определяет интерфейс для работы с базами данных
type DatabaseServiceInterface interface {
	TestConnection(isRemote bool) (*models.ConnectionInfo, error)
	ListDatabases(isRemote bool) (models.DatabaseList, error)
	ListTables(databaseName string, isRemote bool) ([]models.Table, error)
	ListTableDependencies(databaseName string, tableNames []string, isRemote bool) ([]models.TableDependency, error)
	ValidateDatabaseName(name string) error
	DatabaseExists(name string, isRemote bool) (bool, error)
	GetDatabaseInfo(name string, isRemote bool) (*models.Database, error)
}

// DumpServiceInterface определяет интерфейс для работы с дампами
type DumpServiceInterface interface {
	ValidateDumpOperation(databaseName string) error
	CreateDump(options *models.SyncOptions) (*models.SyncResult, error)
	RestoreDump(options *models.SyncOptions) (*models.SyncResult, error)
}

// SyncServiceInterface определяет event-driven контракт выполнения синхронизации.
type SyncServiceInterface interface {
	ExecutePlan(plan *models.SyncPlan, runtime models.RuntimeOptions, observer models.ProgressObserver) ([]models.SyncResult, error)
}
