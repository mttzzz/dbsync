package models

import "time"

// UsesTableSelection сообщает, что target синхронизируется по списку таблиц.
func (t SyncTarget) UsesTableSelection() bool {
	return len(t.SelectedTables) > 0
}

// EffectiveTables возвращает полный список таблиц с учетом auto-include.
func (t SyncTarget) EffectiveTables() []string {
	if len(t.AutoIncludedTables) == 0 {
		return append([]string(nil), t.SelectedTables...)
	}

	combined := make([]string, 0, len(t.SelectedTables)+len(t.AutoIncludedTables))
	seen := make(map[string]struct{}, len(t.SelectedTables)+len(t.AutoIncludedTables))
	for _, tableName := range t.SelectedTables {
		if _, ok := seen[tableName]; ok {
			continue
		}
		seen[tableName] = struct{}{}
		combined = append(combined, tableName)
	}
	for _, tableName := range t.AutoIncludedTables {
		if _, ok := seen[tableName]; ok {
			continue
		}
		seen[tableName] = struct{}{}
		combined = append(combined, tableName)
	}

	return combined
}

// TotalBytes возвращает суммарный входящий и исходящий сетевой трафик.
func (m TrafficMetrics) TotalBytes() int64 {
	return m.BytesIn + m.BytesOut
}

// DownloadedBytes возвращает байты, пришедшие с remote сервера в локальный mysqlsh.
func (m TrafficMetrics) DownloadedBytes() int64 {
	return m.BytesIn
}

// UploadedBytes возвращает управляющий и исходящий трафик от локального mysqlsh к remote серверу.
func (m TrafficMetrics) UploadedBytes() int64 {
	return m.BytesOut
}

// HasETA сообщает, есть ли оценка времени завершения.
func (p ProgressSnapshot) HasETA() bool {
	return p.ETA > 0
}

// Duration возвращает длительность выполнения результата.
func (r SyncResult) DurationOrZero() time.Duration {
	if r.EndTime.IsZero() || r.StartTime.IsZero() {
		return r.Duration
	}
	return r.EndTime.Sub(r.StartTime)
}
