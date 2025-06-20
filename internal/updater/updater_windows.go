//go:build windows

package updater

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

// replaceExecutableWindows заменяет исполняемый файл на Windows
func (u *Updater) replaceExecutableWindows(currentPath, tempPath, backupPath string) error {
	// На Windows сначала попробуем прямую замену
	err := os.Rename(tempPath, currentPath)
	if err != nil {
		// Если не получилось заменить немедленно, используем MoveFileEx
		// с флагом MOVEFILE_DELAY_UNTIL_REBOOT
		currentPathPtr, _ := windows.UTF16PtrFromString(currentPath)
		tempPathPtr, _ := windows.UTF16PtrFromString(tempPath)

		// MOVEFILE_DELAY_UNTIL_REBOOT = 0x4
		err := windows.MoveFileEx(tempPathPtr, currentPathPtr, windows.MOVEFILE_DELAY_UNTIL_REBOOT)
		if err != nil {
			// Восстанавливаем из резервной копии
			u.copyFile(backupPath, currentPath)
			os.Remove(backupPath)
			os.Remove(tempPath)
			return fmt.Errorf("failed to schedule file replacement: %w", err)
		}

		// Не удаляем файлы, так как замена произойдет при перезагрузке
		return fmt.Errorf("update scheduled for next system restart - please restart your computer")
	}

	// Если замена прошла успешно, устанавливаем права доступа
	err = os.Chmod(currentPath, 0755)
	if err != nil {
		return fmt.Errorf("failed to set executable permissions: %w", err)
	}

	// Удаляем временные файлы
	os.Remove(backupPath)
	os.Remove(tempPath)
	return nil
}
