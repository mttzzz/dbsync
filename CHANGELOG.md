# Changelog

## [2.0.0] - 2025-12-01

### 🚀 Major Changes
- **Complete rewrite using mydumper/myloader**: Replaced mysqldump with mydumper for 2-3x faster database synchronization
- **Docker-based**: Now requires Docker to run mydumper/myloader containers
- **Simplified interface**: Run `dbsync` without arguments for interactive mode, or `dbsync database_name` for direct sync
- **Removed mysqldump support**: The old mysqldump-based sync has been completely removed

### ✨ New Features
- **Parallel dump/restore**: Uses multiple threads (default: 8) for significantly faster operations
- **Network compression**: `--compress-protocol` for faster remote transfers
- **Optimized restore**: Indexes and foreign keys are created after data import for maximum speed
- **Automatic cleanup**: Temporary files are created in system temp directory and cleaned up automatically

### 🔧 Configuration Changes
- Removed `DBSYNC_DUMP_TEMP_DIR` - now uses system temp directory
- Removed `DBSYNC_DUMP_MYSQLDUMP_PATH` - no longer needed
- Removed `DBSYNC_DUMP_USE_MYDUMPER` - mydumper is now the only option
- Simplified `.env.example` to essential settings only

### 📊 Performance
- **2.3 GB database**: ~36 seconds (vs ~3+ minutes with mysqldump)
- Dump: ~19s with 8 threads and network compression
- Restore: ~16s with parallel import and deferred index creation

### 🗑️ Removed
- `sync` subcommand (now root command handles sync)
- `benchmark` command
- All mysqldump-related code
- Size column from database list (was inaccurate)

### 📦 Build
- Now builds only for Windows x64 and macOS Apple Silicon

## [1.1.2] - 2025-06-20

### 🔧 Fixed
- Fixed cross-platform compilation issues with build constraints
- Added Unix stub for Windows-specific functions to ensure proper compilation on all platforms
- Improved build constraint compatibility for go vet across platforms

## [1.1.1] - 2025-06-20

### 🔧 Fixed
- Fixed Windows compilation issues with syscall imports
- Improved cross-platform compatibility for the updater module
- Added proper build constraints for Windows-specific code

### 🛠️ Technical
- Split Windows-specific updater code into separate file with build constraints
- Updated dependencies to use golang.org/x/sys/windows instead of deprecated syscall functions
- Added go vet and staticcheck to the testing pipeline

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
