# Changelog

## [1.1.0] - 2025-06-20

### ✨ Added
- **Auto-update functionality**: New `upgrade` command to check for and install updates from GitHub releases
  - `dbsync upgrade` - Check and install latest version
  - `dbsync upgrade --check-only` - Only check for updates without installing
  - `dbsync upgrade --force` - Skip confirmation prompt
- **Improved database selection**: Databases in interactive mode are now sorted by size (largest first)
- **Better user experience**: Enhanced visual feedback during update process

### 🔧 Enhanced
- Interactive database selector now shows databases sorted by size for better prioritization
- Added comprehensive error handling for update process
- Cross-platform update support (Windows, Linux, macOS)

### 🛠️ Technical
- New `internal/updater` package for handling GitHub releases
- Added methods for sorting database lists (`SortBySize`, `SortBySizeAsc`, `SortByName`)
- Improved version comparison and platform detection

## [1.0.0] - Initial Release

First version of the project.
