package version

import (
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	// Проверяем, что версия не пустая
	if Version == "" {
		t.Error("Version should not be empty")
	}

	// Проверяем, что версия имеет ожидаемый формат (хотя бы "dev")
	if len(Version) < 3 {
		t.Errorf("Version format seems incorrect: %s", Version)
	}
}

func TestBuildInfo(t *testing.T) {
	// Эти поля могут быть пустыми в тестах, но должны существовать
	_ = BuildDate
	_ = GitCommit
	_ = GoVersion

	// Проверяем, что структура Get работает
	info := Get()

	if info.Version != Version {
		t.Errorf("Info.Version = %v, want %v", info.Version, Version)
	}

	if info.GitCommit != GitCommit {
		t.Errorf("Info.GitCommit = %v, want %v", info.GitCommit, GitCommit)
	}

	if info.BuildDate != BuildDate {
		t.Errorf("Info.BuildDate = %v, want %v", info.BuildDate, BuildDate)
	}

	if info.GoVersion != GoVersion {
		t.Errorf("Info.GoVersion = %v, want %v", info.GoVersion, GoVersion)
	}
}

func TestGet(t *testing.T) {
	info := Get()

	// Проверяем все поля структуры
	if info.Version == "" {
		t.Error("Info.Version should not be empty")
	}

	if info.GoVersion == "" {
		t.Error("Info.GoVersion should not be empty")
	}

	if info.Platform == "" {
		t.Error("Info.Platform should not be empty")
	}

	// Platform должна содержать OS/ARCH
	if !strings.Contains(info.Platform, "/") {
		t.Errorf("Info.Platform should contain OS/ARCH format, got: %s", info.Platform)
	}
}

func TestInfoString(t *testing.T) {
	info := Get()
	str := info.String()

	// Проверяем, что строка содержит ожидаемые элементы
	if !strings.Contains(str, "dbsync version") {
		t.Error("String() should contain 'dbsync version'")
	}

	if !strings.Contains(str, info.Version) {
		t.Error("String() should contain version")
	}

	if !strings.Contains(str, info.Platform) {
		t.Error("String() should contain platform")
	}

	if !strings.Contains(str, "Build:") {
		t.Error("String() should contain 'Build:'")
	}

	if !strings.Contains(str, "Date:") {
		t.Error("String() should contain 'Date:'")
	}
}
