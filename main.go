package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/captain-duster/anwin-agent/internal/client"
	"github.com/captain-duster/anwin-agent/internal/commander"
	"github.com/captain-duster/anwin-agent/internal/config"
	"github.com/captain-duster/anwin-agent/internal/scanner"
	"github.com/captain-duster/anwin-agent/internal/watcher"
)

const version = "2.0.0"

const banner = `
  ╔══════════════════════════════════════════╗
  ║                                          ║
  ║        █████╗ ███╗   ██╗██╗    ██╗      ║
  ║       ██╔══██╗████╗  ██║██║    ██║      ║
  ║       ███████║██╔██╗ ██║██║ █╗ ██║      ║
  ║       ██╔══██║██║╚██╗██║██║███╗██║      ║
  ║       ██║  ██║██║ ╚████║╚███╔███╔╝      ║
  ║       ╚═╝  ╚═╝╚═╝  ╚═══╝ ╚══╝╚══╝       ║
  ║                                          ║
  ║         Local Code Sync Agent            ║
  ║             Version v%s               ║
  ║                                          ║
  ╚══════════════════════════════════════════╝
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
	fmt.Println("  Usage:")
	fmt.Println("    anwin-agent setup     Configure the agent (run once)")
	fmt.Println("    anwin-agent start     Start syncing your project")
	fmt.Println("    anwin-agent status    Show connection and config status")
	fmt.Println("    anwin-agent reset     Remove saved configuration")
	fmt.Println("    anwin-agent version   Show agent version")
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

	fmt.Println("  Setup — You need the following from ANWIN → Project Settings → Local Agent")
	fmt.Println("  ──────────────────────────────────────────────────────────────────────────")
	fmt.Println("")

	serverURL := prompt(reader, "ANWIN Server URL   (e.g. https://app.anwin.ai)")
	token := prompt(reader, "Agent Token        (generated in Project Settings)")
	watchPath := prompt(reader, "Directory to watch (full absolute path to your codebase)")

	fmt.Println("")

	if serverURL == "" || token == "" || watchPath == "" {
		fmt.Println("  ✗ All fields are required. Please run setup again.")
		os.Exit(1)
	}

	serverURL = strings.TrimRight(serverURL, "/")

	absPath, err := filepath.Abs(watchPath)
	if err != nil {
		fmt.Printf("  ✗ Invalid path: %s\n", err.Error())
		os.Exit(1)
	}
	watchPath = absPath

	info, err := os.Stat(watchPath)
	if os.IsNotExist(err) {
		fmt.Printf("  ✗ Directory not found: %s\n", watchPath)
		os.Exit(1)
	}
	if !info.IsDir() {
		fmt.Printf("  ✗ Not a directory: %s\n", watchPath)
		os.Exit(1)
	}

	cfg := &config.AgentConfig{
		ServerURL: serverURL,
		Token:     token,
		WatchPath: watchPath,
	}

	if err := config.Save(cfg); err != nil {
		fmt.Printf("  ✗ Failed to save config: %s\n", err.Error())
		os.Exit(1)
	}

	anwinClient := client.New(serverURL, token)
	fmt.Print("  ◆ Verifying connection... ")

	if err := anwinClient.Ping(); err != nil {
		fmt.Printf("FAILED\n    %s\n", err.Error())
		fmt.Println("    Config saved anyway. Fix the issue and run: anwin-agent start")
		os.Exit(1)
	}

	fmt.Println("OK")
	writeReadme(watchPath)

	fmt.Println("")
	fmt.Println("  ╔══════════════════════════════════════════╗")
	fmt.Println("  ║   ✓  Setup complete!                     ║")
	fmt.Println("  ║                                          ║")
	fmt.Println("  ║   Next: anwin-agent start                ║")
	fmt.Println("  ╚══════════════════════════════════════════╝")
	fmt.Println("")
}

func runStart() {
	fmt.Printf(banner, version)

	cfg, err := config.Load()
	if err != nil {
		fmt.Println("  ✗ Not configured. Run: anwin-agent setup")
		os.Exit(1)
	}

	anwinClient := client.New(cfg.ServerURL, cfg.Token)

	fmt.Print("  ◆ Connecting to ANWIN... ")
	if err := anwinClient.Ping(); err != nil {
		fmt.Printf("FAILED\n    %s\n", err.Error())
		os.Exit(1)
	}
	fmt.Println("OK")

	fingerprint := config.MachineFingerprint()
	platform := config.DetectPlatform()

	fmt.Printf("  ◆ Platform   →  %s\n", platform)
	fmt.Printf("  ◆ Machine    →  %s\n", fingerprint[:16]+"...")
	fmt.Printf("  ◆ Watching   →  %s\n", cfg.WatchPath)

	fmt.Print("  ◆ Registering agent... ")
	if err := anwinClient.Register(platform, version, fingerprint, cfg.WatchPath); err != nil {
		fmt.Printf("FAILED\n    %s\n", err.Error())
		os.Exit(1)
	}
	fmt.Println("OK")

	fmt.Print("  ◆ Running initial scan... ")
	files := scanner.ScanDirectory(cfg.WatchPath)
	fmt.Printf("found %d file(s)\n", len(files))

	if len(files) > 0 {
		fmt.Print("  ◆ Syncing initial snapshot... ")
		batches := client.SplitBatches(files, 200)
		for i, batch := range batches {
			if err := anwinClient.Sync(batch, true); err != nil {
				fmt.Printf("FAILED (batch %d/%d)\n    %s\n", i+1, len(batches), err.Error())
				os.Exit(1)
			}
		}
		fmt.Printf("OK (%d batch(es))\n", len(batches))
	}

	fmt.Println("")
	fmt.Println("  ════════════════════════════════════════════")
	fmt.Println("  Agent is running. Press Ctrl+C to stop.")
	fmt.Println("  ════════════════════════════════════════════")
	fmt.Println("")

	w := watcher.New(cfg.WatchPath, anwinClient)
	go w.Start()

	cmd := commander.New(cfg.WatchPath, anwinClient)
	go cmd.Start()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("")
	fmt.Printf("  [%s] Agent stopped.\n", timestamp())
}

func runStatus() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Println("  ✗ Not configured. Run: anwin-agent setup")
		os.Exit(1)
	}

	fmt.Printf(banner, version)
	fmt.Println("  Configuration")
	fmt.Println("  ─────────────────────────────────────")
	fmt.Printf("  Server     →  %s\n", cfg.ServerURL)
	fmt.Printf("  Token      →  %s••••\n", cfg.Token[:8])
	fmt.Printf("  Watch Path →  %s\n", cfg.WatchPath)
	fmt.Printf("  Platform   →  %s\n", config.DetectPlatform())
	fmt.Printf("  Machine    →  %s\n", config.MachineFingerprint()[:16]+"...")
	fmt.Println("")

	anwinClient := client.New(cfg.ServerURL, cfg.Token)
	fmt.Print("  Connection →  ")
	if err := anwinClient.Ping(); err != nil {
		fmt.Println("DISCONNECTED")
		fmt.Printf("    %s\n", err.Error())
	} else {
		fmt.Println("CONNECTED")
	}

	files := scanner.ScanDirectory(cfg.WatchPath)
	fmt.Printf("  Files      →  %d trackable file(s)\n", len(files))
	fmt.Println("")
}

func runReset() {
	if err := config.Delete(); err != nil {
		fmt.Printf("  ✗ Failed to reset: %s\n", err.Error())
		os.Exit(1)
	}
	fmt.Println("  ✓ Configuration removed. Run: anwin-agent setup")
}

func writeReadme(watchPath string) {
	readmePath := filepath.Join(watchPath, "ANWIN-AGENT.md")
	if _, err := os.Stat(readmePath); err == nil {
		return
	}
	content := `# ANWIN Agent

This directory is synced with ANWIN for code intelligence.

## What does the agent do?
- Monitors this directory for file changes
- Sends new and modified files to ANWIN for parsing and analysis
- Notifies ANWIN when files are deleted
- Keeps your code graph and compliance data up to date

## Commands
- ` + "`anwin-agent start`" + `  — Start syncing
- ` + "`anwin-agent status`" + ` — Check connection
- ` + "`anwin-agent reset`" + `  — Remove configuration

## Ignored
The agent ignores: .git, node_modules, vendor, build, dist, __pycache__, .idea, .vscode, target, bin, obj, .next, .nuxt, coverage, .terraform
`
	_ = os.WriteFile(readmePath, []byte(content), 0644)
}

func timestamp() string {
	return time.Now().Format("15:04:05")
}
