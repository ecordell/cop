package jira

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"

	"github.com/juju/persistent-cookiejar"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"gopkg.in/andygrunwald/go-jira.v1"
)

// TODO: this is mostly a POC of authenticating to jira and needs a lot of cleanup

// homeDir returns the OS-specific home path as specified in the environment.
func homeDir() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("HOMEDRIVE"), os.Getenv("HOMEPATH"))
	}
	return os.Getenv("HOME")
}

func NewClient(username, password string) (*jira.Client, error) {
	jar, err := cookiejar.New(&cookiejar.Options{
		Filename:              filepath.Join(homeDir(), ".olmcop-cookies"),
		PersistSessionCookies: true,
	})
	if err != nil {
		return nil, err
	}
	client := &http.Client{
		Jar: jar,
	}

	jiraclient, err := jira.NewClient(client, "https://issues.redhat.com")
	if err != nil {
		return nil, err
	}

	_, _, err = jiraclient.Project.Get("OLM")
	if err == nil {
		fmt.Println("Already authenticated")
		return jiraclient, nil
	}

	resp, err := client.Get("https://issues.redhat.com/login.jsp?os_destination=%2Fdefault.jsp")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("could not reach jira (%d)", resp.StatusCode)
	}

	samlRequest := getSAMLRequest(resp)
	if samlRequest == "" {
		return nil, fmt.Errorf("could not get saml request")
	}
	if err := resp.Body.Close(); err != nil {
		return nil, err
	}

	formData := url.Values{"SAMLRequest": {samlRequest}}
	resp, err = client.PostForm("https://sso.redhat.com/auth/realms/redhat-external/protocol/saml", formData)
	if err != nil {
		return nil, err
	}
	loginURL := getFormURL(resp)
	if samlRequest == "" {
		return nil, fmt.Errorf("could not get login url")
	}
	if err := resp.Body.Close(); err != nil {
		return nil, err
	}

	loginData := url.Values{"username": {username}, "password": {password}}
	resp, err = client.PostForm(loginURL, loginData)
	if err != nil {
		return nil, err
	}

	// tokenizer misses the input obj for this response for some reason, parse the whole doc
	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, err
	}
	samlResp := getSAMLResponse(doc)
	if err := resp.Body.Close(); err != nil {
		return nil, err
	}

	samlRespFormData := url.Values{"SAMLResponse": {samlResp}}
	resp, err = client.PostForm("https://sso.jboss.org/login?provider=RedHatExternalProvider", samlRespFormData)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("could not login with saml response (%d)", resp.StatusCode)
	}
	if err := jar.Save(); err != nil {
		return nil, err
	}
	return jiraclient, nil
}

func getSAMLRequest(response *http.Response) string {
	doc := html.NewTokenizer(response.Body)
	for tokenType := doc.Next(); tokenType != html.ErrorToken; {
		token := doc.Token()

		if tokenType == html.StartTagToken {
			if token.DataAtom != atom.Textarea {
				tokenType = doc.Next()
				continue
			}
			for _, attr := range token.Attr {
				if attr.Key == "name" && attr.Val == "SAMLRequest" {
					if doc.Next() == html.TextToken {
						return doc.Token().String()
					}
				}
			}
		}
		tokenType = doc.Next()
	}
	return ""
}

func getSAMLResponse(n *html.Node) string {
	if n.Type == html.ElementNode && n.DataAtom == atom.Input {
		isSamlInput := false
		for _, attr := range n.Attr {
			if attr.Key == "name" && attr.Val == "SAMLResponse" {
				isSamlInput = true
			}
		}
		if !isSamlInput {
			return ""
		}
		for _, attr := range n.Attr {
			if attr.Key == "value" {
				return attr.Val
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := getSAMLResponse(c); found != "" {
			return found
		}
	}

	return ""
}

func getFormURL(response *http.Response) string {
	doc := html.NewTokenizer(response.Body)

	for tokenType := doc.Next(); tokenType != html.ErrorToken; {
		token := doc.Token()

		if tokenType == html.StartTagToken {
			if token.DataAtom != atom.Form {
				tokenType = doc.Next()
				continue
			}
			for _, attr := range token.Attr {
				if attr.Key == "action" {
					return attr.Val
				}
			}
		}
		tokenType = doc.Next()
	}
	return ""
}
