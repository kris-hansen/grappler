package main

import (
	"fmt"
	"os"

	"github.com/krish/grappler/internal/cli"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "grappler",
		Short: "Grappler orchestrates multiple git worktree groups",
		Long:  `Grappler is a lightweight orchestration tool for running multiple git worktrees (backend + frontend pairs) simultaneously with port isolation.`,
	}

	rootCmd.AddCommand(cli.InitCmd())
	rootCmd.AddCommand(cli.StartCmd())
	rootCmd.AddCommand(cli.StopCmd())
	rootCmd.AddCommand(cli.StatusCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
