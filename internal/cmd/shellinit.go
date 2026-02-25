package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

const shellFunction = `wt() {
  if [ "$1" = "cd" ]; then
    shift
    local dir
    dir="$(command wt cd "$@")"
    if [ $? -eq 0 ] && [ -n "$dir" ]; then
      cd "$dir"
    else
      return 1
    fi
  elif [ "$1" = "remove" ]; then
    command wt "$@" || return $?
    local dir
    dir="$(command wt cd 2>/dev/null)"
    if [ $? -eq 0 ] && [ -n "$dir" ]; then
      cd "$dir"
    fi
  elif [ "$1" = "create" ] || [ "$1" = "pr" ]; then
    local name="$2"
    command wt "$@" || return $?
    local dir
    dir="$(command wt cd "$name")"
    if [ $? -eq 0 ] && [ -n "$dir" ]; then
      cd "$dir"
    fi
  else
    command wt "$@"
  fi
}`

func ShellInitCmd(d *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "shell-init",
		Short: "Output shell function for wt cd integration",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(d.Stdout, shellFunction)
		},
	}
}
