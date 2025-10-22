package slack

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	slack "github.com/slack-go/slack"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	slack_log = logf.Log.WithName("slack_api_alerts")
)

type SlackAPI interface {
	PostMessage(channelID string, options ...slack.MsgOption) (string, string, error)
	GetConversationHistoryContext(ctx context.Context, params *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error)
}

type SLACKAPI struct {
	api SlackAPI
}

func NewSlackAPI() (*SLACKAPI, error) {
	slack_log.Info("Creating Slack API")

	token := os.Getenv("SLACK_TOKEN")
	if token == "" {
		slack_log.Info("Error when calling `NewSlackAPI` - token empty \n")
		return nil, nil
	}

	return &SLACKAPI{slack.New(token)}, nil
}

func InjectSlackAPI(a SlackAPI) *SLACKAPI {
	return &SLACKAPI{a}
}

// test channel #eng-fredbottest: C02EKA9SB
// if message already present, do not post again
func (a SLACKAPI) PostMessage(ctx context.Context, app string, tag string, eventAttributes *datadogV2.EventAttributes) error {
	monitor_name := *eventAttributes.Monitor.Get().Name
	monitor_id := *eventAttributes.Monitor.Get().Id

	iris_url := fmt.Sprintf("https://iris.medium.build/servers/%s/revisions/%s?target=production", app, tag)
	dataog_url := "https://app.datadoghq.com/monitors/" + fmt.Sprint(monitor_id) + "?q=version%3A" + tag

	elements := make([]slack.BlockElement, 0, 3)
	addLink := func(label, link string) {
		textBlock := slack.NewTextBlockObject("plain_text", label, true, false)
		button := slack.NewButtonBlockElement("", label, textBlock)
		button.URL = link
		elements = append(elements, button)
	}
	addLink(":iris: Iris", iris_url)
	addLink(":datadog: Datadog Triggered Monitor", dataog_url)
	addLink(":notion: Canary Runbook", "")

	block := slack.NewActionBlock("useful_links", elements...)

	text := ":failure-error-cross-x: :canary: *" + strings.ToUpper(app) + "* release `" + "main-20251017-181807-e7e818488e" + "` is failing Datadog Canary\n:datadog: Triggered Monitor `" + monitor_name + "`\n"
	textBlock := slack.NewTextBlockObject("mrkdwn", text, false, false)

	opts := []slack.MsgOption{
		slack.MsgOptionText(text, false),
		slack.MsgOptionAsUser(true),
		slack.MsgOptionBlocks(slack.NewSectionBlock(textBlock, nil, nil), block),
	}

	// get channel history first to see if message already present
	params := slack.GetConversationHistoryParameters{
		ChannelID: "C02EKA9SB",
		Limit:     5,
	}
	messages, err := a.api.GetConversationHistoryContext(context.Background(), &params)
	if err != nil {
		slack_log.Error(err, "Error when calling `GetConversationHistoryContextostMessage`\n", "error", err, "messages", messages)
		return err
	}

	send := true
	for _, message := range messages.Messages {
		if strings.Contains(message.Text, ":failure-error-cross-x: :canary:") && strings.Contains(message.Text, tag) && strings.Contains(message.Text, strings.ToUpper(app)) {
			slack_log.Info("FOUND CANARY MESSAGE:\n", "error", err, "tag", tag, "app", app)
			send = false
			break
		}
	}

	// message not found, send it
	if send {
		respChannelID, timestamp, err := a.api.PostMessage(
			"C02EKA9SB",
			opts...,
		)
		if err != nil {
			slack_log.Error(err, "Error when calling `PostMessage`\n", "error", err, "respChannelID", respChannelID)
			return err
		}
		slack_log.Info("Slack message successfully sent to channel #eng-fredbottest", "respChannelID", respChannelID, "app", app, "tag", tag, "timestamp", timestamp)
		return nil
	}

	return nil
}
