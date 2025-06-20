package updater

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"db-sync-cli/internal/version"
)

const (
	// GitHub API URL для получения последнего релиза
	githubAPIURL = "https://api.github.com/repos/mttzzz/dbsync/releases/latest"

	// Таймаут для HTTP запросов
	httpTimeout = 30 * time.Second
)

// GitHubRelease представляет информацию о релизе из GitHub API
type GitHubRelease struct {
	TagName     string        `json:"tag_name"`
	Name        string        `json:"name"`
	Body        string        `json:"body"`
	Draft       bool          `json:"draft"`
	Prerelease  bool          `json:"prerelease"`
	Assets      []GitHubAsset `json:"assets"`
	CreatedAt   time.Time     `json:"created_at"`
	PublishedAt time.Time     `json:"published_at"`
}

// GitHubAsset представляет ассет из релиза
type GitHubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
	ContentType        string `json:"content_type"`
}

// UpdateInfo содержит информацию об обновлении
type UpdateInfo struct {
	Available      bool      `json:"available"`
	CurrentVersion string    `json:"current_version"`
	LatestVersion  string    `json:"latest_version"`
	ReleaseNotes   string    `json:"release_notes,omitempty"`
	DownloadURL    string    `json:"download_url,omitempty"`
	AssetSize      int64     `json:"asset_size,omitempty"`
	PublishedAt    time.Time `json:"published_at,omitempty"`
}

// UpdateResult содержит результат обновления
type UpdateResult struct {
	Success         bool          `json:"success"`
	PreviousVersion string        `json:"previous_version"`
	NewVersion      string        `json:"new_version"`
	Duration        time.Duration `json:"duration"`
	Error           string        `json:"error,omitempty"`
}

// Updater отвечает за проверку и выполнение обновлений
type Updater struct {
	client *http.Client
}

// NewUpdater создает новый экземпляр Updater
func NewUpdater() *Updater {
	return &Updater{
		client: &http.Client{
			Timeout: httpTimeout,
		},
	}
}

// CheckForUpdates проверяет наличие новых версий
func (u *Updater) CheckForUpdates() (*UpdateInfo, error) {
	currentVersion := version.Version
	if currentVersion == "dev" {
		return &UpdateInfo{
			Available:      false,
			CurrentVersion: currentVersion,
			LatestVersion:  currentVersion,
		}, nil
	}

	// Получаем информацию о последнем релизе
	release, err := u.getLatestRelease()
	if err != nil {
		return nil, fmt.Errorf("failed to get latest release: %w", err)
	}

	// Проверяем, есть ли обновление
	isNewer, err := u.isVersionNewer(release.TagName, currentVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to compare versions: %w", err)
	}

	updateInfo := &UpdateInfo{
		Available:      isNewer,
		CurrentVersion: currentVersion,
		LatestVersion:  release.TagName,
		ReleaseNotes:   release.Body,
		PublishedAt:    release.PublishedAt,
	}

	if isNewer {
		// Находим подходящий ассет для текущей платформы
		asset, err := u.findAssetForCurrentPlatform(release.Assets)
		if err != nil {
			return nil, fmt.Errorf("no suitable asset found: %w", err)
		}

		updateInfo.DownloadURL = asset.BrowserDownloadURL
		updateInfo.AssetSize = asset.Size
	}

	return updateInfo, nil
}

// PerformUpdate выполняет обновление приложения
func (u *Updater) PerformUpdate(updateInfo *UpdateInfo) (*UpdateResult, error) {
	startTime := time.Now()
	result := &UpdateResult{
		PreviousVersion: updateInfo.CurrentVersion,
		NewVersion:      updateInfo.LatestVersion,
	}

	if !updateInfo.Available {
		result.Error = "no update available"
		return result, fmt.Errorf("no update available")
	}

	// Получаем путь к текущему исполняемому файлу
	currentExePath, err := os.Executable()
	if err != nil {
		result.Error = fmt.Sprintf("failed to get executable path: %v", err)
		return result, fmt.Errorf("failed to get executable path: %w", err)
	}

	// Создаем временную директорию
	tempDir, err := os.MkdirTemp("", "dbsync-update-*")
	if err != nil {
		result.Error = fmt.Sprintf("failed to create temp directory: %v", err)
		return result, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Скачиваем архив
	archiveName := filepath.Base(updateInfo.DownloadURL)
	archivePath := filepath.Join(tempDir, archiveName)
	err = u.downloadFile(updateInfo.DownloadURL, archivePath)
	if err != nil {
		result.Error = fmt.Sprintf("failed to download update: %v", err)
		return result, fmt.Errorf("failed to download update: %w", err)
	}

	// Извлекаем архив в зависимости от типа
	extractDir := filepath.Join(tempDir, "extracted")
	if strings.HasSuffix(strings.ToLower(archiveName), ".zip") {
		err = u.extractZip(archivePath, extractDir)
	} else if strings.HasSuffix(strings.ToLower(archiveName), ".tar.gz") {
		err = u.extractTarGz(archivePath, extractDir)
	} else {
		err = fmt.Errorf("unsupported archive format: %s", archiveName)
	}

	if err != nil {
		result.Error = fmt.Sprintf("failed to extract archive: %v", err)
		return result, fmt.Errorf("failed to extract archive: %w", err)
	}

	// Находим новый исполняемый файл
	newExePath, err := u.findExecutableInDir(extractDir)
	if err != nil {
		result.Error = fmt.Sprintf("failed to find executable in archive: %v", err)
		return result, fmt.Errorf("failed to find executable in archive: %w", err)
	}

	// Заменяем текущий исполняемый файл
	err = u.replaceExecutable(currentExePath, newExePath)
	if err != nil {
		result.Error = fmt.Sprintf("failed to replace executable: %v", err)
		return result, fmt.Errorf("failed to replace executable: %w", err)
	}

	result.Success = true
	result.Duration = time.Since(startTime)
	return result, nil
}

// getLatestRelease получает информацию о последнем релизе из GitHub API
func (u *Updater) getLatestRelease() (*GitHubRelease, error) {
	resp, err := u.client.Get(githubAPIURL)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	err = json.NewDecoder(resp.Body).Decode(&release)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &release, nil
}

// isVersionNewer проверяет, является ли новая версия более новой
func (u *Updater) isVersionNewer(newVersion, currentVersion string) (bool, error) {
	// Удаляем префикс 'v' если он есть
	newVersion = strings.TrimPrefix(newVersion, "v")
	currentVersion = strings.TrimPrefix(currentVersion, "v")

	// Простое сравнение версий с семантическим версионированием
	newParts := strings.Split(newVersion, ".")
	currentParts := strings.Split(currentVersion, ".")

	// Дополняем до 3 частей если нужно
	for len(newParts) < 3 {
		newParts = append(newParts, "0")
	}
	for len(currentParts) < 3 {
		currentParts = append(currentParts, "0")
	}

	// Сравниваем по частям: major.minor.patch
	for i := 0; i < 3; i++ {
		newPart := 0
		currentPart := 0

		// Парсим числа из строк
		if newParts[i] != "" {
			if n, err := fmt.Sscanf(newParts[i], "%d", &newPart); n != 1 || err != nil {
				newPart = 0
			}
		}
		if currentParts[i] != "" {
			if n, err := fmt.Sscanf(currentParts[i], "%d", &currentPart); n != 1 || err != nil {
				currentPart = 0
			}
		}

		if newPart > currentPart {
			return true, nil
		} else if newPart < currentPart {
			return false, nil
		}
		// Если равны, продолжаем сравнение следующей части
	}

	// Все части равны
	return false, nil
}

// findAssetForCurrentPlatform находит подходящий ассет для текущей платформы
func (u *Updater) findAssetForCurrentPlatform(assets []GitHubAsset) (*GitHubAsset, error) {
	osName := runtime.GOOS
	archName := runtime.GOARCH

	// Ожидаемые имена файлов для вашего формата: dbsync-v1.0.0-windows-amd64.zip
	expectedPatterns := []string{
		fmt.Sprintf("-%s-%s", osName, archName),  // -windows-amd64
		fmt.Sprintf("_%s_%s", osName, archName),  // _windows_amd64
		fmt.Sprintf("-%s-%s.", osName, archName), // -windows-amd64.
		fmt.Sprintf("_%s_%s.", osName, archName), // _windows_amd64.
	}

	for _, asset := range assets {
		assetNameLower := strings.ToLower(asset.Name)

		// Пропускаем checksums.txt и другие не-исполняемые файлы
		if strings.Contains(assetNameLower, "checksum") ||
			strings.Contains(assetNameLower, "sha256") {
			continue
		}

		for _, pattern := range expectedPatterns {
			if strings.Contains(assetNameLower, strings.ToLower(pattern)) {
				return &asset, nil
			}
		}
	}

	return nil, fmt.Errorf("no asset found for platform %s/%s", osName, archName)
}

// downloadFile скачивает файл по URL
func (u *Updater) downloadFile(url, filepath string) error {
	resp, err := u.client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// extractZip извлекает ZIP архив
func (u *Updater) extractZip(src, dest string) error {
	reader, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}
	defer reader.Close()

	err = os.MkdirAll(dest, 0755)
	if err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	for _, file := range reader.File {
		path := filepath.Join(dest, file.Name)

		// Проверяем безопасность пути
		if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path: %s", file.Name)
		}

		if file.FileInfo().IsDir() {
			os.MkdirAll(path, file.FileInfo().Mode())
			continue
		}

		rc, err := file.Open()
		if err != nil {
			return fmt.Errorf("failed to open file in zip: %w", err)
		}

		outFile, err := os.Create(path)
		if err != nil {
			rc.Close()
			return fmt.Errorf("failed to create extracted file: %w", err)
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return fmt.Errorf("failed to extract file: %w", err)
		}

		// Устанавливаем права доступа
		err = os.Chmod(path, file.FileInfo().Mode())
		if err != nil {
			return fmt.Errorf("failed to set file permissions: %w", err)
		}
	}

	return nil
}

// extractTarGz извлекает TAR.GZ архив
func (u *Updater) extractTarGz(src, dest string) error {
	file, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open tar.gz: %w", err)
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	err = os.MkdirAll(dest, 0755)
	if err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		path := filepath.Join(dest, header.Name)

		// Проверяем безопасность пути
		if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			err = os.MkdirAll(path, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		case tar.TypeReg:
			outFile, err := os.Create(path)
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}

			_, err = io.Copy(outFile, tr)
			outFile.Close()

			if err != nil {
				return fmt.Errorf("failed to extract file: %w", err)
			}

			// Устанавливаем права доступа
			err = os.Chmod(path, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to set file permissions: %w", err)
			}
		}
	}

	return nil
}

// findExecutableInDir находит исполняемый файл в директории
func (u *Updater) findExecutableInDir(dir string) (string, error) {
	var executablePath string
	var candidates []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Пропускаем директории
		if info.IsDir() {
			return nil
		}

		fileName := strings.ToLower(info.Name())

		// На Windows ищем .exe файлы
		if runtime.GOOS == "windows" {
			if strings.HasSuffix(fileName, ".exe") {
				// Сначала ищем файлы с "dbsync" в имени
				if strings.Contains(fileName, "dbsync") {
					candidates = append(candidates, path)
				}
			}
		} else {
			// На Unix-системах ищем исполняемые файлы
			if info.Mode()&0111 != 0 {
				// Сначала ищем файлы с "dbsync" в имени
				if strings.Contains(fileName, "dbsync") {
					candidates = append(candidates, path)
				}
			}
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to search for executable: %w", err)
	}

	// Если нашли кандидатов с "dbsync" в имени, выбираем первого
	if len(candidates) > 0 {
		executablePath = candidates[0]
	} else {
		// Если не нашли файлы с "dbsync", ищем любой исполняемый файл
		err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			if runtime.GOOS == "windows" {
				if strings.HasSuffix(strings.ToLower(info.Name()), ".exe") {
					executablePath = path
					return filepath.SkipDir
				}
			} else {
				if info.Mode()&0111 != 0 {
					executablePath = path
					return filepath.SkipDir
				}
			}

			return nil
		})

		if err != nil {
			return "", fmt.Errorf("failed to search for executable: %w", err)
		}
	}

	if executablePath == "" {
		return "", fmt.Errorf("executable not found in archive")
	}

	return executablePath, nil
}

// replaceExecutable заменяет текущий исполняемый файл новым
func (u *Updater) replaceExecutable(currentPath, newPath string) error {
	// Получаем имя текущего исполняемого файла
	currentFileName := filepath.Base(currentPath)

	// Создаем резервную копию с тем же именем + .backup
	backupPath := currentPath + ".backup"
	err := u.copyFile(currentPath, backupPath)
	if err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Создаем временный файл с правильным именем
	tempDir := filepath.Dir(currentPath)
	tempPath := filepath.Join(tempDir, "temp_"+currentFileName)

	// Копируем новый файл во временный с правильным именем
	err = u.copyFile(newPath, tempPath)
	if err != nil {
		return fmt.Errorf("failed to copy new executable: %w", err)
	}

	// На Windows может потребоваться особая обработка
	if runtime.GOOS == "windows" {
		return u.replaceExecutableWindows(currentPath, tempPath, backupPath)
	}

	// Заменяем файл (атомарная операция на Unix-системах)
	err = os.Rename(tempPath, currentPath)
	if err != nil {
		// Если не получилось, восстанавливаем из резервной копии
		u.copyFile(backupPath, currentPath)
		os.Remove(backupPath)
		os.Remove(tempPath)
		return fmt.Errorf("failed to replace executable: %w", err)
	}

	// Устанавливаем права доступа
	err = os.Chmod(currentPath, 0755)
	if err != nil {
		return fmt.Errorf("failed to set executable permissions: %w", err)
	}

	// Удаляем временные файлы
	os.Remove(backupPath)
	os.Remove(tempPath)
	return nil
}

// copyFile копирует файл
func (u *Updater) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}
