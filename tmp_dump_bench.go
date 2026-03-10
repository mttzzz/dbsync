package main

import (
	"fmt"
	"os"
	"time"

	"db-sync-cli/internal/config"
	"db-sync-cli/internal/services"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	if value := os.Getenv("BENCH_THREADS"); value != "" {
		_, _ = fmt.Sscanf(value, "%d", &cfg.Dump.Threads)
	}
	if value := os.Getenv("BENCH_COMPRESS"); value != "" {
		cfg.Dump.Compress = value == "1" || value == "true"
	}
	if value := os.Getenv("BENCH_NET_COMPRESS"); value != "" {
		cfg.Dump.NetworkCompress = value == "1" || value == "true"
	}
	if value := os.Getenv("BENCH_NET_ZSTD"); value != "" {
		_, _ = fmt.Sscanf(value, "%d", &cfg.Dump.NetworkZstdLevel)
	}

	dbService := services.NewDatabaseService(cfg)
	shellService := services.NewMySQLShellService(cfg, dbService)
	shellService.SetQuiet(true)

	startedAt := time.Now()
	result, dumpDir, err := shellService.CreateDump("kp_modmb_com", false)
	if dumpDir != "" {
		defer os.RemoveAll(dumpDir)
	}
	elapsed := time.Since(startedAt)

	if err != nil {
		fmt.Printf("ERR\n%s\n", err)
		os.Exit(1)
	}

	fmt.Printf("threads=%d dumpCompress=%v netCompress=%v netZstd=%d elapsed=%s logical=%d dump=%d downloaded=%d uploaded=%d totalNet=%d dumpDuration=%s\n",
		cfg.Dump.Threads,
		cfg.Dump.Compress,
		cfg.Dump.NetworkCompress,
		cfg.Dump.NetworkZstdLevel,
		elapsed.Round(time.Millisecond),
		result.LogicalSize,
		result.DumpSizeOnDisk,
		result.Traffic.BytesIn,
		result.Traffic.BytesOut,
		result.Traffic.TotalBytes(),
		result.Duration.Round(time.Millisecond),
	)
}
