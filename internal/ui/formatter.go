package ui

import (
	"fmt"
	"strings"

	"db-sync-cli/internal/models"
)

// Formatter предоставляет методы для красивого форматирования вывода
type Formatter struct{}

// NewFormatter создает новый форматтер
func NewFormatter() *Formatter {
	return &Formatter{}
}

// FormatDatabaseList форматирует список баз данных
func (f *Formatter) FormatDatabaseList(databases models.DatabaseList, serverName string) string {
	if len(databases) == 0 {
		return InfoStyle.Render("No databases found")
	}

	var result strings.Builder

	// Заголовок
	title := fmt.Sprintf("📋 Available databases on %s", serverName)
	result.WriteString(HighlightStyle.Render(title))
	result.WriteString("\n\n")

	// Подготавливаем данные для таблицы
	headers := []string{"NAME", "TABLES"}
	rows := make([][]string, len(databases))

	for i, db := range databases {
		rows[i] = []string{
			db.Name,
			fmt.Sprintf("%d", db.Tables),
		}
	}

	// Рендерим таблицу
	result.WriteString(RenderTable(headers, rows))

	// Итого
	totalTables := 0
	for _, db := range databases {
		totalTables += db.Tables
	}

	result.WriteString("\n")
	summary := fmt.Sprintf("Total: %d databases, %d tables",
		len(databases), totalTables)
	result.WriteString(MutedStyle.Render(summary))

	return result.String()
}

// FormatConnectionStatus форматирует статус подключения
func (f *Formatter) FormatConnectionStatus(info *models.ConnectionInfo, label string) string {
	var result strings.Builder

	title := fmt.Sprintf("%s server (%s:%d):", label, info.Host, info.Port)
	result.WriteString(HighlightStyle.Render(title))
	result.WriteString("\n")

	if info.Connected {
		result.WriteString(FormatStatus("success", "Connected successfully"))
		result.WriteString("\n")

		if info.Version != "" {
			result.WriteString(fmt.Sprintf("  📊 MySQL version: %s", info.Version))
			result.WriteString("\n")
		}

		result.WriteString(fmt.Sprintf("  👤 User: %s", info.User))
		result.WriteString("\n")
	} else {
		result.WriteString(FormatStatus("error", fmt.Sprintf("Connection failed: %s", info.Error)))
		result.WriteString("\n")
	}

	return result.String()
}

// FormatSyncPlan форматирует план синхронизации
func (f *Formatter) FormatSyncPlan(db *models.Database, willReplace bool, dryRun bool) string {
	var result strings.Builder

	// Заголовок
	if dryRun {
		result.WriteString(WarningStyle.Render("🧪 DRY RUN MODE - No changes will be made"))
	} else {
		result.WriteString(InfoStyle.Render("📋 Sync Operation Plan"))
	}
	result.WriteString("\n\n")

	// Информация о БД
	displaySize := db.Size
	if db.DataSize > 0 {
		displaySize = db.DataSize
	}
	dbInfo := fmt.Sprintf("Database: %s\nSource data estimate: %s\nTables: %d",
		db.Name, FormatSize(displaySize), db.Tables)
	result.WriteString(RenderBox("Database Information", dbInfo))
	result.WriteString("\n")

	// Операция
	var operation string
	if willReplace {
		operation = fmt.Sprintf("⚠️  Local database '%s' will be REPLACED", db.Name)
		if dryRun {
			operation = fmt.Sprintf("DRY RUN: Would replace local database '%s' with %d tables (%s)",
				db.Name, db.Tables, FormatSize(displaySize))
		}
		result.WriteString(WarningStyle.Render(operation))
	} else {
		operation = fmt.Sprintf("✅ Local database '%s' will be created", db.Name)
		if dryRun {
			operation = fmt.Sprintf("DRY RUN: Would create local database '%s' with %d tables (%s)",
				db.Name, db.Tables, FormatSize(displaySize))
		}
		result.WriteString(SuccessStyle.Render(operation))
	}
	result.WriteString("\n")

	return result.String()
}

// FormatSafetyChecks форматирует результаты проверок безопасности
func (f *Formatter) FormatSafetyChecks(checks []SafetyCheck) string {
	var result strings.Builder

	result.WriteString(InfoStyle.Render("🔍 Running safety checks..."))
	result.WriteString("\n")

	for _, check := range checks {
		if check.Passed {
			result.WriteString(FormatStatus("success", check.Message))
		} else {
			result.WriteString(FormatStatus("error", check.Message))
		}
		result.WriteString("\n")
	}

	// Итоговый статус
	allPassed := true
	for _, check := range checks {
		if !check.Passed {
			allPassed = false
			break
		}
	}

	if allPassed {
		result.WriteString(FormatStatus("success", "All safety checks passed!"))
	} else {
		result.WriteString(FormatStatus("error", "Some safety checks failed!"))
	}
	result.WriteString("\n")

	return result.String()
}

// SafetyCheck представляет результат проверки безопасности
type SafetyCheck struct {
	Name    string
	Passed  bool
	Message string
}

// FormatCommands форматирует команды которые будут выполнены
func (f *Formatter) FormatCommands(dumpCmd, restoreCmd string, dryRun bool) string {
	var result strings.Builder

	if dryRun {
		result.WriteString(MutedStyle.Render("📝 Commands that would be executed:"))
	} else {
		result.WriteString(InfoStyle.Render("📝 Executing commands:"))
	}
	result.WriteString("\n")

	result.WriteString(fmt.Sprintf("   Dump: %s", dumpCmd))
	result.WriteString("\n")
	result.WriteString(fmt.Sprintf("   Restore: %s", restoreCmd))
	result.WriteString("\n")

	return result.String()
}

// FormatSyncResult форматирует результат синхронизации
func (f *Formatter) FormatSyncResult(result *models.SyncResult) string {
	var output strings.Builder

	if result.Success {
		output.WriteString(FormatStatus("success",
			fmt.Sprintf("Successfully synchronized database '%s'", result.DatabaseName)))
	} else {
		output.WriteString(FormatStatus("error",
			fmt.Sprintf("Failed to synchronize database '%s': %s", result.DatabaseName, result.Error)))
	}
	output.WriteString("\n")

	if result.Success {
		// Статистика
		statsLines := []string{
			fmt.Sprintf("Duration: %s", FormatDuration(result.Duration)),
			fmt.Sprintf("Tables: %d", result.TablesCount),
		}
		if result.Traffic.DownloadedBytes() > 0 {
			statsLines = append(statsLines, fmt.Sprintf("Downloaded from remote: %s", FormatSize(result.Traffic.DownloadedBytes())))
		}
		if result.DumpDuration > 0 && result.Traffic.DownloadedBytes() > 0 {
			bytesPerSecond := float64(result.Traffic.DownloadedBytes()) / result.DumpDuration.Seconds()
			if bytesPerSecond > 0 {
				statsLines = append(statsLines, fmt.Sprintf("Average dump speed: %s/s", FormatSize(int64(bytesPerSecond))))
			}
		}
		if result.DumpDuration > 0 || result.RestoreDuration > 0 {
			statsLines = append(statsLines, fmt.Sprintf("Dump phase: %s", FormatDuration(result.DumpDuration)))
			statsLines = append(statsLines, fmt.Sprintf("Restore phase: %s", FormatDuration(result.RestoreDuration)))
		}
		if result.LogicalSize > 0 {
			statsLines = append(statsLines, fmt.Sprintf("Source data estimate: %s", FormatSize(result.LogicalSize)))
		}
		if result.IndexSize > 0 {
			statsLines = append(statsLines, fmt.Sprintf("Source index estimate: %s", FormatSize(result.IndexSize)))
		}
		if result.DumpSizeOnDisk > 0 {
			statsLines = append(statsLines, fmt.Sprintf("Compressed dump on disk: %s", FormatSize(result.DumpSizeOnDisk)))
		} else if result.DumpSize > 0 {
			statsLines = append(statsLines, fmt.Sprintf("Dump size: %s", FormatSize(result.DumpSize)))
		}
		if result.Traffic.TotalBytes() > 0 {
			statsLines = append(statsLines,
				fmt.Sprintf("Uploaded control traffic: %s", FormatSize(result.Traffic.UploadedBytes())),
				fmt.Sprintf("Total network I/O: %s", FormatSize(result.Traffic.TotalBytes())),
				fmt.Sprintf("Transport: %s", strings.ToUpper(string(result.TransportMode))),
			)
		}
		if result.IndexSize > 0 {
			statsLines = append(statsLines, "Source index estimate is storage context, not a network metric.")
		}
		if result.CompressionRatio > 0 {
			statsLines = append(statsLines, fmt.Sprintf("Compression ratio: %.2fx", result.CompressionRatio))
		}

		stats := strings.Join(statsLines, "\n")
		output.WriteString("\n")
		output.WriteString(RenderBox("Sync Statistics", stats))
	}

	return output.String()
}

// FormatConfirmation форматирует запрос подтверждения
func (f *Formatter) FormatConfirmation(message string) string {
	var result strings.Builder

	result.WriteString(WarningStyle.Render("⚠️  CONFIRMATION REQUIRED"))
	result.WriteString("\n\n")
	result.WriteString(message)
	result.WriteString("\n\n")
	result.WriteString(MutedStyle.Render("Type 'yes' to continue, or 'no' to cancel: "))

	return result.String()
}
