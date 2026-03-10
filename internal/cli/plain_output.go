package cli

import (
	"fmt"
	"strings"
	"time"

	"db-sync-cli/internal/models"
)

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh %dm", hours, minutes)
}

func printDatabaseList(databases models.DatabaseList, serverName string) {
	fmt.Printf("Available databases on %s\n\n", serverName)
	for _, database := range databases {
		size := database.Size
		if database.DataSize > 0 {
			size = database.DataSize
		}
		fmt.Printf("- %-40s %10s  %5d tables\n", database.Name, formatBytes(size), database.Tables)
	}
}

func formatConnectionStatus(info *models.ConnectionInfo, label string) string {
	if info == nil {
		return fmt.Sprintf("%s server: unavailable\n", label)
	}
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("%s server (%s:%d):\n", label, info.Host, info.Port))
	if info.Connected {
		builder.WriteString("  OK: Connected successfully\n")
		if info.Version != "" {
			builder.WriteString(fmt.Sprintf("  MySQL version: %s\n", info.Version))
		}
		builder.WriteString(fmt.Sprintf("  User: %s\n", info.User))
	} else {
		builder.WriteString(fmt.Sprintf("  ERROR: %s\n", info.Error))
	}
	return builder.String()
}

func printSyncResult(result *models.SyncResult) {
	if result == nil {
		return
	}
	if result.Success {
		fmt.Printf("Successfully synchronized database '%s'\n", result.DatabaseName)
	} else {
		fmt.Printf("Failed to synchronize database '%s': %s\n", result.DatabaseName, result.Error)
		return
	}
	if result.LogicalSize > 0 {
		fmt.Printf("Source data estimate: %s\n", formatBytes(result.LogicalSize))
	}
	if result.IndexSize > 0 {
		fmt.Printf("Source index estimate: %s\n", formatBytes(result.IndexSize))
	}
	if result.DumpSizeOnDisk > 0 {
		fmt.Printf("Compressed dump on disk: %s\n", formatBytes(result.DumpSizeOnDisk))
	}
	if result.Traffic.TotalBytes() > 0 {
		fmt.Printf("Network I/O: %s\n", formatBytes(result.Traffic.TotalBytes()))
	}
}
