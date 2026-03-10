//go:build integration
// +build integration

package integration

import (
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"db-sync-cli/internal/config"
	"db-sync-cli/internal/models"
	"db-sync-cli/internal/services"
)

func TestMySQLShellService_CreateDumpTarget_Integration(t *testing.T) {
	targetDatabase := strings.TrimSpace(os.Getenv("DBSYNC_TEST_DATABASE"))
	if targetDatabase == "" {
		t.Skip("Integration dump test requires DBSYNC_TEST_DATABASE")
	}

	cfg := integrationConfig(t)
	dbService := services.NewDatabaseService(cfg)
	service := services.NewMySQLShellService(cfg, dbService)
	service.SetQuiet(true)

	result, dumpDir, err := service.CreateDumpTarget(models.SyncTarget{DatabaseName: targetDatabase, ReplaceEntireDatabase: true}, false)
	if dumpDir != "" {
		defer os.RemoveAll(dumpDir)
	}
	if err != nil {
		t.Fatalf("CreateDumpTarget() error = %v", err)
	}
	if result == nil {
		t.Fatal("CreateDumpTarget() returned nil result")
	}
	if !result.Success {
		t.Fatalf("CreateDumpTarget() returned unsuccessful result: %+v", result)
	}
	if result.DatabaseName != targetDatabase {
		t.Fatalf("DatabaseName = %q, want %q", result.DatabaseName, targetDatabase)
	}
	if result.DumpSizeOnDisk <= 0 {
		t.Fatalf("DumpSizeOnDisk = %d, want > 0", result.DumpSizeOnDisk)
	}
	if result.TransportMode == "" {
		t.Fatal("TransportMode should not be empty")
	}
	if result.Traffic.TotalBytes() <= 0 {
		t.Fatalf("Traffic.TotalBytes() = %d, want > 0", result.Traffic.TotalBytes())
	}
	if _, err := os.Stat(dumpDir); err != nil {
		t.Fatalf("dump directory missing: %v", err)
	}
	if result.Duration <= 0 {
		t.Fatalf("Duration = %v, want > 0", result.Duration)
	}
	if result.EndTime.Before(result.StartTime) {
		t.Fatalf("invalid timestamps: start=%v end=%v", result.StartTime, result.EndTime)
	}
	if t.Failed() {
		t.Logf("result: %+v", result)
	}
}

func TestMySQLShellService_CreateDumpTarget_SelectedTables_Integration(t *testing.T) {
	targetDatabase := strings.TrimSpace(os.Getenv("DBSYNC_TEST_DATABASE"))
	if targetDatabase == "" {
		t.Skip("Integration dump test requires DBSYNC_TEST_DATABASE")
	}

	cfg := integrationConfig(t)
	dbService := services.NewDatabaseService(cfg)
	service := services.NewMySQLShellService(cfg, dbService)
	service.SetQuiet(true)

	tables, err := dbService.ListTables(targetDatabase, true)
	if err != nil {
		t.Fatalf("ListTables() error = %v", err)
	}
	if len(tables) < 2 {
		t.Skip("selected-table integration test requires at least 2 tables")
	}

	target := models.SyncTarget{
		DatabaseName:          targetDatabase,
		SelectedTables:        []string{tables[0].Name},
		AutoIncludedTables:    []string{tables[1].Name},
		ReplaceEntireDatabase: false,
	}

	result, dumpDir, err := service.CreateDumpTarget(target, false)
	if dumpDir != "" {
		defer os.RemoveAll(dumpDir)
	}
	if err != nil {
		t.Fatalf("CreateDumpTarget(selected tables) error = %v", err)
	}
	if result == nil {
		t.Fatal("CreateDumpTarget(selected tables) returned nil result")
	}
	if len(result.SelectedTables) != 1 || result.SelectedTables[0] != tables[0].Name {
		t.Fatalf("SelectedTables = %+v, want [%s]", result.SelectedTables, tables[0].Name)
	}
	if len(result.AutoIncludedTables) != 1 || result.AutoIncludedTables[0] != tables[1].Name {
		t.Fatalf("AutoIncludedTables = %+v, want [%s]", result.AutoIncludedTables, tables[1].Name)
	}
	if result.TablesCount < 2 {
		t.Fatalf("TablesCount = %d, want >= 2 for selected+auto-included tables", result.TablesCount)
	}
	if result.DumpSizeOnDisk <= 0 {
		t.Fatalf("DumpSizeOnDisk = %d, want > 0", result.DumpSizeOnDisk)
	}
}

func TestMySQLShellService_ExecuteTarget_Integration(t *testing.T) {
	if os.Getenv("DBSYNC_TEST_DESTRUCTIVE") != "1" {
		t.Skip("Full sync integration test requires DBSYNC_TEST_DESTRUCTIVE=1")
	}

	targetDatabase := strings.TrimSpace(os.Getenv("DBSYNC_TEST_DATABASE"))
	if targetDatabase == "" {
		t.Skip("Full sync integration test requires DBSYNC_TEST_DATABASE")
	}

	cfg := integrationConfig(t)
	dbService := services.NewDatabaseService(cfg)
	service := services.NewMySQLShellService(cfg, dbService)
	service.SetQuiet(true)

	result, err := service.ExecuteTarget(models.SyncTarget{DatabaseName: targetDatabase, ReplaceEntireDatabase: true})
	if err != nil {
		t.Fatalf("ExecuteTarget() error = %v", err)
	}
	if result == nil {
		t.Fatal("ExecuteTarget() returned nil result")
	}
	if !result.Success {
		t.Fatalf("ExecuteTarget() returned unsuccessful result: %+v", result)
	}
	if result.DatabaseName != targetDatabase {
		t.Fatalf("DatabaseName = %q, want %q", result.DatabaseName, targetDatabase)
	}
	if result.Traffic.TotalBytes() <= 0 {
		t.Fatalf("Traffic.TotalBytes() = %d, want > 0", result.Traffic.TotalBytes())
	}
	if result.Duration <= 0 || result.DumpDuration <= 0 || result.RestoreDuration <= 0 {
		t.Fatalf("durations should be > 0, got total=%v dump=%v restore=%v", result.Duration, result.DumpDuration, result.RestoreDuration)
	}
}

func integrationConfig(t *testing.T) *config.Config {
	t.Helper()

	cfg, err := config.Load()
	if err != nil {
		cfg, err = config.LoadForTest()
		if err != nil {
			t.Fatalf("failed to load config: %v", err)
		}
	}

	overrideString(&cfg.Remote.Host, "DBSYNC_TEST_REMOTE_HOST")
	overrideInt(&cfg.Remote.Port, "DBSYNC_TEST_REMOTE_PORT")
	overrideString(&cfg.Remote.User, "DBSYNC_TEST_REMOTE_USER")
	overrideString(&cfg.Remote.Password, "DBSYNC_TEST_REMOTE_PASSWORD")
	overrideString(&cfg.Remote.ProxyURL, "DBSYNC_TEST_REMOTE_PROXY_URL")

	overrideString(&cfg.Local.Host, "DBSYNC_TEST_LOCAL_HOST")
	overrideInt(&cfg.Local.Port, "DBSYNC_TEST_LOCAL_PORT")
	overrideString(&cfg.Local.User, "DBSYNC_TEST_LOCAL_USER")
	overrideString(&cfg.Local.Password, "DBSYNC_TEST_LOCAL_PASSWORD")

	if timeout := strings.TrimSpace(os.Getenv("DBSYNC_TEST_DUMP_TIMEOUT")); timeout != "" {
		parsed, err := time.ParseDuration(timeout)
		if err != nil {
			t.Fatalf("invalid DBSYNC_TEST_DUMP_TIMEOUT: %v", err)
		}
		cfg.Dump.Timeout = parsed
	}
	if cfg.Dump.Timeout <= 0 {
		cfg.Dump.Timeout = 10 * time.Minute
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("invalid integration config: %v", err)
	}

	return cfg
}

func overrideString(target *string, envKey string) {
	if value := strings.TrimSpace(os.Getenv(envKey)); value != "" {
		*target = value
	}
}

func overrideInt(target *int, envKey string) {
	if value := strings.TrimSpace(os.Getenv(envKey)); value != "" {
		parsed, err := strconv.Atoi(value)
		if err == nil {
			*target = parsed
		}
	}
}
