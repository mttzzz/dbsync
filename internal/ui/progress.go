package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// ProgressBar представляет прогресс-бар
type ProgressBar struct {
	Width    int
	Progress float64 // 0.0 - 1.0
	Text     string
}

// NewProgressBar создает новый прогресс-бар
func NewProgressBar(width int) *ProgressBar {
	return &ProgressBar{
		Width:    width,
		Progress: 0.0,
	}
}

// SetProgress устанавливает прогресс (0.0 - 1.0)
func (pb *ProgressBar) SetProgress(progress float64, text string) {
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}
	pb.Progress = progress
	pb.Text = text
}

// Render рендерит прогресс-бар
func (pb *ProgressBar) Render() string {
	filled := int(pb.Progress * float64(pb.Width))
	empty := pb.Width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	percentage := int(pb.Progress * 100)

	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#04B575")).
		Bold(true)

	if pb.Text != "" {
		return fmt.Sprintf("%s %s [%d%%]", pb.Text, style.Render(bar), percentage)
	}
	return fmt.Sprintf("%s [%d%%]", style.Render(bar), percentage)
}

// Styles для различных типов сообщений
var (
	SuccessStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575")).
			Bold(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5555")).
			Bold(true)

	WarningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFF55")).
			Bold(true)

	InfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#5555FF")).
			Bold(true)

	MutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))

	HighlightStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true)

	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#777777")).
			Padding(1, 2)
)

// FormatStatus форматирует статус с иконкой
func FormatStatus(status string, message string) string {
	var icon string
	var style lipgloss.Style

	switch status {
	case "success":
		icon = "✅"
		style = SuccessStyle
	case "error":
		icon = "❌"
		style = ErrorStyle
	case "warning":
		icon = "⚠️"
		style = WarningStyle
	case "info":
		icon = "ℹ️"
		style = InfoStyle
	case "running":
		icon = "🔄"
		style = InfoStyle
	default:
		icon = "•"
		style = MutedStyle
	}

	return fmt.Sprintf("%s %s", icon, style.Render(message))
}

// FormatDuration форматирует продолжительность
func FormatDuration(d time.Duration) string {
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

// FormatSize форматирует размер файла
func FormatSize(bytes int64) string {
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

// RenderBox рендерит содержимое в рамке
func RenderBox(title, content string) string {
	if title != "" {
		titleStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#04B575"))

		return BoxStyle.Render(titleStyle.Render(title) + "\n\n" + content)
	}
	return BoxStyle.Render(content)
}

// RenderTable рендерит простую таблицу
func RenderTable(headers []string, rows [][]string) string {
	if len(headers) == 0 || len(rows) == 0 {
		return ""
	}

	// Вычисляем ширину колонок
	colWidths := make([]int, len(headers))
	for i, header := range headers {
		colWidths[i] = len(header)
	}

	for _, row := range rows {
		for i, cell := range row {
			if i < len(colWidths) && len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	var result strings.Builder

	// Заголовки
	for i, header := range headers {
		result.WriteString(HighlightStyle.Render(fmt.Sprintf("%-*s", colWidths[i], header)))
		if i < len(headers)-1 {
			result.WriteString("  ")
		}
	}
	result.WriteString("\n")

	// Разделитель
	for i := range headers {
		result.WriteString(MutedStyle.Render(strings.Repeat("-", colWidths[i])))
		if i < len(headers)-1 {
			result.WriteString("  ")
		}
	}
	result.WriteString("\n")

	// Строки данных
	for _, row := range rows {
		for i, cell := range row {
			if i < len(colWidths) {
				result.WriteString(fmt.Sprintf("%-*s", colWidths[i], cell))
			}
			if i < len(row)-1 {
				result.WriteString("  ")
			}
		}
		result.WriteString("\n")
	}

	return result.String()
}
