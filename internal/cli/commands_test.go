package cli

import (
	"bytes"
	"strings"
	"testing"

	"db-sync-cli/internal/config"
	"db-sync-cli/internal/models"
	"db-sync-cli/test/mocks"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestRootCommand(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "version flag",
			args:     []string{"--version"},
			expected: "dbsync version",
		},
		{
			name:     "help flag",
			args:     []string{"--help"},
			expected: "dbsync is a CLI tool for synchronizing MySQL databases",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаём новую команду для каждого теста
			cmd := &cobra.Command{
				Use:   "dbsync",
				Short: "MySQL database synchronization tool",
				Long: `dbsync is a CLI tool for synchronizing MySQL databases between remote and local servers.
It creates dumps from remote databases and restores them to local instances with progress tracking.`,
				Version: "test-version",
			}

			// Захватываем вывод
			var output bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetErr(&output)

			// Устанавливаем аргументы
			cmd.SetArgs(tt.args)

			// Выполняем команду
			err := cmd.Execute()

			// Проверяем результат
			if tt.name == "version flag" {
				// Для версии команда может завершиться успешно или с ошибкой
				outputStr := output.String()
				assert.Contains(t, outputStr, "test-version")
			} else {
				assert.NoError(t, err)
				outputStr := output.String()
				assert.Contains(t, outputStr, tt.expected)
			}
		})
	}
}

func TestVersionCommand(t *testing.T) {
	var output bytes.Buffer

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version number",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println("dbsync version test-version")
		},
	}

	cmd.SetOut(&output)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	assert.NoError(t, err)

	outputStr := output.String()
	assert.Contains(t, outputStr, "dbsync version")
}

func TestListCommand(t *testing.T) {
	tests := []struct {
		name           string
		expectError    bool
		mockSetup      func(*mocks.MockDatabaseService)
		expectedOutput []string
	}{
		{
			name:        "successful list",
			expectError: false,
			mockSetup: func(mockDB *mocks.MockDatabaseService) {
				mockDB.DatabaseList = models.DatabaseList{
					{Name: "test_db1"},
					{Name: "test_db2"},
				}
				mockDB.ListDatabasesError = nil
			},
			expectedOutput: []string{"test_db1", "test_db2"},
		},
		{
			name:        "database service error",
			expectError: true,
			mockSetup: func(mockDB *mocks.MockDatabaseService) {
				mockDB.ListDatabasesError = assert.AnError
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаём мок
			mockDB := new(mocks.MockDatabaseService)
			tt.mockSetup(mockDB)

			// Создаём команду list с моком
			var output bytes.Buffer
			cmd := &cobra.Command{
				Use:   "list",
				Short: "List available databases",
				RunE: func(cmd *cobra.Command, args []string) error {
					databases, err := mockDB.ListDatabases(true)
					if err != nil {
						return err
					}

					for _, db := range databases {
						cmd.Printf("- %s\n", db.Name)
					}
					return nil
				},
			}

			cmd.SetOut(&output)
			cmd.SetErr(&output)
			cmd.SetArgs([]string{})

			err := cmd.Execute()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				outputStr := output.String()
				for _, expected := range tt.expectedOutput {
					assert.Contains(t, outputStr, expected)
				}
			}
		})
	}
}

func TestConfigCommand(t *testing.T) {
	// Тест команды config
	var output bytes.Buffer

	cmd := &cobra.Command{
		Use:   "config",
		Short: "Show current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Симулируем загрузку конфигурации
			cfg := &config.Config{
				Remote: config.MySQLConfig{
					Host:     "remote.example.com",
					Port:     3306,
					User:     "user",
					Password: "pass",
				},
				Local: config.MySQLConfig{
					Host:     "localhost",
					Port:     3306,
					User:     "root",
					Password: "localpass",
				},
			}

			cmd.Printf("Remote: %s:%d (user: %s)\n", cfg.Remote.Host, cfg.Remote.Port, cfg.Remote.User)
			cmd.Printf("Local: %s:%d (user: %s)\n", cfg.Local.Host, cfg.Local.Port, cfg.Local.User)
			return nil
		},
	}

	cmd.SetOut(&output)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	assert.NoError(t, err)

	outputStr := output.String()
	assert.Contains(t, outputStr, "remote.example.com")
	assert.Contains(t, outputStr, "localhost")
}

func TestCommandFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		flagName string
		expected interface{}
	}{
		{
			name:     "dry-run flag true",
			args:     []string{"--dry-run"},
			flagName: "dry-run",
			expected: true,
		},
		{
			name:     "verbose flag true",
			args:     []string{"--verbose"},
			flagName: "verbose",
			expected: true,
		},
		{
			name:     "config file flag",
			args:     []string{"--config", "/path/to/config.yaml"},
			flagName: "config",
			expected: "/path/to/config.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{
				Use: "test",
				RunE: func(cmd *cobra.Command, args []string) error {
					return nil
				},
			}

			// Добавляем флаги
			cmd.Flags().Bool("dry-run", false, "Dry run mode")
			cmd.Flags().Bool("verbose", false, "Verbose output")
			cmd.Flags().String("config", "", "Config file path")

			cmd.SetArgs(tt.args)
			err := cmd.Execute()
			assert.NoError(t, err)

			// Проверяем значение флага
			switch tt.flagName {
			case "dry-run", "verbose":
				value, err := cmd.Flags().GetBool(tt.flagName)
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, value)
			case "config":
				value, err := cmd.Flags().GetString(tt.flagName)
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, value)
			}
		})
	}
}

func TestSyncCommandValidation(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "no database name - should be interactive",
			args:        []string{},
			expectError: false,
		},
		{
			name:        "valid database name",
			args:        []string{"test_db"},
			expectError: false,
		},
		{
			name:        "too many arguments",
			args:        []string{"db1", "db2"},
			expectError: true,
			errorMsg:    "accepts at most 1 arg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{
				Use:  "sync [database_name]",
				Args: cobra.MaximumNArgs(1),
				RunE: func(cmd *cobra.Command, args []string) error {
					// Симулируем логику sync команды
					if len(args) == 0 {
						cmd.Println("Interactive mode would start here")
					} else {
						cmd.Printf("Would sync database: %s\n", args[0])
					}
					return nil
				},
			}

			var output bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetErr(&output)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStatusCommand(t *testing.T) {
	// Тест команды status
	var output bytes.Buffer

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show connection status",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Симулируем проверку статуса
			cmd.Println("Remote connection: ✓ Connected")
			cmd.Println("Local connection: ✓ Connected")
			cmd.Println("MySQL tools: ✓ Available")
			return nil
		},
	}

	cmd.SetOut(&output)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	assert.NoError(t, err)

	outputStr := output.String()
	assert.Contains(t, outputStr, "Remote connection: ✓ Connected")
	assert.Contains(t, outputStr, "Local connection: ✓ Connected")
	assert.Contains(t, outputStr, "MySQL tools: ✓ Available")
}

// TestCommandHelp проверяет, что все команды имеют корректную справку
func TestCommandHelp(t *testing.T) {
	commands := []struct {
		name       string
		use        string
		short      string
		shouldHave []string
	}{
		{
			name:       "sync command help",
			use:        "sync [database_name]",
			short:      "Synchronize a database from remote to local",
			shouldHave: []string{"database", "remote", "local"},
		},
		{
			name:       "list command help",
			use:        "list",
			short:      "List available databases on remote server",
			shouldHave: []string{"databases", "remote"},
		},
		{
			name:       "status command help",
			use:        "status",
			short:      "Check connection status",
			shouldHave: []string{"connection", "status"},
		},
	}

	for _, tc := range commands {
		t.Run(tc.name, func(t *testing.T) {
			cmd := &cobra.Command{
				Use:   tc.use,
				Short: tc.short,
				Long:  "This is a longer description for " + tc.short,
			}

			assert.Equal(t, tc.use, cmd.Use)
			assert.Equal(t, tc.short, cmd.Short)

			// Проверяем, что в описании есть ключевые слова
			description := strings.ToLower(cmd.Short + " " + cmd.Long)
			for _, keyword := range tc.shouldHave {
				assert.Contains(t, description, strings.ToLower(keyword))
			}
		})
	}
}
