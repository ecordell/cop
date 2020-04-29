package login

import (
	"fmt"
	"github.com/zalando/go-keyring"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

const (
	jiraUserService = "jirauser"
	jiraPassService = "jirapass"
)

var JiraLoginCmd = &cobra.Command{
	Use:   "jira",
	Short: "jira login",
	Long:  `log in to jira`,
	RunE: func(cmd *cobra.Command, args []string) error {
		prompt := promptui.Prompt{
			Label: "Username: ",
		}

		username, err := prompt.Run()
		if err != nil {
			return fmt.Errorf("Failed to get user: %v\n", err)
		}
		if err := keyring.Set(jiraUserService, user, username); err != nil {
			return err
		}

		prompt = promptui.Prompt{
			Label: "Password: ",
			Mask: '*',
		}

		pass, err := prompt.Run()
		if err != nil {
			return fmt.Errorf("Failed to get password: %v\n", err)
		}
		if err := keyring.Set(jiraPassService, user, pass); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	LoginCmd.AddCommand(BugzillaLoginCmd)
}