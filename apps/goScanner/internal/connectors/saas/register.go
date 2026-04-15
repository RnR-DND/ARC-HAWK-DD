package saas

import "github.com/arc-platform/go-scanner/internal/connectors"

func init() {
	connectors.Register("slack", func() connectors.Connector { return &SlackConnector{} })
	connectors.Register("jira", func() connectors.Connector { return &JiraConnector{} })
	connectors.Register("salesforce", func() connectors.Connector { return &SalesforceConnector{} })
	connectors.Register("hubspot", func() connectors.Connector { return &HubSpotConnector{} })
	connectors.Register("ms_teams", func() connectors.Connector { return &TeamsConnector{} })
}
