package lib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"text/template"
)

type Client struct {
	SearchRepositoryURL   *url.URL
	TrendingRepositoryURL *url.URL
	HTTPClient            *http.Client
}

type Item struct {
	ID              int      `json:"id"`
	Name            string   `json:"name"`
	FullName        string   `json:"full_name"`
	URL             string   `json:"url"`
	HTMLURL         string   `json:"html_url"`
	CloneURL        string   `json:"clone_url"`
	Description     string   `json:"description"`
	StargazersCount int      `json:"stargazers_count,stars"`
	Stars           int      `json:"stars"`
	Watchers        int      `json:"watchers"`
	Topics          []string `json:"topics"`
	Language        string   `json:"language"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
	DataSource      string
}

type Result struct {
	Items []Item `json:"items"`
}

func (item *Item) GetRepositoryName() string {
	name := item.FullName
	if name == "" {
		url, err := url.Parse(item.URL)
		if err != nil {
			return ""
		}
		name = url.Path[1:]
	}
	return name
}

func (item *Item) GetRepositoryURL() string {
	url := item.HTMLURL
	if url == "" {
		return item.URL
	}
	return url
}
func (item *Item) GetCloneURL() string {
	url := item.GetRepositoryURL()
	if !strings.HasSuffix(url, ".git") {
		return url + ".git"
	}
	return url
}

func (item *Item) String() string {
	const officialTemplateText = `
	Name: {{.GetRepositoryName}}
	URL: {{.GetRepositoryURL}}
	Star: {{.StargazersCount}}
	Clone URL: {{.GetCloneURL}}
	Description: {{.Description}}
	Watchers: {{.Watchers}}
	Topics: {{.Topics}}
	Language: {{.Language}}
	CreatedAt: {{.CreatedAt}}
	UpdatedAt: {{.UpdatedAt}}
	`
	const trendingTemplateText = `
	Name: {{.GetRepositoryName}}
	URL: {{.GetRepositoryURL}}
	Star: {{.Stars}}
	Clone URL: {{.GetCloneURL}}
	Description: {{.Description}}
	Language: {{.Language}}
	`
	templateText := trendingTemplateText
	if item.DataSource == "OfficialAPI" {
		templateText = officialTemplateText
	}
	template, err := template.New("Repository").Parse(templateText)
	if err != nil {
		panic(err)
	}
	var doc bytes.Buffer
	if err := template.Execute(&doc, item); err != nil {
		panic(err)
	}
	return doc.String()
}

func (result *Result) Draw(writer io.Writer) error {
	for _, item := range result.Items {
		fmt.Fprintf(writer, " %s\n", item.GetRepositoryName())
	}
	return nil
}

func NewClient() (*Client, error) {
	searchRepositoryURL, err := url.Parse("https://api.github.com/search/repositories")
	if err != nil {
		return nil, err
	}
	trendingRepositoryURL, err := url.Parse("https://github-trending-api.now.sh/repositories")
	if err != nil {
		return nil, err
	}
	return &Client{
		SearchRepositoryURL:   searchRepositoryURL,
		TrendingRepositoryURL: trendingRepositoryURL,
		HTTPClient:            http.DefaultClient,
	}, nil
}

func (client *Client) SearchRepository(query string) (*Result, error) {
	req, err := http.NewRequest("GET", client.SearchRepositoryURL.String()+"?q="+query, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept", "application/vnd.github.mercy-preview+json")
	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var result *Result
	if err = json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	items := result.Items
	for i := range items {
		result.Items[i].DataSource = "OfficialAPI"
	}
	return result, nil
}

func (client *Client) GetTrendingRepository(language string, since string) (*Result, error) {
	q := client.TrendingRepositoryURL.Query()
	if language != "" {
		q.Set("language", language)
	}
	if since != "" {
		q.Set("since", since)
	}
	url := client.TrendingRepositoryURL
	if len(q) != 0 {
		url.RawQuery = q.Encode()
	}
	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept", "application/vnd.github.mercy-preview+json")
	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var items []Item
	if err = json.Unmarshal(body, &items); err != nil {
		return nil, err
	}
	for i := range items {
		items[i].DataSource = "TrendingAPI"
	}
	return &Result{
		Items: items,
	}, nil
}