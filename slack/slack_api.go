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
func (a SLACKAPI) PostMessage(ctx context.Context, app string, tag string, eventAttributes *datadogV2.EventAttributes) (bool, error) {
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

	text := ":failure-error-cross-x: :canary: *" + strings.ToUpper("echo") + "* release `" + "main-20251017-181807-e7e818488e" + "` is failing Datadog Canary\n:datadog: Triggered Monitor `" + monitor_name + "`\n"
	textBlock := slack.NewTextBlockObject("mrkdwn", text, false, false)

	opts := []slack.MsgOption{
		slack.MsgOptionText(text, false),
		slack.MsgOptionAsUser(true),
		slack.MsgOptionBlocks(slack.NewSectionBlock(textBlock, nil, nil), block),
	}

	respChannelID, timestamp, err := a.api.PostMessage(
		"C02EKA9SB",
		opts...,
	)
	if err != nil {
		slack_log.Error(err, "Error when calling `PostMessage`\n", "error", err, "respChannelID", respChannelID)
		return false, err
	}
	slack_log.Info("Slack message successfully sent to channel #eng-fredbottest", "respChannelID", respChannelID, "app", app, "tag", tag, "timestamp", timestamp)

	return false, nil
}
