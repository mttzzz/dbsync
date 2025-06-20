package models

import (
	"sort"
	"time"
)

// Database представляет информацию о базе данных
type Database struct {
	Name    string    `json:"name"`
	Size    int64     `json:"size_bytes"`
	Tables  int       `json:"tables_count"`
	Created time.Time `json:"created_at,omitempty"`
}

// DatabaseList представляет список баз данных
type DatabaseList []Database

// SortBySize сортирует базы данных по размеру (сначала большие)
func (dl DatabaseList) SortBySize() {
	sort.Slice(dl, func(i, j int) bool {
		return dl[i].Size > dl[j].Size
	})
}

// SortBySizeAsc сортирует базы данных по размеру (сначала маленькие)
func (dl DatabaseList) SortBySizeAsc() {
	sort.Slice(dl, func(i, j int) bool {
		return dl[i].Size < dl[j].Size
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

// SyncOptions содержит опции для синхронизации
type SyncOptions struct {
	DatabaseName string
	DryRun       bool
	Force        bool
	Verbose      bool
	RemoteHost   string
	LocalHost    string
}

// SyncResult содержит результат синхронизации
type SyncResult struct {
	Success         bool          `json:"success"`
	DatabaseName    string        `json:"database_name"`
	Duration        time.Duration `json:"duration"`
	DumpDuration    time.Duration `json:"dump_duration"`
	RestoreDuration time.Duration `json:"restore_duration"`
	DumpSize        int64         `json:"dump_size_bytes"`
	TablesCount     int           `json:"tables_count"`
	Error           string        `json:"error,omitempty"`
	StartTime       time.Time     `json:"start_time"`
	EndTime         time.Time     `json:"end_time"`
}
