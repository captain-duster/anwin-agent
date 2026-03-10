package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/anwin/agent/internal/client"
	"github.com/anwin/agent/internal/config"
	"github.com/anwin/agent/internal/logger"
	"github.com/anwin/agent/internal/scanner"
	"github.com/anwin/agent/internal/watcher"
)

const version = "1.0.0"

const banner = `
  █████╗ ███╗   ██╗██╗    ██╗██╗███╗   ██╗
 ██╔══██╗████╗  ██║██║    ██║██║████╗  ██║
 ███████║██╔██╗ ██║██║ █╗ ██║██║██╔██╗ ██║
 ██╔══██║██║╚██╗██║██║███╗██║██║██║╚██╗██║
 ██║  ██║██║ ╚████║╚███╔███╔╝██║██║ ╚████║
 ╚═╝  ╚═╝╚═╝  ╚═══╝ ╚══╝╚══╝ ╚═╝╚═╝  ╚═══╝
 Local Code Sync Agent  v%s
`

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "setup":
		runSetup()
	case "start":
		runStart()
	case "status":
		runStatus()
	case "reset":
		runReset()
	case "version":
		fmt.Printf("ANWIN Agent v%s\n", version)
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf(banner, version)
	fmt.Println("Usage:")
	fmt.Println("  anwin-agent setup     Configure the agent (run once)")
	fmt.Println("  anwin-agent start     Start syncing your project")
	fmt.Println("  anwin-agent status    Show connection and config status")
	fmt.Println("  anwin-agent reset     Remove saved configuration")
	fmt.Println("  anwin-agent version   Show agent version")
	fmt.Println("")
}

func prompt(reader *bufio.Reader, label string) string {
	fmt.Print("  " + label + ": ")
	val, _ := reader.ReadString('\n')
	return strings.TrimSpace(val)
}

func runSetup() {
	fmt.Printf(banner, version)
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Setup — You will need the following from ANWIN → Project Settings → Local Agent")
	fmt.Println("──────────────────────────────────────────────────────────────────────────────")
	fmt.Println("")

	serverURL := prompt(reader, "ANWIN Server URL   (e.g. https://app.anwin.ai)")
	token := prompt(reader, "Agent Token        (generated in Project Settings)")
	projectID := prompt(reader, "Project ID         (shown in Project Settings)")
	watchPath := prompt(reader, "Directory to watch (full absolute path to your codebase)")

	fmt.Println("")

	if serverURL == "" || token == "" || projectID == "" || watchPath == "" {
		fmt.Println("  All fields are required. Please run setup again.")
		os.Exit(1)
	}

	serverURL = strings.TrimRight(serverURL, "/")

	info, err := os.Stat(watchPath)
	if os.IsNotExist(err) {
		fmt.Printf("  Directory not found: %s\n", watchPath)
		fmt.Println("  Please check the path and run setup again.")
		os.Exit(1)
	}
	if !info.IsDir() {
		fmt.Printf("  Path is not a directory: %s\n", watchPath)
		os.Exit(1)
	}

	cfg := &config.Config{
		ServerURL: serverURL,
		Token:     token,
		ProjectID: projectID,
		WatchPath: watchPath,
		Version:   version,
	}

	fmt.Println("  Saving configuration...")
	if err := config.Save(cfg); err != nil {
		fmt.Printf("  Failed to save config: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("  Registering with ANWIN server...")
	anwinClient := client.New(serverURL, token, version)
	if err := anwinClient.RegisterAgent(projectID, watchPath); err != nil {
		fmt.Printf("  Registration failed: %v\n", err)
		fmt.Println("  Check your token and server URL, then run setup again.")
		_ = config.Delete()
		os.Exit(1)
	}

	fmt.Println("")
	fmt.Println("  ✓ Setup complete")
	fmt.Println("")
	fmt.Println("  Run 'anwin-agent start' to begin syncing.")
	fmt.Println("")
}

func runStart() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Println("Agent is not configured. Run 'anwin-agent setup' first.")
		os.Exit(1)
	}

	if _, err := os.Stat(cfg.WatchPath); os.IsNotExist(err) {
		fmt.Printf("Watch directory not found: %s\n", cfg.WatchPath)
		fmt.Println("If the path changed, run 'anwin-agent setup' again.")
		os.Exit(1)
	}

	fmt.Printf(banner, version)

	logger.Info("Agent starting",
		"version", version,
		"project", cfg.ProjectID,
		"path", cfg.WatchPath,
		"server", cfg.ServerURL,
	)

	anwinClient := client.New(cfg.ServerURL, cfg.Token, cfg.Version)

	logger.Info("Connecting to server...")
	if !anwinClient.Ping() {
		logger.Fatal("Cannot reach ANWIN server. Check your connection and server URL.")
	}
	logger.Info("Server connection established")

	logger.Info("Starting initial directory scan...")
	start := time.Now()
	scanResult, err := scanner.ScanDirectory(cfg.WatchPath)
	if err != nil {
		logger.Fatal("Directory scan failed", "error", err.Error())
	}
	logger.Info("Scan complete",
		"files", scanResult.Total,
		"skipped", scanResult.Skipped,
		"batches", len(scanResult.Batches),
	)

	for i, batch := range scanResult.Batches {
		logger.Info("Uploading batch",
			"batch", fmt.Sprintf("%d/%d", i+1, len(scanResult.Batches)),
			"files", len(batch),
		)
		if err := anwinClient.SyncFiles(cfg.ProjectID, batch, true); err != nil {
			logger.Fatal("Initial sync failed", "error", err.Error())
		}
	}
	logger.Info("Initial sync complete",
		"files", scanResult.Total,
		"duration", fmt.Sprintf("%.2fs", time.Since(start).Seconds()),
	)

	fmt.Println("")
	logger.Info("Watching for changes. Press Ctrl+C to stop.")
	fmt.Println("")

	w := watcher.New(cfg.WatchPath, cfg.ProjectID, anwinClient)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		errCh <- w.Start()
	}()

	select {
	case sig := <-sigCh:
		fmt.Println("")
		logger.Info("Shutdown signal received", "signal", sig.String())
		w.Stop()
		time.Sleep(500 * time.Millisecond)
		logger.Info("Agent stopped cleanly")

	case err := <-errCh:
		if err != nil {
			logger.Fatal("Watcher stopped unexpectedly", "error", err.Error())
		}
	}
}

func runStatus() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Println("Agent not configured. Run 'anwin-agent setup' first.")
		return
	}

	fmt.Println("")
	fmt.Println("  ANWIN Agent Status")
	fmt.Println("  ─────────────────────────────────────")
	fmt.Printf("  Version  : %s\n", cfg.Version)
	fmt.Printf("  Server   : %s\n", cfg.ServerURL)
	fmt.Printf("  Project  : %s\n", cfg.ProjectID)
	fmt.Printf("  Watching : %s\n", cfg.WatchPath)

	anwinClient := client.New(cfg.ServerURL, cfg.Token, cfg.Version)
	if anwinClient.Ping() {
		fmt.Println("  Status   : ✓ Connected")
	} else {
		fmt.Println("  Status   : ✗ Server unreachable")
	}

	info, err := os.Stat(cfg.WatchPath)
	if err != nil || !info.IsDir() {
		fmt.Println("  Directory: ✗ Not found")
	} else {
		fmt.Println("  Directory: ✓ Exists")
	}
	fmt.Println("")
}

func runReset() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("This will remove all saved configuration. Are you sure? (yes/no): ")
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))

	if answer != "yes" {
		fmt.Println("Reset cancelled.")
		return
	}

	if err := config.Delete(); err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No configuration found.")
			return
		}
		fmt.Printf("Failed to remove config: %v\n", err)
		return
	}
	fmt.Println("Configuration removed. Run 'anwin-agent setup' to configure again.")
}
