package cli

import (
	"fmt"

	"db-sync-cli/internal/config"
	"db-sync-cli/internal/services"
	"db-sync-cli/internal/tui"
	"db-sync-cli/internal/updater"
	"db-sync-cli/internal/version"

	"github.com/spf13/cobra"
)

var (
	// Глобальные флаги
	verbose    bool
	configFile string
)

// rootCmd представляет основную команду
var rootCmd = &cobra.Command{
	Use:   "dbsync",
	Short: "MySQL database synchronization tool",
	Long: `dbsync is a CLI tool for synchronizing MySQL databases between remote and local servers.
Uses MySQL Shell (mysqlsh) for fast parallel dump and restore operations.

	Run without arguments to launch the full-screen terminal UI.`,
	Version: version.Version,
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTUI(cmd)
	},
}

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the full-screen terminal UI",
	Long:  `Launch the full-screen terminal UI for database selection, settings and sync execution.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTUI(cmd)
	},
}

var textCmd = &cobra.Command{
	Use:   "text",
	Short: "Launch the text-mode interface",
	Long:  `Launch the fallback text-mode interface without the full-screen terminal UI.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadCLIConfig(cmd)
		if err != nil {
			return err
		}

		dbService := services.NewDatabaseService(cfg)
		return runTextInteractiveFlow(cfg, dbService, cmd)
	},
}

func loadCLIConfig(cmd *cobra.Command) (*config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	threads, _ := cmd.Flags().GetInt("threads")
	if threads > 0 {
		cfg.Dump.Threads = threads
	}

	return cfg, nil
}

func runTUI(cmd *cobra.Command) error {
	cfg, err := loadCLIConfig(cmd)
	if err != nil {
		return err
	}

	dbService := services.NewDatabaseService(cfg)
	shellService := services.NewMySQLShellService(cfg, dbService)

	_, err = tui.RunApp(cfg, dbService, shellService, nil)
	if err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}

	return nil
}

// executeSyncOperation выполняет синхронизацию через MySQL Shell
func executeSyncOperation(cfg *config.Config, dbService *services.DatabaseService, databaseName string, dryRun bool, skipConfirmation bool, cmd *cobra.Command) error {
	shellService := services.NewMySQLShellService(cfg, dbService)

	if dryRun {
		fmt.Printf("🧪 DRY RUN - no changes will be made\n")
		return nil
	}

	// Запрашиваем подтверждение у пользователя
	force, _ := cmd.Flags().GetBool("force")
	if !force && !skipConfirmation {
		message := fmt.Sprintf("This will replace the local database '%s'", databaseName)
		confirmed, err := promptForConfirmation(message)
		if err != nil {
			return fmt.Errorf("confirmation failed: %w", err)
		}

		if !confirmed {
			fmt.Printf("❌ Operation cancelled\n")
			return nil
		}
	}

	// Выполняем синхронизацию
	syncResult, err := shellService.ExecuteSync(databaseName)
	if err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	// Показываем результат
	fmt.Printf("\n✅ Done! %s in %s (dump: %s, restore: %s)\n",
		formatBytes(syncResult.DumpSizeOnDisk),
		formatDuration(syncResult.Duration),
		formatDuration(syncResult.DumpDuration),
		formatDuration(syncResult.RestoreDuration))
	if syncResult.LogicalSize > 0 || syncResult.Traffic.TotalBytes() > 0 {
		printSyncResult(syncResult)
	}

	return nil
}

// listCmd команда получения списка БД
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available databases on remote server",
	Long:  `Show a list of all databases available on the remote MySQL server.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		dbService := services.NewDatabaseService(cfg)

		fmt.Printf("Connecting to remote server %s:%d...\n", cfg.Remote.Host, cfg.Remote.Port)

		databases, err := dbService.ListDatabases(true) // true = remote
		if err != nil {
			return fmt.Errorf("failed to list databases: %w", err)
		}

		if len(databases) == 0 {
			fmt.Println("No databases found on remote server")
			return nil
		}

		printDatabaseList(databases, cfg.Remote.Host)

		return nil
	},
}

// statusCmd команда проверки статуса
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check connection status to remote and local servers",
	Long:  `Check if both remote and local MySQL servers are accessible.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		dbService := services.NewDatabaseService(cfg)

		fmt.Println("Checking MySQL server connections...")
		fmt.Println()

		// Проверяем удаленный сервер
		remoteInfo, _ := dbService.TestConnection(true)
		output := formatConnectionStatus(remoteInfo, "Remote")
		fmt.Print(output)

		fmt.Println()

		// Проверяем локальный сервер
		localInfo, _ := dbService.TestConnection(false)
		output = formatConnectionStatus(localInfo, "Local")
		fmt.Print(output)

		// Общий статус
		fmt.Println()
		if remoteInfo.Connected && localInfo.Connected {
			fmt.Println("OK: All connections are working!")
		} else {
			fmt.Println("WARN: Some connections have issues. Please check your configuration.")
		}

		return nil
	},
}

// configCmd команда показа конфигурации
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show current configuration",
	Long:  `Display the current configuration settings.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		fmt.Printf("Remote MySQL: %s:%d (user: %s)\n",
			cfg.Remote.Host, cfg.Remote.Port, cfg.Remote.User)
		if cfg.Remote.HasProxy() {
			fmt.Printf("Remote Proxy: %s\n", cfg.Remote.RedactedProxyURL())
		}
		fmt.Printf("Local MySQL: %s:%d (user: %s)\n",
			cfg.Local.Host, cfg.Local.Port, cfg.Local.User)
		if cfg.Local.HasProxy() {
			fmt.Printf("Local Proxy: %s\n", cfg.Local.RedactedProxyURL())
		}
		fmt.Printf("Dump Timeout: %s\n", cfg.Dump.Timeout)
		fmt.Printf("\n--- MySQL Shell Settings ---\n")
		fmt.Printf("Threads: %d\n", cfg.Dump.Threads)
		fmt.Printf("Compress: %v (zstd)\n", cfg.Dump.Compress)

		return nil
	},
}

// versionCmd команда показа версии
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  `Display version, build information and platform details.`,
	Run: func(cmd *cobra.Command, args []string) {
		info := version.Get()
		fmt.Println(info.String())
	},
}

// upgradeCmd команда обновления приложения
var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Check for updates and upgrade the application",
	Long: `Check for the latest version of dbsync on GitHub and upgrade if a newer version is available.
This command will download and replace the current executable with the latest version.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Checking for updates...")

		up := updater.NewUpdater()
		updateInfo, err := up.CheckForUpdates()
		if err != nil {
			return fmt.Errorf("failed to check for updates: %w", err)
		}

		if !updateInfo.Available {
			fmt.Printf("✅ You are already using the latest version (%s)\n", updateInfo.CurrentVersion)
			return nil
		}

		// Показываем информацию об обновлении
		fmt.Printf("\n🎉 New version available!\n")
		fmt.Printf("   Current: %s\n", updateInfo.CurrentVersion)
		fmt.Printf("   Latest:  %s\n", updateInfo.LatestVersion)
		fmt.Printf("   Size:    %s\n", formatBytes(updateInfo.AssetSize))
		fmt.Printf("   Released: %s\n", updateInfo.PublishedAt.Format("2006-01-02"))

		// Запрашиваем подтверждение
		checkOnly, _ := cmd.Flags().GetBool("check-only")
		if checkOnly {
			return nil
		}

		force, _ := cmd.Flags().GetBool("force")
		if !force {
			message := fmt.Sprintf("Download and install version %s?", updateInfo.LatestVersion)
			confirmed, err := promptForConfirmation(message)
			if err != nil {
				return fmt.Errorf("confirmation failed: %w", err)
			}

			if !confirmed {
				fmt.Printf("❌ Update cancelled by user\n")
				return nil
			}
		}

		// Выполняем обновление
		fmt.Printf("\n🚀 Downloading and installing update...\n")
		result, err := up.PerformUpdate(updateInfo)
		if err != nil {
			return fmt.Errorf("update failed: %w", err)
		}

		if result.Success {
			fmt.Printf("✅ Update completed successfully!\n")
			fmt.Printf("   Updated from %s to %s\n", result.PreviousVersion, result.NewVersion)
			fmt.Printf("   Duration: %s\n", formatDuration(result.Duration))
			fmt.Printf("\n💡 The application has been updated. You can continue using it immediately.\n")
		} else {
			return fmt.Errorf("update failed: %s", result.Error)
		}

		return nil
	},
}

// Execute добавляет все дочерние команды к корневой команде и устанавливает флаги
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Глобальные флаги
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default is .env)")

	// Флаги для синхронизации (теперь в rootCmd)
	rootCmd.Flags().Bool("dry-run", false, "show what would be done without executing")
	rootCmd.Flags().Bool("force", false, "skip confirmation prompts for destructive operations")
	rootCmd.Flags().Int("threads", 8, "number of threads for parallel dump/restore")

	// Флаги для команды upgrade
	upgradeCmd.Flags().Bool("check-only", false, "only check for updates without installing")
	upgradeCmd.Flags().Bool("force", false, "skip confirmation prompt for update")

	// Добавляем команды
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(upgradeCmd)
	rootCmd.AddCommand(tuiCmd)
	rootCmd.AddCommand(textCmd)
}
