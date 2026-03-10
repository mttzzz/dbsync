package models

import (
	"sort"
	"time"
)

// Database представляет информацию о базе данных
type Database struct {
	Name      string    `json:"name"`
	Size      int64     `json:"size_bytes"`
	DataSize  int64     `json:"data_size_bytes,omitempty"`
	IndexSize int64     `json:"index_size_bytes,omitempty"`
	Tables    int       `json:"tables_count"`
	Created   time.Time `json:"created_at,omitempty"`
}

// DatabaseList представляет список баз данных
type DatabaseList []Database

// SortBySize сортирует базы данных по размеру (сначала большие)
func (dl DatabaseList) SortBySize() {
	sort.Slice(dl, func(i, j int) bool {
		left := dl[i].Size
		if dl[i].DataSize > 0 {
			left = dl[i].DataSize
		}
		right := dl[j].Size
		if dl[j].DataSize > 0 {
			right = dl[j].DataSize
		}
		return left > right
	})
}

// SortBySizeAsc сортирует базы данных по размеру (сначала маленькие)
func (dl DatabaseList) SortBySizeAsc() {
	sort.Slice(dl, func(i, j int) bool {
		left := dl[i].Size
		if dl[i].DataSize > 0 {
			left = dl[i].DataSize
		}
		right := dl[j].Size
		if dl[j].DataSize > 0 {
			right = dl[j].DataSize
		}
		return left < right
	})
}

// SortByName сортирует базы данных по имени
func (dl DatabaseList) SortByName() {
	sort.Slice(dl, func(i, j int) bool {
		return dl[i].Name < dl[j].Name
	})
}

// ConnectionInfo содержит информацию о подключении
type ConnectionInfo struct {
	Host      string `json:"host"`
	Port      int    `json:"port"`
	User      string `json:"user"`
	Connected bool   `json:"connected"`
	Version   string `json:"version,omitempty"`
	Error     string `json:"error,omitempty"`
}

// Table представляет информацию о таблице базы данных.
type Table struct {
	DatabaseName string `json:"database_name"`
	Name         string `json:"name"`
	Size         int64  `json:"size_bytes"`
	DataSize     int64  `json:"data_size_bytes,omitempty"`
	IndexSize    int64  `json:"index_size_bytes,omitempty"`
	Rows         int64  `json:"rows"`
	RowsApprox   bool   `json:"rows_approximate,omitempty"`
	Engine       string `json:"engine,omitempty"`
	Collation    string `json:"collation,omitempty"`
	DataFree     int64  `json:"data_free_bytes,omitempty"`
}

// TableDependency представляет внешнюю зависимость таблицы.
type TableDependency struct {
	DatabaseName    string `json:"database_name"`
	TableName       string `json:"table_name"`
	ReferencedTable string `json:"referenced_table"`
	ConstraintName  string `json:"constraint_name"`
	AutoIncluded    bool   `json:"auto_included,omitempty"`
	Reason          string `json:"reason,omitempty"`
}

// RuntimeOptions содержит runtime-only опции выполнения.
type RuntimeOptions struct {
	DryRun     bool   `json:"dry_run"`
	Force      bool   `json:"force"`
	Verbose    bool   `json:"verbose"`
	Threads    int    `json:"threads,omitempty"`
	ConfigFile string `json:"config_file,omitempty"`
}

// TransportMode описывает способ подключения к remote MySQL.
type TransportMode string

const (
	TransportModeDirect TransportMode = "direct"
	TransportModeProxy  TransportMode = "proxy"
)

// SyncPhase описывает текущую фазу выполнения синхронизации.
type SyncPhase string

const (
	SyncPhaseValidation SyncPhase = "validation"
	SyncPhasePlanning   SyncPhase = "planning"
	SyncPhaseDump       SyncPhase = "dump"
	SyncPhaseRestore    SyncPhase = "restore"
	SyncPhaseCleanup    SyncPhase = "cleanup"
	SyncPhaseDone       SyncPhase = "done"
	SyncPhaseFailed     SyncPhase = "failed"
)

// SyncTarget описывает одну цель синхронизации.
type SyncTarget struct {
	DatabaseName          string   `json:"database_name"`
	SelectedTables        []string `json:"selected_tables,omitempty"`
	AutoIncludedTables    []string `json:"auto_included_tables,omitempty"`
	ReplaceEntireDatabase bool     `json:"replace_entire_database"`
}

// SyncPlan описывает итоговый план синхронизации.
type SyncPlan struct {
	Targets                []SyncTarget  `json:"targets"`
	TransportMode          TransportMode `json:"transport_mode"`
	EstimatedLogicalSize   int64         `json:"estimated_logical_size_bytes,omitempty"`
	EstimatedTransferBytes int64         `json:"estimated_transfer_bytes,omitempty"`
	EstimatedDumpBytes     int64         `json:"estimated_dump_bytes,omitempty"`
	EstimatedDuration      time.Duration `json:"estimated_duration,omitempty"`
	CreatedAt              time.Time     `json:"created_at"`
}

// TrafficMetrics хранит сетевые метрики выполнения.
type TrafficMetrics struct {
	Mode                  TransportMode `json:"mode"`
	BytesIn               int64         `json:"bytes_in"`
	BytesOut              int64         `json:"bytes_out"`
	AverageBytesPerSecond float64       `json:"average_bytes_per_second,omitempty"`
	CurrentBytesPerSecond float64       `json:"current_bytes_per_second,omitempty"`
	SampleWindow          time.Duration `json:"sample_window,omitempty"`
}

// ProgressSnapshot хранит срез состояния во время синхронизации.
type ProgressSnapshot struct {
	Phase          SyncPhase      `json:"phase"`
	DatabaseName   string         `json:"database_name"`
	TableName      string         `json:"table_name,omitempty"`
	Message        string         `json:"message,omitempty"`
	Current        int64          `json:"current,omitempty"`
	Total          int64          `json:"total,omitempty"`
	Percent        float64        `json:"percent,omitempty"`
	BytesCompleted int64          `json:"bytes_completed,omitempty"`
	BytesTotal     int64          `json:"bytes_total,omitempty"`
	ETA            time.Duration  `json:"eta,omitempty"`
	Traffic        TrafficMetrics `json:"traffic,omitempty"`
	Timestamp      time.Time      `json:"timestamp"`
}

// ProgressObserver получает события выполнения синхронизации.
type ProgressObserver func(ProgressSnapshot)

// SyncOptions содержит опции для синхронизации
type SyncOptions struct {
	DatabaseName   string
	DryRun         bool
	Force          bool
	Verbose        bool
	RemoteHost     string
	LocalHost      string
	SelectedTables []string
	Runtime        RuntimeOptions
}

// SyncResult содержит результат синхронизации
type SyncResult struct {
	Success            bool               `json:"success"`
	DatabaseName       string             `json:"database_name"`
	Duration           time.Duration      `json:"duration"`
	DumpDuration       time.Duration      `json:"dump_duration"`
	RestoreDuration    time.Duration      `json:"restore_duration"`
	DumpSize           int64              `json:"dump_size_bytes"`
	TablesCount        int                `json:"tables_count"`
	Error              string             `json:"error,omitempty"`
	StartTime          time.Time          `json:"start_time"`
	EndTime            time.Time          `json:"end_time"`
	SelectedTables     []string           `json:"selected_tables,omitempty"`
	AutoIncludedTables []string           `json:"auto_included_tables,omitempty"`
	TransportMode      TransportMode      `json:"transport_mode,omitempty"`
	LogicalSize        int64              `json:"logical_size_bytes,omitempty"`
	IndexSize          int64              `json:"index_size_bytes,omitempty"`
	DumpSizeOnDisk     int64              `json:"dump_size_on_disk_bytes,omitempty"`
	CompressionRatio   float64            `json:"compression_ratio,omitempty"`
	Traffic            TrafficMetrics     `json:"traffic,omitempty"`
	Progress           []ProgressSnapshot `json:"progress,omitempty"`
}
