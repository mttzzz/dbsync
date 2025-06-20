//go:build !windows
// +build !windows

package updater

import (
	"fmt"
)

// replaceExecutableWindows заглушка для не-Windows платформ
func (u *Updater) replaceExecutableWindows(currentPath, tempPath, backupPath string) error {
	// Эта функция не должна вызываться на не-Windows платформах
	// но нужна для совместимости компиляции
	return fmt.Errorf("replaceExecutableWindows should not be called on non-Windows platforms")
}
