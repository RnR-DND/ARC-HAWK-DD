package saas

import (
	"context"
	"fmt"
	"net/http"

	"github.com/andygrunwald/go-jira/v2/cloud"
	"github.com/arc-platform/go-scanner/internal/connectors"
)

// JiraConnector scans Jira Cloud issues and comments.
// Config keys: url (e.g. https://yoursite.atlassian.net), user (email), token (API token)
type JiraConnector struct {
	client *cloud.Client
}

func (c *JiraConnector) SourceType() string { return "jira" }

func (c *JiraConnector) Connect(_ context.Context, cfg map[string]any) error {
	url := fmt.Sprintf("%v", cfg["url"])
	if url == "" || url == "<nil>" {
		return fmt.Errorf("jira: url is required (e.g. https://yoursite.atlassian.net)")
	}
	user := fmt.Sprintf("%v", cfg["user"])
	token := fmt.Sprintf("%v", cfg["token"])
	if user == "" || token == "" || user == "<nil>" || token == "<nil>" {
		return fmt.Errorf("jira: user and token are required")
	}

	tp := cloud.BasicAuthTransport{
		Username: user,
		APIToken: token,
	}
	client, err := cloud.NewClient(url, &http.Client{Transport: &tp})
	if err != nil {
		return err
	}
	c.client = client
	return nil
}

func (c *JiraConnector) Close() error { return nil }

func (c *JiraConnector) StreamFields(ctx context.Context) (<-chan connectors.FieldRecord, <-chan error) {
	out := make(chan connectors.FieldRecord, 100)
	errc := make(chan error, 1)
	go func() {
		defer close(out)
		defer close(errc)

		startAt := 0
		maxResults := 100
		for {
			issues, _, err := c.client.Issue.Search(ctx,
				"ORDER BY created DESC",
				&cloud.SearchOptions{StartAt: startAt, MaxResults: maxResults, Fields: []string{"summary", "description", "comment"}})
			if err != nil {
				errc <- fmt.Errorf("jira: search issues: %w", err)
				return
			}
			if len(issues) == 0 {
				break
			}
			for _, issue := range issues {
				fields := issue.Fields
				if fields == nil {
					continue
				}
				if fields.Summary != "" {
					select {
					case out <- connectors.FieldRecord{
						Value:        fields.Summary,
						FieldName:    "summary",
						SourcePath:   fmt.Sprintf("jira://%s:summary", issue.Key),
						IsStructured: false,
					}:
					case <-ctx.Done():
						return
					}
				}
				if fields.Description != "" {
					select {
					case out <- connectors.FieldRecord{
						Value:        fields.Description,
						FieldName:    "description",
						SourcePath:   fmt.Sprintf("jira://%s:description", issue.Key),
						IsStructured: false,
					}:
					case <-ctx.Done():
						return
					}
				}
				if fields.Comments != nil {
					for _, comment := range fields.Comments.Comments {
						if comment.Body == "" {
							continue
						}
						select {
						case out <- connectors.FieldRecord{
							Value:        comment.Body,
							FieldName:    "comment",
							SourcePath:   fmt.Sprintf("jira://%s:comment/%s", issue.Key, comment.ID),
							IsStructured: false,
						}:
						case <-ctx.Done():
							return
						}
					}
				}
			}
			startAt += len(issues)
			if len(issues) < maxResults {
				break
			}
		}
	}()
	return out, errc
}
