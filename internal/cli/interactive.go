package cli

import (
	"fmt"
	"strings"

	"db-sync-cli/internal/models"
	"db-sync-cli/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DatabaseSelector представляет интерактивный селектор БД
type DatabaseSelector struct {
	databases  models.DatabaseList
	cursor     int
	selected   int
	searchTerm string
	searching  bool
	filtered   models.DatabaseList
	maxHeight  int
}

// NewDatabaseSelector создает новый селектор БД
func NewDatabaseSelector(databases models.DatabaseList) *DatabaseSelector {
	selector := &DatabaseSelector{
		databases: databases,
		cursor:    0,
		selected:  -1,
		searching: true, // Сразу включаем режим поиска
		maxHeight: 10,
	}
	selector.updateFiltered()
	return selector
}

// Styles for the TUI
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#04B575")).
			Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#04B575")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true)

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#777777"))

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Italic(true)

	searchStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#1A1A1A")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Italic(true)
)

// Init implements tea.Model
func (m *DatabaseSelector) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m *DatabaseSelector) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
				m.selected = m.cursor
				return m, tea.Quit
			}
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
		case "/":
			m.searching = true
			m.searchTerm = ""
			m.updateFiltered()
		case "backspace":
			if m.searching && len(m.searchTerm) > 0 {
				m.searchTerm = m.searchTerm[:len(m.searchTerm)-1]
				m.updateFiltered()
			}
		default:
			if m.searching && len(msg.String()) == 1 {
				m.searchTerm += msg.String()
				m.updateFiltered()
			}
		}
	}
	return m, nil
}

// View implements tea.Model
func (m *DatabaseSelector) View() string {
	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("📋 Select Database to Sync"))
	b.WriteString("\n\n")

	// Search bar
	if m.searching {
		searchText := fmt.Sprintf("🔍 Search: %s", m.searchTerm)
		b.WriteString(searchStyle.Render(searchText))
		b.WriteString("\n\n")
	} else {
		b.WriteString(helpStyle.Render("Press '/' to search, ↑/↓ to navigate, Enter to select, q to quit"))
		b.WriteString("\n\n")
	}

	// Database list
	if len(m.filtered) == 0 {
		b.WriteString(infoStyle.Render("No databases found"))
		b.WriteString("\n")
	} else {
		// Calculate visible range
		start := 0
		end := len(m.filtered)

		if len(m.filtered) > m.maxHeight {
			start = m.cursor - m.maxHeight/2
			if start < 0 {
				start = 0
			}
			end = start + m.maxHeight
			if end > len(m.filtered) {
				end = len(m.filtered)
				start = end - m.maxHeight
				if start < 0 {
					start = 0
				}
			}
		}

		for i := start; i < end; i++ {
			db := m.filtered[i]
			line := fmt.Sprintf("%-30s %10s %8d tables",
				db.Name,
				ui.FormatSize(db.Size),
				db.Tables)

			if i == m.cursor {
				b.WriteString(selectedStyle.Render(" > " + line))
			} else {
				b.WriteString(normalStyle.Render("   " + line))
			}
			b.WriteString("\n")
		}

		// Show pagination info if needed
		if len(m.filtered) > m.maxHeight {
			info := fmt.Sprintf("Showing %d-%d of %d databases", start+1, end, len(m.filtered))
			b.WriteString("\n")
			b.WriteString(infoStyle.Render(info))
		}
	}

	return b.String()
}

// updateFiltered обновляет отфильтрованный список БД
func (m *DatabaseSelector) updateFiltered() {
	if m.searchTerm == "" {
		m.filtered = m.databases
	} else {
		m.filtered = models.DatabaseList{}
		searchLower := strings.ToLower(m.searchTerm)
		for _, db := range m.databases {
			if strings.Contains(strings.ToLower(db.Name), searchLower) {
				m.filtered = append(m.filtered, db)
			}
		}
	}

	// Reset cursor if out of bounds
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
	if m.cursor < 0 && len(m.filtered) > 0 {
		m.cursor = 0
	}
}

// GetSelected возвращает выбранную БД
func (m *DatabaseSelector) GetSelected() *models.Database {
	if m.selected >= 0 && m.selected < len(m.filtered) {
		return &m.filtered[m.selected]
	}
	return nil
}

// RunDatabaseSelector запускает интерактивный селектор БД
func RunDatabaseSelector(databases models.DatabaseList) (*models.Database, error) {
	if len(databases) == 0 {
		return nil, fmt.Errorf("no databases available")
	}

	selector := NewDatabaseSelector(databases)
	program := tea.NewProgram(selector)

	model, err := program.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to run selector: %w", err)
	}

	finalSelector := model.(*DatabaseSelector)
	selected := finalSelector.GetSelected()
	if selected == nil {
		return nil, fmt.Errorf("no database selected")
	}

	return selected, nil
}

// ConfirmationSelector представляет интерактивный селектор для подтверждения
type ConfirmationSelector struct {
	message  string
	options  []string
	cursor   int
	selected int
}

// NewConfirmationSelector создает новый селектор подтверждения
func NewConfirmationSelector(message string) *ConfirmationSelector {
	return &ConfirmationSelector{
		message:  message,
		options:  []string{"Yes", "No"},
		cursor:   1, // По умолчанию выбираем "No" для безопасности
		selected: -1,
	}
}

// Init implements tea.Model
func (m *ConfirmationSelector) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m *ConfirmationSelector) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.selected = 1 // "No"
			return m, tea.Quit
		case "enter":
			m.selected = m.cursor
			return m, tea.Quit
		case "up", "k", "left", "h":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j", "right", "l":
			if m.cursor < len(m.options)-1 {
				m.cursor++
			}
		case "y", "Y":
			m.selected = 0 // "Yes"
			return m, tea.Quit
		case "n", "N":
			m.selected = 1 // "No"
			return m, tea.Quit
		}
	}
	return m, nil
}

// View implements tea.Model
func (m *ConfirmationSelector) View() string {
	var b strings.Builder

	// Warning title
	warningStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF6B6B")).
		Padding(0, 1)

	b.WriteString(warningStyle.Render("⚠️  CONFIRMATION REQUIRED"))
	b.WriteString("\n\n")

	// Message
	messageStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Bold(true)

	b.WriteString(messageStyle.Render(m.message))
	b.WriteString("\n\n")

	// Options
	for i, option := range m.options {
		if i == m.cursor {
			if option == "Yes" {
				// Yes option - green when selected
				style := lipgloss.NewStyle().
					Background(lipgloss.Color("#28A745")).
					Foreground(lipgloss.Color("#FFFFFF")).
					Bold(true).
					Padding(0, 2)
				b.WriteString(style.Render(" > " + option + " "))
			} else {
				// No option - red when selected
				style := lipgloss.NewStyle().
					Background(lipgloss.Color("#DC3545")).
					Foreground(lipgloss.Color("#FFFFFF")).
					Bold(true).
					Padding(0, 2)
				b.WriteString(style.Render(" > " + option + " "))
			}
		} else {
			// Unselected option
			style := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#888888")).
				Padding(0, 2)
			b.WriteString(style.Render("   " + option + " "))
		}

		if i < len(m.options)-1 {
			b.WriteString("   ")
		}
	}

	b.WriteString("\n\n")

	// Help text
	helpText := "Use ← → arrows or Y/N keys to choose, Enter to confirm, Esc to cancel"
	b.WriteString(helpStyle.Render(helpText))

	return b.String()
}

// IsConfirmed возвращает true если пользователь выбрал "Yes"
func (m *ConfirmationSelector) IsConfirmed() bool {
	return m.selected == 0
}

// RunConfirmationSelector запускает интерактивный селектор подтверждения
func RunConfirmationSelector(message string) (bool, error) {
	selector := NewConfirmationSelector(message)
	program := tea.NewProgram(selector)

	model, err := program.Run()
	if err != nil {
		return false, fmt.Errorf("failed to run confirmation selector: %w", err)
	}

	finalSelector := model.(*ConfirmationSelector)
	return finalSelector.IsConfirmed(), nil
}
