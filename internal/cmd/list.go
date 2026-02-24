package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func ListCmd(d *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all worktrees",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, _, err := d.Config.LoadConfig()
			if err != nil {
				return err
			}
			output, err := d.Git.GitInBare("worktree list", root)
			if err != nil {
				return err
			}
			fmt.Fprintln(d.Stdout, output)
			return nil
		},
	}
}
