package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"
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
					ProxyURL: "",
				},
				Local: MySQLConfig{
					Host:     "localhost",
					Port:     3306,
					User:     "local_user",
					Password: "local_pass",
					ProxyURL: "",
				},
				Dump: DumpConfig{
					Timeout:          5 * time.Minute,
					NetworkCompress:  true,
					NetworkZstdLevel: 7,
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
			name: "remote proxy configuration",
			envVars: map[string]string{
				"DBSYNC_REMOTE_HOST":      "remote.example.com",
				"DBSYNC_REMOTE_USER":      "remote_user",
				"DBSYNC_REMOTE_PASSWORD":  "remote_pass",
				"DBSYNC_REMOTE_PROXY_URL": "socks5://proxy.example.com:1080",
				"DBSYNC_LOCAL_HOST":       "localhost",
				"DBSYNC_LOCAL_USER":       "local_user",
				"DBSYNC_LOCAL_PASSWORD":   "local_pass",
			},
			expected: &Config{
				Remote: MySQLConfig{
					Host:     "remote.example.com",
					Port:     3306,
					User:     "remote_user",
					Password: "remote_pass",
					ProxyURL: "socks5://proxy.example.com:1080",
				},
				Local: MySQLConfig{
					Host:     "localhost",
					Port:     3306,
					User:     "local_user",
					Password: "local_pass",
					ProxyURL: "",
				},
				Dump: DumpConfig{
					Timeout:          5 * time.Minute,
					Threads:          8,
					Compress:         true,
					NetworkCompress:  true,
					NetworkZstdLevel: 7,
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
				if tt.expected.Remote.ProxyURL != got.Remote.ProxyURL {
					t.Errorf("Remote.ProxyURL = %v, want %v", got.Remote.ProxyURL, tt.expected.Remote.ProxyURL)
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
					Timeout:          30 * time.Minute,
					NetworkZstdLevel: 7,
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
			name: "invalid proxy URL",
			config: &Config{
				Remote: MySQLConfig{
					Host:     "remote.example.com",
					Port:     3306,
					User:     "remote_user",
					Password: "remote_pass",
					ProxyURL: "proxy.example.com:1080",
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

func TestConfig_ToEnvString(t *testing.T) {
	cfg := &Config{
		Remote: MySQLConfig{
			Host:     "remote.example.com",
			Port:     3306,
			User:     "remote_user",
			Password: "remote pass",
			ProxyURL: "socks5://proxy.example.com:1080",
		},
		Local: MySQLConfig{
			Host:     "127.0.0.1",
			Port:     3307,
			User:     "local_user",
			Password: "local_pass",
		},
		Dump: DumpConfig{
			Timeout:          90 * time.Second,
			Threads:          12,
			Compress:         true,
			NetworkCompress:  true,
			NetworkZstdLevel: 9,
		},
		CLI: CLIConfig{
			DefaultCharset:     "utf8mb4",
			InteractiveMode:    true,
			ConfirmDestructive: false,
		},
		Log: LogConfig{
			Level:  "debug",
			Format: "json",
		},
	}

	content, err := cfg.ToEnvString()
	if err != nil {
		t.Fatalf("ToEnvString() error = %v", err)
	}

	assertContains := func(needle string) {
		t.Helper()
		if !strings.Contains(content, needle) {
			t.Fatalf("env content does not contain %q\n%s", needle, content)
		}
	}

	assertContains("# Remote MySQL")
	assertContains("DBSYNC_REMOTE_HOST=remote.example.com")
	assertContains("DBSYNC_REMOTE_PASSWORD=\"remote pass\"")
	assertContains("DBSYNC_DUMP_THREADS=12")
	assertContains("DBSYNC_DUMP_NETWORK_COMPRESS=true")
	assertContains("DBSYNC_DUMP_NETWORK_ZSTD_LEVEL=9")
	assertContains("DBSYNC_LOG_FORMAT=json")
}

func TestConfig_SaveEnvRoundTrip(t *testing.T) {
	tempDir := t.TempDir()
	envPath := filepath.Join(tempDir, ".env")

	cfg := &Config{
		Remote: MySQLConfig{
			Host:     "prod.example.com",
			Port:     3308,
			User:     "prod_user",
			Password: "prod_pass",
			ProxyURL: "http://proxy.example.com:8080",
		},
		Local: MySQLConfig{
			Host:     "localhost",
			Port:     3306,
			User:     "root",
			Password: "root_pass",
		},
		Dump: DumpConfig{
			Timeout:          10 * time.Minute,
			Threads:          6,
			Compress:         false,
			NetworkCompress:  true,
			NetworkZstdLevel: 11,
		},
		CLI: CLIConfig{
			DefaultCharset:     "utf8mb4",
			InteractiveMode:    true,
			ConfirmDestructive: true,
		},
		Log: LogConfig{
			Level:  "warn",
			Format: "text",
		},
	}

	if err := cfg.SaveEnv(envPath); err != nil {
		t.Fatalf("SaveEnv() error = %v", err)
	}

	values, err := godotenv.Read(envPath)
	if err != nil {
		t.Fatalf("godotenv.Read() error = %v", err)
	}

	if values["DBSYNC_REMOTE_HOST"] != cfg.Remote.Host {
		t.Fatalf("remote host = %q, want %q", values["DBSYNC_REMOTE_HOST"], cfg.Remote.Host)
	}
	if values["DBSYNC_DUMP_THREADS"] != "6" {
		t.Fatalf("dump threads = %q, want %q", values["DBSYNC_DUMP_THREADS"], "6")
	}
	if values["DBSYNC_DUMP_COMPRESS"] != "false" {
		t.Fatalf("dump compress = %q, want %q", values["DBSYNC_DUMP_COMPRESS"], "false")
	}
	if values["DBSYNC_DUMP_NETWORK_COMPRESS"] != "true" {
		t.Fatalf("dump network compress = %q, want %q", values["DBSYNC_DUMP_NETWORK_COMPRESS"], "true")
	}
	if values["DBSYNC_DUMP_NETWORK_ZSTD_LEVEL"] != "11" {
		t.Fatalf("dump network zstd level = %q, want %q", values["DBSYNC_DUMP_NETWORK_ZSTD_LEVEL"], "11")
	}
	if values["DBSYNC_LOG_LEVEL"] != "warn" {
		t.Fatalf("log level = %q, want %q", values["DBSYNC_LOG_LEVEL"], "warn")
	}

	stat, err := os.Stat(envPath)
	if err != nil {
		t.Fatalf("os.Stat() error = %v", err)
	}
	if stat.Mode().Perm() != 0600 {
		t.Fatalf("file mode = %v, want 0600", stat.Mode().Perm())
	}
}

// Вспомогательные функции
func clearEnvVars() {
	envVars := []string{
		"DBSYNC_REMOTE_HOST",
		"DBSYNC_REMOTE_PORT",
		"DBSYNC_REMOTE_USER",
		"DBSYNC_REMOTE_PASSWORD",
		"DBSYNC_REMOTE_PROXY_URL",
		"DBSYNC_LOCAL_HOST",
		"DBSYNC_LOCAL_PORT",
		"DBSYNC_LOCAL_USER",
		"DBSYNC_LOCAL_PASSWORD",
		"DBSYNC_LOCAL_PROXY_URL",
		"DBSYNC_DUMP_TIMEOUT",
		"DBSYNC_DUMP_THREADS",
		"DBSYNC_DUMP_COMPRESS",
		"DBSYNC_DUMP_NETWORK_COMPRESS",
		"DBSYNC_DUMP_NETWORK_ZSTD_LEVEL",
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
