package cmd

import (
  "fmt"
  "github.com/ecordell/cop/cmd/bug"
  "os"

  "github.com/spf13/cobra"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
  Use:   "cop",
  Short: "tools for managing bugs and docs",
  Long: `A set of tools that can be used to manage bugs and docs for operator-framework.`,
}

func Execute() {
  RootCmd.AddCommand(bug.BugCmd)
  if err := RootCmd.Execute(); err != nil {
    fmt.Println(err)
    os.Exit(1)
  }
}
