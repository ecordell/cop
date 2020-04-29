package bug

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/jinzhu/copier"
	"github.com/manifoldco/promptui"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"

	"github.com/ecordell/cop/pkg/bugzilla"
)

const (
	service = "bugzilla"
	user = "io.olm.cop"
)

func init() {
	BugCmd.AddCommand(backportCmd)
	backportCmd.PersistentFlags().StringSliceVarP(&backportOpts.targetVersions,"versions", "v", []string{"4.5.0"}, "target versions to query")
}

type backportOptions struct {
	targetVersions []string
	client         bugzilla.Client
}

var backportOpts backportOptions

// TODO: check for backport labels, ensure links
// alert on anything that doesn't have them or doesn't match expected

const baseQuery =  "bug_status=NEW&bug_status=ASSIGNED&bug_status=POST&bug_status=MODIFIED&bug_status=ON_DEV&bug_status=ON_QA&classification=Red Hat&component=OLM&product=OpenShift Container Platform&query_format=advanced"

var backportCmd = &cobra.Command{
	Use:   "backport",
	Short: "Check backport status",
	Long: `Check backport status`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if bugOpts.debug {
			logrus.SetLevel(logrus.DebugLevel)
		}
		var err error

		apikey, err := keyring.Get(service, user)
		if err != nil && !errors.Is(err, keyring.ErrNotFound) {
			return err
		}
		if apikey == "" {
			apikey = bugOpts.apiKey
		}
		if bugOpts.apiKey != "" {
			if err := keyring.Set(service, user, bugOpts.apiKey); err != nil {
				return err
			}
		}
		if apikey == "" {
			return fmt.Errorf("must provide apikey or login with `cop login bugzilla`")
		}

		endpoint := "https://bugzilla.redhat.com/"
		backportOpts.client = bugzilla.NewClient(func() []byte {
			return []byte(apikey)
		}, endpoint)

		// TODO check BZ API key - api returns an error if wrong
		query := baseQuery
		for _, v := range backportOpts.targetVersions {
			query = query+"&target_release="+v
		}
		bs, err := backportOpts.client.SearchBugs(query)
		if err != nil {
			return err
		}

		sbs := []CLIMarshaller{}
		for _, bug := range bs {
			sbs = append(sbs, NewSimpleBugView(*bug))
		}
		return NewMultiSelectView(sbs).Prompt()
	},
}

const cliTag = "cli"

type CLIMarshaller interface {
	MarshallCLI() ([]string, error)
}

type SimpleBugView struct {
	bugzilla.Bug
	// ID is the unique numeric ID of this bug.
	ID int `cli:"ID"`
	// Status is the current status of the bug.
	Status string `cli:"Status"`
	// AssignedTo is the login name of the user to whom the bug is assigned.
	AssignedTo string `cli:"Assignee"`
	// Summary is the summary of this bug.
	Summary string `cli:"Summary,50"`
	// Priority is the priority of the bug.
	Priority string `cli:"Priority"`
	// Severity is the current severity of the bug.
	Severity string `cli:"Severity"`
	// Backport is desired backport version of the bug, derived from the internal whiteboard
	Backport string `cli:"Backport"`
	// InternalWhiteboard is used for internal team notes
	InternalWhiteboard string
}

func NewSimpleBugView(bug bugzilla.Bug) *SimpleBugView {
	backport := "⚠️"
	whiteboard := strings.Split(bug.InternalWhiteboard, ":")
	if len(whiteboard) > 1 {
		backport = strings.Trim(whiteboard[1], " ")
	}
	view := &SimpleBugView{
		Bug: bug,
		Backport: backport,
	}
	if err := copier.Copy(&view, bug); err != nil {
		logrus.Error(err)
	}
	return view
}

func (b SimpleBugView) MarshallCLI() ([]string, error) {
	values := []string{}
	var err error
	val := reflect.ValueOf(b)
	for i := 0; i < val.Type().NumField(); i++ {
		tag := val.Type().Field(i).Tag.Get(cliTag)

		var maxLen int
		parts := strings.Split(tag, ",")
		if len(parts) > 2 {
			return nil, fmt.Errorf("too many parts to field tag %s", tag)
		}
		if len(parts) == 2 {
			maxLen, err = strconv.Atoi(parts[1])
			if err != nil {
				return nil, fmt.Errorf("couldn't get length from struct tag %s: %v", tag, err)
			}
		}

		// Skip if tag is not defined or ignored
		if tag == "" || tag == "-" {
			continue
		}

		if maxLen == 0 {
			values = append(values, fmt.Sprintf("%v", val.Field(i).Interface()))
			continue
		}

		// if maxlen, assume string
		if maxLen > len(val.Field(i).String()) {
			maxLen = len(val.Field(i).String())
		}

		values = append(values, val.Field(i).String()[:maxLen])
	}
	return values, err
}

func Fields(b CLIMarshaller) ([]string) {
	fields := []string{}

	val := reflect.ValueOf(b).Elem()
	for i := 0; i < val.Type().NumField(); i++ {
		tag := val.Type().Field(i).Tag.Get(cliTag)

		// Skip if tag is not defined or ignored
		if tag == "" || tag == "-" {
			continue
		}
		parts := strings.Split(tag, ",")
		fields = append(fields, parts[0])
	}
	return fields
}

var _ CLIMarshaller = &SimpleBugView{}

func TabHeader(s []string) string {
	return "  " + strings.Join(s, "\t  ") + "\n"
}

func TabLine(s []string) string {
	return strings.Join(s, "\t") + "\n"
}

type MultiSelectView struct {
	options []CLIMarshaller
}

func NewMultiSelectView(options []CLIMarshaller) *MultiSelectView {
	return &MultiSelectView{
		options: options,
	}
}

func (v *MultiSelectView) Prompt() error {
	var buffer bytes.Buffer
	w := tabwriter.NewWriter(&buffer, 0, 0, 2, ' ', 0)
	if len(v.options) == 0 {
		return fmt.Errorf("can't prompt with no options")
	}
	if _, err := fmt.Fprint(w, TabHeader(Fields(v.options[0]))); err != nil {
		return err
	}
	for _, o := range v.options {
		values, err := o.MarshallCLI()
		if err != nil {
			return err
		}
		if _, err := fmt.Fprint(w, TabLine(values)); err != nil {
			logrus.Error(err)
		}
	}
	if err := w.Flush(); err != nil {
		return err
	}
	lines := strings.Split(buffer.String(), "\n")
	prompt := promptui.Select{
		Label: lines[0],
		Size: 10,
		Items: lines[1:],
		HideSelected: true,
	}

	row, _, err := prompt.Run()
	if err != nil {
		return err
	}
	fmt.Println(row)
	return nil
}

func bugPrompt(bug *bugzilla.Bug, bugs []*bugzilla.Bug) error {
	options := []string{
		"View Bug on Bugzilla",
		"Set Backport Version",
	}

	prompt := promptui.Select{
		Label: "Select Action",
		Items: options,
	}
	actions := []func() error {
		func() error {
			openbrowser(fmt.Sprintf("https://bugzilla.redhat.com/show_bug.cgi?id=%d",bug.ID))
			return bugPrompt(bug, bugs)
		},
		func() error {
			return backportSelect(bug, bugs)
		},
	}
	i, _, err := prompt.Run()
	if err != nil {
		return err
	}
	if i < len(actions) {
		return actions[i]()
	}
	fmt.Printf("invalid selection")
	//printBZs(bugs)
	return nil
}

func backportSelect(bug *bugzilla.Bug, bugs []*bugzilla.Bug) error {
	options := []string{
		"4.1",
		"4.2",
		"4.3",
		"4.4",
		"4.5",
	}
	prompt := promptui.Select{
		Label: "Backport To: ",
		Items: options,
	}
	_, to, err := prompt.Run()
	if err != nil {
		return err
	}

	//fmt.Print(to)
	_, err = backportOpts.client.UpdateInternalWhiteboard(bug.ID, "backport-to: "+to)
	if err != nil {
		return err
	}
	fmt.Println("Updated whiteboard with selection.")
	//printBZs(bugs)
	return nil
}

// have not tested outside of macos
func openbrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Fatal(err)
	}
}