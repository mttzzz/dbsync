package mocks

import (
	"db-sync-cli/internal/models"
)

// MockDatabaseService provides a mock implementation of DatabaseServiceInterface
type MockDatabaseService struct {
	// Поля для управления поведением моков
	TestConnectionError  error
	ListDatabasesError   error
	ValidateNameError    error
	DatabaseExistsError  error
	GetDatabaseInfoError error

	RemoteConnInfo       *models.ConnectionInfo
	LocalConnInfo        *models.ConnectionInfo
	DatabaseList         models.DatabaseList
	DatabaseExistsResult bool
	DatabaseInfo         *models.Database

	// Поля для отслеживания вызовов
	TestConnectionCalled  bool
	ListDatabasesCalled   bool
	ValidateNameCalled    bool
	DatabaseExistsCalled  bool
	GetDatabaseInfoCalled bool

	LastIsRemote      bool
	LastDatabaseName  string
	LastValidatedName string
}

// TestConnection имитирует тестирование подключения
func (m *MockDatabaseService) TestConnection(isRemote bool) (*models.ConnectionInfo, error) {
	m.TestConnectionCalled = true
	m.LastIsRemote = isRemote

	if m.TestConnectionError != nil {
		return nil, m.TestConnectionError
	}

	if isRemote && m.RemoteConnInfo != nil {
		return m.RemoteConnInfo, nil
	}

	if !isRemote && m.LocalConnInfo != nil {
		return m.LocalConnInfo, nil
	}

	// Значения по умолчанию
	return &models.ConnectionInfo{
		Host:      "localhost",
		Port:      3306,
		User:      "root",
		Connected: true,
	}, nil
}

// ListDatabases имитирует получение списка баз данных
func (m *MockDatabaseService) ListDatabases(isRemote bool) (models.DatabaseList, error) {
	m.ListDatabasesCalled = true
	m.LastIsRemote = isRemote

	if m.ListDatabasesError != nil {
		return models.DatabaseList{}, m.ListDatabasesError
	}

	if len(m.DatabaseList) > 0 {
		return m.DatabaseList, nil
	}

	// Значения по умолчанию
	return models.DatabaseList{
		{Name: "test_db1", Size: 1024000, Tables: 5},
		{Name: "test_db2", Size: 2048000, Tables: 10},
	}, nil
}

// ValidateDatabaseName имитирует валидацию имени базы данных
func (m *MockDatabaseService) ValidateDatabaseName(name string) error {
	m.ValidateNameCalled = true
	m.LastValidatedName = name
	return m.ValidateNameError
}

// DatabaseExists имитирует проверку существования базы данных
func (m *MockDatabaseService) DatabaseExists(name string, isRemote bool) (bool, error) {
	m.DatabaseExistsCalled = true
	m.LastDatabaseName = name
	m.LastIsRemote = isRemote

	if m.DatabaseExistsError != nil {
		return false, m.DatabaseExistsError
	}

	return m.DatabaseExistsResult, nil
}

// GetDatabaseInfo имитирует получение информации о базе данных
func (m *MockDatabaseService) GetDatabaseInfo(name string, isRemote bool) (*models.Database, error) {
	m.GetDatabaseInfoCalled = true
	m.LastDatabaseName = name
	m.LastIsRemote = isRemote

	if m.GetDatabaseInfoError != nil {
		return nil, m.GetDatabaseInfoError
	}

	if m.DatabaseInfo != nil {
		return m.DatabaseInfo, nil
	}

	// Значения по умолчанию
	return &models.Database{
		Name:   name,
		Size:   1024000,
		Tables: 5,
	}, nil
}

// Reset сбрасывает состояние мока для нового теста
func (m *MockDatabaseService) Reset() {
	m.TestConnectionError = nil
	m.ListDatabasesError = nil
	m.ValidateNameError = nil
	m.DatabaseExistsError = nil
	m.GetDatabaseInfoError = nil
	m.RemoteConnInfo = nil
	m.LocalConnInfo = nil
	m.DatabaseList = models.DatabaseList{}
	m.DatabaseExistsResult = false
	m.DatabaseInfo = nil
	m.TestConnectionCalled = false
	m.ListDatabasesCalled = false
	m.ValidateNameCalled = false
	m.DatabaseExistsCalled = false
	m.GetDatabaseInfoCalled = false
	m.LastIsRemote = false
	m.LastDatabaseName = ""
	m.LastValidatedName = ""
}

// SetupSuccessfulConnection настраивает мок для успешного подключения
func (m *MockDatabaseService) SetupSuccessfulConnection() {
	m.RemoteConnInfo = &models.ConnectionInfo{
		Host:      "remote.example.com",
		Port:      3306,
		User:      "root",
		Connected: true,
	}
	m.LocalConnInfo = &models.ConnectionInfo{
		Host:      "localhost",
		Port:      3306,
		User:      "root",
		Connected: true,
	}
}

// SetupDatabaseList настраивает мок с тестовыми базами данных
func (m *MockDatabaseService) SetupDatabaseList() {
	m.DatabaseList = models.DatabaseList{
		{Name: "production_db", Size: 5242880, Tables: 15},
		{Name: "staging_db", Size: 2097152, Tables: 8},
		{Name: "test_db", Size: 1048576, Tables: 5},
	}
}

// NewMockDatabaseService создает новый мок сервиса базы данных
func NewMockDatabaseService() *MockDatabaseService {
	return &MockDatabaseService{}
}
