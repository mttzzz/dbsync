package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"db-sync-cli/internal/config"
	"db-sync-cli/internal/models"
	"db-sync-cli/internal/services"
	"db-sync-cli/internal/version"

	"github.com/spf13/cobra"
)

func runTextInteractiveFlow(cfg *config.Config, dbService *services.DatabaseService, cmd *cobra.Command) error {
	reader := bufio.NewReader(os.Stdin)
	for {
		printSafeInteractiveMenu(cfg)
		action, err := promptForMenuAction(reader)
		if err != nil {
			return err
		}
		switch action {
		case "1":
			if err := runTextSyncSelection(cfg, dbService, cmd, reader); err != nil {
				return err
			}
		case "2":
			if err := printConnectionStatus(dbService); err != nil {
				return err
			}
		case "3":
			printCurrentConfig(cfg)
		case "4":
			fmt.Println("Run the full-screen UI with: dbsync tui")
		case "q":
			fmt.Println("Operation cancelled")
			return nil
		}
		fmt.Println()
	}
}

func printSafeInteractiveMenu(cfg *config.Config) {
	info := version.Get()
	fmt.Printf("dbsync %s\n", info.Version)
	fmt.Printf("Remote: %s:%d\n", cfg.Remote.Host, cfg.Remote.Port)
	fmt.Printf("Local:  %s:%d\n", cfg.Local.Host, cfg.Local.Port)
	fmt.Println()
	fmt.Println("1. List databases and sync")
	fmt.Println("2. Check connection status")
	fmt.Println("3. Show current config")
	fmt.Println("4. Show TUI command")
	fmt.Println("q. Quit")
	if cfg.Remote.HasProxy() {
		fmt.Println("Mode: proxy enabled")
	} else {
		fmt.Println("Mode: direct")
	}
	fmt.Println()
}

func promptForMenuAction(reader *bufio.Reader) (string, error) {
	for {
		fmt.Print("Select action [1-4/q]: ")
		raw, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		value := strings.ToLower(strings.TrimSpace(raw))
		switch value {
		case "1", "2", "3", "4", "q":
			return value, nil
		default:
			fmt.Println("Unknown action. Choose 1, 2, 3, 4 or q.")
		}
	}
}

func runTextSyncSelection(cfg *config.Config, dbService *services.DatabaseService, cmd *cobra.Command, reader *bufio.Reader) error {
	fmt.Println("Loading remote databases...")
	databases, err := dbService.ListDatabases(true)
	if err != nil {
		return fmt.Errorf("failed to list databases: %w", err)
	}
	if len(databases) == 0 {
		fmt.Println("No databases found on remote server")
		return nil
	}
	databases.SortBySize()

	printDatabaseChoices(databases)
	selected, err := promptForDatabaseSelection(reader, databases)
	if err != nil {
		return err
	}
	if selected == nil {
		fmt.Println("Selection cancelled")
		return nil
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	return executeSyncOperation(cfg, dbService, selected.Name, dryRun, false, cmd)
}

func printDatabaseChoices(databases models.DatabaseList) {
	for index, database := range databases {
		size := database.Size
		if database.DataSize > 0 {
			size = database.DataSize
		}
		fmt.Printf("%2d. %-40s %10s  %5d tables\n", index+1, database.Name, formatBytes(size), database.Tables)
	}
	fmt.Println()
	fmt.Println("Enter database number or exact name. Press Enter or q to cancel.")
}

func promptForDatabaseSelection(reader *bufio.Reader, databases models.DatabaseList) (*models.Database, error) {
	for {
		fmt.Print("> ")
		raw, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		value := strings.TrimSpace(raw)
		if value == "" || strings.EqualFold(value, "q") {
			return nil, nil
		}
		selected := selectDatabase(databases, value)
		if selected != nil {
			return selected, nil
		}
		fmt.Println("Unknown database selection. Enter a number from the list or an exact database name.")
	}
}

func selectDatabase(databases models.DatabaseList, value string) *models.Database {
	if index, err := strconv.Atoi(value); err == nil {
		if index >= 1 && index <= len(databases) {
			return &databases[index-1]
		}
		return nil
	}
	for index := range databases {
		if databases[index].Name == value {
			return &databases[index]
		}
	}
	return nil
}

func promptForConfirmation(message string) (bool, error) {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("%s [y/N]: ", message)
		raw, err := reader.ReadString('\n')
		if err != nil {
			return false, err
		}
		switch strings.ToLower(strings.TrimSpace(raw)) {
		case "y", "yes":
			return true, nil
		case "", "n", "no":
			return false, nil
		default:
			fmt.Println("Please answer y or n.")
		}
	}
}

func printConnectionStatus(dbService *services.DatabaseService) error {
	fmt.Println("Checking MySQL server connections...")
	remoteInfo, remoteErr := dbService.TestConnection(true)
	if remoteInfo != nil {
		fmt.Print(formatConnectionStatus(remoteInfo, "Remote"))
	}
	if remoteErr != nil {
		fmt.Println(remoteErr)
	}
	fmt.Println()
	localInfo, localErr := dbService.TestConnection(false)
	if localInfo != nil {
		fmt.Print(formatConnectionStatus(localInfo, "Local"))
	}
	if localErr != nil {
		fmt.Println(localErr)
	}
	return nil
}

func printCurrentConfig(cfg *config.Config) {
	fmt.Printf("Remote MySQL: %s:%d (user: %s)\n", cfg.Remote.Host, cfg.Remote.Port, cfg.Remote.User)
	if cfg.Remote.HasProxy() {
		fmt.Printf("Remote Proxy: %s\n", cfg.Remote.RedactedProxyURL())
	}
	fmt.Printf("Local MySQL: %s:%d (user: %s)\n", cfg.Local.Host, cfg.Local.Port, cfg.Local.User)
	fmt.Printf("Dump Timeout: %s\n", cfg.Dump.Timeout)
	fmt.Printf("Threads: %d\n", cfg.Dump.Threads)
	fmt.Printf("Compress: %v\n", cfg.Dump.Compress)
	fmt.Printf("Network Compress: %v\n", cfg.Dump.NetworkCompress)
	fmt.Printf("Network Zstd Level: %d\n", cfg.Dump.NetworkZstdLevel)
}
