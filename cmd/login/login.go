package login

import (
	"github.com/spf13/cobra"
)

type loginOptions struct {
	debug bool

	apiKey   string
	jiraUser string
	jiraPass string
}

var loginOpts loginOptions

var LoginCmd = &cobra.Command{
	Use:   "login",
	Short: "manage credentials for required services.",
	Long:  `manage credentials for required services.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}

func init() {
	LoginCmd.PersistentFlags().BoolVarP(&loginOpts.debug, "debug", "d", false, "enable debug logging")
}
