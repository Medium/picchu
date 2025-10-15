package datadog

import (
	"context"
	"fmt"
	"os"

	slack "github.com/slack-go/slack"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	slack_log = logf.Log.WithName("slack_api_alerts")
)

type SlackAPI interface {
	PostMessage(channelID string, options ...slack.MsgOption) (string, string, error)
}

type SLACKAPI struct {
	api SlackAPI
}

func NewSlackAPI() (*SLACKAPI, error) {
	slack_log.Info("Creating Datadog Monitor API")

	token := os.Getenv("SLACK_BOT_TOKEN")
	if token == "" {
		fmt.Println("SLACK_BOT_TOKEN environment variable is required")
		// idk
		os.Exit(1)
	}

	return &SLACKAPI{slack.New(token)}, nil
}

func InjectSlackAPI(a SlackAPI) *SLACKAPI {
	return &SLACKAPI{a}
}

// test channel #eng-fredbottest: C02EKA9SB
func (a SLACKAPI) PostMessage(ctx context.Context, app string, tag string) (bool, error) {
	respChannelID, timestamp, err := a.api.PostMessage(
		"C02EKA9SB",
		slack.MsgOptionText("Canary is Failing for "+app+" - Picchu detected an issue with revision "+tag, false),
		slack.MsgOptionAsUser(true), // Add this if you want that the bot would post message as a user, otherwise it will send response using the default slackbot
	)
	if err != nil {
		slack_log.Error(err, "Error when calling `PostMessage`\n", "error", err, "respChannelID", respChannelID)
		return false, err
	}
	slack_log.Info("Slack message successfully sent to channel #eng-fredbottest", "respChannelID", respChannelID, "app", app, "tag", tag, "timestamp", timestamp)

	return false, nil
}
