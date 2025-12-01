package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config содержит всю конфигурацию приложения
type Config struct {
	// Настройки удаленного MySQL сервера
	Remote MySQLConfig `mapstructure:"remote"`

	// Настройки локального MySQL сервера
	Local MySQLConfig `mapstructure:"local"`

	// Настройки дампа
	Dump DumpConfig `mapstructure:"dump"`

	// Настройки CLI
	CLI CLIConfig `mapstructure:"cli"`

	// Настройки логирования
	Log LogConfig `mapstructure:"log"`
}

// MySQLConfig содержит настройки подключения к MySQL
type MySQLConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
}

// DumpConfig содержит настройки для создания дампов
type DumpConfig struct {
	Timeout time.Duration `mapstructure:"timeout"`

	// MyDumper настройки
	MyDumperImage string `mapstructure:"mydumper_image"`
	Threads       int    `mapstructure:"threads"`
	ChunkSize     int    `mapstructure:"chunk_size"` // Размер чанка в строках
	Compress      bool   `mapstructure:"compress"`
}

// CLIConfig содержит настройки CLI интерфейса
type CLIConfig struct {
	DefaultCharset     string `mapstructure:"default_charset"`
	InteractiveMode    bool   `mapstructure:"interactive_mode"`
	ConfirmDestructive bool   `mapstructure:"confirm_destructive"`
}

// LogConfig содержит настройки логирования
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// Load загружает конфигурацию из переменных окружения и файлов
func Load() (*Config, error) {
	// Пытаемся загрузить .env файл из нескольких возможных местоположений
	loadEnvFile()

	v := viper.New()

	// Настройка значений по умолчанию
	setDefaults(v)

	// Настройка чтения переменных окружения
	v.SetEnvPrefix("DBSYNC")
	v.AutomaticEnv()

	// Явно связываем переменные окружения с полями конфигурации
	v.BindEnv("remote.host", "DBSYNC_REMOTE_HOST")
	v.BindEnv("remote.port", "DBSYNC_REMOTE_PORT")
	v.BindEnv("remote.user", "DBSYNC_REMOTE_USER")
	v.BindEnv("remote.password", "DBSYNC_REMOTE_PASSWORD")

	v.BindEnv("local.host", "DBSYNC_LOCAL_HOST")
	v.BindEnv("local.port", "DBSYNC_LOCAL_PORT")
	v.BindEnv("local.user", "DBSYNC_LOCAL_USER")
	v.BindEnv("local.password", "DBSYNC_LOCAL_PASSWORD")

	v.BindEnv("dump.timeout", "DBSYNC_DUMP_TIMEOUT")
	v.BindEnv("dump.mydumper_image", "DBSYNC_DUMP_MYDUMPER_IMAGE")
	v.BindEnv("dump.threads", "DBSYNC_DUMP_THREADS")
	v.BindEnv("dump.chunk_size", "DBSYNC_DUMP_CHUNK_SIZE")
	v.BindEnv("dump.compress", "DBSYNC_DUMP_COMPRESS")

	v.BindEnv("cli.default_charset", "DBSYNC_CLI_DEFAULT_CHARSET")
	v.BindEnv("cli.interactive_mode", "DBSYNC_CLI_INTERACTIVE_MODE")
	v.BindEnv("cli.confirm_destructive", "DBSYNC_CLI_CONFIRM_DESTRUCTIVE")

	v.BindEnv("log.level", "DBSYNC_LOG_LEVEL")
	v.BindEnv("log.format", "DBSYNC_LOG_FORMAT")
	// Попытка загрузить конфигурацию из файла
	v.SetConfigName(".env")
	v.SetConfigType("dotenv")
	v.AddConfigPath(".")
	v.AddConfigPath("$HOME")

	// Игнорируем ошибку, если файл не найден
	_ = v.ReadInConfig()

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Валидация конфигурации
	if err := validate(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

// LoadForTest загружает конфигурацию с изолированным viper экземпляром для тестов
func LoadForTest() (*Config, error) {
	v := viper.New()
	setDefaults(v)

	// Настройка чтения переменных окружения
	v.SetEnvPrefix("DBSYNC")
	v.AutomaticEnv()

	// Явно связываем переменные окружения с полями конфигурации
	v.BindEnv("remote.host", "DBSYNC_REMOTE_HOST")
	v.BindEnv("remote.port", "DBSYNC_REMOTE_PORT")
	v.BindEnv("remote.user", "DBSYNC_REMOTE_USER")
	v.BindEnv("remote.password", "DBSYNC_REMOTE_PASSWORD")

	v.BindEnv("local.host", "DBSYNC_LOCAL_HOST")
	v.BindEnv("local.port", "DBSYNC_LOCAL_PORT")
	v.BindEnv("local.user", "DBSYNC_LOCAL_USER")
	v.BindEnv("local.password", "DBSYNC_LOCAL_PASSWORD")

	v.BindEnv("dump.timeout", "DBSYNC_DUMP_TIMEOUT")
	v.BindEnv("dump.mydumper_image", "DBSYNC_DUMP_MYDUMPER_IMAGE")
	v.BindEnv("dump.threads", "DBSYNC_DUMP_THREADS")
	v.BindEnv("dump.chunk_size", "DBSYNC_DUMP_CHUNK_SIZE")
	v.BindEnv("dump.compress", "DBSYNC_DUMP_COMPRESS")

	v.BindEnv("cli.default_charset", "DBSYNC_CLI_DEFAULT_CHARSET")
	v.BindEnv("cli.interactive_mode", "DBSYNC_CLI_INTERACTIVE_MODE")
	v.BindEnv("cli.confirm_destructive", "DBSYNC_CLI_CONFIRM_DESTRUCTIVE")

	v.BindEnv("log.level", "DBSYNC_LOG_LEVEL")
	v.BindEnv("log.format", "DBSYNC_LOG_FORMAT")

	// НЕ читаем файлы конфигурации в тестах

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Валидация конфигурации
	if err := validate(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

// setDefaults устанавливает значения по умолчанию
func setDefaults(v *viper.Viper) {
	// Удаленный MySQL сервер
	v.SetDefault("remote.host", "localhost")
	v.SetDefault("remote.port", 3306)
	v.SetDefault("remote.user", "root")
	v.SetDefault("remote.password", "")

	// Локальный MySQL сервер
	v.SetDefault("local.host", "localhost")
	v.SetDefault("local.port", 3306)
	v.SetDefault("local.user", "root")
	v.SetDefault("local.password", "")

	// Настройки дампа
	v.SetDefault("dump.timeout", "300s")
	v.SetDefault("dump.mydumper_image", "mydumper/mydumper:latest")
	v.SetDefault("dump.threads", 8)
	v.SetDefault("dump.chunk_size", 100000) // 100k строк на чанк
	v.SetDefault("dump.compress", false)

	// Настройки CLI
	v.SetDefault("cli.default_charset", "utf8mb4")
	v.SetDefault("cli.interactive_mode", true)
	v.SetDefault("cli.confirm_destructive", true)

	// Настройки логирования
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "text")
}

// Validate валидирует конфигурацию
func (c *Config) Validate() error {
	return validate(c)
}

// validate валидирует конфигурацию
func validate(config *Config) error {
	// Проверяем обязательные поля
	if config.Remote.Host == "" {
		return fmt.Errorf("remote.host is required")
	}

	if config.Local.Host == "" {
		return fmt.Errorf("local.host is required")
	}

	return nil
}

// GetConnectionString возвращает строку подключения для MySQL
func (m MySQLConfig) GetConnectionString(database string) string {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		m.User, m.Password, m.Host, m.Port, database)
	return dsn
}

// GetMysqldumpArgs возвращает аргументы для mysqldump
func (m MySQLConfig) GetMysqldumpArgs(database string) []string {
	args := []string{
		"--single-transaction",
		"--routines",
		"--triggers",
		"--add-drop-table",
		"--create-options",
		"--extended-insert",
		"--set-charset",
		"--disable-keys",
		fmt.Sprintf("--host=%s", m.Host),
		fmt.Sprintf("--port=%d", m.Port),
		fmt.Sprintf("--user=%s", m.User),
	}

	if m.Password != "" {
		args = append(args, fmt.Sprintf("--password=%s", m.Password))
	}

	args = append(args, database)
	return args
}

// GetMysqlArgs возвращает аргументы для mysql
func (m MySQLConfig) GetMysqlArgs(database string) []string {
	args := []string{
		fmt.Sprintf("--host=%s", m.Host),
		fmt.Sprintf("--port=%d", m.Port),
		fmt.Sprintf("--user=%s", m.User),
	}

	if m.Password != "" {
		args = append(args, fmt.Sprintf("--password=%s", m.Password))
	}

	args = append(args, database)
	return args
}

// GetTempDumpPath возвращает путь для временного файла дампа в системной temp директории
func (d DumpConfig) GetTempDumpPath(database string) string {
	filename := fmt.Sprintf("dbsync_%s_%d.sql", database, time.Now().Unix())
	return filepath.Join(os.TempDir(), filename)
}

// loadEnvFile пытается загрузить .env файл из двух возможных местоположений
func loadEnvFile() {
	// Список возможных путей к .env файлу
	var envPaths []string

	// 1. Домашняя директория пользователя
	if homeDir, err := os.UserHomeDir(); err == nil {
		envPaths = append(envPaths, filepath.Join(homeDir, ".dbsync.env"))
	}

	// 2. Директория с исполняемым файлом
	if execPath, err := os.Executable(); err == nil {
		execDir := filepath.Dir(execPath)
		envPaths = append(envPaths, filepath.Join(execDir, ".env"))
	}

	// Пытаемся загрузить из каждого пути
	for _, envPath := range envPaths {
		if _, err := os.Stat(envPath); err == nil {
			// Файл существует, пытаемся загрузить
			if err := godotenv.Load(envPath); err == nil {
				// Успешно загружен
				return
			}
		}
	}

	// Если ни один файл не найден, это не критично
	// Приложение будет использовать переменные окружения или значения по умолчанию
}
