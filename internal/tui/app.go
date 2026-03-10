package tui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"db-sync-cli/internal/config"
	"db-sync-cli/internal/models"
	"db-sync-cli/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type DatabaseBrowser interface {
	TestConnection(isRemote bool) (*models.ConnectionInfo, error)
	ListDatabases(isRemote bool) (models.DatabaseList, error)
	ListTables(databaseName string, isRemote bool) ([]models.Table, error)
	ListTableDependencies(databaseName string, tableNames []string, isRemote bool) ([]models.TableDependency, error)
}

type SyncExecutor interface {
	ExecuteTarget(target models.SyncTarget) (*models.SyncResult, error)
	ExecutePlan(plan *models.SyncPlan, runtime models.RuntimeOptions, observer models.ProgressObserver) ([]models.SyncResult, error)
}

type view int

const (
	viewList view = iota
	viewTables
	viewPlan
	viewConfirm
	viewSettings
	viewRunning
	viewReport
)

type confirmChoice int

const (
	confirmCancel confirmChoice = iota
	confirmSync
)

type settingsFieldKind int

const (
	settingsFieldString settingsFieldKind = iota
	settingsFieldInt
	settingsFieldBool
	settingsFieldDuration
	settingsFieldPassword
)

type settingsField struct {
	Label       string
	Description string
	Kind        settingsFieldKind
	Get         func(*config.Config) string
	Set         func(*config.Config, string) error
	MaskValue   bool
}

type databaseTableState struct {
	Tables        []models.Table
	Dependencies  []models.TableDependency
	Selected      map[string]bool
	AutoIncluded  map[string]bool
	Initialized   bool
	Cursor        int
	Loaded        bool
	Loading       bool
	Error         string
	VisibleTables []models.Table
	Filter        string
	Filtering     bool
}

type tablesLoadedMsg struct {
	DatabaseName string
	Tables       []models.Table
	Dependencies []models.TableDependency
	Err          error
}

type connectionTestMsg struct {
	Remote bool
	Info   *models.ConnectionInfo
	Err    error
}

type databasesReloadedMsg struct {
	Databases models.DatabaseList
	Err       error
}

type syncTargetDoneMsg struct {
	Target models.SyncTarget
	Result *models.SyncResult
	Err    error
}

type planRunDone struct {
	Results []models.SyncResult
	Err     error
}

type runTickMsg time.Time

// AppResult хранит результат работы unified TUI shell.
type AppResult struct {
	Cancelled bool
	Results   []models.SyncResult
}

type phaseTimingTracker struct {
	currentPhase  models.SyncPhase
	currentDetail string
	currentAt     time.Time
	durations     map[models.SyncPhase]map[string]time.Duration
}

type AppModel struct {
	cfg     *config.Config
	browser DatabaseBrowser
	runner  SyncExecutor

	databases models.DatabaseList
	filtered  models.DatabaseList
	cursor    int
	search    string
	searching bool

	selectedDatabases map[string]bool
	tableStates       map[string]*databaseTableState
	previewDatabase   string

	view          view
	previousView  view
	showHelp      bool
	confirmChoice confirmChoice
	planCursor    int

	savePath          string
	settingsFields    []settingsField
	settingsCursor    int
	settingsEditing   bool
	settingsBuffer    string
	settingsStatus    string
	notice            string
	settingsDirty     bool
	remoteTestStatus  string
	localTestStatus   string
	connectionTesting bool

	runningPlan          *models.SyncPlan
	runningResults       []models.SyncResult
	runningCompleted     int
	runningStartedAt     time.Time
	runningTargetStarted time.Time
	runningNow           time.Time
	runningTargetName    string
	running              bool
	currentProgress      models.ProgressSnapshot
	runProgressCh        chan models.ProgressSnapshot
	runDoneCh            chan planRunDone
	runningError         string
	phaseTimings         map[string]*phaseTimingTracker

	result AppResult

	width  int
	height int
}

var (
	pageStyle        = lipgloss.NewStyle().Padding(1, 2)
	headerStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F6F2FF")).Background(lipgloss.Color("#6C2BD9")).Padding(0, 2)
	subtleStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#7E7A9A"))
	panelStyle       = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#353160")).Padding(1, 2)
	selectedRowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#A855F7")).Bold(true)
	keyStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED")).Bold(true)
	sizeStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#00D2FF")).Bold(true)
	okStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("#00D084")).Bold(true)
	warnStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")).Bold(true)
	dangerStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4D6D")).Bold(true)
	mutedValueStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#A09ABF"))
)

func NewAppModel(cfg *config.Config, browser DatabaseBrowser, runner SyncExecutor, databases models.DatabaseList) *AppModel {
	copyList := append(models.DatabaseList(nil), databases...)
	copyList.SortBySize()
	model := &AppModel{
		cfg:               cfg,
		browser:           browser,
		runner:            runner,
		databases:         copyList,
		view:              viewList,
		previousView:      viewList,
		confirmChoice:     confirmCancel,
		selectedDatabases: make(map[string]bool),
		tableStates:       make(map[string]*databaseTableState),
		savePath:          config.DefaultEnvPath(),
		width:             120,
		height:            36,
		runningNow:        time.Now(),
		phaseTimings:      make(map[string]*phaseTimingTracker),
	}
	model.initSettingsFields()
	model.updateFilter()
	return model
}

func RunApp(cfg *config.Config, browser DatabaseBrowser, runner SyncExecutor, databases models.DatabaseList) (*AppResult, error) {
	program := tea.NewProgram(NewAppModel(cfg, browser, runner, databases), tea.WithAltScreen())
	finalModel, err := program.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to run app shell: %w", err)
	}
	result := finalModel.(*AppModel).result
	return &result, nil
}

func (m *AppModel) Init() tea.Cmd { return nil }

func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tablesLoadedMsg:
		state := m.tableState(msg.DatabaseName)
		state.Loading = false
		state.Loaded = msg.Err == nil
		state.Error = ""
		if msg.Err != nil {
			state.Error = msg.Err.Error()
		} else {
			state.Tables = msg.Tables
			state.Dependencies = msg.Dependencies
			state.VisibleTables = append([]models.Table(nil), msg.Tables...)
			m.initializeDefaultTableSelection(msg.DatabaseName)
			m.refreshAutoIncluded(msg.DatabaseName)
		}
		return m, nil
	case databasesReloadedMsg:
		if msg.Err != nil {
			m.setNotice(dangerStyle.Render(msg.Err.Error()))
			return m, nil
		}
		m.applyDatabases(msg.Databases)
		m.setNotice(okStyle.Render(fmt.Sprintf("Loaded %d remote databases", len(msg.Databases))))
		if m.previewDatabase != "" {
			state := m.tableState(m.previewDatabase)
			state.Tables = nil
			state.Dependencies = nil
			state.VisibleTables = nil
			state.Loaded = false
			state.Loading = true
			state.Error = ""
			return m, m.loadTablesCmd(m.previewDatabase)
		}
		return m, nil
	case connectionTestMsg:
		m.connectionTesting = false
		status := dangerStyle.Render("connection failed")
		if msg.Info != nil && msg.Info.Connected {
			status = okStyle.Render(fmt.Sprintf("connected to %s %s", msg.Info.Host, msg.Info.Version))
		} else if msg.Info != nil && msg.Info.Error != "" {
			status = dangerStyle.Render(msg.Info.Error)
		}
		if msg.Err != nil {
			status = dangerStyle.Render(msg.Err.Error())
		}
		if msg.Remote {
			m.remoteTestStatus = status
		} else {
			m.localTestStatus = status
		}
		m.setNotice(status)
		return m, nil
	case runTickMsg:
		m.runningNow = time.Time(msg)
		if done, hasDone := m.drainRunChannels(); hasDone {
			m.runningResults = append([]models.SyncResult(nil), done.Results...)
			m.result.Results = append([]models.SyncResult(nil), done.Results...)
			m.running = false
			m.runningTargetName = ""
			if done.Err != nil {
				m.runningError = done.Err.Error()
			}
			m.view = viewReport
			return m, nil
		}
		if m.running {
			return m, tickCmd()
		}
	case tea.KeyMsg:
		if m.showHelp {
			return m.handleHelpKey(msg)
		}
		switch m.view {
		case viewList:
			return m.handleListKey(msg)
		case viewTables:
			return m.handleTablesKey(msg)
		case viewPlan:
			return m.handlePlanKey(msg)
		case viewConfirm:
			return m.handleConfirmKey(msg)
		case viewSettings:
			return m.handleSettingsKey(msg)
		case viewRunning:
			return m.handleRunningKey(msg)
		case viewReport:
			return m.handleReportKey(msg)
		}
	}
	return m, nil
}

func (m *AppModel) handleHelpKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "?", "esc", "q", "enter", " ":
		m.showHelp = false
	}
	return m, nil
}

func (m *AppModel) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.searching {
		switch msg.String() {
		case "esc":
			m.searching = false
			m.search = ""
			m.updateFilter()
		case "enter", "ctrl+m":
			m.searching = false
		case "backspace":
			if len(m.search) > 0 {
				m.search = m.search[:len(m.search)-1]
				m.updateFilter()
			}
		default:
			if len(msg.String()) == 1 {
				m.search += msg.String()
				m.updateFilter()
			}
		}
		return m, nil
	}

	switch msg.String() {
	case "ctrl+c", "q":
		m.result.Cancelled = true
		return m, tea.Quit
	case "?":
		m.showHelp = true
	case "/":
		m.searching = true
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
		}
	case "home":
		m.cursor = 0
	case "end":
		if len(m.filtered) > 0 {
			m.cursor = len(m.filtered) - 1
		}
	case "space", " ":
		if db := m.currentDatabase(); db != nil {
			m.toggleDatabaseSelection(db.Name)
		}
	case "a":
		for _, db := range m.filtered {
			m.selectedDatabases[db.Name] = true
		}
	case "c":
		m.selectedDatabases = make(map[string]bool)
	case "r":
		return m, m.reloadDatabasesCmd()
	case "s":
		m.previousView = m.view
		m.view = viewSettings
	case "y", "Y":
		if len(m.buildPlan().Targets) > 0 {
			m.view = viewPlan
		}
	case "enter", "ctrl+m", "right", "l":
		if db := m.currentDatabase(); db != nil {
			m.previewDatabase = db.Name
			m.selectedDatabases[db.Name] = true
			state := m.tableState(db.Name)
			m.view = viewTables
			if !state.Loaded && !state.Loading {
				state.Loading = true
				return m, m.loadTablesCmd(db.Name)
			}
		}
	}
	return m, nil
}

func (m *AppModel) handleTablesKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	state := m.tableState(m.previewDatabase)
	if state.Filtering {
		switch msg.String() {
		case "esc":
			state.Filter = ""
			state.Filtering = false
			m.updateVisibleTables(m.previewDatabase)
		case "enter", "ctrl+m":
			state.Filtering = false
		case "backspace":
			if len(state.Filter) > 0 {
				state.Filter = state.Filter[:len(state.Filter)-1]
				m.updateVisibleTables(m.previewDatabase)
			}
		default:
			if len(msg.String()) == 1 {
				state.Filter += msg.String()
				m.updateVisibleTables(m.previewDatabase)
			}
		}
		return m, nil
	}

	switch msg.String() {
	case "ctrl+c", "q":
		m.result.Cancelled = true
		return m, tea.Quit
	case "?":
		m.showHelp = true
	case "esc", "left", "h":
		m.view = viewList
	case "up", "k":
		if state.Cursor > 0 {
			state.Cursor--
		}
	case "down", "j":
		if state.Cursor < len(state.VisibleTables)-1 {
			state.Cursor++
		}
	case "/":
		state.Filtering = true
	case "space", " ":
		if table, ok := m.currentTable(); ok {
			if state.Selected[table.Name] {
				delete(state.Selected, table.Name)
			} else {
				state.Selected[table.Name] = true
			}
			m.refreshAutoIncluded(m.previewDatabase)
		}
	case "a":
		for _, table := range state.Tables {
			state.Selected[table.Name] = true
		}
		m.refreshAutoIncluded(m.previewDatabase)
	case "c":
		state.Selected = make(map[string]bool)
		state.AutoIncluded = make(map[string]bool)
	case "s":
		m.previousView = m.view
		m.view = viewSettings
	case "y", "Y", "enter", "ctrl+m":
		m.view = viewPlan
	}
	return m, nil
}

func (m *AppModel) handlePlanKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	plan := m.buildPlan()
	switch msg.String() {
	case "ctrl+c", "q":
		m.result.Cancelled = true
		return m, tea.Quit
	case "?":
		m.showHelp = true
	case "esc", "left", "h":
		if m.previewDatabase != "" {
			m.view = viewTables
		} else {
			m.view = viewList
		}
	case "up", "k":
		if m.planCursor > 0 {
			m.planCursor--
		}
	case "down", "j":
		if m.planCursor < len(plan.Targets)-1 {
			m.planCursor++
		}
	case "x", "backspace", "delete":
		if target, ok := m.currentPlanTarget(plan); ok {
			delete(m.selectedDatabases, target.DatabaseName)
			if m.planCursor >= len(m.buildPlan().Targets) && m.planCursor > 0 {
				m.planCursor--
			}
		}
	case "enter", "ctrl+m":
		if target, ok := m.currentPlanTarget(plan); ok {
			m.previewDatabase = target.DatabaseName
			m.view = viewTables
			state := m.tableState(target.DatabaseName)
			if !state.Loaded && !state.Loading {
				state.Loading = true
				return m, m.loadTablesCmd(target.DatabaseName)
			}
		}
	case "c":
		m.selectedDatabases = make(map[string]bool)
		m.planCursor = 0
	case "y", "Y":
		if len(plan.Targets) > 0 {
			m.view = viewConfirm
			m.confirmChoice = confirmSync
		}
	case "s":
		m.previousView = m.view
		m.view = viewSettings
	}
	return m, nil
}

func (m *AppModel) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		m.result.Cancelled = true
		return m, tea.Quit
	case "?":
		m.showHelp = true
	case "esc":
		if m.viewBeforeConfirm() == viewPlan {
			m.view = viewPlan
		} else if m.previewDatabase != "" {
			m.view = viewTables
		} else {
			m.view = viewList
		}
	case "left", "h":
		m.confirmChoice = confirmCancel
	case "right", "l":
		m.confirmChoice = confirmSync
	case "y", "Y":
		m.confirmChoice = confirmSync
		plan := m.buildPlan()
		if len(plan.Targets) == 0 {
			return m, nil
		}
		m.runningPlan = plan
		m.runningResults = nil
		m.runningCompleted = 0
		m.runningStartedAt = time.Now()
		m.runningTargetStarted = m.runningStartedAt
		m.runningTargetName = plan.Targets[0].DatabaseName
		m.currentProgress = models.ProgressSnapshot{Phase: models.SyncPhasePlanning, DatabaseName: plan.Targets[0].DatabaseName, Message: "Launching sync plan", Timestamp: time.Now()}
		m.runningError = ""
		m.runProgressCh = make(chan models.ProgressSnapshot, 256)
		m.runDoneCh = make(chan planRunDone, 1)
		m.running = true
		m.view = viewRunning
		return m, tea.Batch(m.startSyncCmd(), tickCmd())
	case "tab":
		if m.confirmChoice == confirmCancel {
			m.confirmChoice = confirmSync
		} else {
			m.confirmChoice = confirmCancel
		}
	case "enter", "ctrl+m":
		if m.confirmChoice == confirmSync {
			plan := m.buildPlan()
			if len(plan.Targets) == 0 {
				return m, nil
			}
			m.runningPlan = plan
			m.runningResults = nil
			m.runningCompleted = 0
			m.runningStartedAt = time.Now()
			m.runningTargetStarted = m.runningStartedAt
			m.runningTargetName = plan.Targets[0].DatabaseName
			m.currentProgress = models.ProgressSnapshot{Phase: models.SyncPhasePlanning, DatabaseName: plan.Targets[0].DatabaseName, Message: "Launching sync plan", Timestamp: time.Now()}
			m.runningError = ""
			m.runProgressCh = make(chan models.ProgressSnapshot, 256)
			m.runDoneCh = make(chan planRunDone, 1)
			m.running = true
			m.view = viewRunning
			return m, tea.Batch(m.startSyncCmd(), tickCmd())
		}
		if m.viewBeforeConfirm() == viewPlan {
			m.view = viewPlan
		} else if m.previewDatabase != "" {
			m.view = viewTables
		} else {
			m.view = viewList
		}
	}
	return m, nil
}

func (m *AppModel) handleSettingsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.settingsEditing {
		return m.handleSettingsEditingKey(msg)
	}
	if m.connectionTesting {
		switch msg.String() {
		case "ctrl+c", "q":
			m.result.Cancelled = true
			return m, tea.Quit
		case "?":
			m.showHelp = true
		case "esc", "left", "h", "s":
			m.view = m.previousView
		}
		return m, nil
	}

	switch msg.String() {
	case "ctrl+c", "q":
		m.result.Cancelled = true
		return m, tea.Quit
	case "?":
		m.showHelp = true
	case "up", "k":
		if m.settingsCursor > 0 {
			m.settingsCursor--
		}
	case "down", "j":
		if m.settingsCursor < len(m.settingsFields)-1 {
			m.settingsCursor++
		}
	case "home":
		m.settingsCursor = 0
	case "end":
		if len(m.settingsFields) > 0 {
			m.settingsCursor = len(m.settingsFields) - 1
		}
	case "enter", "ctrl+m":
		m.openSettingsEditor()
	case "space", " ":
		if m.currentSettingsField().Kind == settingsFieldBool {
			m.toggleCurrentBoolField()
		}
	case "w":
		return m, m.saveSettingsAndReloadCmd()
	case "r":
		m.connectionTesting = true
		m.remoteTestStatus = warnStyle.Render("checking remote...")
		return m, m.connectionTestCmd(true)
	case "l":
		m.connectionTesting = true
		m.localTestStatus = warnStyle.Render("checking local...")
		return m, m.connectionTestCmd(false)
	case "esc", "left", "h", "s":
		m.view = m.previousView
	}
	return m, nil
}

func (m *AppModel) handleSettingsEditingKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.settingsEditing = false
		m.settingsBuffer = ""
		m.settingsStatus = subtleStyle.Render("Edit cancelled")
		m.setNotice(m.settingsStatus)
	case "enter", "ctrl+m":
		field := m.currentSettingsField()
		if err := field.Set(m.cfg, m.settingsBuffer); err != nil {
			m.settingsStatus = dangerStyle.Render(err.Error())
			m.setNotice(m.settingsStatus)
			return m, nil
		}
		m.settingsEditing = false
		m.settingsDirty = true
		m.settingsStatus = okStyle.Render(field.Label + " updated")
		m.setNotice(m.settingsStatus)
		m.settingsBuffer = ""
	case "backspace":
		if len(m.settingsBuffer) > 0 {
			m.settingsBuffer = m.settingsBuffer[:len(m.settingsBuffer)-1]
		}
	default:
		if len(msg.String()) == 1 {
			m.settingsBuffer += msg.String()
		}
	}
	return m, nil
}

func (m *AppModel) handleRunningKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "?":
		m.showHelp = true
	case "q", "ctrl+c":
		if !m.running {
			m.result.Cancelled = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *AppModel) handleReportKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "?":
		m.showHelp = true
	case "enter", "q", "ctrl+c", "esc":
		return m, tea.Quit
	case "b":
		m.view = viewList
	}
	return m, nil
}

func (m *AppModel) View() string {
	base := pageStyle.Render(strings.Join([]string{m.renderHeader(), "", m.renderBody(), "", m.renderFooter()}, "\n"))
	if m.showHelp {
		return overlayCentered(base, m.renderHelpDialog(), m.width, m.height)
	}
	return base
}

func (m *AppModel) renderHeader() string {
	selectedCount := len(m.selectedDatabaseNames())
	stats := []string{
		fmt.Sprintf("Remote: %s:%d", m.cfg.Remote.Host, m.cfg.Remote.Port),
		fmt.Sprintf("Local: %s:%d", m.cfg.Local.Host, m.cfg.Local.Port),
		fmt.Sprintf("Threads: %d", m.cfg.Dump.Threads),
		fmt.Sprintf("Selected DBs: %s", sizeStyle.Render(strconv.Itoa(selectedCount))),
	}
	if m.cfg.Remote.HasProxy() {
		stats = append(stats, fmt.Sprintf("Mode: %s", warnStyle.Render("PROXY")))
	} else {
		stats = append(stats, fmt.Sprintf("Mode: %s", okStyle.Render("DIRECT")))
	}
	return strings.Join([]string{headerStyle.Render("DBSync Control Center"), subtleStyle.Render(strings.Join(stats, "   "))}, "\n")
}

func (m *AppModel) renderBody() string {
	bodyWidth := maxInt(m.width-4, 64)
	return panelStyle.Width(bodyWidth).Render(m.renderMainContent(bodyWidth - 4))
}

func (m *AppModel) renderMainContent(width int) string {
	switch m.view {
	case viewList:
		return m.renderListView(width)
	case viewTables:
		return m.renderTablesView(width)
	case viewPlan:
		return m.renderPlanView(width)
	case viewConfirm:
		return m.renderConfirmView(width)
	case viewSettings:
		return m.renderSettingsView(width)
	case viewRunning:
		return m.renderRunningView(width)
	case viewReport:
		return m.renderReportView(width)
	default:
		return ""
	}

}

func (m *AppModel) renderListView(width int) string {
	lines := []string{headerStyle.UnsetBackground().Render("Database Queue Builder"), "", m.listSummary(), ""}
	if m.searching {
		lines = append(lines, fmt.Sprintf("Search: %s", keyStyle.Render(m.search+cursorSuffix())), "")
	} else if m.search != "" {
		lines = append(lines, subtleStyle.Render("Filter: "+m.search), "")
	}
	if len(m.filtered) == 0 {
		return wrapLines(append(lines, subtleStyle.Render("No databases match the current filter.")), width)
	}
	visible := clampInt(m.height-14, 8, 18)
	start, end := visibleRange(m.cursor, len(m.filtered), visible)
	nameWidth, sizeWidth, tablesWidth := databaseColumnWidths(m.filtered)
	for index := start; index < end; index++ {
		db := m.filtered[index]
		mark := "[ ]"
		if m.selectedDatabases[db.Name] {
			mark = okStyle.Render("[x]")
		}
		prefix := "  "
		tableCount := formatGroupedInt64(int64(db.Tables))
		row := fmt.Sprintf("%s %s  %s  %s %s", mark, padRight(db.Name, nameWidth), padLeft(sizeStyle.Render(ui.FormatSize(displayDatabaseBytes(db))), sizeWidth), padLeft(subtleStyle.Render(tableCount), tablesWidth), subtleStyle.Render("tables"))
		if index == m.cursor {
			prefix = keyStyle.Render("▸ ")
			row = selectedRowStyle.Render(row)
		}
		lines = append(lines, prefix+row)
	}
	if len(m.filtered) > visible {
		lines = append(lines, "", subtleStyle.Render(fmt.Sprintf("[%d-%d / %d]", start+1, end, len(m.filtered))))
	}
	return wrapLines(lines, width)
}

func (m *AppModel) renderTablesView(width int) string {
	if m.previewDatabase == "" {
		return wrapLines([]string{"No database selected."}, width)
	}
	state := m.tableState(m.previewDatabase)
	lines := []string{headerStyle.UnsetBackground().Render("Table Selection"), "", fmt.Sprintf("Database: %s", selectedRowStyle.Render(m.previewDatabase)), subtleStyle.Render("Sizes show source data estimate; index/storage overhead is reported separately later."), ""}
	if state.Loading {
		return wrapLines(append(lines, warnStyle.Render("Loading tables and FK dependencies...")), width)
	}
	if state.Error != "" {
		return wrapLines(append(lines, dangerStyle.Render(state.Error)), width)
	}
	if state.Filtering {
		lines = append(lines, fmt.Sprintf("Filter tables: %s", keyStyle.Render(state.Filter+cursorSuffix())), "")
	} else if state.Filter != "" {
		lines = append(lines, subtleStyle.Render("Filter: "+state.Filter), "")
	}
	if len(state.VisibleTables) == 0 {
		return wrapLines(append(lines, subtleStyle.Render("No tables available for this filter.")), width)
	}
	visible := clampInt(m.height-16, 8, 18)
	start, end := visibleRange(state.Cursor, len(state.VisibleTables), visible)
	nameWidth, sizeWidth, rowsWidth := tableColumnWidths(state.VisibleTables)
	hasApproxRows := false
	for index := start; index < end; index++ {
		table := state.VisibleTables[index]
		mark := "[ ]"
		if state.Selected[table.Name] {
			mark = okStyle.Render("[x]")
		} else if state.AutoIncluded[table.Name] {
			mark = warnStyle.Render("[+] ")
		}
		prefix := "  "
		rowCount := formatGroupedInt64(table.Rows)
		if table.RowsApprox {
			rowCount = "~" + rowCount
			hasApproxRows = true
		}
		row := fmt.Sprintf("%s %s  %s  %s %s", mark, padRight(table.Name, nameWidth), padLeft(sizeStyle.Render(ui.FormatSize(displayTableBytes(table))), sizeWidth), padLeft(mutedValueStyle.Render(rowCount), rowsWidth), mutedValueStyle.Render("rows"))
		if index == state.Cursor {
			prefix = keyStyle.Render("▸ ")
			row = selectedRowStyle.Render(row)
		}
		lines = append(lines, prefix+row)
	}
	if len(state.VisibleTables) > visible {
		lines = append(lines, "", subtleStyle.Render(fmt.Sprintf("[%d-%d / %d]", start+1, end, len(state.VisibleTables))))
	}
	if len(state.AutoIncluded) > 0 {
		auto := make([]string, 0, len(state.AutoIncluded))
		for tableName := range state.AutoIncluded {
			auto = append(auto, tableName)
		}
		sort.Strings(auto)
		lines = append(lines, "", warnStyle.Render("Auto-included by FK: "+strings.Join(auto, ", ")))
	}
	if hasApproxRows {
		lines = append(lines, "", subtleStyle.Render("Rows prefixed with ~ are fallback estimates when exact COUNT(*) was too expensive or timed out."))
	}
	return wrapLines(lines, width)
}

func (m *AppModel) renderConfirmView(width int) string {
	plan := m.buildPlan()
	if len(plan.Targets) == 0 {
		return wrapLines([]string{"No sync targets selected."}, width)
	}
	lines := []string{headerStyle.UnsetBackground().Render("Confirm Sync Plan"), "", fmt.Sprintf("Databases: %d", len(plan.Targets)), fmt.Sprintf("Estimated source data: %s", ui.FormatSize(plan.EstimatedLogicalSize)), ""}
	for _, target := range plan.Targets {
		mode := okStyle.Render("FULL DB")
		if len(target.SelectedTables) > 0 {
			mode = warnStyle.Render(fmt.Sprintf("%d selected tables", len(target.SelectedTables)))
		}
		line := fmt.Sprintf("%s  %s", selectedRowStyle.Render(target.DatabaseName), mode)
		lines = append(lines, line)
		if len(target.AutoIncludedTables) > 0 {
			lines = append(lines, subtleStyle.Render("  auto: "+strings.Join(target.AutoIncludedTables, ", ")))
		}
	}
	cancelButton := renderButton("Cancel", m.confirmChoice == confirmCancel, false)
	syncButton := renderButton("Sync", m.confirmChoice == confirmSync, true)
	lines = append(lines, "", lipgloss.JoinHorizontal(lipgloss.Top, cancelButton, "   ", syncButton))
	return wrapLines(lines, width)
}

func (m *AppModel) renderPlanView(width int) string {
	plan := m.buildPlan()
	if len(plan.Targets) == 0 {
		return wrapLines([]string{"No sync targets selected."}, width)
	}
	lines := []string{headerStyle.UnsetBackground().Render("Plan Editor"), "", subtleStyle.Render("Review the queue before destructive sync. Enter edits tables, X removes a target, Y continues."), ""}
	for index, target := range plan.Targets {
		prefix := "  "
		mode := okStyle.Render("full database")
		if len(target.SelectedTables) > 0 {
			mode = warnStyle.Render(fmt.Sprintf("%d tables", len(target.SelectedTables)))
		}
		row := fmt.Sprintf("%s  %s  %s", padRight(target.DatabaseName, 28), mode, subtleStyle.Render(ui.FormatSize(m.targetLogicalSize(target))))
		if index == m.planCursor {
			prefix = keyStyle.Render("▸ ")
			row = selectedRowStyle.Render(row)
		}
		lines = append(lines, prefix+row)
		if index == m.planCursor && len(target.AutoIncludedTables) > 0 {
			lines = append(lines, subtleStyle.Render("    auto: "+strings.Join(target.AutoIncludedTables, ", ")))
		}
	}
	lines = append(lines, "", fmt.Sprintf("Estimated source data: %s", sizeStyle.Render(ui.FormatSize(plan.EstimatedLogicalSize))))
	return wrapLines(lines, width)
}

func (m *AppModel) renderSettingsView(width int) string {
	lines := []string{headerStyle.UnsetBackground().Render("Settings"), "", subtleStyle.Render("Enter edits, Space toggles booleans, R/L run connection tests, W saves .env."), ""}
	visible := clampInt(m.height-18, 8, 16)
	start, end := visibleRange(m.settingsCursor, len(m.settingsFields), visible)
	for index := start; index < end; index++ {
		field := m.settingsFields[index]
		prefix := "  "
		row := fmt.Sprintf("%s  %s", padRight(field.Label, 24), m.fieldDisplayValue(field))
		if index == m.settingsCursor {
			prefix = keyStyle.Render("▸ ")
			row = selectedRowStyle.Render(row)
		}
		lines = append(lines, prefix+row)
		if index == m.settingsCursor {
			lines = append(lines, subtleStyle.Render("    "+field.Description))
		}
	}
	if len(m.settingsFields) > visible {
		lines = append(lines, "", subtleStyle.Render(fmt.Sprintf("[%d-%d / %d]", start+1, end, len(m.settingsFields))))
	}
	lines = append(lines, "", fmt.Sprintf("Save path: %s", subtleStyle.Render(m.savePath)))
	if m.settingsEditing {
		lines = append(lines, "", headerStyle.UnsetBackground().Render("Editing"), fmt.Sprintf("%s%s", m.settingsBuffer, cursorSuffix()))
	}
	if m.settingsStatus != "" {
		lines = append(lines, "", m.settingsStatus)
	}
	return wrapLines(lines, width)
}

func (m *AppModel) renderRunningView(width int) string {
	if m.runningPlan == nil {
		return wrapLines([]string{"No active sync."}, width)
	}
	progress := m.runningProgress()
	eta := m.runningETA()
	avgSpeed := m.currentSpeedLabel()
	phaseProgress := m.currentProgress.Percent / 100
	if phaseProgress < 0 {
		phaseProgress = 0
	}
	if phaseProgress > 1 {
		phaseProgress = 1
	}
	progressBar := renderProgressBar(minInt(width-10, 48), progress)
	phaseBar := renderProgressBar(minInt(width-10, 48), phaseProgress)
	phase := m.currentProgress.Phase
	if phase == "" {
		phase = models.SyncPhasePlanning
	}
	bytesProgress := "measuring..."
	if m.currentProgress.BytesTotal > 0 {
		bytesProgress = fmt.Sprintf("%s / %s", ui.FormatSize(m.currentProgress.BytesCompleted), ui.FormatSize(m.currentProgress.BytesTotal))
	} else if m.currentProgress.BytesCompleted > 0 {
		bytesProgress = ui.FormatSize(m.currentProgress.BytesCompleted)
	} else if m.currentProgress.Traffic.DownloadedBytes() > 0 {
		bytesProgress = ui.FormatSize(m.currentProgress.Traffic.DownloadedBytes())
	}
	trafficLabel := m.trafficSnapshotLabel()
	etaLabel := "Remaining ETA"
	if phase == models.SyncPhaseDump {
		etaLabel = "Dump ETA"
	} else if phase == models.SyncPhaseRestore {
		etaLabel = "Restore ETA"
	}
	phaseDetail := m.runningPhaseDetail()
	phaseDetailLabel := m.runningPhaseDetailLabel()
	lines := []string{
		headerStyle.UnsetBackground().Render("Running Sync"),
		"",
		fmt.Sprintf("Current target: %s", selectedRowStyle.Render(m.runningTargetName)),
		fmt.Sprintf("Phase: %s", strings.ToUpper(string(phase))),
		fmt.Sprintf("%s: %s", phaseDetailLabel, phaseDetail),
		fmt.Sprintf("Completed: %d/%d", m.runningCompleted, len(m.runningPlan.Targets)),
		fmt.Sprintf("Elapsed: %s", ui.FormatDuration(m.runningNow.Sub(m.runningStartedAt))),
		fmt.Sprintf("%s: %s", etaLabel, eta),
		fmt.Sprintf("Download speed: %s", avgSpeed),
		fmt.Sprintf("Dump progress (downloaded): %s", bytesProgress),
		fmt.Sprintf("Traffic snapshot: %s", trafficLabel),
		fmt.Sprintf("Queue progress: %s", progressBar),
		fmt.Sprintf("Phase progress: %s", phaseBar),
		fmt.Sprintf("Current step: %s", m.runningMessage()),
	}
	lines = append(lines, m.renderRunningPhaseBreakdown()...)
	lines = append(lines,
		"",
		subtleStyle.Render("Speed and ETA are oriented around downloaded dump traffic; compressed dump size is a separate local disk metric."),
	)
	return wrapLines(lines, width)
}

func (m *AppModel) renderReportView(width int) string {
	lines := []string{headerStyle.UnsetBackground().Render("Sync Report"), ""}
	if len(m.runningResults) == 0 {
		return wrapLines(append(lines, subtleStyle.Render("No results available.")), width)
	}
	if m.runningError != "" {
		lines = append(lines, dangerStyle.Render("Run stopped with error: "+m.runningError), "")
	}
	var totalLogical int64
	var totalIndex int64
	var totalDownloaded int64
	var totalUploaded int64
	var totalNetwork int64
	var totalDuration time.Duration
	for _, result := range m.runningResults {
		totalLogical += result.LogicalSize
		totalIndex += result.IndexSize
		totalDownloaded += result.Traffic.DownloadedBytes()
		totalUploaded += result.Traffic.UploadedBytes()
		totalNetwork += result.Traffic.TotalBytes()
		totalDuration += result.Duration
		status := okStyle.Render("OK")
		if !result.Success {
			status = dangerStyle.Render("FAILED")
		}
		downloadLabel := fmt.Sprintf("downloaded from remote: %s", ui.FormatSize(result.Traffic.DownloadedBytes()))
		speedLabel := fmt.Sprintf("avg dump speed: %s", m.resultDumpSpeedLabel(result))
		dumpLabel := fmt.Sprintf("compressed dump on disk: %s", ui.FormatSize(result.DumpSizeOnDisk))
		uploadLabel := fmt.Sprintf("uploaded control traffic: %s", ui.FormatSize(result.Traffic.UploadedBytes()))
		networkLabel := fmt.Sprintf("total network I/O: %s", ui.FormatSize(result.Traffic.TotalBytes()))
		sourceLabel := fmt.Sprintf("source data estimate: %s", ui.FormatSize(result.LogicalSize))
		indexLabel := fmt.Sprintf("source index estimate: %s", ui.FormatSize(result.IndexSize))
		lines = append(lines,
			fmt.Sprintf("%s  %s", status, selectedRowStyle.Render(result.DatabaseName)),
			fmt.Sprintf("  %s", downloadLabel),
			fmt.Sprintf("  %s", speedLabel),
			fmt.Sprintf("  dump phase: %s   restore phase: %s", ui.FormatDuration(result.DumpDuration), ui.FormatDuration(result.RestoreDuration)),
		)
		lines = append(lines, m.renderPhaseBreakdown(result.DatabaseName)...)
		lines = append(lines,
			fmt.Sprintf("  %s", dumpLabel),
			fmt.Sprintf("  total duration: %s", ui.FormatDuration(result.Duration)),
			fmt.Sprintf("  %s", uploadLabel),
			fmt.Sprintf("  %s", networkLabel),
			subtleStyle.Render("  source footprint context:"),
			fmt.Sprintf("  %s", sourceLabel),
			fmt.Sprintf("  %s", indexLabel),
			"",
		)
		if result.Error != "" {
			lines = append(lines, dangerStyle.Render("  "+result.Error))
		}
	}
	if len(m.runningResults) > 1 {
		lines = append(lines,
			fmt.Sprintf("Total downloaded from remote: %s", sizeStyle.Render(ui.FormatSize(totalDownloaded))),
			fmt.Sprintf("Total uploaded control traffic: %s", sizeStyle.Render(ui.FormatSize(totalUploaded))),
			fmt.Sprintf("Total network I/O: %s", sizeStyle.Render(ui.FormatSize(totalNetwork))),
			fmt.Sprintf("Total duration: %s", sizeStyle.Render(ui.FormatDuration(totalDuration))),
			subtleStyle.Render("Source footprint context across the queue:"),
			fmt.Sprintf("Total source data estimate: %s", sizeStyle.Render(ui.FormatSize(totalLogical))),
			fmt.Sprintf("Total source index estimate: %s", sizeStyle.Render(ui.FormatSize(totalIndex))),
		)
	}
	return wrapLines(lines, width)
}

func (m *AppModel) renderFooter() string {
	switch m.view {
	case viewList:
		return subtleStyle.Render(fmt.Sprintf("%s move   %s select DB   %s tables   %s select all   %s clear   %s reload   %s confirm   %s settings", keyStyle.Render("↑/↓"), keyStyle.Render("Space"), keyStyle.Render("Enter"), keyStyle.Render("A"), keyStyle.Render("C"), keyStyle.Render("R"), keyStyle.Render("Y"), keyStyle.Render("S")))
	case viewTables:
		return subtleStyle.Render(fmt.Sprintf("%s move   %s toggle table   %s filter   %s select all   %s clear   %s confirm", keyStyle.Render("↑/↓"), keyStyle.Render("Space"), keyStyle.Render("/"), keyStyle.Render("A"), keyStyle.Render("C"), keyStyle.Render("Y/Enter")))
	case viewPlan:
		return subtleStyle.Render(fmt.Sprintf("%s move   %s edit target   %s remove   %s clear   %s continue", keyStyle.Render("↑/↓"), keyStyle.Render("Enter"), keyStyle.Render("X"), keyStyle.Render("C"), keyStyle.Render("Y")))
	case viewConfirm:
		return subtleStyle.Render(fmt.Sprintf("%s switch   %s start sync   %s arm/start   %s back", keyStyle.Render("←/→/Tab"), keyStyle.Render("Enter"), keyStyle.Render("Y"), keyStyle.Render("Esc")))
	case viewSettings:
		if m.settingsEditing {
			return subtleStyle.Render(fmt.Sprintf("%s input   %s apply   %s cancel", keyStyle.Render("Type"), keyStyle.Render("Enter"), keyStyle.Render("Esc")))
		}
		return subtleStyle.Render(fmt.Sprintf("%s move   %s edit   %s toggle   %s remote test   %s local test   %s save", keyStyle.Render("↑/↓"), keyStyle.Render("Enter"), keyStyle.Render("Space"), keyStyle.Render("R"), keyStyle.Render("L"), keyStyle.Render("W")))
	case viewRunning:
		return subtleStyle.Render("Sync is running. Press ? for help.")
	case viewReport:
		return subtleStyle.Render(fmt.Sprintf("%s quit   %s back to list", keyStyle.Render("Enter/Q/Esc"), keyStyle.Render("B")))
	default:
		return ""
	}
}

func (m *AppModel) renderHelpDialog() string {
	lines := []string{
		headerStyle.Render("Help"),
		"",
		"List view",
		"  Space toggles databases into the sync queue",
		"  Enter opens table drill-down for the current database",
		"  R reloads the remote database inventory with current settings",
		"  Y opens the plan editor for all selected databases",
		"",
		"Tables view",
		"  Space toggles selected tables",
		"  [+] marks FK auto-included tables",
		"  Enter or Y opens the plan editor with the current queue",
		"",
		"Plan view",
		"  Enter re-opens the highlighted target for editing",
		"  X removes a target from the queue",
		"  Y continues to destructive confirmation",
		"",
		"Confirm view",
		"  Sync is selected by default",
		"  Enter starts the sync",
		"  Y also arms sync immediately",
		"",
		"Settings view",
		"  Enter edits the selected field",
		"  Space toggles boolean values",
		"  R and L test remote/local connections",
		"  W saves to the configured .env path",
		"",
		"Running view",
		"  Shows queue progress, elapsed time, ETA estimate and average transfer metrics",
		"",
		subtleStyle.Render("Press Esc, Enter, Space or ? to close help."),
	}
	content := panelStyle.Width(minInt(72, maxInt(m.width-8, 40))).Render(wrapLines(lines, 64))
	return content
}

func (m *AppModel) listSummary() string {
	var totalSize int64
	var totalTables int
	for _, db := range m.filtered {
		totalSize += displayDatabaseBytes(db)
		totalTables += db.Tables
	}
	return fmt.Sprintf("Visible: %s   Source data est.: %s   Tables: %d", sizeStyle.Render(strconv.Itoa(len(m.filtered))), sizeStyle.Render(ui.FormatSize(totalSize)), totalTables)
}

func (m *AppModel) viewLabel() string {
	switch m.view {
	case viewList:
		return "Building multi-database queue"
	case viewTables:
		return "Selecting tables and FK dependencies"
	case viewPlan:
		return "Reviewing and editing sync plan"
	case viewConfirm:
		return dangerStyle.Render("Awaiting destructive confirmation")
	case viewSettings:
		if m.settingsEditing {
			return warnStyle.Render("Editing config field")
		}
		return "Editing settings and testing connections"
	case viewRunning:
		return warnStyle.Render("Running destructive sync queue")
	case viewReport:
		return okStyle.Render("Run finished")
	default:
		return ""
	}
}

func (m *AppModel) currentDatabase() *models.Database {
	if len(m.filtered) == 0 || m.cursor < 0 || m.cursor >= len(m.filtered) {
		return nil
	}
	return &m.filtered[m.cursor]
}

func (m *AppModel) selectedDatabaseNames() []string {
	names := make([]string, 0, len(m.selectedDatabases))
	for _, db := range m.databases {
		if m.selectedDatabases[db.Name] {
			names = append(names, db.Name)
		}
	}
	return names
}

func (m *AppModel) toggleDatabaseSelection(name string) {
	if m.selectedDatabases[name] {
		delete(m.selectedDatabases, name)
		return
	}
	m.selectedDatabases[name] = true
}

func (m *AppModel) tableState(databaseName string) *databaseTableState {
	state, ok := m.tableStates[databaseName]
	if ok {
		return state
	}
	state = &databaseTableState{Selected: make(map[string]bool), AutoIncluded: make(map[string]bool)}
	m.tableStates[databaseName] = state
	return state
}

func (m *AppModel) currentTable() (models.Table, bool) {
	state := m.tableState(m.previewDatabase)
	if len(state.VisibleTables) == 0 || state.Cursor < 0 || state.Cursor >= len(state.VisibleTables) {
		return models.Table{}, false
	}
	return state.VisibleTables[state.Cursor], true
}

func (m *AppModel) updateFilter() {
	if m.search == "" {
		m.filtered = append(models.DatabaseList(nil), m.databases...)
	} else {
		needle := strings.ToLower(m.search)
		m.filtered = m.filtered[:0]
		for _, db := range m.databases {
			if strings.Contains(strings.ToLower(db.Name), needle) {
				m.filtered = append(m.filtered, db)
			}
		}
	}
	sort.Slice(m.filtered, func(i, j int) bool {
		if m.filtered[i].Size == m.filtered[j].Size {
			return m.filtered[i].Name < m.filtered[j].Name
		}
		return m.filtered[i].Size > m.filtered[j].Size
	})
	if len(m.filtered) == 0 {
		m.cursor = 0
		return
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
}

func (m *AppModel) applyDatabases(databases models.DatabaseList) {
	copyList := append(models.DatabaseList(nil), databases...)
	copyList.SortBySize()
	m.databases = copyList
	m.updateFilter()
	valid := make(map[string]struct{}, len(copyList))
	for _, db := range copyList {
		valid[db.Name] = struct{}{}
	}
	for name := range m.selectedDatabases {
		if _, ok := valid[name]; !ok {
			delete(m.selectedDatabases, name)
			delete(m.tableStates, name)
		}
	}
	if m.previewDatabase != "" {
		if _, ok := valid[m.previewDatabase]; !ok {
			m.previewDatabase = ""
			if m.view == viewTables {
				m.view = viewList
			}
		}
	}
}

func (m *AppModel) updateVisibleTables(databaseName string) {
	state := m.tableState(databaseName)
	if state.Filter == "" {
		state.VisibleTables = append(state.VisibleTables[:0], state.Tables...)
	} else {
		needle := strings.ToLower(state.Filter)
		state.VisibleTables = state.VisibleTables[:0]
		for _, table := range state.Tables {
			if strings.Contains(strings.ToLower(table.Name), needle) {
				state.VisibleTables = append(state.VisibleTables, table)
			}
		}
	}
	if len(state.VisibleTables) == 0 {
		state.Cursor = 0
		return
	}
	if state.Cursor >= len(state.VisibleTables) {
		state.Cursor = len(state.VisibleTables) - 1
	}
}

func (m *AppModel) initializeDefaultTableSelection(databaseName string) {
	state := m.tableState(databaseName)
	if state.Initialized {
		return
	}
	state.Selected = make(map[string]bool, len(state.Tables))
	for _, table := range state.Tables {
		state.Selected[table.Name] = true
	}
	state.Initialized = true
}

func (m *AppModel) refreshAutoIncluded(databaseName string) {
	state := m.tableState(databaseName)
	state.AutoIncluded = make(map[string]bool)
	if len(state.Selected) == 0 {
		return
	}
	closure := make(map[string]bool, len(state.Selected))
	for tableName := range state.Selected {
		closure[tableName] = true
	}
	changed := true
	for changed {
		changed = false
		for _, dep := range state.Dependencies {
			if !closure[dep.TableName] {
				continue
			}
			if closure[dep.ReferencedTable] {
				continue
			}
			closure[dep.ReferencedTable] = true
			if !state.Selected[dep.ReferencedTable] {
				state.AutoIncluded[dep.ReferencedTable] = true
			}
			changed = true
		}
	}
}

func (m *AppModel) effectiveTables(databaseName string) []string {
	state := m.tableState(databaseName)
	if len(state.Selected) == 0 {
		return nil
	}
	effective := make([]string, 0, len(state.Selected)+len(state.AutoIncluded))
	for tableName := range state.Selected {
		effective = append(effective, tableName)
	}
	for tableName := range state.AutoIncluded {
		effective = append(effective, tableName)
	}
	sort.Strings(effective)
	return effective
}

func (m *AppModel) targetForDatabase(name string) models.SyncTarget {
	state := m.tableState(name)
	effective := m.effectiveTables(name)
	auto := make([]string, 0, len(state.AutoIncluded))
	for tableName := range state.AutoIncluded {
		auto = append(auto, tableName)
	}
	sort.Strings(auto)
	target := models.SyncTarget{DatabaseName: name, ReplaceEntireDatabase: true}
	if len(effective) > 0 {
		target.SelectedTables = effective
		target.AutoIncludedTables = auto
	}
	return target
}

func (m *AppModel) buildPlan() *models.SyncPlan {
	plan := &models.SyncPlan{TransportMode: models.TransportModeDirect, CreatedAt: time.Now()}
	if m.cfg.Remote.HasProxy() {
		plan.TransportMode = models.TransportModeProxy
	}
	for _, name := range m.selectedDatabaseNames() {
		target := m.targetForDatabase(name)
		plan.Targets = append(plan.Targets, target)
		plan.EstimatedLogicalSize += m.targetLogicalSize(target)
	}
	return plan
}

func (m *AppModel) targetLogicalSize(target models.SyncTarget) int64 {
	if len(target.SelectedTables) == 0 {
		if db := m.databaseByName(target.DatabaseName); db != nil {
			if db.DataSize > 0 {
				return db.DataSize
			}
			return db.Size
		}
		return 0
	}
	state := m.tableState(target.DatabaseName)
	set := make(map[string]struct{}, len(target.SelectedTables))
	for _, name := range target.SelectedTables {
		set[name] = struct{}{}
	}
	var total int64
	for _, table := range state.Tables {
		if _, ok := set[table.Name]; ok {
			if table.DataSize > 0 {
				total += table.DataSize
			} else {
				total += table.Size
			}
		}
	}
	return total
}

func (m *AppModel) databaseByName(name string) *models.Database {
	for index := range m.databases {
		if m.databases[index].Name == name {
			return &m.databases[index]
		}
	}
	return nil
}

func (m *AppModel) loadTablesCmd(databaseName string) tea.Cmd {
	return func() tea.Msg {
		if m.browser == nil {
			return tablesLoadedMsg{DatabaseName: databaseName, Err: fmt.Errorf("database browser is not configured")}
		}
		tables, err := m.browser.ListTables(databaseName, true)
		if err != nil {
			return tablesLoadedMsg{DatabaseName: databaseName, Err: err}
		}
		tableNames := make([]string, 0, len(tables))
		for _, table := range tables {
			tableNames = append(tableNames, table.Name)
		}
		deps, err := m.browser.ListTableDependencies(databaseName, tableNames, true)
		if err != nil {
			return tablesLoadedMsg{DatabaseName: databaseName, Err: err}
		}
		return tablesLoadedMsg{DatabaseName: databaseName, Tables: tables, Dependencies: deps}
	}
}

func (m *AppModel) connectionTestCmd(remote bool) tea.Cmd {
	return func() tea.Msg {
		if m.browser == nil {
			return connectionTestMsg{Remote: remote, Err: fmt.Errorf("database browser is not configured")}
		}
		info, err := m.browser.TestConnection(remote)
		return connectionTestMsg{Remote: remote, Info: info, Err: err}
	}
}

func (m *AppModel) reloadDatabasesCmd() tea.Cmd {
	return func() tea.Msg {
		if m.browser == nil {
			return databasesReloadedMsg{Err: fmt.Errorf("database browser is not configured")}
		}
		databases, err := m.browser.ListDatabases(true)
		return databasesReloadedMsg{Databases: databases, Err: err}
	}
}

func (m *AppModel) startSyncCmd() tea.Cmd {
	if m.runningPlan == nil {
		return nil
	}
	plan := m.runningPlan
	progressCh := m.runProgressCh
	doneCh := m.runDoneCh
	return func() tea.Msg {
		if m.runner == nil {
			doneCh <- planRunDone{Err: fmt.Errorf("sync executor is not configured")}
			return runTickMsg(time.Now())
		}
		go func() {
			results, err := m.runner.ExecutePlan(plan, models.RuntimeOptions{Threads: m.cfg.Dump.Threads}, func(snapshot models.ProgressSnapshot) {
				select {
				case progressCh <- snapshot:
				default:
				}
			})
			doneCh <- planRunDone{Results: results, Err: err}
		}()
		return runTickMsg(time.Now())
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(250*time.Millisecond, func(t time.Time) tea.Msg { return runTickMsg(t) })
}

func (m *AppModel) runningProgress() float64 {
	if m.runningPlan == nil || len(m.runningPlan.Targets) == 0 {
		return 0
	}
	completed := float64(m.runningCompleted)
	if !m.running {
		return 1
	}
	frac := m.currentProgress.Percent / 100
	if frac <= 0 {
		estimate := m.estimateTargetDuration(m.targetForDatabase(m.runningTargetName))
		if estimate > 0 {
			frac = time.Since(m.runningTargetStarted).Seconds() / estimate.Seconds()
		}
	}
	if frac > 0.95 {
		frac = 0.95
	}
	progress := (completed + frac) / float64(len(m.runningPlan.Targets))
	if progress < 0 {
		return 0
	}
	if progress > 1 {
		return 1
	}
	return progress
}

func (m *AppModel) runningETA() string {
	if m.runningPlan == nil || len(m.runningPlan.Targets) == 0 {
		return "n/a"
	}
	if m.currentProgress.HasETA() {
		return ui.FormatDuration(m.currentProgress.ETA)
	}
	if !m.etaReadyForCurrentPhase() {
		return "warming up..."
	}
	var remaining time.Duration
	if m.running {
		if trafficETA, ok := m.currentTrafficETA(); ok {
			remaining += trafficETA
		} else {
			currentTarget := m.targetForDatabase(m.runningTargetName)
			estimate := m.estimateTargetDuration(currentTarget)
			elapsed := time.Since(m.runningTargetStarted)
			if estimate > elapsed {
				remaining += estimate - elapsed
			}
		}
		for index := m.runningCompleted + 1; index < len(m.runningPlan.Targets); index++ {
			remaining += m.estimateTargetDuration(m.runningPlan.Targets[index])
		}
	}
	if remaining <= 0 {
		return "finishing..."
	}
	return ui.FormatDuration(remaining)
}

func (m *AppModel) currentTrafficETA() (time.Duration, bool) {
	if !m.etaReadyForCurrentPhase() {
		return 0, false
	}
	remainingBytes := m.currentProgress.BytesTotal - m.currentProgress.BytesCompleted
	if remainingBytes <= 0 {
		return 0, false
	}

	bytesPerSecond := m.observedBytesPerSecond()
	if bytesPerSecond <= 0 {
		return 0, false
	}

	seconds := float64(remainingBytes) / bytesPerSecond
	if seconds <= 0 {
		return 0, false
	}

	return time.Duration(seconds * float64(time.Second)), true
}

func (m *AppModel) observedBytesPerSecond() float64 {
	if m.currentProgress.Traffic.AverageBytesPerSecond > 0 {
		return m.currentProgress.Traffic.AverageBytesPerSecond
	}
	if m.currentProgress.Traffic.CurrentBytesPerSecond > 0 {
		return m.currentProgress.Traffic.CurrentBytesPerSecond
	}
	return 0
}

func (m *AppModel) completedAverageSpeed() string {
	var totalBytes int64
	var totalDuration time.Duration
	for _, result := range m.runningResults {
		totalBytes += result.Traffic.DownloadedBytes()
		totalDuration += result.Duration
	}
	if totalBytes <= 0 || totalDuration <= 0 {
		return "measuring..."
	}
	bytesPerSecond := float64(totalBytes) / totalDuration.Seconds()
	return ui.FormatSize(int64(bytesPerSecond)) + "/s"
}

func (m *AppModel) currentSpeedLabel() string {
	if m.currentProgress.Traffic.AverageBytesPerSecond > 0 {
		return ui.FormatSize(int64(m.currentProgress.Traffic.AverageBytesPerSecond)) + "/s avg"
	}
	if m.currentProgress.Traffic.CurrentBytesPerSecond > 0 {
		return ui.FormatSize(int64(m.currentProgress.Traffic.CurrentBytesPerSecond)) + "/s"
	}
	return m.completedAverageSpeed()
}

func (m *AppModel) resultDumpSpeedLabel(result models.SyncResult) string {
	if result.DumpDuration <= 0 || result.Traffic.DownloadedBytes() <= 0 {
		return "n/a"
	}
	bytesPerSecond := float64(result.Traffic.DownloadedBytes()) / result.DumpDuration.Seconds()
	if bytesPerSecond <= 0 {
		return "n/a"
	}
	return ui.FormatSize(int64(bytesPerSecond)) + "/s"
}

func (m *AppModel) runningMessage() string {
	if m.currentProgress.Message != "" {
		return m.currentProgress.Message
	}
	return "waiting for mysqlsh progress..."
}

func (m *AppModel) runningPhaseDetailLabel() string {
	switch m.currentProgress.Phase {
	case models.SyncPhaseDump:
		return "Dump subphase"
	case models.SyncPhaseRestore:
		return "Restore subphase"
	default:
		return "Phase detail"
	}
}

func (m *AppModel) runningPhaseDetail() string {
	return phaseDetailForSnapshot(m.currentProgress)
}

func (m *AppModel) etaReadyForCurrentPhase() bool {
	if m.currentProgress.Phase != models.SyncPhaseDump {
		return true
	}
	return !m.isEarlyDumpWarmup()
}

func (m *AppModel) isEarlyDumpWarmup() bool {
	return isEarlyDumpWarmupSnapshot(m.currentProgress)
}

func (m *AppModel) currentPlanTarget(plan *models.SyncPlan) (models.SyncTarget, bool) {
	if plan == nil || len(plan.Targets) == 0 || m.planCursor < 0 || m.planCursor >= len(plan.Targets) {
		return models.SyncTarget{}, false
	}
	return plan.Targets[m.planCursor], true
}

func (m *AppModel) viewBeforeConfirm() view {
	if len(m.buildPlan().Targets) > 0 {
		return viewPlan
	}
	if m.previewDatabase != "" {
		return viewTables
	}
	return viewList
}

func (m *AppModel) drainRunChannels() (planRunDone, bool) {
	if m.runProgressCh != nil {
		for {
			select {
			case snapshot := <-m.runProgressCh:
				m.currentProgress = mergeProgressSnapshot(m.currentProgress, snapshot)
				m.recordPhaseTiming(m.currentProgress)
				if snapshot.DatabaseName != "" && snapshot.DatabaseName != m.runningTargetName {
					m.runningTargetName = snapshot.DatabaseName
					m.runningTargetStarted = snapshot.Timestamp
				}
				if snapshot.Phase == models.SyncPhaseDone {
					m.finalizePhaseTiming(snapshot.DatabaseName, snapshot.Timestamp)
					m.runningCompleted++
				}
			default:
				goto doneProgress
			}
		}
	}

doneProgress:
	if m.runDoneCh != nil {
		select {
		case done := <-m.runDoneCh:
			m.finalizeAllPhaseTimings(time.Now())
			return done, true
		default:
		}
	}
	return planRunDone{}, false
}

func (m *AppModel) recordPhaseTiming(snapshot models.ProgressSnapshot) {
	if snapshot.DatabaseName == "" || snapshot.Timestamp.IsZero() {
		return
	}
	detail := phaseDetailForSnapshot(snapshot)
	tracker := m.phaseTimings[snapshot.DatabaseName]
	if tracker == nil {
		tracker = &phaseTimingTracker{durations: make(map[models.SyncPhase]map[string]time.Duration)}
		m.phaseTimings[snapshot.DatabaseName] = tracker
	}
	if !tracker.currentAt.IsZero() && tracker.currentPhase != "" && tracker.currentDetail != "" {
		elapsed := snapshot.Timestamp.Sub(tracker.currentAt)
		if elapsed > 0 {
			phaseDurations := tracker.durations[tracker.currentPhase]
			if phaseDurations == nil {
				phaseDurations = make(map[string]time.Duration)
				tracker.durations[tracker.currentPhase] = phaseDurations
			}
			phaseDurations[tracker.currentDetail] += elapsed
		}
	}
	tracker.currentPhase = snapshot.Phase
	tracker.currentDetail = detail
	tracker.currentAt = snapshot.Timestamp
}

func (m *AppModel) finalizePhaseTiming(databaseName string, at time.Time) {
	tracker := m.phaseTimings[databaseName]
	if tracker == nil || tracker.currentAt.IsZero() || tracker.currentPhase == "" || tracker.currentDetail == "" || at.IsZero() {
		return
	}
	elapsed := at.Sub(tracker.currentAt)
	if elapsed > 0 {
		phaseDurations := tracker.durations[tracker.currentPhase]
		if phaseDurations == nil {
			phaseDurations = make(map[string]time.Duration)
			tracker.durations[tracker.currentPhase] = phaseDurations
		}
		phaseDurations[tracker.currentDetail] += elapsed
	}
	tracker.currentAt = at
}

func (m *AppModel) finalizeAllPhaseTimings(at time.Time) {
	for databaseName := range m.phaseTimings {
		m.finalizePhaseTiming(databaseName, at)
	}
}

func (m *AppModel) renderPhaseBreakdown(databaseName string) []string {
	tracker := m.phaseTimings[databaseName]
	if tracker == nil {
		return nil
	}
	lines := make([]string, 0)
	if dumpLines := renderPhaseDurationLines("  dump breakdown:", tracker.durations[models.SyncPhaseDump], []string{"Preparing dump metadata", "Writing schema metadata", "Writing table metadata", "Streaming table data", "Finalizing dump files"}); len(dumpLines) > 0 {
		lines = append(lines, dumpLines...)
	}
	if restoreLines := renderPhaseDurationLines("  restore breakdown:", tracker.durations[models.SyncPhaseRestore], []string{"Preparing local restore", "Applying schema metadata", "Loading table data", "Rebuilding indexes", "Finalizing restore"}); len(restoreLines) > 0 {
		lines = append(lines, restoreLines...)
	}
	return lines
}

func (m *AppModel) renderRunningPhaseBreakdown() []string {
	if m.runningTargetName == "" {
		return nil
	}
	durations := m.runningPhaseDurationsSnapshot(m.runningTargetName, m.runningNow)
	if len(durations) == 0 {
		return nil
	}
	lines := []string{"", subtleStyle.Render("Live phase timers:")}
	if dumpLines := renderPhaseDurationLines("  dump breakdown:", durations[models.SyncPhaseDump], []string{"Preparing dump metadata", "Writing schema metadata", "Writing table metadata", "Streaming table data", "Finalizing dump files"}); len(dumpLines) > 0 {
		lines = append(lines, dumpLines...)
	}
	if restoreLines := renderPhaseDurationLines("  restore breakdown:", durations[models.SyncPhaseRestore], []string{"Preparing local restore", "Applying schema metadata", "Loading table data", "Rebuilding indexes", "Finalizing restore"}); len(restoreLines) > 0 {
		lines = append(lines, restoreLines...)
	}
	return lines
}

func (m *AppModel) runningPhaseDurationsSnapshot(databaseName string, now time.Time) map[models.SyncPhase]map[string]time.Duration {
	tracker := m.phaseTimings[databaseName]
	if tracker == nil {
		return nil
	}
	copyDurations := clonePhaseDurations(tracker.durations)
	if tracker.currentPhase == "" || tracker.currentDetail == "" || tracker.currentAt.IsZero() || now.IsZero() {
		return copyDurations
	}
	elapsed := now.Sub(tracker.currentAt)
	if elapsed <= 0 {
		return copyDurations
	}
	if copyDurations == nil {
		copyDurations = make(map[models.SyncPhase]map[string]time.Duration)
	}
	phaseDurations := copyDurations[tracker.currentPhase]
	if phaseDurations == nil {
		phaseDurations = make(map[string]time.Duration)
		copyDurations[tracker.currentPhase] = phaseDurations
	}
	phaseDurations[tracker.currentDetail] += elapsed
	return copyDurations
}

func clonePhaseDurations(source map[models.SyncPhase]map[string]time.Duration) map[models.SyncPhase]map[string]time.Duration {
	if len(source) == 0 {
		return nil
	}
	copyDurations := make(map[models.SyncPhase]map[string]time.Duration, len(source))
	for phase, durations := range source {
		inner := make(map[string]time.Duration, len(durations))
		for key, value := range durations {
			inner[key] = value
		}
		copyDurations[phase] = inner
	}
	return copyDurations
}

func renderPhaseDurationLines(header string, durations map[string]time.Duration, order []string) []string {
	if len(durations) == 0 {
		return nil
	}
	lines := []string{subtleStyle.Render(header)}
	rendered := make(map[string]bool, len(durations))
	for _, key := range order {
		duration := durations[key]
		if duration <= 0 {
			continue
		}
		lines = append(lines, fmt.Sprintf("    %s: %s", phaseDurationLabel(key), ui.FormatDuration(duration)))
		rendered[key] = true
	}
	for key, duration := range durations {
		if duration <= 0 || rendered[key] {
			continue
		}
		lines = append(lines, fmt.Sprintf("    %s: %s", phaseDurationLabel(key), ui.FormatDuration(duration)))
	}
	return lines
}

func phaseDurationLabel(name string) string {
	if name == "" {
		return "unknown subphase"
	}
	runes := []rune(name)
	if len(runes) == 0 {
		return "unknown subphase"
	}
	runes[0] = []rune(strings.ToLower(string(runes[0])))[0]
	return string(runes)
}

func phaseDetailForSnapshot(snapshot models.ProgressSnapshot) string {
	if detail, ok := normalizedSnapshotPhaseDetail(snapshot); ok {
		return detail
	}
	if snapshot.Phase == models.SyncPhaseRestore {
		return "Preparing local restore"
	}
	if snapshot.Phase != models.SyncPhaseDump {
		if snapshot.Message != "" {
			return snapshot.Message
		}
		return "n/a"
	}
	if snapshot.Message != "" && snapshot.Message != "Streaming remote dump" {
		return snapshot.Message
	}
	if isEarlyDumpWarmupSnapshot(snapshot) {
		return "Preparing dump metadata"
	}
	if snapshot.Percent >= 99 {
		return "Finalizing dump files"
	}
	if snapshot.Traffic.DownloadedBytes() > 0 || snapshot.BytesCompleted > 0 {
		return "Streaming table data"
	}
	return "Preparing dump metadata"
}

func normalizedSnapshotPhaseDetail(snapshot models.ProgressSnapshot) (string, bool) {
	if snapshot.Message == "" {
		return "", false
	}
	if detail, ok := normalizePhaseMessage(snapshot.Phase, snapshot.Message); ok {
		return detail, true
	}
	switch snapshot.Message {
	case "Streaming remote dump":
		return "Streaming table data", true
	case "Dump complete":
		return "Finalizing dump files", true
	case "Loading dump into local MySQL":
		return "Loading table data", true
	case "Restore complete":
		return "Finalizing restore", true
	default:
		return "", false
	}
}

func normalizePhaseMessage(phase models.SyncPhase, message string) (string, bool) {
	lower := strings.ToLower(strings.TrimSpace(message))
	if lower == "" {
		return "", false
	}
	if strings.HasPrefix(lower, "warning:") || strings.HasPrefix(lower, "note:") {
		return "", false
	}
	if phase == models.SyncPhaseRestore {
		switch {
		case strings.Contains(lower, "loading ddl and data") || strings.Contains(lower, "opening dump") || strings.Contains(lower, "dump is complete") || strings.Contains(lower, "target is mysql") || strings.Contains(lower, "scanning metadata") || strings.Contains(lower, "checking for pre-existing objects") || strings.Contains(lower, "prepar") || strings.Contains(lower, "open") || strings.Contains(lower, "validat"):
			return "Preparing local restore", true
		case strings.Contains(lower, "postamble") || strings.Contains(lower, "restore complete"):
			return "Finalizing restore", true
		case strings.Contains(lower, "schema") || strings.Contains(lower, "table ddl") || strings.Contains(lower, "view ddl") || strings.Contains(lower, "common preamble") || strings.Contains(lower, "ddl") || strings.Contains(lower, "metadata"):
			return "Applying schema metadata", true
		case strings.Contains(lower, "starting data load") || strings.Contains(lower, "loading dump into local mysql") || strings.Contains(lower, "load") || strings.Contains(lower, "import") || strings.Contains(lower, "chunk") || strings.Contains(lower, "rows") || strings.Contains(lower, "data"):
			return "Loading table data", true
		case strings.Contains(lower, "building indexes") || strings.Contains(lower, "indexing") || strings.Contains(lower, "index") || strings.Contains(lower, "analy") || strings.Contains(lower, "constraint"):
			return "Rebuilding indexes", true
		default:
			return "", false
		}
	}
	switch {
	case strings.Contains(lower, "initializ") || strings.Contains(lower, "schemas will be dumped") || strings.Contains(lower, "gather") || strings.Contains(lower, "analy") || strings.Contains(lower, "check") || strings.Contains(lower, "discover") || strings.Contains(lower, "prepar"):
		return "Preparing dump metadata", true
	case strings.Contains(lower, "table metadata"):
		return "Writing table metadata", true
	case strings.Contains(lower, "global ddl") || strings.Contains(lower, "writing ddl") || strings.Contains(lower, "schema") || strings.Contains(lower, "metadata"):
		return "Writing schema metadata", true
	case strings.Contains(lower, "running data dump") || strings.Contains(lower, "starting data dump") || strings.Contains(lower, "streaming remote dump") || strings.Contains(lower, "dumping") || strings.Contains(lower, "chunk") || strings.Contains(lower, "writing data") || strings.Contains(lower, "rows"):
		return "Streaming table data", true
	case strings.Contains(lower, "dump complete") || strings.Contains(lower, "final") || strings.Contains(lower, "compress") || strings.Contains(lower, "finish") || strings.Contains(lower, "complete"):
		return "Finalizing dump files", true
	default:
		return "", false
	}
}

func isEarlyDumpWarmupSnapshot(snapshot models.ProgressSnapshot) bool {
	if snapshot.Phase != models.SyncPhaseDump {
		return false
	}
	if snapshot.Percent >= 1 {
		return false
	}
	downloaded := snapshot.Traffic.DownloadedBytes()
	if downloaded == 0 {
		downloaded = snapshot.BytesCompleted
	}
	if downloaded >= 1024*1024 {
		return false
	}
	bytesPerSecond := snapshot.Traffic.AverageBytesPerSecond
	if bytesPerSecond <= 0 {
		bytesPerSecond = snapshot.Traffic.CurrentBytesPerSecond
	}
	return bytesPerSecond <= 128*1024
}

func mergeProgressSnapshot(previous models.ProgressSnapshot, next models.ProgressSnapshot) models.ProgressSnapshot {
	if next.DatabaseName == "" {
		next.DatabaseName = previous.DatabaseName
	}
	if next.Timestamp.IsZero() {
		next.Timestamp = previous.Timestamp
	}
	if previous.DatabaseName != next.DatabaseName || previous.Phase != next.Phase {
		return next
	}
	if next.Message == "" {
		next.Message = previous.Message
	}
	if next.BytesCompleted == 0 && previous.BytesCompleted > 0 {
		next.BytesCompleted = previous.BytesCompleted
	}
	if next.BytesTotal == 0 && previous.BytesTotal > 0 {
		next.BytesTotal = previous.BytesTotal
	}
	if next.Percent == 0 && previous.Percent > 0 {
		next.Percent = previous.Percent
	}
	if next.ETA == 0 && previous.ETA > 0 {
		next.ETA = previous.ETA
	}
	if next.Traffic.Mode == "" {
		next.Traffic.Mode = previous.Traffic.Mode
	}
	if next.Traffic.BytesIn == 0 && previous.Traffic.BytesIn > 0 {
		next.Traffic.BytesIn = previous.Traffic.BytesIn
	}
	if next.Traffic.BytesOut == 0 && previous.Traffic.BytesOut > 0 {
		next.Traffic.BytesOut = previous.Traffic.BytesOut
	}
	if next.Traffic.CurrentBytesPerSecond == 0 && previous.Traffic.CurrentBytesPerSecond > 0 {
		next.Traffic.CurrentBytesPerSecond = previous.Traffic.CurrentBytesPerSecond
	}
	if next.Traffic.AverageBytesPerSecond == 0 && previous.Traffic.AverageBytesPerSecond > 0 {
		next.Traffic.AverageBytesPerSecond = previous.Traffic.AverageBytesPerSecond
	}
	if next.Traffic.SampleWindow == 0 && previous.Traffic.SampleWindow > 0 {
		next.Traffic.SampleWindow = previous.Traffic.SampleWindow
	}
	return next
}

func (m *AppModel) estimateTargetDuration(target models.SyncTarget) time.Duration {
	logicalSize := m.targetLogicalSize(target)
	if logicalSize <= 0 {
		return 20 * time.Second
	}
	var totalBytes int64
	var totalSeconds float64
	for _, result := range m.runningResults {
		if result.LogicalSize <= 0 || result.Duration <= 0 {
			continue
		}
		totalBytes += result.LogicalSize
		totalSeconds += result.Duration.Seconds()
	}
	bytesPerSecond := float64(8 * 1024 * 1024)
	if totalBytes > 0 && totalSeconds > 0 {
		bytesPerSecond = float64(totalBytes) / totalSeconds
	}
	seconds := float64(logicalSize)/bytesPerSecond + 2
	if seconds < 3 {
		seconds = 3
	}
	return time.Duration(seconds * float64(time.Second))
}

func (m *AppModel) initSettingsFields() {
	m.settingsFields = []settingsField{
		{Label: "Remote Host", Description: "Hostname or IP of the remote MySQL server.", Kind: settingsFieldString, Get: func(cfg *config.Config) string { return cfg.Remote.Host }, Set: func(cfg *config.Config, value string) error {
			cfg.Remote.Host = strings.TrimSpace(value)
			return cfg.Validate()
		}},
		{Label: "Remote Port", Description: "TCP port for the remote MySQL server.", Kind: settingsFieldInt, Get: func(cfg *config.Config) string { return strconv.Itoa(cfg.Remote.Port) }, Set: func(cfg *config.Config, value string) error {
			port, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				return fmt.Errorf("remote port must be a number")
			}
			cfg.Remote.Port = port
			return cfg.Validate()
		}},
		{Label: "Remote User", Description: "Username used to connect to remote MySQL.", Kind: settingsFieldString, Get: func(cfg *config.Config) string { return cfg.Remote.User }, Set: func(cfg *config.Config, value string) error {
			cfg.Remote.User = strings.TrimSpace(value)
			return cfg.Validate()
		}},
		{Label: "Remote Password", Description: "Password used to connect to remote MySQL.", Kind: settingsFieldPassword, MaskValue: true, Get: func(cfg *config.Config) string { return cfg.Remote.Password }, Set: func(cfg *config.Config, value string) error { cfg.Remote.Password = value; return cfg.Validate() }},
		{Label: "Remote Proxy URL", Description: "Optional socks5/http proxy URL for remote access.", Kind: settingsFieldString, Get: func(cfg *config.Config) string { return cfg.Remote.ProxyURL }, Set: func(cfg *config.Config, value string) error {
			cfg.Remote.ProxyURL = strings.TrimSpace(value)
			return cfg.Validate()
		}},
		{Label: "Local Host", Description: "Hostname or IP of the local MySQL server.", Kind: settingsFieldString, Get: func(cfg *config.Config) string { return cfg.Local.Host }, Set: func(cfg *config.Config, value string) error {
			cfg.Local.Host = strings.TrimSpace(value)
			return cfg.Validate()
		}},
		{Label: "Local Port", Description: "TCP port for the local MySQL server.", Kind: settingsFieldInt, Get: func(cfg *config.Config) string { return strconv.Itoa(cfg.Local.Port) }, Set: func(cfg *config.Config, value string) error {
			port, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				return fmt.Errorf("local port must be a number")
			}
			cfg.Local.Port = port
			return cfg.Validate()
		}},
		{Label: "Local User", Description: "Username used to connect to local MySQL.", Kind: settingsFieldString, Get: func(cfg *config.Config) string { return cfg.Local.User }, Set: func(cfg *config.Config, value string) error {
			cfg.Local.User = strings.TrimSpace(value)
			return cfg.Validate()
		}},
		{Label: "Local Password", Description: "Password used to connect to local MySQL.", Kind: settingsFieldPassword, MaskValue: true, Get: func(cfg *config.Config) string { return cfg.Local.Password }, Set: func(cfg *config.Config, value string) error { cfg.Local.Password = value; return cfg.Validate() }},
		{Label: "Dump Timeout", Description: "Go duration like 300s or 5m.", Kind: settingsFieldDuration, Get: func(cfg *config.Config) string { return cfg.Dump.Timeout.String() }, Set: func(cfg *config.Config, value string) error {
			duration, err := time.ParseDuration(strings.TrimSpace(value))
			if err != nil {
				return fmt.Errorf("dump timeout must be a valid duration")
			}
			cfg.Dump.Timeout = duration
			return cfg.Validate()
		}},
		{Label: "Dump Threads", Description: "Parallel threads used by mysqlsh dump/load.", Kind: settingsFieldInt, Get: func(cfg *config.Config) string { return strconv.Itoa(cfg.Dump.Threads) }, Set: func(cfg *config.Config, value string) error {
			threads, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				return fmt.Errorf("dump threads must be a number")
			}
			if threads <= 0 {
				return fmt.Errorf("dump threads must be greater than zero")
			}
			cfg.Dump.Threads = threads
			return cfg.Validate()
		}},
		{Label: "Dump Compress", Description: "Enable compressed dump output.", Kind: settingsFieldBool, Get: func(cfg *config.Config) string { return strconv.FormatBool(cfg.Dump.Compress) }, Set: func(cfg *config.Config, value string) error {
			parsed, err := strconv.ParseBool(strings.TrimSpace(value))
			if err != nil {
				return fmt.Errorf("dump compress must be true or false")
			}
			cfg.Dump.Compress = parsed
			return cfg.Validate()
		}},
		{Label: "Network Compress", Description: "Enable mysqlsh client/server protocol compression for remote dump traffic.", Kind: settingsFieldBool, Get: func(cfg *config.Config) string { return strconv.FormatBool(cfg.Dump.NetworkCompress) }, Set: func(cfg *config.Config, value string) error {
			parsed, err := strconv.ParseBool(strings.TrimSpace(value))
			if err != nil {
				return fmt.Errorf("network compress must be true or false")
			}
			cfg.Dump.NetworkCompress = parsed
			return cfg.Validate()
		}},
		{Label: "Network Zstd Level", Description: "mysqlsh protocol-compression zstd level from 1 to 22.", Kind: settingsFieldInt, Get: func(cfg *config.Config) string { return strconv.Itoa(cfg.Dump.NetworkZstdLevel) }, Set: func(cfg *config.Config, value string) error {
			level, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				return fmt.Errorf("network zstd level must be a number")
			}
			cfg.Dump.NetworkZstdLevel = level
			return cfg.Validate()
		}},
		{Label: "Interactive Mode", Description: "Keep interactive mode enabled for the TUI flow.", Kind: settingsFieldBool, Get: func(cfg *config.Config) string { return strconv.FormatBool(cfg.CLI.InteractiveMode) }, Set: func(cfg *config.Config, value string) error {
			parsed, err := strconv.ParseBool(strings.TrimSpace(value))
			if err != nil {
				return fmt.Errorf("interactive mode must be true or false")
			}
			cfg.CLI.InteractiveMode = parsed
			return cfg.Validate()
		}},
		{Label: "Confirm Destructive", Description: "Whether destructive sync requires explicit confirmation.", Kind: settingsFieldBool, Get: func(cfg *config.Config) string { return strconv.FormatBool(cfg.CLI.ConfirmDestructive) }, Set: func(cfg *config.Config, value string) error {
			parsed, err := strconv.ParseBool(strings.TrimSpace(value))
			if err != nil {
				return fmt.Errorf("confirm destructive must be true or false")
			}
			cfg.CLI.ConfirmDestructive = parsed
			return cfg.Validate()
		}},
		{Label: "Log Level", Description: "Logging level, for example info, debug or warn.", Kind: settingsFieldString, Get: func(cfg *config.Config) string { return cfg.Log.Level }, Set: func(cfg *config.Config, value string) error {
			cfg.Log.Level = strings.TrimSpace(value)
			return cfg.Validate()
		}},
		{Label: "Log Format", Description: "Output log format, usually text or json.", Kind: settingsFieldString, Get: func(cfg *config.Config) string { return cfg.Log.Format }, Set: func(cfg *config.Config, value string) error {
			cfg.Log.Format = strings.TrimSpace(value)
			return cfg.Validate()
		}},
	}
}

func (m *AppModel) currentSettingsField() settingsField {
	if len(m.settingsFields) == 0 {
		return settingsField{}
	}
	if m.settingsCursor < 0 {
		m.settingsCursor = 0
	}
	if m.settingsCursor >= len(m.settingsFields) {
		m.settingsCursor = len(m.settingsFields) - 1
	}
	return m.settingsFields[m.settingsCursor]
}

func (m *AppModel) fieldDisplayValue(field settingsField) string {
	value := field.Get(m.cfg)
	if field.MaskValue && value != "" {
		value = strings.Repeat("*", minInt(12, len(value)))
	}
	if field.Kind == settingsFieldBool {
		if value == "true" {
			return okStyle.Render("true")
		}
		return dangerStyle.Render("false")
	}
	if value == "" {
		return subtleStyle.Render("(empty)")
	}
	return mutedValueStyle.Render(value)
}

func (m *AppModel) openSettingsEditor() {
	field := m.currentSettingsField()
	if field.Kind == settingsFieldBool {
		m.toggleCurrentBoolField()
		return
	}
	m.settingsEditing = true
	m.settingsBuffer = field.Get(m.cfg)
	m.settingsStatus = subtleStyle.Render("Editing " + field.Label)
}

func (m *AppModel) toggleCurrentBoolField() {
	field := m.currentSettingsField()
	current := field.Get(m.cfg)
	next := "true"
	if current == "true" {
		next = "false"
	}
	if err := field.Set(m.cfg, next); err != nil {
		m.settingsStatus = dangerStyle.Render(err.Error())
		m.setNotice(m.settingsStatus)
		return
	}
	m.settingsDirty = true
	m.settingsStatus = okStyle.Render(field.Label + " updated")
	m.setNotice(m.settingsStatus)
}

func (m *AppModel) saveSettings() {
	if err := m.cfg.SaveEnv(m.savePath); err != nil {
		m.settingsStatus = dangerStyle.Render(err.Error())
		m.setNotice(m.settingsStatus)
		return
	}
	m.settingsDirty = false
	m.settingsStatus = okStyle.Render("Saved to " + m.savePath)
	m.setNotice(m.settingsStatus)
}

func (m *AppModel) saveSettingsAndReloadCmd() tea.Cmd {
	m.saveSettings()
	if m.settingsDirty {
		return nil
	}
	return m.reloadDatabasesCmd()
}

func (m *AppModel) trafficSnapshotLabel() string {
	if m.currentProgress.Traffic.DownloadedBytes() > 0 || m.currentProgress.Traffic.UploadedBytes() > 0 {
		return fmt.Sprintf("down %s  up %s  total %s",
			ui.FormatSize(m.currentProgress.Traffic.DownloadedBytes()),
			ui.FormatSize(m.currentProgress.Traffic.UploadedBytes()),
			ui.FormatSize(m.currentProgress.Traffic.TotalBytes()),
		)
	}
	if len(m.runningResults) == 0 {
		return "n/a"
	}
	var totalDownloaded int64
	var totalUploaded int64
	for _, result := range m.runningResults {
		totalDownloaded += result.Traffic.DownloadedBytes()
		totalUploaded += result.Traffic.UploadedBytes()
	}
	if totalDownloaded == 0 && totalUploaded == 0 {
		return "n/a"
	}
	return fmt.Sprintf("down %s  up %s  total %s",
		ui.FormatSize(totalDownloaded),
		ui.FormatSize(totalUploaded),
		ui.FormatSize(totalDownloaded+totalUploaded),
	)
}

func (m *AppModel) setNotice(message string) {
	m.notice = message
}

func renderButton(label string, selected bool, destructive bool) string {
	style := lipgloss.NewStyle().Padding(0, 2).Border(lipgloss.RoundedBorder())
	if selected {
		if destructive {
			style = style.BorderForeground(lipgloss.Color("#FF4D6D")).Foreground(lipgloss.Color("#FF4D6D")).Bold(true)
		} else {
			style = style.BorderForeground(lipgloss.Color("#7C3AED")).Foreground(lipgloss.Color("#7C3AED")).Bold(true)
		}
	} else {
		style = style.BorderForeground(lipgloss.Color("#353160")).Foreground(lipgloss.Color("#9A95B8"))
	}
	return style.Render(label)
}

func renderProgressBar(width int, progress float64) string {
	if width < 8 {
		width = 8
	}
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}
	filled := int(progress * float64(width))
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	return sizeStyle.Render(bar) + subtleStyle.Render(fmt.Sprintf(" %3.0f%%", progress*100))
}

func wrapLines(lines []string, width int) string {
	trimmed := make([]string, 0, len(lines))
	for _, line := range lines {
		for _, segment := range strings.Split(line, "\n") {
			if width > 0 {
				segment = lipgloss.NewStyle().Width(width).MaxWidth(width).Render(segment)
			}
			trimmed = append(trimmed, segment)
		}
	}
	return strings.Join(trimmed, "\n")
}

func overlayCentered(base, overlay string, width, height int) string {
	if width <= 0 || height <= 0 {
		return base + "\n\n" + overlay
	}
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, overlay, lipgloss.WithWhitespaceChars(" "), lipgloss.WithWhitespaceForeground(lipgloss.Color("#0A0A12")))
}

func visibleRange(cursor, total, visible int) (int, int) {
	if total <= visible {
		return 0, total
	}
	start := cursor - visible/2
	if start < 0 {
		start = 0
	}
	end := start + visible
	if end > total {
		end = total
		start = end - visible
	}
	return start, end
}

func padRight(value string, width int) string {
	if lipgloss.Width(value) >= width {
		return value
	}
	return value + strings.Repeat(" ", width-lipgloss.Width(value))
}

func padLeft(value string, width int) string {
	if lipgloss.Width(value) >= width {
		return value
	}
	return strings.Repeat(" ", width-lipgloss.Width(value)) + value
}

func formatGroupedInt64(value int64) string {
	negative := value < 0
	if negative {
		value = -value
	}

	raw := strconv.FormatInt(value, 10)
	if len(raw) <= 3 {
		if negative {
			return "-" + raw
		}
		return raw
	}

	parts := make([]string, 0, (len(raw)+2)/3)
	for len(raw) > 3 {
		parts = append([]string{raw[len(raw)-3:]}, parts...)
		raw = raw[:len(raw)-3]
	}
	parts = append([]string{raw}, parts...)
	grouped := strings.Join(parts, " ")
	if negative {
		return "-" + grouped
	}
	return grouped
}

func databaseColumnWidths(databases models.DatabaseList) (int, int, int) {
	nameWidth := 0
	sizeWidth := 0
	tablesWidth := 0
	for _, db := range databases {
		nameWidth = maxInt(nameWidth, lipgloss.Width(db.Name))
		sizeWidth = maxInt(sizeWidth, lipgloss.Width(ui.FormatSize(displayDatabaseBytes(db))))
		tablesWidth = maxInt(tablesWidth, lipgloss.Width(formatGroupedInt64(int64(db.Tables))))
	}
	return maxInt(nameWidth, 18), maxInt(sizeWidth, 8), maxInt(tablesWidth, 2)
}

func tableColumnWidths(tables []models.Table) (int, int, int) {
	nameWidth := 0
	sizeWidth := 0
	rowsWidth := 0
	for _, table := range tables {
		nameWidth = maxInt(nameWidth, lipgloss.Width(table.Name))
		sizeWidth = maxInt(sizeWidth, lipgloss.Width(ui.FormatSize(displayTableBytes(table))))
		rowValue := formatGroupedInt64(table.Rows)
		if table.RowsApprox {
			rowValue = "~" + rowValue
		}
		rowsWidth = maxInt(rowsWidth, lipgloss.Width(rowValue))
	}
	return maxInt(nameWidth, 18), maxInt(sizeWidth, 8), maxInt(rowsWidth, 4)
}

func displayDatabaseBytes(db models.Database) int64 {
	if db.DataSize > 0 {
		return db.DataSize
	}
	return db.Size
}

func displayTableBytes(table models.Table) int64 {
	if table.DataSize > 0 {
		return table.DataSize
	}
	return table.Size
}

func cursorSuffix() string { return "_" }

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}

func clampInt(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}
