package updater

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdater_CheckForUpdates(t *testing.T) {
	// Создаем мок-сервер для GitHub API
	mockRelease := GitHubRelease{
		TagName:    "v1.2.0",
		Name:       "Release 1.2.0",
		Body:       "Bug fixes and improvements",
		Draft:      false,
		Prerelease: false,
		Assets: []GitHubAsset{
			{
				Name:               "dbsync-windows-amd64.zip",
				BrowserDownloadURL: "https://github.com/test/dbsync/releases/download/v1.2.0/dbsync-windows-amd64.zip",
				Size:               1024000,
				ContentType:        "application/zip",
			},
			{
				Name:               "dbsync-linux-amd64.zip",
				BrowserDownloadURL: "https://github.com/test/dbsync/releases/download/v1.2.0/dbsync-linux-amd64.zip",
				Size:               1024000,
				ContentType:        "application/zip",
			},
		},
		CreatedAt:   time.Now(),
		PublishedAt: time.Now(),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockRelease)
	}))
	defer server.Close()

	updater := NewUpdater()
	// Для реального тестирования нужно было бы сделать URL конфигурируемым
	// Пока протестируем основные методы

	t.Run("isVersionNewer", func(t *testing.T) {
		testCases := []struct {
			name        string
			newVersion  string
			currVersion string
			expected    bool
		}{
			{"newer version", "v1.2.0", "v1.1.0", true},
			{"same version", "v1.1.0", "v1.1.0", false},
			{"without v prefix", "1.2.0", "1.1.0", true},
			{"mixed prefixes", "v1.2.0", "1.1.0", true},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result, err := updater.isVersionNewer(tc.newVersion, tc.currVersion)
				require.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			})
		}
	})

	t.Run("findAssetForCurrentPlatform", func(t *testing.T) {
		assets := []GitHubAsset{
			{Name: "dbsync-windows-amd64.zip", BrowserDownloadURL: "url1", Size: 1000},
			{Name: "dbsync-linux-amd64.zip", BrowserDownloadURL: "url2", Size: 1000},
			{Name: "dbsync-darwin-amd64.zip", BrowserDownloadURL: "url3", Size: 1000},
		}

		asset, err := updater.findAssetForCurrentPlatform(assets)
		require.NoError(t, err)
		assert.NotNil(t, asset)
		assert.Contains(t, asset.Name, "amd64") // Предполагаем amd64 архитектуру
	})
}

func TestGitHubRelease_JSON(t *testing.T) {
	// Тестируем правильность десериализации JSON
	jsonData := `{
		"tag_name": "v1.1.0",
		"name": "Release 1.1.0",
		"body": "New features",
		"draft": false,
		"prerelease": false,
		"assets": [
			{
				"name": "dbsync-windows-amd64.zip",
				"browser_download_url": "https://example.com/download",
				"size": 1024,
				"content_type": "application/zip"
			}
		],
		"created_at": "2025-06-20T10:00:00Z",
		"published_at": "2025-06-20T10:00:00Z"
	}`

	var release GitHubRelease
	err := json.Unmarshal([]byte(jsonData), &release)
	require.NoError(t, err)

	assert.Equal(t, "v1.1.0", release.TagName)
	assert.Equal(t, "Release 1.1.0", release.Name)
	assert.Equal(t, "New features", release.Body)
	assert.False(t, release.Draft)
	assert.False(t, release.Prerelease)
	assert.Len(t, release.Assets, 1)
	assert.Equal(t, "dbsync-windows-amd64.zip", release.Assets[0].Name)
}

func TestUpdateInfo_Structure(t *testing.T) {
	updateInfo := UpdateInfo{
		Available:      true,
		CurrentVersion: "1.0.0",
		LatestVersion:  "1.1.0",
		ReleaseNotes:   "Bug fixes",
		DownloadURL:    "https://example.com/download",
		AssetSize:      1024000,
		PublishedAt:    time.Now(),
	}

	assert.True(t, updateInfo.Available)
	assert.Equal(t, "1.0.0", updateInfo.CurrentVersion)
	assert.Equal(t, "1.1.0", updateInfo.LatestVersion)
	assert.Equal(t, "Bug fixes", updateInfo.ReleaseNotes)
	assert.Equal(t, "https://example.com/download", updateInfo.DownloadURL)
	assert.Equal(t, int64(1024000), updateInfo.AssetSize)
}

func TestUpdateResult_Structure(t *testing.T) {
	duration := 5 * time.Second
	result := UpdateResult{
		Success:         true,
		PreviousVersion: "1.0.0",
		NewVersion:      "1.1.0",
		Duration:        duration,
		Error:           "",
	}

	assert.True(t, result.Success)
	assert.Equal(t, "1.0.0", result.PreviousVersion)
	assert.Equal(t, "1.1.0", result.NewVersion)
	assert.Equal(t, duration, result.Duration)
	assert.Empty(t, result.Error)
}

func TestUpdater_ReplaceExecutable(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "updater-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Создаем тестовый "исполняемый" файл
	currentExePath := filepath.Join(tempDir, "my-custom-app.exe")
	err = os.WriteFile(currentExePath, []byte("old version"), 0755)
	require.NoError(t, err)

	// Создаем новый "исполняемый" файл
	newExePath := filepath.Join(tempDir, "new-version.exe")
	err = os.WriteFile(newExePath, []byte("new version"), 0755)
	require.NoError(t, err)

	updater := NewUpdater()

	// Тестируем замену файла
	err = updater.replaceExecutable(currentExePath, newExePath)
	require.NoError(t, err)

	// Проверяем, что файл был заменен и имеет правильное имя
	assert.FileExists(t, currentExePath)

	content, err := os.ReadFile(currentExePath)
	require.NoError(t, err)
	assert.Equal(t, "new version", string(content))

	// Проверяем, что временные файлы удалены
	assert.NoFileExists(t, currentExePath+".backup")
	assert.NoFileExists(t, filepath.Join(tempDir, "temp_my-custom-app.exe"))
}

func TestUpdater_FindExecutableInDir(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "updater-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	updater := NewUpdater()

	t.Run("finds dbsync executable", func(t *testing.T) {
		// Создаем файл с именем dbsync
		execPath := filepath.Join(tempDir, "dbsync.exe")
		err = os.WriteFile(execPath, []byte("executable"), 0755)
		require.NoError(t, err)

		found, err := updater.findExecutableInDir(tempDir)
		require.NoError(t, err)
		assert.Equal(t, execPath, found)
	})

	t.Run("finds any executable if no dbsync", func(t *testing.T) {
		// Удаляем файл dbsync
		os.Remove(filepath.Join(tempDir, "dbsync.exe"))

		// Создаем другой исполняемый файл
		execPath := filepath.Join(tempDir, "other-app.exe")
		err = os.WriteFile(execPath, []byte("executable"), 0755)
		require.NoError(t, err)

		found, err := updater.findExecutableInDir(tempDir)
		require.NoError(t, err)
		assert.Equal(t, execPath, found)
	})

	t.Run("no executable found", func(t *testing.T) {
		// Создаем пустую директорию
		emptyDir := filepath.Join(tempDir, "empty")
		err = os.MkdirAll(emptyDir, 0755)
		require.NoError(t, err)

		_, err = updater.findExecutableInDir(emptyDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "executable not found")
	})
}
