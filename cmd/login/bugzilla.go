package login

import (
	"fmt"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

const (
	service = "bugzilla"
	user    = "io.olm.cop"
)

var BugzillaLoginCmd = &cobra.Command{
	Use:   "bugzilla",
	Short: "bugzilla login",
	Long:  `set the apikey for bugzilla`,
	RunE: func(cmd *cobra.Command, args []string) error {
		prompt := promptui.Prompt{
			Label: "API key: ",
			Mask:  '*',
		}

		key, err := prompt.Run()
		if err != nil {
			return fmt.Errorf("Failed to get key: %v\n", err)
		}
		if err := keyring.Set(service, user, key); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	LoginCmd.AddCommand(BugzillaLoginCmd)
}
