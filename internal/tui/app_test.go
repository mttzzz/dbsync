package tui

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"db-sync-cli/internal/config"
	"db-sync-cli/internal/models"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockBrowser struct {
	remoteInfo    *models.ConnectionInfo
	localInfo     *models.ConnectionInfo
	databases     models.DatabaseList
	tablesByDB    map[string][]models.Table
	depsByDB      map[string][]models.TableDependency
	remoteErr     error
	localErr      error
	listTablesErr error
}

func (m *mockBrowser) TestConnection(isRemote bool) (*models.ConnectionInfo, error) {
	if isRemote {
		return m.remoteInfo, m.remoteErr
	}
	return m.localInfo, m.localErr
}

func (m *mockBrowser) ListDatabases(isRemote bool) (models.DatabaseList, error) {
	return append(models.DatabaseList(nil), m.databases...), nil
}

func (m *mockBrowser) ListTables(databaseName string, isRemote bool) ([]models.Table, error) {
	if m.listTablesErr != nil {
		return nil, m.listTablesErr
	}
	return append([]models.Table(nil), m.tablesByDB[databaseName]...), nil
}

func (m *mockBrowser) ListTableDependencies(databaseName string, tableNames []string, isRemote bool) ([]models.TableDependency, error) {
	return append([]models.TableDependency(nil), m.depsByDB[databaseName]...), nil
}

type mockRunner struct {
	results map[string]*models.SyncResult
	errs    map[string]error
}

func (m *mockRunner) ExecuteTarget(target models.SyncTarget) (*models.SyncResult, error) {
	if err := m.errs[target.DatabaseName]; err != nil {
		return nil, err
	}
	if result, ok := m.results[target.DatabaseName]; ok {
		copyResult := *result
		return &copyResult, nil
	}
	return &models.SyncResult{DatabaseName: target.DatabaseName, Success: true}, nil
}

func (m *mockRunner) ExecutePlan(plan *models.SyncPlan, runtime models.RuntimeOptions, observer models.ProgressObserver) ([]models.SyncResult, error) {
	results := make([]models.SyncResult, 0, len(plan.Targets))
	for _, target := range plan.Targets {
		if observer != nil {
			observer(models.ProgressSnapshot{Phase: models.SyncPhaseDump, DatabaseName: target.DatabaseName, Percent: 50, Message: "dumping", Timestamp: time.Now()})
		}
		result, err := m.ExecuteTarget(target)
		if err != nil {
			return results, err
		}
		results = append(results, *result)
		if observer != nil {
			observer(models.ProgressSnapshot{Phase: models.SyncPhaseDone, DatabaseName: target.DatabaseName, Percent: 100, Message: "done", Timestamp: time.Now()})
		}
	}
	_ = runtime
	return results, nil
}

func TestListEnterOpensTables(t *testing.T) {
	model := newTestModel()

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app := updated.(*AppModel)

	assert.Equal(t, viewTables, app.view)
	assert.Equal(t, "beta", app.previewDatabase)
	assert.True(t, app.selectedDatabases["beta"])
}

func TestTablesLoadedAndDependencyAutoIncluded(t *testing.T) {
	model := newTestModel()
	model.previewDatabase = "beta"
	model.view = viewTables

	updated, _ := model.Update(tablesLoadedMsg{
		DatabaseName: "beta",
		Tables: []models.Table{
			{Name: "orders", Size: 200, Rows: 10},
			{Name: "users", Size: 100, Rows: 3},
		},
		Dependencies: []models.TableDependency{{TableName: "orders", ReferencedTable: "users"}},
	})
	app := updated.(*AppModel)
	state := app.tableState("beta")
	require.Len(t, state.VisibleTables, 2)
	state.Selected = make(map[string]bool)
	state.Initialized = true

	updated, _ = app.Update(tea.KeyMsg{Type: tea.KeySpace})
	app = updated.(*AppModel)
	state = app.tableState("beta")

	assert.True(t, state.Selected["orders"])
	assert.True(t, state.AutoIncluded["users"])
}

func TestTablesLoadedSelectsAllByDefault(t *testing.T) {
	model := newTestModel()
	model.previewDatabase = "beta"
	model.view = viewTables

	updated, _ := model.Update(tablesLoadedMsg{
		DatabaseName: "beta",
		Tables: []models.Table{
			{Name: "orders", Size: 200, Rows: 10},
			{Name: "users", Size: 100, Rows: 3},
		},
	})
	app := updated.(*AppModel)
	state := app.tableState("beta")

	assert.True(t, state.Initialized)
	assert.True(t, state.Selected["orders"])
	assert.True(t, state.Selected["users"])
	assert.Empty(t, state.AutoIncluded)
}

func TestTablesReloadKeepsManualClearSelection(t *testing.T) {
	model := newTestModel()
	state := model.tableState("beta")
	state.Initialized = true
	state.Selected = map[string]bool{}
	model.previewDatabase = "beta"
	model.view = viewTables

	updated, _ := model.Update(tablesLoadedMsg{
		DatabaseName: "beta",
		Tables: []models.Table{
			{Name: "orders", Size: 200, Rows: 10},
			{Name: "users", Size: 100, Rows: 3},
		},
	})
	app := updated.(*AppModel)
	state = app.tableState("beta")

	assert.Empty(t, state.Selected)
	assert.False(t, state.Selected["orders"])
	assert.False(t, state.Selected["users"])
}

func TestConfirmEnterStartsRunning(t *testing.T) {
	model := newTestModel()
	model.selectedDatabases["beta"] = true
	model.view = viewConfirm
	model.confirmChoice = confirmSync

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app := updated.(*AppModel)

	require.NotNil(t, cmd)
	assert.Equal(t, viewRunning, app.view)
	assert.True(t, app.running)
	assert.Equal(t, "beta", app.runningTargetName)
}

func TestListYOpensPlanView(t *testing.T) {
	model := newTestModel()
	model.selectedDatabases["beta"] = true

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	app := updated.(*AppModel)

	assert.Equal(t, viewPlan, app.view)
}

func TestPlanUppercaseYOpensConfirmWithSyncSelected(t *testing.T) {
	model := newTestModel()
	model.selectedDatabases["beta"] = true
	model.view = viewPlan

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Y'}})
	app := updated.(*AppModel)

	assert.Equal(t, viewConfirm, app.view)
	assert.Equal(t, confirmSync, app.confirmChoice)
}

func TestConfirmUppercaseYArmsSync(t *testing.T) {
	model := newTestModel()
	model.selectedDatabases["beta"] = true
	model.view = viewConfirm
	model.confirmChoice = confirmCancel

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Y'}})
	app := updated.(*AppModel)

	assert.Equal(t, confirmSync, app.confirmChoice)
}

func TestConfirmCtrlMStartsRunning(t *testing.T) {
	model := newTestModel()
	model.selectedDatabases["beta"] = true
	model.view = viewConfirm
	model.confirmChoice = confirmSync

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlM})
	app := updated.(*AppModel)

	require.NotNil(t, cmd)
	assert.Equal(t, viewRunning, app.view)
	assert.True(t, app.running)
}

func TestRenderConfirmViewPlacesButtonsOnSameRow(t *testing.T) {
	model := newTestModel()
	model.selectedDatabases["beta"] = true
	model.view = viewConfirm
	model.confirmChoice = confirmSync

	rendered := stripANSI(model.renderConfirmView(90))
	assert.NotContains(t, rendered, "╰────────╯   ╭")

	lines := strings.Split(rendered, "\n")
	hasButtonRow := false
	for _, line := range lines {
		if strings.Contains(line, "Cancel") && strings.Contains(line, "Sync") {
			hasButtonRow = true
			break
		}
	}
	assert.True(t, hasButtonRow, "expected confirm buttons to share one rendered row")
}

func TestWrapLinesSplitsEmbeddedNewlines(t *testing.T) {
	rendered := stripANSI(wrapLines([]string{"alpha\nbeta", "gamma"}, 0))
	assert.Equal(t, []string{"alpha", "beta", "gamma"}, strings.Split(rendered, "\n"))
}

func TestInitStartsDatabaseLoadWhenNoPreloadedDatabases(t *testing.T) {
	browser := &mockBrowser{databases: models.DatabaseList{{Name: "alpha", Size: 100, Tables: 1}}}
	model := NewAppModel(&config.Config{
		Remote: config.MySQLConfig{Host: "remote.example.com", Port: 3306},
		Local:  config.MySQLConfig{Host: "localhost", Port: 3306},
		Dump:   config.DumpConfig{Threads: 8, Timeout: 5 * time.Minute, Compress: true},
	}, browser, &mockRunner{}, nil)

	cmd := model.Init()
	assert.NotNil(t, cmd)
	assert.True(t, model.databasesLoading)
	assert.Contains(t, stripANSI(model.notice), "Loading remote databases...")
}

func TestInitSkipsDatabaseLoadWhenPreloadedDatabasesExist(t *testing.T) {
	model := newTestModel()

	cmd := model.Init()
	assert.Nil(t, cmd)
	assert.False(t, model.databasesLoading)
}

func TestRenderListViewShowsLoadingState(t *testing.T) {
	model := newTestModel()
	model.databases = nil
	model.filtered = nil
	model.databasesLoading = true

	rendered := stripANSI(model.renderListView(90))
	assert.Contains(t, rendered, "Loading remote databases...")
	assert.NotContains(t, rendered, "No databases match the current filter.")
}

func TestRenderListViewShowsManualLoadHintWhenNoDatabasesLoaded(t *testing.T) {
	model := newTestModel()
	model.databases = nil
	model.filtered = nil
	model.databasesLoading = false
	model.search = ""

	rendered := stripANSI(model.renderListView(90))
	assert.Contains(t, rendered, "No databases loaded yet. Press R to load remote databases.")
}

func TestFormatGroupedInt64(t *testing.T) {
	assert.Equal(t, "0", formatGroupedInt64(0))
	assert.Equal(t, "62", formatGroupedInt64(62))
	assert.Equal(t, "1 052 553", formatGroupedInt64(1052553))
	assert.Equal(t, "-1 234 567", formatGroupedInt64(-1234567))
}

func TestRenderTablesViewShowsGroupedRowsAndApproximateNote(t *testing.T) {
	model := newTestModel()
	model.previewDatabase = "beta"
	model.view = viewTables
	updated, _ := model.Update(tablesLoadedMsg{
		DatabaseName: "beta",
		Tables: []models.Table{
			{Name: "amocrm_custom_field_value", Size: 353700000, Rows: 1052553},
			{Name: "project_note", Size: 3400000, Rows: 5000, RowsApprox: true},
		},
	})
	app := updated.(*AppModel)

	rendered := stripANSI(app.renderTablesView(120))
	assert.Contains(t, rendered, "1 052 553 rows")
	assert.Contains(t, rendered, "~5 000 rows")
	assert.Contains(t, rendered, "Sizes show source data estimate")
	assert.Contains(t, rendered, "Rows prefixed with ~ are fallback estimates")
}

func TestRenderListViewShowsAlignedGroupedTableCounts(t *testing.T) {
	model := newTestModel()
	model.databases = models.DatabaseList{{Name: "alpha", Size: 2048, Tables: 3}, {Name: "wide_database_name", Size: 2048000, Tables: 1234}}
	model.updateFilter()

	rendered := stripANSI(model.renderListView(120))
	assert.Contains(t, rendered, "1 234 tables")
	assert.Contains(t, rendered, "Source data est.:")
}

func TestRenderRunningViewShowsLiveTrafficMetrics(t *testing.T) {
	model := newTestModel()
	model.view = viewRunning
	model.running = true
	model.runningPlan = &models.SyncPlan{Targets: []models.SyncTarget{{DatabaseName: "beta"}}}
	model.runningTargetName = "beta"
	model.runningStartedAt = time.Now().Add(-5 * time.Second)
	model.runningNow = time.Now()
	model.phaseTimings["beta"] = &phaseTimingTracker{
		currentPhase:  models.SyncPhaseDump,
		currentDetail: "Streaming table data",
		currentAt:     model.runningNow.Add(-3 * time.Second),
		durations: map[models.SyncPhase]map[string]time.Duration{
			models.SyncPhaseDump: {
				"Preparing dump metadata": 2 * time.Second,
			},
		},
	}
	model.currentProgress = models.ProgressSnapshot{
		Phase:          models.SyncPhaseDump,
		DatabaseName:   "beta",
		Message:        "Streaming remote dump",
		BytesCompleted: 2 * 1024 * 1024,
		BytesTotal:     10 * 1024 * 1024,
		Traffic: models.TrafficMetrics{
			BytesIn:               2 * 1024 * 1024,
			BytesOut:              512 * 1024,
			CurrentBytesPerSecond: 256 * 1024,
			AverageBytesPerSecond: 512 * 1024,
		},
	}

	rendered := stripANSI(model.renderRunningView(100))
	assert.Contains(t, rendered, "Dump subphase: Streaming table data")
	assert.Contains(t, rendered, "Dump ETA: 16.0s")
	assert.Contains(t, rendered, "Download speed: 512.0 KB/s avg")
	assert.Contains(t, rendered, "Dump progress (downloaded): 2.0 MB / 10.0 MB")
	assert.Contains(t, rendered, "Traffic snapshot: down 2.0 MB  up 512.0 KB  total 2.5 MB")
	assert.Contains(t, rendered, "Current step: Streaming remote dump")
	assert.Contains(t, rendered, "Live phase timers:")
	assert.Contains(t, rendered, "dump breakdown:")
	assert.Contains(t, rendered, "preparing dump metadata: 2.0s")
	assert.Contains(t, rendered, "streaming table data: 3.0s")
}

func TestRenderRunningViewHandlesUnknownPhaseTimerLabel(t *testing.T) {
	model := newTestModel()
	model.view = viewRunning
	model.running = true
	model.runningPlan = &models.SyncPlan{Targets: []models.SyncTarget{{DatabaseName: "beta"}}}
	model.runningTargetName = "beta"
	model.runningStartedAt = time.Now().Add(-5 * time.Second)
	model.runningNow = time.Now()
	model.phaseTimings["beta"] = &phaseTimingTracker{durations: map[models.SyncPhase]map[string]time.Duration{
		models.SyncPhaseDump: {
			"": 2 * time.Second,
		},
	}}
	model.currentProgress = models.ProgressSnapshot{
		Phase:        models.SyncPhaseDump,
		DatabaseName: "beta",
		Message:      "Streaming remote dump",
	}

	rendered := stripANSI(model.renderRunningView(100))
	assert.Contains(t, rendered, "Live phase timers:")
	assert.Contains(t, rendered, "unknown subphase: 2.0s")
}

func TestRenderRunningViewHandlesActiveTimerWithoutAccumulatedMap(t *testing.T) {
	model := newTestModel()
	model.view = viewRunning
	model.running = true
	model.runningPlan = &models.SyncPlan{Targets: []models.SyncTarget{{DatabaseName: "beta"}}}
	model.runningTargetName = "beta"
	model.runningStartedAt = time.Now().Add(-5 * time.Second)
	model.runningNow = time.Now()
	model.phaseTimings["beta"] = &phaseTimingTracker{
		currentPhase:  models.SyncPhaseDump,
		currentDetail: "Streaming table data",
		currentAt:     model.runningNow.Add(-2 * time.Second),
	}
	model.currentProgress = models.ProgressSnapshot{
		Phase:        models.SyncPhaseDump,
		DatabaseName: "beta",
		Message:      "Streaming remote dump",
	}

	rendered := stripANSI(model.renderRunningView(100))
	assert.Contains(t, rendered, "Live phase timers:")
	assert.Contains(t, rendered, "streaming table data: 2.0s")
}

func TestPhaseDetailForSnapshotIgnoresUnknownRawMessage(t *testing.T) {
	snapshot := models.ProgressSnapshot{
		Phase:        models.SyncPhaseRestore,
		DatabaseName: "beta",
		Message:      "aae455f5-681f-11eb-a5d4-4a2067ec01ac:1-218565949,;",
	}

	assert.Equal(t, "Preparing local restore", phaseDetailForSnapshot(snapshot))
}

func TestRunningETAUsesObservedTraffic(t *testing.T) {
	model := newTestModel()
	model.running = true
	model.runningPlan = &models.SyncPlan{Targets: []models.SyncTarget{{DatabaseName: "beta"}}}
	model.runningTargetName = "beta"
	model.currentProgress = models.ProgressSnapshot{
		Phase:          models.SyncPhaseDump,
		DatabaseName:   "beta",
		BytesCompleted: 2 * 1024 * 1024,
		BytesTotal:     10 * 1024 * 1024,
		Traffic: models.TrafficMetrics{
			AverageBytesPerSecond: 512 * 1024,
		},
	}

	assert.Equal(t, "16.0s", model.runningETA())
}

func TestRunningETAWarmsUpDuringEarlyDumpMetadataPhase(t *testing.T) {
	model := newTestModel()
	model.running = true
	model.runningPlan = &models.SyncPlan{Targets: []models.SyncTarget{{DatabaseName: "beta"}}}
	model.runningTargetName = "beta"
	model.currentProgress = models.ProgressSnapshot{
		Phase:          models.SyncPhaseDump,
		DatabaseName:   "beta",
		BytesCompleted: 105 * 1024,
		BytesTotal:     375 * 1024 * 1024,
		Traffic: models.TrafficMetrics{
			BytesIn:               105 * 1024,
			AverageBytesPerSecond: 3 * 1024,
		},
	}

	assert.Equal(t, "warming up...", model.runningETA())
	assert.Equal(t, "Preparing dump metadata", model.runningPhaseDetail())
}

func TestMergeProgressSnapshotPreservesProgressWithinSamePhase(t *testing.T) {
	previous := models.ProgressSnapshot{
		Phase:          models.SyncPhaseDump,
		DatabaseName:   "beta",
		Message:        "Streaming remote dump",
		BytesCompleted: 375 * 1024 * 1024,
		BytesTotal:     400 * 1024 * 1024,
		Percent:        93.75,
		Traffic: models.TrafficMetrics{
			BytesIn:               375 * 1024 * 1024,
			AverageBytesPerSecond: 8 * 1024 * 1024,
		},
	}
	next := models.ProgressSnapshot{
		Phase:        models.SyncPhaseDump,
		DatabaseName: "beta",
		Message:      "Finalizing dump files",
	}

	merged := mergeProgressSnapshot(previous, next)
	assert.Equal(t, int64(375*1024*1024), merged.BytesCompleted)
	assert.Equal(t, int64(400*1024*1024), merged.BytesTotal)
	assert.Equal(t, 93.75, merged.Percent)
	assert.Equal(t, "Finalizing dump files", merged.Message)
	assert.Equal(t, int64(375*1024*1024), merged.Traffic.BytesIn)
	assert.Equal(t, "Dump subphase", (&AppModel{currentProgress: merged}).runningPhaseDetailLabel())
}

func TestRunningPhaseDetailUsesRestoreMessage(t *testing.T) {
	model := newTestModel()
	model.currentProgress = models.ProgressSnapshot{
		Phase:        models.SyncPhaseRestore,
		DatabaseName: "beta",
		Message:      "Rebuilding indexes",
	}

	assert.Equal(t, "Restore subphase", model.runningPhaseDetailLabel())
	assert.Equal(t, "Rebuilding indexes", model.runningPhaseDetail())
}

func TestRenderReportViewShowsPhaseBreakdown(t *testing.T) {
	model := newTestModel()
	model.view = viewReport
	model.runningResults = []models.SyncResult{{
		DatabaseName:    "beta",
		Success:         true,
		LogicalSize:     7 * 1024 * 1024,
		IndexSize:       3 * 1024 * 1024,
		DumpSizeOnDisk:  2 * 1024 * 1024,
		Duration:        42*time.Second + 500*time.Millisecond,
		DumpDuration:    35 * time.Second,
		RestoreDuration: 7*time.Second + 500*time.Millisecond,
		Traffic:         models.TrafficMetrics{BytesIn: 6 * 1024 * 1024, BytesOut: 2 * 1024 * 1024},
	}}
	model.phaseTimings["beta"] = &phaseTimingTracker{durations: map[models.SyncPhase]map[string]time.Duration{
		models.SyncPhaseDump: {
			"Preparing dump metadata": 4 * time.Second,
			"Streaming table data":    29 * time.Second,
			"Finalizing dump files":   2 * time.Second,
		},
		models.SyncPhaseRestore: {
			"Applying schema metadata": 2 * time.Second,
			"Loading table data":       3 * time.Second,
			"Rebuilding indexes":       2 * time.Second,
		},
	}}

	rendered := stripANSI(model.renderReportView(120))
	assert.Contains(t, rendered, "dump breakdown:")
	assert.Contains(t, rendered, "preparing dump metadata: 4.0s")
	assert.Contains(t, rendered, "streaming table data: 29.0s")
	assert.Contains(t, rendered, "finalizing dump files: 2.0s")
	assert.Contains(t, rendered, "restore breakdown:")
	assert.Contains(t, rendered, "applying schema metadata: 2.0s")
	assert.Contains(t, rendered, "loading table data: 3.0s")
	assert.Contains(t, rendered, "rebuilding indexes: 2.0s")
}

func TestRenderReportViewShowsDumpAndRestoreDurations(t *testing.T) {
	model := newTestModel()
	model.view = viewReport
	model.runningResults = []models.SyncResult{{
		DatabaseName:    "beta",
		Success:         true,
		LogicalSize:     7 * 1024 * 1024,
		IndexSize:       3 * 1024 * 1024,
		DumpSizeOnDisk:  2 * 1024 * 1024,
		Duration:        42*time.Second + 500*time.Millisecond,
		DumpDuration:    35 * time.Second,
		RestoreDuration: 7*time.Second + 500*time.Millisecond,
		Traffic:         models.TrafficMetrics{BytesIn: 6 * 1024 * 1024, BytesOut: 2 * 1024 * 1024},
	}}

	rendered := stripANSI(model.renderReportView(100))
	assert.Contains(t, rendered, "downloaded from remote: 6.0 MB")
	assert.Contains(t, rendered, "avg dump speed: 175.5 KB/s")
	assert.Contains(t, rendered, "compressed dump on disk: 2.0 MB")
	assert.Contains(t, rendered, "uploaded control traffic: 2.0 MB")
	assert.Contains(t, rendered, "total network I/O: 8.0 MB")
	assert.Contains(t, rendered, "source footprint context:")
	assert.Contains(t, rendered, "source data estimate: 7.0 MB")
	assert.Contains(t, rendered, "source index estimate: 3.0 MB")
	assert.Contains(t, rendered, "total duration: 42.5s")
	assert.Contains(t, rendered, "dump phase: 35.0s   restore phase: 7.5s")
	assert.NotContains(t, rendered, "Total source")
}

func TestViewDoesNotRenderSummarySidebar(t *testing.T) {
	model := newTestModel()
	model.selectedDatabases["beta"] = true
	model.previewDatabase = "beta"
	model.view = viewList

	rendered := stripANSI(model.View())
	assert.NotContains(t, rendered, "Summary")
	assert.NotContains(t, rendered, "Active Database")
}

func TestListReloadsDatabases(t *testing.T) {
	model := newTestModel()
	model.browser.(*mockBrowser).databases = models.DatabaseList{{Name: "omega", Size: 8192, Tables: 7}}

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	app := updated.(*AppModel)
	require.NotNil(t, cmd)

	updated, _ = app.Update(cmd().(databasesReloadedMsg))
	app = updated.(*AppModel)
	require.Len(t, app.databases, 1)
	assert.Equal(t, "omega", app.databases[0].Name)
}

func TestReloadRefreshesOpenTableState(t *testing.T) {
	model := newTestModel()
	browser := model.browser.(*mockBrowser)
	model.previewDatabase = "beta"
	model.view = viewTables
	state := model.tableState("beta")
	state.Loaded = true
	state.Tables = []models.Table{{Name: "old_table", Size: 10, Rows: 1}}
	state.VisibleTables = append([]models.Table(nil), state.Tables...)
	browser.databases = models.DatabaseList{{Name: "beta", Size: 4096, Tables: 2}}
	browser.tablesByDB["beta"] = []models.Table{{Name: "fresh_orders", Size: 300, Rows: 12}}
	browser.depsByDB["beta"] = nil

	updated, _ := model.Update(databasesReloadedMsg{Databases: browser.databases})
	app := updated.(*AppModel)
	require.True(t, app.tableState("beta").Loading)

	updated, _ = app.Update(tablesLoadedMsg{DatabaseName: "beta", Tables: browser.tablesByDB["beta"]})
	app = updated.(*AppModel)
	require.Len(t, app.tableState("beta").VisibleTables, 1)
	assert.Equal(t, "fresh_orders", app.tableState("beta").VisibleTables[0].Name)
}

func TestSettingsNavigation(t *testing.T) {
	model := newTestModel()

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	app := updated.(*AppModel)
	assert.Equal(t, viewSettings, app.view)

	updated, _ = app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app = updated.(*AppModel)
	assert.Equal(t, viewList, app.view)
}

func TestSettingsToggleAndSave(t *testing.T) {
	model := newTestModel()
	model.savePath = filepath.Join(t.TempDir(), ".dbsync.env")
	model.view = viewSettings
	model.previousView = viewList
	model.settingsCursor = 11

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeySpace})
	app := updated.(*AppModel)
	assert.False(t, app.cfg.Dump.Compress)
	assert.True(t, app.settingsDirty)

	updated, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	app = updated.(*AppModel)
	assert.False(t, app.settingsDirty)
	assert.FileExists(t, app.savePath)
	content, err := os.ReadFile(app.savePath)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "DBSYNC_DUMP_COMPRESS=false")
}

func TestSettingsConnectionTestUpdatesStatus(t *testing.T) {
	model := newTestModel()
	model.view = viewSettings

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	app := updated.(*AppModel)
	require.NotNil(t, cmd)
	assert.True(t, app.connectionTesting)

	msg := cmd().(connectionTestMsg)
	updated, _ = app.Update(msg)
	app = updated.(*AppModel)
	assert.False(t, app.connectionTesting)
	assert.Contains(t, app.remoteTestStatus, "connected")
}

func TestSettingsEditStringField(t *testing.T) {
	model := newTestModel()
	model.view = viewSettings
	model.settingsCursor = 0

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app := updated.(*AppModel)
	assert.True(t, app.settingsEditing)

	for _, key := range []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune{'-'}},
		{Type: tea.KeyRunes, Runes: []rune{'d'}},
		{Type: tea.KeyRunes, Runes: []rune{'e'}},
		{Type: tea.KeyRunes, Runes: []rune{'v'}},
	} {
		updated, _ = app.Update(key)
		app = updated.(*AppModel)
	}

	updated, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = updated.(*AppModel)
	assert.Equal(t, "remote.example.com-dev", app.cfg.Remote.Host)
	assert.False(t, app.settingsEditing)
	assert.True(t, app.settingsDirty)
}

func TestSearchFiltersDatabases(t *testing.T) {
	model := newTestModel()

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	app := updated.(*AppModel)
	updated, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	app = updated.(*AppModel)

	assert.True(t, app.searching)
	assert.Equal(t, "a", app.search)
	assert.Len(t, app.filtered, 3)

	updated, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	app = updated.(*AppModel)
	assert.Equal(t, "am", app.search)
	assert.Len(t, app.filtered, 1)
	assert.Equal(t, "gamma", app.filtered[0].Name)
}

func TestRunningResultTransitionsToReport(t *testing.T) {
	model := newTestModel()
	model.selectedDatabases["beta"] = true
	model.runningPlan = model.buildPlan()
	model.running = true
	model.runningTargetName = "beta"
	model.view = viewRunning
	model.runDoneCh = make(chan planRunDone, 1)
	model.runProgressCh = make(chan models.ProgressSnapshot, 1)
	model.runDoneCh <- planRunDone{Results: []models.SyncResult{{DatabaseName: "beta", Success: true, Duration: 2 * time.Second, LogicalSize: 4096, Traffic: models.TrafficMetrics{BytesIn: 1024, BytesOut: 2048}}}}

	updated, _ := model.Update(runTickMsg(time.Now()))
	app := updated.(*AppModel)

	assert.False(t, app.running)
	assert.Equal(t, viewReport, app.view)
	require.Len(t, app.runningResults, 1)
	assert.Equal(t, "beta", app.runningResults[0].DatabaseName)
}

func TestHelpOverlayToggles(t *testing.T) {
	model := newTestModel()

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	app := updated.(*AppModel)
	assert.True(t, app.showHelp)

	updated, _ = app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app = updated.(*AppModel)
	assert.False(t, app.showHelp)
}

func newTestModel() *AppModel {
	browser := &mockBrowser{
		remoteInfo: &models.ConnectionInfo{Connected: true, Host: "remote.example.com", Version: "8.0.36"},
		localInfo:  &models.ConnectionInfo{Connected: true, Host: "localhost", Version: "8.0.36"},
		databases: models.DatabaseList{
			{Name: "alpha", Size: 2048, Tables: 3},
			{Name: "beta", Size: 4096, Tables: 5},
			{Name: "gamma", Size: 1024, Tables: 2},
		},
		tablesByDB: map[string][]models.Table{
			"beta": {
				{Name: "orders", Size: 200, Rows: 10},
				{Name: "users", Size: 100, Rows: 3},
			},
		},
		depsByDB: map[string][]models.TableDependency{
			"beta": {
				{TableName: "orders", ReferencedTable: "users"},
			},
		},
	}
	runner := &mockRunner{results: map[string]*models.SyncResult{
		"alpha": {DatabaseName: "alpha", Success: true, Duration: time.Second, LogicalSize: 2048, Traffic: models.TrafficMetrics{BytesIn: 400, BytesOut: 600}},
		"beta":  {DatabaseName: "beta", Success: true, Duration: 2 * time.Second, LogicalSize: 4096, Traffic: models.TrafficMetrics{BytesIn: 800, BytesOut: 1200}},
	}}
	return NewAppModel(&config.Config{
		Remote: config.MySQLConfig{Host: "remote.example.com", Port: 3306},
		Local:  config.MySQLConfig{Host: "localhost", Port: 3306},
		Dump:   config.DumpConfig{Threads: 8, Timeout: 5 * time.Minute, Compress: true},
		CLI:    config.CLIConfig{InteractiveMode: true, ConfirmDestructive: true},
		Log:    config.LogConfig{Level: "info", Format: "text"},
	}, browser, runner, models.DatabaseList{
		{Name: "alpha", Size: 2048, Tables: 3},
		{Name: "beta", Size: 4096, Tables: 5},
		{Name: "gamma", Size: 1024, Tables: 2},
	})
}

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(value string) string {
	return ansiPattern.ReplaceAllString(value, "")
}
