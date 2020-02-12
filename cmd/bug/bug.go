package bug

import (
	"github.com/spf13/cobra"
)

type bugOptions struct {
	debug bool

	apiKey string
	jiraUser string
	jiraPass string
}

var bugOpts bugOptions

var BugCmd = &cobra.Command{
	Use:   "bz",
	Short: "Manage bugs in Bugzilla",
	Long: `Manage bugs in Bugzilla`,
}

func init() {
	BugCmd.PersistentFlags().BoolVarP(&bugOpts.debug, "debug", "d", false, "enable debug logging")
	BugCmd.PersistentFlags().StringVarP(&bugOpts.apiKey, "bz-apikey", "k", "", "apikey for bugzilla")
	BugCmd.PersistentFlags().StringVarP(&bugOpts.jiraUser, "jira-user", "u", "", "username for jboss jira")
	BugCmd.PersistentFlags().StringVarP(&bugOpts.jiraPass, "jira-pass", "p", "", "password for jboss jira")
}
