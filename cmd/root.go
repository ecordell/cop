package cmd

import (
  "fmt"
  "github.com/ecordell/cop/cmd/bug"
  "github.com/ecordell/cop/cmd/login"
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
  RootCmd.AddCommand(login.LoginCmd)
  if err := RootCmd.Execute(); err != nil {
    fmt.Println(err)
    os.Exit(1)
  }
}
