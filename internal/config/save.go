package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// DefaultEnvPath возвращает основной путь сохранения конфигурации.
func DefaultEnvPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ".dbsync.env"
	}
	return filepath.Join(homeDir, ".dbsync.env")
}

var envSections = []struct {
	Title string
	Pairs []struct {
		Key   string
		Value func(*Config) string
	}
}{
	{
		Title: "Remote MySQL",
		Pairs: []struct {
			Key   string
			Value func(*Config) string
		}{
			{Key: "DBSYNC_REMOTE_HOST", Value: func(c *Config) string { return c.Remote.Host }},
			{Key: "DBSYNC_REMOTE_PORT", Value: func(c *Config) string { return strconv.Itoa(c.Remote.Port) }},
			{Key: "DBSYNC_REMOTE_USER", Value: func(c *Config) string { return c.Remote.User }},
			{Key: "DBSYNC_REMOTE_PASSWORD", Value: func(c *Config) string { return c.Remote.Password }},
			{Key: "DBSYNC_REMOTE_PROXY_URL", Value: func(c *Config) string { return c.Remote.ProxyURL }},
		},
	},
	{
		Title: "Local MySQL",
		Pairs: []struct {
			Key   string
			Value func(*Config) string
		}{
			{Key: "DBSYNC_LOCAL_HOST", Value: func(c *Config) string { return c.Local.Host }},
			{Key: "DBSYNC_LOCAL_PORT", Value: func(c *Config) string { return strconv.Itoa(c.Local.Port) }},
			{Key: "DBSYNC_LOCAL_USER", Value: func(c *Config) string { return c.Local.User }},
			{Key: "DBSYNC_LOCAL_PASSWORD", Value: func(c *Config) string { return c.Local.Password }},
			{Key: "DBSYNC_LOCAL_PROXY_URL", Value: func(c *Config) string { return c.Local.ProxyURL }},
		},
	},
	{
		Title: "Dump",
		Pairs: []struct {
			Key   string
			Value func(*Config) string
		}{
			{Key: "DBSYNC_DUMP_TIMEOUT", Value: func(c *Config) string { return c.Dump.Timeout.String() }},
			{Key: "DBSYNC_DUMP_THREADS", Value: func(c *Config) string { return strconv.Itoa(c.Dump.Threads) }},
			{Key: "DBSYNC_DUMP_COMPRESS", Value: func(c *Config) string { return strconv.FormatBool(c.Dump.Compress) }},
			{Key: "DBSYNC_DUMP_NETWORK_COMPRESS", Value: func(c *Config) string { return strconv.FormatBool(c.Dump.NetworkCompress) }},
			{Key: "DBSYNC_DUMP_NETWORK_ZSTD_LEVEL", Value: func(c *Config) string { return strconv.Itoa(c.Dump.NetworkZstdLevel) }},
		},
	},
	{
		Title: "CLI",
		Pairs: []struct {
			Key   string
			Value func(*Config) string
		}{
			{Key: "DBSYNC_CLI_DEFAULT_CHARSET", Value: func(c *Config) string { return c.CLI.DefaultCharset }},
			{Key: "DBSYNC_CLI_INTERACTIVE_MODE", Value: func(c *Config) string { return strconv.FormatBool(c.CLI.InteractiveMode) }},
			{Key: "DBSYNC_CLI_CONFIRM_DESTRUCTIVE", Value: func(c *Config) string { return strconv.FormatBool(c.CLI.ConfirmDestructive) }},
		},
	},
	{
		Title: "Logging",
		Pairs: []struct {
			Key   string
			Value func(*Config) string
		}{
			{Key: "DBSYNC_LOG_LEVEL", Value: func(c *Config) string { return c.Log.Level }},
			{Key: "DBSYNC_LOG_FORMAT", Value: func(c *Config) string { return c.Log.Format }},
		},
	},
}

// ToEnvString сериализует конфигурацию в .env-совместимый текст.
func (c *Config) ToEnvString() (string, error) {
	if err := c.Validate(); err != nil {
		return "", fmt.Errorf("config validation failed: %w", err)
	}

	var builder strings.Builder
	for sectionIndex, section := range envSections {
		if sectionIndex > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString("# ")
		builder.WriteString(section.Title)
		builder.WriteString("\n")
		for _, pair := range section.Pairs {
			builder.WriteString(pair.Key)
			builder.WriteString("=")
			builder.WriteString(escapeEnvValue(pair.Value(c)))
			builder.WriteString("\n")
		}
	}

	return builder.String(), nil
}

// SaveEnv сохраняет конфигурацию в .env файл.
func (c *Config) SaveEnv(path string) error {
	content, err := c.ToEnvString()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func escapeEnvValue(value string) string {
	if value == "" {
		return ""
	}

	if strings.ContainsAny(value, " \t\n\r#\"") {
		return strconv.Quote(value)
	}

	return value
}
