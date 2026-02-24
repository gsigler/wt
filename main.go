package main

import (
	"os"

	"github.com/gradyholmes/wt/internal/cmd"
	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "wt",
		Short: "Git worktree CLI for bare repo workflows",
	}

	d := cmd.DefaultDeps()

	root.AddCommand(cmd.InitCmd(d))
	root.AddCommand(cmd.CreateCmd(d))
	root.AddCommand(cmd.ListCmd(d))
	root.AddCommand(cmd.PrCmd(d))
	root.AddCommand(cmd.RemoveCmd(d))
	root.AddCommand(cmd.CdCmd(d))
	root.AddCommand(cmd.ShellInitCmd(d))
	root.AddCommand(cmd.PruneCmd(d))

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
