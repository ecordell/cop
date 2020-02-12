package jira

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"gopkg.in/andygrunwald/go-jira.v1"
)

func TestLogin(t *testing.T) {
	cookieJar, err := cookiejar.New(nil)
	require.NoError(t, err)
	client := &http.Client{
		Jar: cookieJar,
	}
	resp, err := client.Get("https://issues.redhat.com/login.jsp?os_destination=%2Fdefault.jsp")
	require.NoError(t, err)
	require.Equal(t, resp.StatusCode, http.StatusOK)
	samlToken := getSAMLRequest(resp)
	require.NotEqual(t, "", samlToken)
	require.NoError(t, resp.Body.Close())

	formData := url.Values{"SAMLRequest": {samlToken}}
	t.Logf("%s", formData.Encode())
	resp, err = client.PostForm("https://sso.redhat.com/auth/realms/redhat-external/protocol/saml", formData)
	require.NoError(t, err)
	loginURL := getFormURL(resp)
	t.Log(loginURL)
	require.NoError(t, resp.Body.Close())

	loginData := url.Values{"username": {os.Getenv("TEST_USER")}, "password": {os.Getenv("TEST_PASS")}}
	resp, err = client.PostForm(loginURL, loginData)
	require.NoError(t, err)

	// tokenizer misses the input obj for this response for some reason, parse the whole doc
	doc, err := html.Parse(resp.Body)
	require.NoError(t, err)
	samlResp := getSAMLResponse(doc)
	require.NoError(t, resp.Body.Close())

	samlRespFormData := url.Values{"SAMLResponse": {samlResp}}
	t.Logf("%s", samlRespFormData.Encode())
	resp, err = client.PostForm("https://sso.jboss.org/login?provider=RedHatExternalProvider", samlRespFormData)
	require.NoError(t, err)
	require.Equal(t, resp.StatusCode, http.StatusOK)

	jc, err := jira.NewClient(client, "https://issues.redhat.com")
	i, r, err := jc.Issue.Get("OLM-1378", &jira.GetQueryOptions{})
	require.NoError(t, err)
	t.Logf("%#v", i)
	t.Logf("%#v", i.Fields)
	t.Logf("%#v", r)
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
