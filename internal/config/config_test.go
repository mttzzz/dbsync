package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
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
					Timeout:       5 * time.Minute,
					TempDir:       "./tmp",
					MysqldumpPath: "mysqldump",
					MysqlPath:     "mysql",
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
		{
			name: "custom ports and paths",
			envVars: map[string]string{
				"DBSYNC_REMOTE_HOST":         "remote.example.com",
				"DBSYNC_REMOTE_PORT":         "3307",
				"DBSYNC_REMOTE_USER":         "remote_user",
				"DBSYNC_REMOTE_PASSWORD":     "remote_pass",
				"DBSYNC_LOCAL_HOST":          "localhost",
				"DBSYNC_LOCAL_PORT":          "3308",
				"DBSYNC_LOCAL_USER":          "local_user",
				"DBSYNC_LOCAL_PASSWORD":      "local_pass",
				"DBSYNC_DUMP_MYSQLDUMP_PATH": "/custom/path/mysqldump",
				"DBSYNC_DUMP_MYSQL_PATH":     "/custom/path/mysql",
			},
			expected: &Config{
				Remote: MySQLConfig{
					Host:     "remote.example.com",
					Port:     3307,
					User:     "remote_user",
					Password: "remote_pass",
				},
				Local: MySQLConfig{
					Host:     "localhost",
					Port:     3308,
					User:     "local_user",
					Password: "local_pass",
				},
				Dump: DumpConfig{
					Timeout:       5 * time.Minute,
					TempDir:       "./tmp",
					MysqldumpPath: "/custom/path/mysqldump",
					MysqlPath:     "/custom/path/mysql",
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

			got, err := Load()
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				compareConfigs(t, tt.expected, got)
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
					Timeout:       30 * time.Minute,
					TempDir:       os.TempDir(),
					MysqldumpPath: "mysqldump",
					MysqlPath:     "mysql",
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
		{
			name: "invalid port",
			config: &Config{
				Remote: MySQLConfig{
					Host:     "remote.example.com",
					Port:     -1,
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
		"DBSYNC_DUMP_TEMP_DIR",
		"DBSYNC_DUMP_MYSQLDUMP_PATH",
		"DBSYNC_DUMP_MYSQL_PATH",
		"DBSYNC_CLI_DEFAULT_CHARSET",
		"DBSYNC_CLI_INTERACTIVE_MODE",
		"DBSYNC_CLI_CONFIRM_DESTRUCTIVE",
		"DBSYNC_LOG_LEVEL",
		"DBSYNC_LOG_FORMAT",
	}
	for _, envVar := range envVars {
		os.Unsetenv(envVar)
	}
}

func compareConfigs(t *testing.T, expected, got *Config) {
	if expected.Remote.Host != got.Remote.Host {
		t.Errorf("Remote.Host = %v, want %v", got.Remote.Host, expected.Remote.Host)
	}
	if expected.Remote.Port != got.Remote.Port {
		t.Errorf("Remote.Port = %v, want %v", got.Remote.Port, expected.Remote.Port)
	}
	if expected.Remote.User != got.Remote.User {
		t.Errorf("Remote.User = %v, want %v", got.Remote.User, expected.Remote.User)
	}
	if expected.Remote.Password != got.Remote.Password {
		t.Errorf("Remote.Password = %v, want %v", got.Remote.Password, expected.Remote.Password)
	}

	if expected.Local.Host != got.Local.Host {
		t.Errorf("Local.Host = %v, want %v", got.Local.Host, expected.Local.Host)
	}
	if expected.Local.Port != got.Local.Port {
		t.Errorf("Local.Port = %v, want %v", got.Local.Port, expected.Local.Port)
	}
	if expected.Local.User != got.Local.User {
		t.Errorf("Local.User = %v, want %v", got.Local.User, expected.Local.User)
	}
	if expected.Local.Password != got.Local.Password {
		t.Errorf("Local.Password = %v, want %v", got.Local.Password, expected.Local.Password)
	}

	if expected.Dump.Timeout != got.Dump.Timeout {
		t.Errorf("Dump.Timeout = %v, want %v", got.Dump.Timeout, expected.Dump.Timeout)
	}
	if expected.Dump.TempDir != got.Dump.TempDir {
		t.Errorf("Dump.TempDir = %v, want %v", got.Dump.TempDir, expected.Dump.TempDir)
	}
	if expected.Dump.MysqldumpPath != got.Dump.MysqldumpPath {
		t.Errorf("Dump.MysqldumpPath = %v, want %v", got.Dump.MysqldumpPath, expected.Dump.MysqldumpPath)
	}
	if expected.Dump.MysqlPath != got.Dump.MysqlPath {
		t.Errorf("Dump.MysqlPath = %v, want %v", got.Dump.MysqlPath, expected.Dump.MysqlPath)
	}

	if expected.CLI.DefaultCharset != got.CLI.DefaultCharset {
		t.Errorf("CLI.DefaultCharset = %v, want %v", got.CLI.DefaultCharset, expected.CLI.DefaultCharset)
	}
	if expected.CLI.InteractiveMode != got.CLI.InteractiveMode {
		t.Errorf("CLI.InteractiveMode = %v, want %v", got.CLI.InteractiveMode, expected.CLI.InteractiveMode)
	}
	if expected.CLI.ConfirmDestructive != got.CLI.ConfirmDestructive {
		t.Errorf("CLI.ConfirmDestructive = %v, want %v", got.CLI.ConfirmDestructive, expected.CLI.ConfirmDestructive)
	}

	if expected.Log.Level != got.Log.Level {
		t.Errorf("Log.Level = %v, want %v", got.Log.Level, expected.Log.Level)
	}
	if expected.Log.Format != got.Log.Format {
		t.Errorf("Log.Format = %v, want %v", got.Log.Format, expected.Log.Format)
	}
}
