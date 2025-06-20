package version

import (
	"fmt"
	"runtime"
)

var (
	// Эти переменные будут заполнены во время сборки через ldflags
	Version   = "1.1.2"   // Версия приложения
	GitCommit = "unknown" // Git коммит
	BuildDate = "unknown" // Дата сборки
	GoVersion = runtime.Version()
)

// Info содержит информацию о версии
type Info struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
}

// Get возвращает информацию о версии
func Get() Info {
	return Info{
		Version:   Version,
		GitCommit: GitCommit,
		BuildDate: BuildDate,
		GoVersion: GoVersion,
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// String возвращает строковое представление версии
func (i Info) String() string {
	return fmt.Sprintf(`dbsync version %s
Build: %s
Date: %s
Go: %s
Platform: %s`, i.Version, i.GitCommit, i.BuildDate, i.GoVersion, i.Platform)
}
