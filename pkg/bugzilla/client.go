package bugzilla

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

type Client interface {
	Endpoint() string
	GetBug(id int) (*Bug, error)
	GetExternalBugPRsOnBug(id int) ([]GithubExternalBug, error)
	GetJiraIssueForBug(id int) ([]JiraExternalBug, error)
	//UpdateBug(id int, update BugUpdate) error
	//AddPullRequestAsExternalBug(id int, org, repo string, num int) (bool, error)
}

func NewClient(getAPIKey func() []byte, endpoint string) Client {
	return &client{
		logger:    logrus.WithField("client", "bugzilla"),
		client:    &http.Client{},
		endpoint:  endpoint,
		getAPIKey: getAPIKey,
	}
}

type client struct {
	logger    *logrus.Entry
	client    *http.Client
	endpoint  string
	getAPIKey func() []byte
}

// the client is a Client impl
var _ Client = &client{}

func (c *client) request(req *http.Request, logger *logrus.Entry) ([]byte, error) {
	if apiKey := c.getAPIKey(); len(apiKey) > 0 {
		// some BugZilla servers are too old and can't handle the header.
		// some don't want the query parameter. We can set both and keep
		// everyone happy without negotiating on versions
		req.Header.Set("X-BUGZILLA-API-KEY", string(apiKey))
		values := req.URL.Query()
		values.Add("api_key", string(apiKey))
		req.URL.RawQuery = values.Encode()
	}
	resp, err := c.client.Do(req)
	if resp != nil {
		logger.WithField("response", resp.StatusCode).Debug("Got response from Bugzilla.")
	}
	if err != nil {
		code := -1
		if resp != nil {
			code = resp.StatusCode
		}
		fmt.Printf("%#v", resp)
		return nil, &requestError{statusCode: code, message: err.Error()}
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.WithError(err).Warn("could not close response body")
		}
	}()
	if resp.StatusCode != http.StatusOK {
		return nil, &requestError{statusCode: resp.StatusCode, message: fmt.Sprintf("response code %d not %d", resp.StatusCode, http.StatusOK)}
	}
	raw, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read response body: %v", err)
	}
	return raw, nil
}

func (c *client) Endpoint() string {
	return c.endpoint
}

// GetBug retrieves a Bug from the server
// https://bugzilla.readthedocs.io/en/latest/api/core/v1/bug.html#get-bug
func (c *client) GetBug(id int) (*Bug, error) {
	logger := c.logger.WithFields(logrus.Fields{"method": "GetBug", "id": id})
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/rest/bug/%d", c.endpoint, id), nil)
	if err != nil {
		return nil, err
	}
	raw, err := c.request(req, logger)
	if err != nil {
		return nil, err
	}
	fmt.Println(string(raw))
	var parsedResponse struct {
		Bugs []*Bug `json:"bugs,omitempty"`
	}
	if err := json.Unmarshal(raw, &parsedResponse); err != nil {
		return nil, fmt.Errorf("could not unmarshal response body: %v", err)
	}
	if len(parsedResponse.Bugs) != 1 {
		return nil, fmt.Errorf("did not get one bug, but %d: %v", len(parsedResponse.Bugs), parsedResponse)
	}
	return parsedResponse.Bugs[0], nil
}

// GetJiraIssueForBug retrieves external bugs on a Bug from the server
// and returns any that reference a Jira issue
// https://bugzilla.readthedocs.io/en/latest/api/core/v1/bug.html#get-bug
func (c *client) GetJiraIssueForBug(id int) ([]JiraExternalBug, error) {
	logger := c.logger.WithFields(logrus.Fields{"method": "GetJiraIssueForBug", "id": id})
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/rest/bug/%d", c.endpoint, id), nil)
	if err != nil {
		return nil, err
	}
	values := req.URL.Query()
	values.Add("include_fields", "external_bugs")
	req.URL.RawQuery = values.Encode()
	raw, err := c.request(req, logger)
	if err != nil {
		return nil, err
	}
	var parsedResponse struct {
		Bugs []struct {
			ExternalBugs []ExternalBug `json:"external_bugs"`
		} `json:"bugs"`
	}
	if err := json.Unmarshal(raw, &parsedResponse); err != nil {
		return nil, fmt.Errorf("could not unmarshal response body: %v", err)
	}
	if len(parsedResponse.Bugs) != 1 {
		return nil, fmt.Errorf("did not get one bug, but %d: %v", len(parsedResponse.Bugs), parsedResponse)
	}
	var prs []JiraExternalBug
	for _, bug := range parsedResponse.Bugs[0].ExternalBugs {
		if bug.BugzillaBugID != id {
			continue
		}
		if bug.Type.URL != "https://jira.coreos.com/" && bug.Type.URL != "https://issues.redhat.com/" {
			continue
		}
		prs = append(prs, JiraExternalBug{ExternalBug: bug})
	}
	return prs, nil
}


// GetExternalBugPRsOnBug retrieves external bugs on a Bug from the server
// and returns any that reference a Pull Request in GitHub
// https://bugzilla.readthedocs.io/en/latest/api/core/v1/bug.html#get-bug
func (c *client) GetExternalBugPRsOnBug(id int) ([]GithubExternalBug, error) {
	logger := c.logger.WithFields(logrus.Fields{"method": "GetExternalBugPRsOnBug", "id": id})
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/rest/bug/%d", c.endpoint, id), nil)
	if err != nil {
		return nil, err
	}
	values := req.URL.Query()
	values.Add("include_fields", "external_bugs")
	req.URL.RawQuery = values.Encode()
	raw, err := c.request(req, logger)
	if err != nil {
		return nil, err
	}
	var parsedResponse struct {
		Bugs []struct {
			ExternalBugs []ExternalBug `json:"external_bugs"`
		} `json:"bugs"`
	}
	if err := json.Unmarshal(raw, &parsedResponse); err != nil {
		return nil, fmt.Errorf("could not unmarshal response body: %v", err)
	}
	if len(parsedResponse.Bugs) != 1 {
		return nil, fmt.Errorf("did not get one bug, but %d: %v", len(parsedResponse.Bugs), parsedResponse)
	}
	var prs []GithubExternalBug
	for _, bug := range parsedResponse.Bugs[0].ExternalBugs {
		if bug.BugzillaBugID != id {
			continue
		}
		if bug.Type.URL != "https://github.com/" {
			continue
		}
		org, repo, num, err := PullFromIdentifier(bug.ExternalBugID)
		if IsIdentifierNotForPullErr(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("could not parse external identifier %q as pull: %v", bug.ExternalBugID, err)
		}
		prs = append(prs, NewGithubExternalBug(bug, org, repo, num))
	}
	return prs, nil
}

func PullFromIdentifier(identifier string) (org, repo string, num int, err error) {
	parts := strings.Split(identifier, "/")
	if len(parts) != 4 {
		return "", "", 0, fmt.Errorf("invalid pull identifier with %d parts: %q", len(parts), identifier)
	}
	if parts[2] != "pull" {
		return "", "", 0, &identifierNotForPull{identifier: identifier}
	}
	number, err := strconv.Atoi(parts[3])
	if err != nil {
		return "", "", 0, fmt.Errorf("invalid pull identifier: could not parse %s as number: %v", parts[3], err)
	}

	return parts[0], parts[1], number, nil
}




