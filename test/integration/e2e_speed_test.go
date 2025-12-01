//go:build e2e
// +build e2e

package integration

import (
	"fmt"
	"testing"
	"time"

	"db-sync-cli/internal/config"
	"db-sync-cli/internal/services"
)

const (
	// TestDatabaseName - база данных для тестирования
	TestDatabaseName = "octane_pushka_biz"
)

// TestE2E_Sync выполняет тест синхронизации через mydumper
func TestE2E_Sync(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	dbService := services.NewDatabaseService(cfg)
	mydumperService := services.NewMyDumperService(cfg, dbService)

	// Проверяем Docker
	if err := mydumperService.ValidateDumpOperation(TestDatabaseName); err != nil {
		t.Skipf("Skipping test: %v", err)
		return
	}

	// Получаем информацию о БД
	dbInfo, err := dbService.GetDatabaseInfo(TestDatabaseName, true)
	if err != nil {
		t.Fatalf("Failed to get database info: %v", err)
	}

	t.Logf("📊 Database: %s (%d tables)", TestDatabaseName, dbInfo.Tables)
	t.Logf("🚀 Starting sync with mydumper (%d threads)...", cfg.Dump.Threads)

	startTime := time.Now()
	result, err := mydumperService.ExecuteSync(TestDatabaseName)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	t.Logf("✅ Completed in %s", time.Since(startTime).Round(time.Second))
	t.Logf("   Dump: %s, Restore: %s", result.DumpDuration.Round(time.Second), result.RestoreDuration.Round(time.Second))
	t.Logf("   Size: %s", services.FormatSize(result.DumpSize))
}

// TestE2E_Threads тестирует разное количество потоков
func TestE2E_Threads(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	dbService := services.NewDatabaseService(cfg)

	// Получаем информацию о БД
	dbInfo, err := dbService.GetDatabaseInfo(TestDatabaseName, true)
	if err != nil {
		t.Fatalf("Failed to get database info: %v", err)
	}

	t.Logf("📊 Database: %s (%d tables)", TestDatabaseName, dbInfo.Tables)

	threadCounts := []int{1, 4, 8}
	results := make(map[int]time.Duration)

	for _, threads := range threadCounts {
		t.Run(fmt.Sprintf("Threads_%d", threads), func(t *testing.T) {
			cfg.Dump.Threads = threads
			mydumperService := services.NewMyDumperService(cfg, dbService)

			if err := mydumperService.ValidateDumpOperation(TestDatabaseName); err != nil {
				t.Skipf("Skipping: %v", err)
				return
			}

			t.Logf("🚀 Testing with %d threads...", threads)
			startTime := time.Now()

			_, err := mydumperService.ExecuteSync(TestDatabaseName)
			if err != nil {
				t.Fatalf("Sync failed: %v", err)
			}

			duration := time.Since(startTime)
			results[threads] = duration
			t.Logf("   Completed in %s", duration.Round(time.Second))
		})
	}

	// Выводим сводку
	t.Run("Summary", func(t *testing.T) {
		fmt.Println("\n═══ Thread Scaling Results ═══")
		for _, threads := range threadCounts {
			if d, ok := results[threads]; ok {
				fmt.Printf("  %d threads: %s\n", threads, d.Round(time.Second))
			}
		}
	})
}
