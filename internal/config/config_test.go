package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	// Очищаем все переменные окружения перед тестами
	clearEnvVars()
	defer clearEnvVars()

	tests := []struct {
		name     string
		envVars  map[string]string
		expected *Config
		wantErr  bool
	}{
		{
			name: "default configuration",
			envVars: map[string]string{
				"DBSYNC_REMOTE_HOST":     "remote.example.com",
				"DBSYNC_REMOTE_USER":     "remote_user",
				"DBSYNC_REMOTE_PASSWORD": "remote_pass",
				"DBSYNC_LOCAL_HOST":      "localhost",
				"DBSYNC_LOCAL_USER":      "local_user",
				"DBSYNC_LOCAL_PASSWORD":  "local_pass",
			},
			expected: &Config{
				Remote: MySQLConfig{
					Host:     "remote.example.com",
					Port:     3306,
					User:     "remote_user",
					Password: "remote_pass",
				},
				Local: MySQLConfig{
					Host:     "localhost",
					Port:     3306,
					User:     "local_user",
					Password: "local_pass",
				},
				Dump: DumpConfig{
					Timeout: 5 * time.Minute,
				},
				CLI: CLIConfig{
					DefaultCharset:     "utf8mb4",
					InteractiveMode:    true,
					ConfirmDestructive: true,
				},
				Log: LogConfig{
					Level:  "info",
					Format: "text",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Очищаем переменные окружения
			clearEnvVars()

			// Устанавливаем тестовые переменные окружения
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			got, err := LoadForTest()
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadForTest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Проверяем основные поля
				if tt.expected.Remote.Host != got.Remote.Host {
					t.Errorf("Remote.Host = %v, want %v", got.Remote.Host, tt.expected.Remote.Host)
				}
				if tt.expected.Remote.Port != got.Remote.Port {
					t.Errorf("Remote.Port = %v, want %v", got.Remote.Port, tt.expected.Remote.Port)
				}
				if tt.expected.Local.Host != got.Local.Host {
					t.Errorf("Local.Host = %v, want %v", got.Local.Host, tt.expected.Local.Host)
				}
				if tt.expected.Dump.Timeout != got.Dump.Timeout {
					t.Errorf("Dump.Timeout = %v, want %v", got.Dump.Timeout, tt.expected.Dump.Timeout)
				}
			}

			// Очищаем переменные окружения после теста
			clearEnvVars()
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid configuration",
			config: &Config{
				Remote: MySQLConfig{
					Host:     "remote.example.com",
					Port:     3306,
					User:     "remote_user",
					Password: "remote_pass",
				},
				Local: MySQLConfig{
					Host:     "localhost",
					Port:     3306,
					User:     "local_user",
					Password: "local_pass",
				},
				Dump: DumpConfig{
					Timeout: 30 * time.Minute,
				},
			},
			wantErr: false,
		},
		{
			name: "missing remote host",
			config: &Config{
				Remote: MySQLConfig{
					Host:     "",
					Port:     3306,
					User:     "remote_user",
					Password: "remote_pass",
				},
				Local: MySQLConfig{
					Host:     "localhost",
					Port:     3306,
					User:     "local_user",
					Password: "local_pass",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Вспомогательные функции
func clearEnvVars() {
	envVars := []string{
		"DBSYNC_REMOTE_HOST",
		"DBSYNC_REMOTE_PORT",
		"DBSYNC_REMOTE_USER",
		"DBSYNC_REMOTE_PASSWORD",
		"DBSYNC_LOCAL_HOST",
		"DBSYNC_LOCAL_PORT",
		"DBSYNC_LOCAL_USER",
		"DBSYNC_LOCAL_PASSWORD",
		"DBSYNC_DUMP_TIMEOUT",
		"DBSYNC_DUMP_THREADS",
		"DBSYNC_DUMP_COMPRESS",
		"DBSYNC_CLI_DEFAULT_CHARSET",
		"DBSYNC_CLI_INTERACTIVE_MODE",
		"DBSYNC_CLI_CONFIRM_DESTRUCTIVE",
		"DBSYNC_LOG_LEVEL",
		"DBSYNC_LOG_FORMAT",
	}
	for _, envVar := range envVars {
		os.Unsetenv(envVar)
	}

	// Также очищаем рабочую директорию от .env файлов для тестов
	os.Remove(".env")
}
