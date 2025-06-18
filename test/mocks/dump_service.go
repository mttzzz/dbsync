package mocks

import (
	"db-sync-cli/internal/models"
)

// MockDumpService provides a mock implementation of DumpServiceInterface
type MockDumpService struct {
	// Поля для управления поведением моков
	ValidateOperationError error
	CreateDumpError        error
	RestoreDumpError       error
	CreateDumpResult       *models.SyncResult
	RestoreDumpResult      *models.SyncResult

	// Поля для отслеживания вызовов
	ValidateOperationCalled    bool
	CreateDumpCalled           bool
	RestoreDumpCalled          bool
	ValidateOperationCallCount int
	CreateDumpCallCount        int
	RestoreDumpCallCount       int
	LastDatabaseName           string
	LastSyncOptions            *models.SyncOptions

	// Поля для совместимости с legacy тестами
	CreatePlanError      error
	ExecutePlanError     error
	Plan                 *models.SyncResult
	ExecuteResult        *models.SyncResult
	CreatePlanCalled     bool
	ExecutePlanCalled    bool
	CreatePlanCallCount  int
	ExecutePlanCallCount int
	LastOptions          *models.SyncOptions
}

// ValidateDumpOperation имитирует валидацию операции дампа
func (m *MockDumpService) ValidateDumpOperation(databaseName string) error {
	m.ValidateOperationCalled = true
	m.ValidateOperationCallCount++
	m.LastDatabaseName = databaseName
	return m.ValidateOperationError
}

// CreateDump имитирует создание дампа
func (m *MockDumpService) CreateDump(options *models.SyncOptions) (*models.SyncResult, error) {
	m.CreateDumpCalled = true
	m.CreateDumpCallCount++
	m.LastSyncOptions = options

	if m.CreateDumpError != nil {
		return nil, m.CreateDumpError
	}

	if m.CreateDumpResult != nil {
		return m.CreateDumpResult, nil
	}

	return &models.SyncResult{
		Success:      true,
		DatabaseName: options.DatabaseName,
		DumpSize:     1024000,
		TablesCount:  10,
	}, nil
}

// RestoreDump имитирует восстановление дампа
func (m *MockDumpService) RestoreDump(options *models.SyncOptions) (*models.SyncResult, error) {
	m.RestoreDumpCalled = true
	m.RestoreDumpCallCount++
	m.LastSyncOptions = options

	if m.RestoreDumpError != nil {
		return nil, m.RestoreDumpError
	}

	if m.RestoreDumpResult != nil {
		return m.RestoreDumpResult, nil
	}

	return &models.SyncResult{
		Success:      true,
		DatabaseName: options.DatabaseName,
		DumpSize:     1024000,
		TablesCount:  10,
	}, nil
}

// Legacy методы для совместимости с существующими тестами
func (m *MockDumpService) CreatePlan(options *models.SyncOptions) (*models.SyncResult, error) {
	m.CreatePlanCalled = true
	m.CreatePlanCallCount++
	m.LastOptions = options

	if m.CreatePlanError != nil {
		return nil, m.CreatePlanError
	}

	if m.Plan != nil {
		return m.Plan, nil
	}

	return &models.SyncResult{
		Success:      true,
		DatabaseName: options.DatabaseName,
		DumpSize:     1024000,
		TablesCount:  10,
	}, nil
}

func (m *MockDumpService) ExecutePlan(options *models.SyncOptions) (*models.SyncResult, error) {
	m.ExecutePlanCalled = true
	m.ExecutePlanCallCount++
	m.LastOptions = options

	if m.ExecutePlanError != nil {
		return nil, m.ExecutePlanError
	}

	if m.ExecuteResult != nil {
		return m.ExecuteResult, nil
	}

	return &models.SyncResult{
		Success:      true,
		DatabaseName: options.DatabaseName,
		DumpSize:     1024000,
		TablesCount:  10,
	}, nil
}

// Reset сбрасывает состояние мока для нового теста
func (m *MockDumpService) Reset() {
	m.ValidateOperationError = nil
	m.CreateDumpError = nil
	m.RestoreDumpError = nil
	m.CreateDumpResult = nil
	m.RestoreDumpResult = nil
	m.ValidateOperationCalled = false
	m.CreateDumpCalled = false
	m.RestoreDumpCalled = false
	m.ValidateOperationCallCount = 0
	m.CreateDumpCallCount = 0
	m.RestoreDumpCallCount = 0
	m.LastDatabaseName = ""
	m.LastSyncOptions = nil

	// Reset legacy поля
	m.CreatePlanError = nil
	m.ExecutePlanError = nil
	m.Plan = nil
	m.ExecuteResult = nil
	m.CreatePlanCalled = false
	m.ExecutePlanCalled = false
	m.CreatePlanCallCount = 0
	m.ExecutePlanCallCount = 0
	m.LastOptions = nil
}

// NewMockDumpService создает новый мок сервиса дампа
func NewMockDumpService() *MockDumpService {
	return &MockDumpService{}
}
