package utils

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
)

// ValidateHost проверяет корректность хоста (IP или домен)
func ValidateHost(host string) error {
	if host == "" {
		return fmt.Errorf("host cannot be empty")
	}

	// Проверяем является ли это IP адресом
	if ip := net.ParseIP(host); ip != nil {
		return nil
	}

	// Проверяем является ли это корректным доменным именем
	domainRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)
	if !domainRegex.MatchString(host) {
		return fmt.Errorf("invalid host format: %s", host)
	}

	return nil
}

// ValidatePort проверяет корректность порта
func ValidatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", port)
	}
	return nil
}

// ValidatePortString проверяет корректность порта из строки
func ValidatePortString(portStr string) (int, error) {
	if portStr == "" {
		return 0, fmt.Errorf("port cannot be empty")
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 0, fmt.Errorf("invalid port format: %s", portStr)
	}

	if err := ValidatePort(port); err != nil {
		return 0, err
	}

	return port, nil
}

// ValidateUsername проверяет корректность имени пользователя MySQL
func ValidateUsername(username string) error {
	if username == "" {
		return fmt.Errorf("username cannot be empty")
	}

	if len(username) > 32 {
		return fmt.Errorf("username too long (max 32 characters)")
	}

	// MySQL usernames can contain alphanumeric characters, underscore, and some special chars
	usernameRegex := regexp.MustCompile(`^[a-zA-Z0-9_@%.]+$`)
	if !usernameRegex.MatchString(username) {
		return fmt.Errorf("username contains invalid characters")
	}

	return nil
}

// ValidateMySQLIdentifier проверяет корректность MySQL идентификатора (БД, таблица, колонка)
func ValidateMySQLIdentifier(identifier string) error {
	if identifier == "" {
		return fmt.Errorf("identifier cannot be empty")
	}

	if len(identifier) > 64 {
		return fmt.Errorf("identifier too long (max 64 characters)")
	}

	// MySQL identifiers can contain letters, digits, underscore, and dollar sign
	identifierRegex := regexp.MustCompile(`^[a-zA-Z0-9_$]+$`)
	if !identifierRegex.MatchString(identifier) {
		return fmt.Errorf("identifier contains invalid characters")
	}

	// Не должен начинаться с цифры
	if identifier[0] >= '0' && identifier[0] <= '9' {
		return fmt.Errorf("identifier cannot start with a digit")
	}

	return nil
}

// SanitizeString очищает строку от потенциально опасных символов
func SanitizeString(input string) string {
	// Убираем переводы строк и другие управляющие символы
	input = strings.ReplaceAll(input, "\n", "")
	input = strings.ReplaceAll(input, "\r", "")
	input = strings.ReplaceAll(input, "\t", "")

	// Убираем ведущие и завершающие пробелы
	input = strings.TrimSpace(input)

	return input
}

// IsSafeForShell проверяет безопасность строки для использования в shell команде
func IsSafeForShell(input string) bool {
	// Запрещенные символы для shell
	dangerousChars := []string{";", "&", "|", "`", "$", "(", ")", "{", "}", "[", "]", "<", ">", "\"", "'", "\\"}

	for _, char := range dangerousChars {
		if strings.Contains(input, char) {
			return false
		}
	}

	return true
}

// EscapeShellArg экранирует аргумент для безопасного использования в shell
func EscapeShellArg(arg string) string {
	// Простое экранирование - оборачиваем в одинарные кавычки
	// и экранируем существующие одинарные кавычки
	escaped := strings.ReplaceAll(arg, "'", "'\"'\"'")
	return "'" + escaped + "'"
}

// ValidateFileName проверяет корректность имени файла
func ValidateFileName(filename string) error {
	if filename == "" {
		return fmt.Errorf("filename cannot be empty")
	}

	// Проверяем длину
	if len(filename) > 255 {
		return fmt.Errorf("filename too long: %d characters (max 255)", len(filename))
	}

	// Проверяем на потенциально опасные символы и пути
	dangerousChars := []string{"<", ">", ":", "\"", "|", "?", "*", "/", "\\", "..", "\x00"}
	for _, char := range dangerousChars {
		if strings.Contains(filename, char) {
			return fmt.Errorf("filename contains invalid character: %s", char)
		}
	}

	return nil
}

// SanitizeInput очищает пользовательский ввод
func SanitizeInput(input string) string {
	// Убираем переводы строк и табуляции, заменяя их пробелами
	input = strings.ReplaceAll(input, "\n", " ")
	input = strings.ReplaceAll(input, "\r", " ")
	input = strings.ReplaceAll(input, "\t", " ")

	// Убираем множественные пробелы
	spaceRegex := regexp.MustCompile(`\s+`)
	input = spaceRegex.ReplaceAllString(input, " ")

	// Убираем ведущие и завершающие пробелы
	return strings.TrimSpace(input)
}

// FormatBytes форматирует размер в байтах в читаемый вид
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	units := []string{"KB", "MB", "GB", "TB", "PB"}
	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp])
}

// IsDangerous проверяет, является ли имя базы данных потенциально опасным
func IsDangerous(dbName string) bool {
	dangerousNames := []string{
		"production", "prod", "live", "master", "main",
		"root", "admin", "system", "mysql", "information_schema",
		"performance_schema", "sys",
	}

	dbNameLower := strings.ToLower(dbName)
	for _, dangerous := range dangerousNames {
		if dbNameLower == dangerous {
			return true
		}
	}

	return false
}
