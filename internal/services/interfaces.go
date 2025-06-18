package services

import (
	"db-sync-cli/internal/models"
)

// DatabaseServiceInterface определяет интерфейс для работы с базами данных
type DatabaseServiceInterface interface {
	TestConnection(isRemote bool) (*models.ConnectionInfo, error)
	ListDatabases(isRemote bool) (models.DatabaseList, error)
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
