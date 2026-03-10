# Changelog

## [4.0.0] - 2026-03-10

### 🚀 Major Changes
- **TUI-first workflow**: `dbsync` now launches a full-screen terminal UI for database discovery, table selection, execution, and reporting
- **Interactive settings management**: remote/local connection settings and dump options can be edited from the UI and persisted to `$HOME/.dbsync.env`
- **Plan-based synchronization**: sync now works with queued targets, selected tables, and runtime options instead of a single positional database argument

### ✨ New Features
- **Multi-database queue**: select and run several databases in one session
- **Table-level sync**: choose specific tables per database with automatic foreign-key dependency inclusion
- **Live progress view**: running screen now shows normalized dump and restore phases, timers, traffic, and ETA
- **Richer metrics**: reports distinguish source data size, index size, compressed dump size, download traffic, upload traffic, and total network I/O
- **Config save/load support**: UI settings can be written to and loaded from `.env`-compatible config files
- **Transport compression controls**: MySQL Shell protocol compression and zstd level are configurable

### 🔧 Improvements
- **More accurate table metadata**: database lists and plans use data size separately from index size, with best-effort exact row counts where possible
- **Cleaner mysqlsh integration**: partial dumps use `--includeTables`, errors retain stdout/stderr context, and transport metrics are tracked in both direct and proxy modes
- **Better observability**: dump and restore subphases are classified into human-readable states instead of raw mysqlsh output

### 🐛 Fixed
- Fixed progress resets between dump and restore subphases
- Fixed warm-up ETA noise before enough traffic data is available
- Fixed Bubble Tea panics in live phase timers caused by empty labels and nil duration maps
- Fixed raw mysqlsh summary, throughput, and GTID-like lines leaking into phase breakdowns

### ⚠️ Breaking Changes
- Removed direct positional sync from the root command; the primary sync flow now runs through the interactive TUI
- Updated configuration workflow to center around in-app editing and persisted `.env` files

## [3.0.0] - 2025-12-01

### 🚀 Major Changes
- **Switched to MySQL Shell**: Replaced mydumper with MySQL Shell for 10x faster sync
- **No Docker required**: MySQL Shell runs natively, no container overhead
- **Cleaner output**: Minimal, non-repetitive console output

### ✨ New Features
- **util.dump-schemas / util.load-dump**: Uses MySQL Shell's parallel dump utilities
- **zstd compression**: 665 MB → 182 MB (3.5x compression ratio)
- **Deferred indexes**: `--deferTableIndexes=all` for faster restore
- **Skip consistency checks**: Works with managed MySQL (DigitalOcean, etc.)

### ⚡ Performance
- **665 MB database (63 tables, 2.3M rows)**:
  - Dump: 19 sec
  - Restore: 13 sec (including 11 sec for 113 indexes)
  - **Total: ~32 sec** (was 5+ min with mydumper)

### 🗑️ Removed
- Docker dependency
- mydumper/myloader code
- Verbose output and warnings

### 📋 Requirements
- MySQL Shell 8.4+ (install via `winget install Oracle.MySQLShell`)
- `SET GLOBAL local_infile = 1` on local MySQL

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
