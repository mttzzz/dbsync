package services

import (
	"strings"
	"testing"
	"time"

	"db-sync-cli/internal/config"
	"db-sync-cli/internal/models"
)

func TestParseMySQLShellProgressLine(t *testing.T) {
	tests := []struct {
		name           string
		line           string
		wantMatch      bool
		wantPercent    float64
		wantCompleted  int64
		wantTotal      int64
		wantSpeedBytes int64
		wantETA        time.Duration
	}{
		{
			name:           "percent size speed eta",
			line:           "41% 512 MB / 1 GB 32 MB/s remaining: 00:15",
			wantMatch:      true,
			wantPercent:    41,
			wantCompleted:  512 * 1024 * 1024,
			wantTotal:      1024 * 1024 * 1024,
			wantSpeedBytes: 32 * 1024 * 1024,
			wantETA:        15 * time.Second,
		},
		{
			name:           "iec units and comma decimals",
			line:           "[73,5%] 1.5 GiB/2.0 GiB 12.5 MiB/sec ETA 1m 05s",
			wantMatch:      true,
			wantPercent:    73.5,
			wantCompleted:  int64(1.5 * 1024 * 1024 * 1024),
			wantTotal:      int64(2.0 * 1024 * 1024 * 1024),
			wantSpeedBytes: int64(12.5 * 1024 * 1024),
			wantETA:        65 * time.Second,
		},
		{
			name:           "compact bytes syntax",
			line:           "99% 950MB/1GB 40MB/sec left 00:02",
			wantMatch:      true,
			wantPercent:    99,
			wantCompleted:  950 * 1024 * 1024,
			wantTotal:      1024 * 1024 * 1024,
			wantSpeedBytes: 40 * 1024 * 1024,
			wantETA:        2 * time.Second,
		},
		{
			name:        "plain non progress line",
			line:        "starting thread pool",
			wantMatch:   false,
			wantPercent: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, ok := parseMySQLShellProgressLine(tt.line)
			if ok != tt.wantMatch {
				t.Fatalf("parseMySQLShellProgressLine() matched = %v, want %v", ok, tt.wantMatch)
			}
			if !tt.wantMatch {
				return
			}
			if parsed.Percent != tt.wantPercent {
				t.Fatalf("Percent = %v, want %v", parsed.Percent, tt.wantPercent)
			}
			if parsed.BytesCompleted != tt.wantCompleted {
				t.Fatalf("BytesCompleted = %d, want %d", parsed.BytesCompleted, tt.wantCompleted)
			}
			if parsed.BytesTotal != tt.wantTotal {
				t.Fatalf("BytesTotal = %d, want %d", parsed.BytesTotal, tt.wantTotal)
			}
			if int64(parsed.BytesPerSecond) != tt.wantSpeedBytes {
				t.Fatalf("BytesPerSecond = %d, want %d", int64(parsed.BytesPerSecond), tt.wantSpeedBytes)
			}
			if parsed.ETA != tt.wantETA {
				t.Fatalf("ETA = %v, want %v", parsed.ETA, tt.wantETA)
			}
		})
	}
}

func TestParseLooseDuration(t *testing.T) {
	tests := []struct {
		input    string
		want     time.Duration
		wantOkay bool
	}{
		{input: "01:30", want: 90 * time.Second, wantOkay: true},
		{input: "1m 5s", want: 65 * time.Second, wantOkay: true},
		{input: "2:01:00", want: 2*time.Hour + time.Minute, wantOkay: true},
		{input: "soon", wantOkay: false},
	}

	for _, tt := range tests {
		got, ok := parseLooseDuration(tt.input)
		if ok != tt.wantOkay {
			t.Fatalf("parseLooseDuration(%q) ok = %v, want %v", tt.input, ok, tt.wantOkay)
		}
		if ok && got != tt.want {
			t.Fatalf("parseLooseDuration(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestClassifyMySQLShellStatusLine(t *testing.T) {
	tests := []struct {
		phase  models.SyncPhase
		line   string
		want   string
		wantOK bool
	}{
		{phase: models.SyncPhaseDump, line: "Gathering information about schemas", want: "Preparing dump metadata", wantOK: true},
		{phase: models.SyncPhaseDump, line: "Initializing - done", want: "Preparing dump metadata", wantOK: true},
		{phase: models.SyncPhaseDump, line: "1 schemas will be dumped and within them 62 tables, 0 views.", want: "Preparing dump metadata", wantOK: true},
		{phase: models.SyncPhaseDump, line: "Writing global DDL files", want: "Writing schema metadata", wantOK: true},
		{phase: models.SyncPhaseDump, line: "Writing table metadata - done", want: "Writing table metadata", wantOK: true},
		{phase: models.SyncPhaseDump, line: "Starting data dump", want: "Streaming table data", wantOK: true},
		{phase: models.SyncPhaseDump, line: "Writing metadata for schema kp_modmb_com", want: "Writing schema metadata", wantOK: true},
		{phase: models.SyncPhaseDump, line: "Dumping data chunks", want: "Streaming table data", wantOK: true},
		{phase: models.SyncPhaseDump, line: "Finalizing output files", want: "Finalizing dump files", wantOK: true},
		{phase: models.SyncPhaseDump, line: "NOTE: Progress information uses estimated values and may not be accurate.", wantOK: false},
		{phase: models.SyncPhaseDump, line: "Dump duration: 00:00:01s", wantOK: false},
		{phase: models.SyncPhaseRestore, line: "Loading DDL and Data from '/tmp/foo' using 2 threads", want: "Preparing local restore", wantOK: true},
		{phase: models.SyncPhaseRestore, line: "Opening dump - done", want: "Preparing local restore", wantOK: true},
		{phase: models.SyncPhaseRestore, line: "Scanning metadata - done", want: "Preparing local restore", wantOK: true},
		{phase: models.SyncPhaseRestore, line: "Checking for pre-existing objects - done", want: "Preparing local restore", wantOK: true},
		{phase: models.SyncPhaseRestore, line: "Executing common preamble SQL - done", want: "Applying schema metadata", wantOK: true},
		{phase: models.SyncPhaseRestore, line: "Executing table DDL - done", want: "Applying schema metadata", wantOK: true},
		{phase: models.SyncPhaseRestore, line: "Starting data load", want: "Loading table data", wantOK: true},
		{phase: models.SyncPhaseRestore, line: "Building indexes - done", want: "Rebuilding indexes", wantOK: true},
		{phase: models.SyncPhaseRestore, line: "Executing common postamble SQL - done", want: "Finalizing restore", wantOK: true},
		{phase: models.SyncPhaseRestore, line: "Data load duration: 20 sec", wantOK: false},
		{phase: models.SyncPhaseRestore, line: "Preparing load", want: "Preparing local restore", wantOK: true},
		{phase: models.SyncPhaseRestore, line: "Loading data chunks", want: "Loading table data", wantOK: true},
		{phase: models.SyncPhaseRestore, line: "Rebuilding deferred indexes", want: "Rebuilding indexes", wantOK: true},
		{phase: models.SyncPhaseRestore, line: "WARNING: Using a password on the command line interface can be insecure.", wantOK: false},
	}

	for _, tt := range tests {
		got, ok := classifyMySQLShellStatusLine(tt.phase, tt.line)
		if ok != tt.wantOK {
			t.Fatalf("classifyMySQLShellStatusLine(%q) ok = %v, want %v", tt.line, ok, tt.wantOK)
		}
		if ok && got != tt.want {
			t.Fatalf("classifyMySQLShellStatusLine(%q) = %q, want %q", tt.line, got, tt.want)
		}
	}
}

func TestFormatMySQLShellError(t *testing.T) {
	err := formatMySQLShellError("dump", assertErr("exit status 10"), "dump stdout", "dump stderr")
	message := err.Error()

	if !strings.Contains(message, "mysqlsh dump failed: exit status 10") {
		t.Fatalf("error message = %q, want dump failure header", message)
	}
	if !strings.Contains(message, "stderr: dump stderr") {
		t.Fatalf("error message = %q, want stderr details", message)
	}
	if !strings.Contains(message, "stdout: dump stdout") {
		t.Fatalf("error message = %q, want stdout details", message)
	}
}

func TestBuildDumpArgsUsesIncludeTables(t *testing.T) {
	service := NewMySQLShellService(&config.Config{
		Remote: config.MySQLConfig{User: "remote_user", Password: "secret"},
		Dump:   config.DumpConfig{Threads: 6, NetworkCompress: true, NetworkZstdLevel: 7},
	}, nil)

	args := service.buildDumpArgs("mysql://remote_user@127.0.0.1:3307", "kp_modmb_com", "/tmp/dumpdir", 512*1024*1024, []string{"orders", "users"})
	joined := strings.Join(args, " ")

	if !strings.Contains(joined, "util dump-schemas kp_modmb_com") {
		t.Fatalf("dump args = %q, want dump-schemas command", joined)
	}
	if !strings.Contains(joined, "--includeTables=kp_modmb_com.orders,kp_modmb_com.users") {
		t.Fatalf("dump args = %q, want includeTables for selected tables", joined)
	}
	if strings.Contains(joined, "--tables=") {
		t.Fatalf("dump args = %q, deprecated --tables flag should not be used", joined)
	}
}

func TestBuildDumpArgsHonorsCompressionFlag(t *testing.T) {
	service := NewMySQLShellService(&config.Config{
		Remote: config.MySQLConfig{User: "remote_user", Password: "secret"},
		Dump:   config.DumpConfig{Threads: 8, Compress: false, NetworkCompress: true, NetworkZstdLevel: 7},
	}, nil)

	args := service.buildDumpArgs("mysql://remote_user@127.0.0.1:3307", "kp_modmb_com", "/tmp/dumpdir", 512*1024*1024, nil)
	joined := strings.Join(args, " ")

	if !strings.Contains(joined, "--compression=none") {
		t.Fatalf("dump args = %q, want compression=none", joined)
	}
	if strings.Contains(joined, "--compression=zstd") {
		t.Fatalf("dump args = %q, unexpected zstd compression", joined)
	}
}

func TestEffectiveDumpThreadsUsesLowerParallelismForSmallSchemas(t *testing.T) {
	service := NewMySQLShellService(&config.Config{Dump: config.DumpConfig{Threads: 8, Compress: true}}, nil)

	if got := service.effectiveDumpThreads(7 * 1024 * 1024); got != 2 {
		t.Fatalf("effectiveDumpThreads(small) = %d, want 2", got)
	}
	if got := service.effectiveDumpThreads(128 * 1024 * 1024); got != 4 {
		t.Fatalf("effectiveDumpThreads(medium) = %d, want 4", got)
	}
	if got := service.effectiveDumpThreads(1024 * 1024 * 1024); got != 8 {
		t.Fatalf("effectiveDumpThreads(large) = %d, want 8", got)
	}
}

func TestBuildDumpArgsAddsTransportCompression(t *testing.T) {
	service := NewMySQLShellService(&config.Config{
		Remote: config.MySQLConfig{User: "remote_user", Password: "secret"},
		Dump:   config.DumpConfig{Threads: 8, Compress: true, NetworkCompress: true, NetworkZstdLevel: 13},
	}, nil)

	args := service.buildDumpArgs("mysql://remote_user@127.0.0.1:3307", "kp_modmb_com", "/tmp/dumpdir", 512*1024*1024, nil)
	joined := strings.Join(args, " ")

	if !strings.Contains(joined, "--compress=REQUIRED") {
		t.Fatalf("dump args = %q, want protocol compression enabled", joined)
	}
	if !strings.Contains(joined, "--compression-algorithms=zstd,zlib") {
		t.Fatalf("dump args = %q, want explicit compression algorithms", joined)
	}
	if !strings.Contains(joined, "--zstd-compression-level=13") {
		t.Fatalf("dump args = %q, want zstd compression level", joined)
	}
}

type assertErr string

func (e assertErr) Error() string {
	return string(e)
}
