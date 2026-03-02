package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
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
