package ui

import (
	"fmt"
	"testing"
	"time"

	"db-sync-cli/internal/models"

	"github.com/stretchr/testify/assert"
)

func TestNewFormatter(t *testing.T) {
	formatter := NewFormatter()
	assert.NotNil(t, formatter)
	assert.IsType(t, &Formatter{}, formatter)
}

func TestFormatDatabaseList(t *testing.T) {
	formatter := NewFormatter()

	tests := []struct {
		name         string
		databases    models.DatabaseList
		serverName   string
		expectedText []string
	}{
		{
			name:         "empty database list",
			databases:    models.DatabaseList{},
			serverName:   "localhost",
			expectedText: []string{"No databases found"},
		},
		{
			name: "single database",
			databases: models.DatabaseList{
				{Name: "test_db", Size: 1024, Tables: 5},
			},
			serverName:   "remote",
			expectedText: []string{"Available databases on remote", "test_db", "5"},
		},
		{
			name: "multiple databases",
			databases: models.DatabaseList{
				{Name: "db1", Size: 1024, Tables: 3},
				{Name: "db2", Size: 2048, Tables: 7},
			},
			serverName:   "production",
			expectedText: []string{"Available databases on production", "db1", "db2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.FormatDatabaseList(tt.databases, tt.serverName)

			for _, expected := range tt.expectedText {
				assert.Contains(t, result, expected)
			}
		})
	}
}

func TestFormatConnectionStatus(t *testing.T) {
	formatter := NewFormatter()

	tests := []struct {
		name         string
		connInfo     *models.ConnectionInfo
		label        string
		expectedText []string
	}{
		{
			name: "successful connection",
			connInfo: &models.ConnectionInfo{
				Host:      "localhost",
				Port:      3306,
				User:      "root",
				Connected: true,
				Version:   "8.0.30",
			},
			label:        "Local",
			expectedText: []string{"Local", "localhost:3306", "root", "8.0.30"},
		},
		{
			name: "failed connection",
			connInfo: &models.ConnectionInfo{
				Host:      "remote.db.com",
				Port:      3306,
				User:      "user",
				Connected: false,
				Error:     "Connection refused",
			},
			label:        "Remote",
			expectedText: []string{"Remote", "remote.db.com:3306", "Connection refused"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.FormatConnectionStatus(tt.connInfo, tt.label)

			for _, expected := range tt.expectedText {
				assert.Contains(t, result, expected)
			}
		})
	}
}

func TestFormatSyncPlan(t *testing.T) {
	formatter := NewFormatter()

	tests := []struct {
		name         string
		db           *models.Database
		willReplace  bool
		dryRun       bool
		expectedText []string
	}{
		{
			name: "dry run mode",
			db: &models.Database{
				Name:   "test_db",
				Size:   1024 * 1024,
				Tables: 10,
			},
			willReplace:  false,
			dryRun:       true,
			expectedText: []string{"test_db", "1.0 MB", "10", "DRY RUN"},
		},
		{
			name: "replace existing database",
			db: &models.Database{
				Name:   "prod_db",
				Size:   5 * 1024 * 1024,
				Tables: 25,
			},
			willReplace:  true,
			dryRun:       false,
			expectedText: []string{"prod_db", "5.0 MB", "25", "REPLACED"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.FormatSyncPlan(tt.db, tt.willReplace, tt.dryRun)

			for _, expected := range tt.expectedText {
				assert.Contains(t, result, expected)
			}
		})
	}
}

func TestFormatSyncResult(t *testing.T) {
	formatter := NewFormatter()

	tests := []struct {
		name         string
		result       *models.SyncResult
		expectedText []string
	}{
		{
			name: "successful sync",
			result: &models.SyncResult{
				Success:      true,
				DatabaseName: "test_db",
				Duration:     2 * time.Minute,
				DumpSize:     1024 * 1024, // 1MB
				TablesCount:  15,
				StartTime:    time.Now().Add(-2 * time.Minute),
				EndTime:      time.Now(),
			},
			expectedText: []string{"Success", "test_db", "2m 0s", "1.0 MB", "15"},
		},
		{
			name: "failed sync",
			result: &models.SyncResult{
				Success:      false,
				DatabaseName: "fail_db",
				Duration:     30 * time.Second,
				Error:        "Connection timeout",
				StartTime:    time.Now().Add(-30 * time.Second),
				EndTime:      time.Now(),
			},
			expectedText: []string{"Failed", "fail_db", "Connection timeout"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.FormatSyncResult(tt.result)

			for _, expected := range tt.expectedText {
				assert.Contains(t, result, expected)
			}
		})
	}
}

func TestFormatConfirmation(t *testing.T) {
	formatter := NewFormatter()

	tests := []struct {
		name         string
		message      string
		expectedText []string
	}{
		{
			name:         "simple confirmation",
			message:      "Are you sure you want to proceed?",
			expectedText: []string{"CONFIRMATION REQUIRED", "Are you sure you want to proceed?", "Type 'yes'"},
		},
		{
			name:         "dangerous operation",
			message:      "This will delete all data in the database!",
			expectedText: []string{"CONFIRMATION REQUIRED", "delete all data", "Type 'yes'"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.FormatConfirmation(tt.message)

			for _, expected := range tt.expectedText {
				assert.Contains(t, result, expected)
			}
		})
	}
}

func TestFormatTable(t *testing.T) {
	tests := []struct {
		name         string
		headers      []string
		rows         [][]string
		expectedText []string
	}{
		{
			name:         "simple table",
			headers:      []string{"Name", "Size"},
			rows:         [][]string{{"db1", "1KB"}, {"db2", "2KB"}},
			expectedText: []string{"Name", "Size", "db1", "1KB", "db2", "2KB"},
		},
		{
			name:         "empty table",
			headers:      []string{"Col1", "Col2"},
			rows:         [][]string{},
			expectedText: []string{}, // пустая таблица возвращает пустую строку
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderTable(tt.headers, tt.rows)

			if len(tt.expectedText) == 0 {
				assert.Empty(t, result)
			} else {
				assert.NotEmpty(t, result)
				for _, expected := range tt.expectedText {
					assert.Contains(t, result, expected)
				}
			}
		})
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		name     string
		size     int64
		expected string
	}{
		{name: "zero bytes", size: 0, expected: "0 B"},
		{name: "bytes", size: 512, expected: "512 B"},
		{name: "kilobytes", size: 1024, expected: "1.0 KB"},
		{name: "megabytes", size: 1024 * 1024, expected: "1.0 MB"},
		{name: "gigabytes", size: 1024 * 1024 * 1024, expected: "1.0 GB"},
		{name: "large value", size: 1536 * 1024 * 1024, expected: "1.5 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatSize(tt.size)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{name: "seconds", duration: 30 * time.Second, expected: "30.0s"},
		{name: "minutes", duration: 2 * time.Minute, expected: "2m 0s"},
		{name: "hours", duration: time.Hour + 30*time.Minute, expected: "1h 30m"},
		{name: "zero", duration: 0, expected: "0ms"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDuration(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStylesExist(t *testing.T) {
	// Проверяем, что стили определены и не вызывают панику
	tests := []func() string{
		func() string { return SuccessStyle.Render("test") },
		func() string { return ErrorStyle.Render("test") },
		func() string { return WarningStyle.Render("test") },
		func() string { return InfoStyle.Render("test") },
		func() string { return HighlightStyle.Render("test") },
		func() string { return MutedStyle.Render("test") },
	}

	for i, styleFunc := range tests {
		t.Run(fmt.Sprintf("style_%d", i), func(t *testing.T) {
			assert.NotPanics(t, func() {
				result := styleFunc()
				assert.NotEmpty(t, result)
				assert.Contains(t, result, "test")
			})
		})
	}
}
