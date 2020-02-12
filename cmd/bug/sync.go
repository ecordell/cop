package bug

import (
	"fmt"
	"gopkg.in/andygrunwald/go-jira.v1"
	"strconv"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/ecordell/cop/pkg/bugzilla"
	jiraclient "github.com/ecordell/cop/pkg/jira"
)

type syncOptions struct {
}

var syncOpts syncOptions

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync bug between bz and jira",
	Long: `Sync bug between bz and jira`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if bugOpts.debug {
			logrus.SetLevel(logrus.DebugLevel)
		}
		var bzId int
		var err error
		if len(args) == 1 {
			bzId, err = strconv.Atoi(args[0])

			// right now, we only know about bz ids
			if err != nil {
				return err
			}
		}

		endpoint := "https://bugzilla.redhat.com/"
		c := bugzilla.NewClient(func() []byte {
			return []byte(bugOpts.apiKey)
		}, endpoint)

		client, err := jiraclient.NewClient(bugOpts.jiraUser, bugOpts.jiraPass)
		if err != nil {
			return err
		}

		// TODO check BZ API key - api returns an error if wrong
		fmt.Printf("Checking BZ %d\n", bzId)
		bs, err := c.GetJiraIssueForBug(bzId)
		if err != nil {
			return err
		}

		return printBug(bs[0].ExternalBugID, client)
	},
}

func printBug(jiraIssueId string, client *jira.Client) error {
	fmt.Printf("JIRA issue link found: %s\n", jiraIssueId)
	issue, _, err := client.Issue.Get(jiraIssueId, nil)
	if err != nil {
		return err
	}
	fmt.Printf("%s: %+v\n", issue.Key, issue.Fields.Summary)
	fmt.Printf("Type: %s\n", issue.Fields.Type.Name)
	fmt.Printf("Priority: %s\n", issue.Fields.Priority.Name)
	return nil
}

func init() {
	BugCmd.AddCommand(syncCmd)
}
