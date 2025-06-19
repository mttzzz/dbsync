package cli

import (
	"fmt"

	"db-sync-cli/internal/config"
	"db-sync-cli/internal/services"
	"db-sync-cli/internal/ui"
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
It creates dumps from remote databases and restores them to local instances with progress tracking.`,
	Version: version.Version,
}

// syncCmd команда синхронизации
var syncCmd = &cobra.Command{
	Use:   "sync [database_name]",
	Short: "Synchronize a database from remote to local",
	Long: `Synchronize a specific database from remote server to local server.
If database name is not provided, an interactive selection will be shown.

Available flags:
  --dry-run   Show what would be done without making changes
  --force     Skip confirmation prompts for destructive operations`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Получаем флаги
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		dbService := services.NewDatabaseService(cfg)
		dumpService := services.NewDumpService(cfg, dbService)

		var databaseName string

		// Если имя БД не указано, показываем интерактивный выбор
		if len(args) == 0 {
			databases, err := dbService.ListDatabases(true)
			if err != nil {
				return fmt.Errorf("failed to list databases: %w", err)
			}

			if len(databases) == 0 {
				fmt.Println("No databases found on remote server")
				return nil
			}

			// Запускаем интерактивный селектор
			selected, err := RunDatabaseSelector(databases)
			if err != nil {
				return fmt.Errorf("database selection failed: %w", err)
			}

			databaseName = selected.Name
		} else {
			databaseName = args[0]
		}

		// Выполняем проверки безопасности
		fmt.Printf("🔍 Running safety checks for database '%s'...\n\n", databaseName)

		checks, err := dumpService.GetSafetyChecks(databaseName)
		for _, check := range checks {
			fmt.Println(check)
		}

		if err != nil {
			fmt.Printf("\n❌ Safety checks failed: %v\n", err)
			return err
		}

		fmt.Println("\n✅ All safety checks passed!")

		// Показываем план операции
		result, err := dumpService.PlanDumpOperation(databaseName)
		if err != nil {
			return fmt.Errorf("failed to plan operation: %w", err)
		}

		fmt.Printf("\n📋 Operation Plan:\n")
		fmt.Printf("   Database: %s\n", result.DatabaseName)
		fmt.Printf("   Size: %s\n", ui.FormatSize(result.DumpSize))
		fmt.Printf("   Tables: %d\n", result.TablesCount)

		if dryRun {
			fmt.Printf("\n🧪 DRY RUN MODE - No changes will be made\n")
			fmt.Printf("   %s\n", result.Error)

			// Показываем команды которые будут выполнены
			fmt.Printf("\n📝 Commands that would be executed:\n")
			dumpCmd := dumpService.GetDumpCommand(databaseName)
			fmt.Printf("   Dump: %s\n", dumpCmd[0])

			restoreCmd := dumpService.GetRestoreCommand(databaseName)
			fmt.Printf("   Restore: %s\n", restoreCmd[0])

			return nil
		}

		// Реальная синхронизация
		fmt.Printf("\n🚀 Starting synchronization...\n")

		// Запрашиваем подтверждение у пользователя
		force, _ := cmd.Flags().GetBool("force")
		if !force {
			message := fmt.Sprintf("This will replace the local database '%s'", databaseName)
			confirmed, err := RunConfirmationSelector(message)
			if err != nil {
				return fmt.Errorf("confirmation failed: %w", err)
			}

			if !confirmed {
				fmt.Printf("❌ Operation cancelled by user\n")
				return nil
			}
		}

		// Выполняем синхронизацию
		syncResult, err := dumpService.ExecuteSync(databaseName)
		if err != nil {
			return fmt.Errorf("sync failed: %w", err)
		}

		// Показываем результат
		fmt.Printf("\n✅ Synchronization completed successfully!\n")
		fmt.Printf("   Database: %s\n", syncResult.DatabaseName)
		fmt.Printf("   Total Duration: %s\n", ui.FormatDuration(syncResult.Duration))
		fmt.Printf("     ├─ Dump: %s\n", ui.FormatDuration(syncResult.DumpDuration))
		fmt.Printf("     └─ Restore: %s\n", ui.FormatDuration(syncResult.RestoreDuration))
		fmt.Printf("   Size: %s\n", ui.FormatSize(syncResult.DumpSize))
		fmt.Printf("   Tables: %d\n", syncResult.TablesCount)

		return nil
	},
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
		formatter := ui.NewFormatter()

		fmt.Printf("Connecting to remote server %s:%d...\n", cfg.Remote.Host, cfg.Remote.Port)

		databases, err := dbService.ListDatabases(true) // true = remote
		if err != nil {
			return fmt.Errorf("failed to list databases: %w", err)
		}

		if len(databases) == 0 {
			fmt.Println(ui.InfoStyle.Render("No databases found on remote server"))
			return nil
		}

		output := formatter.FormatDatabaseList(databases, cfg.Remote.Host)
		fmt.Println(output)

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
		formatter := ui.NewFormatter()

		fmt.Println(ui.InfoStyle.Render("Checking MySQL server connections..."))
		fmt.Println()

		// Проверяем удаленный сервер
		remoteInfo, _ := dbService.TestConnection(true)
		output := formatter.FormatConnectionStatus(remoteInfo, "Remote")
		fmt.Print(output)

		fmt.Println()

		// Проверяем локальный сервер
		localInfo, _ := dbService.TestConnection(false)
		output = formatter.FormatConnectionStatus(localInfo, "Local")
		fmt.Print(output)

		// Общий статус
		fmt.Println()
		if remoteInfo.Connected && localInfo.Connected {
			fmt.Println(ui.FormatStatus("success", "All connections are working!"))
		} else {
			fmt.Println(ui.FormatStatus("warning", "Some connections have issues. Please check your configuration."))
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
		fmt.Printf("Local MySQL: %s:%d (user: %s)\n",
			cfg.Local.Host, cfg.Local.Port, cfg.Local.User)
		fmt.Printf("Temp Directory: %s\n", cfg.Dump.TempDir)
		fmt.Printf("Dump Timeout: %s\n", cfg.Dump.Timeout)

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

// Execute добавляет все дочерние команды к корневой команде и устанавливает флаги
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Глобальные флаги
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default is .env)")

	// Флаги для команды sync
	syncCmd.Flags().String("remote-host", "", "remote MySQL host (overrides config)")
	syncCmd.Flags().String("local-host", "", "local MySQL host (overrides config)")
	syncCmd.Flags().Bool("dry-run", false, "show what would be done without executing")
	syncCmd.Flags().Bool("force", false, "skip confirmation prompts for destructive operations")

	// Добавляем команды
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(versionCmd)
}
