package saas

import (
	"context"
	"fmt"

	"github.com/arc-platform/go-scanner/internal/connectors"
	"github.com/slack-go/slack"
)

// SlackConnector scans Slack workspace messages via the Slack API.
// Config keys: token (Bot User OAuth Token, xoxb-...)
type SlackConnector struct {
	client *slack.Client
}

func (c *SlackConnector) SourceType() string { return "slack" }

func (c *SlackConnector) Connect(_ context.Context, cfg map[string]any) error {
	token := fmt.Sprintf("%v", cfg["token"])
	if token == "" || token == "<nil>" {
		return fmt.Errorf("slack: token is required (xoxb-...)")
	}
	c.client = slack.New(token)
	return nil
}

func (c *SlackConnector) Close() error { return nil }

func (c *SlackConnector) StreamFields(ctx context.Context) (<-chan connectors.FieldRecord, <-chan error) {
	out := make(chan connectors.FieldRecord, 100)
	errc := make(chan error, 1)
	go func() {
		defer close(out)
		defer close(errc)

		channels, _, err := c.client.GetConversationsContext(ctx, &slack.GetConversationsParameters{
			Types: []string{"public_channel", "private_channel"},
			Limit: 200,
		})
		if err != nil {
			errc <- fmt.Errorf("slack: list channels: %w", err)
			return
		}

		for _, ch := range channels {
			params := &slack.GetConversationHistoryParameters{
				ChannelID: ch.ID,
				Limit:     1000,
			}
			history, err := c.client.GetConversationHistoryContext(ctx, params)
			if err != nil {
				continue
			}
			for _, msg := range history.Messages {
				if msg.Text == "" {
					continue
				}
				select {
				case out <- connectors.FieldRecord{
					Value:        msg.Text,
					FieldName:    "message",
					SourcePath:   fmt.Sprintf("slack://%s/%s", ch.Name, msg.Timestamp),
					IsStructured: false,
				}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out, errc
}
