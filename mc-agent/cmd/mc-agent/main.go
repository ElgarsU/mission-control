package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"mission-control/mc-agent/internal/daemon"
	"mission-control/mc-agent/internal/tmux"
)

var rootCmd = &cobra.Command{
	Use:   "mc-agent",
	Short: "Mission Control agent — manages Claude Code sessions on this machine",
	Long: `mc-agent is the local daemon for Mission Control.

It manages tmux sessions running Claude Code, connects to the relay server
on the VPS via WebSocket, and exposes local frontends (menu bar, TUI).

Run 'mc-agent start' to launch the daemon.
Run 'mc-agent tui' for the interactive terminal UI.`,
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the mc-agent daemon",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("starting daemon...")

		// Set up tmux manager and daemon
		mgr := tmux.NewManager()
		adapter := daemon.NewTmuxAdapter(mgr)
		d := daemon.New(adapter)

		// Sync existing sessions
		if err := d.Sync(); err != nil {
			log.Printf("warning: sync existing sessions: %v", err)
		}

		sessions := d.ListSessions()
		if len(sessions) > 0 {
			fmt.Printf("discovered %d existing session(s)\n", len(sessions))
		}

		fmt.Println("daemon running. press ctrl+c to stop.")

		// Wait for shutdown signal
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig

		fmt.Println("\nshutting down...")
	},
}

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the interactive terminal UI",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("starting TUI...")
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(tuiCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
