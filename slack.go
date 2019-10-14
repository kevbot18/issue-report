package main

import (
	"bytes"
	"encoding/json"
	"net/http"
)

// SlackMessage contains the fields used to respond to the slack message.
// View slack documentation for which fields you need
type SlackMessage struct {
	Text        string            `json:"text"`
	Channel     string            `json:"channel,omitempty"`
	Response    string            `json:"response_type,omitempty"`
	Blocks      []SlackBlock      `json:"blocks,omitempty"`
	Attachments []SlackAttachment `json:"attachments,omitempty"`
	ThreadTS    string            `json:"thread_ts,omitempty"`
	Markdown    bool              `json:"mrkdwn,omitempty"`
}

// SlackBlock layout blocks for Slack's new messages
// A mess. Should refactor/separate each block to match slack documentation
// TODO: separate, implement validateSlackMessage for each type and use an interface
type SlackBlock struct {
	Type string `json:"type"`
	Text interface {
		validateSlackMessage() bool
	} `json:"text,omitempty"`
	Emoji      bool          `json:"emoji,omitempty"`
	Verbatil   bool          `json:"verbatim,omitempty"`
	BlockID    string        `json:"block_id,omitempty"`
	Fields     []*SlackBlock `json:"fields,omitempty"`
	Accessory  *SlackBlock   `json:"accessory,omitempty"`
	ImageURL   string        `json:"image_url,omitempty"`
	AltText    string        `json:"alt_text,omitempty"`
	Title      *SlackBlock   `json:"title,omitempty"`
	Elements   []*SlackBlock `json:"elements,omitempty"`
	Label      *SlackBlock   `json:"label,omitempty"`
	Hint       *SlackBlock   `json:"hint,omitempty"`
	Optional   bool          `json:"optional,omitempty"`
	ExternalID string        `json:"external_id,omitempty"`
	Source     string        `json:"source,omitempty"`
	ActionID   string        `json:"action_id,omitempty"`
	URL        string        `json:"url,omitempty"`
	Value      string        `json:"value,omitempty"`
	Style      string        `json:"style,omitempty"`
	Confirm    *SlackBlock   `json:"confirm,omitempty"`
}

func (block SlackBlock) validateSlackMessage() bool {
	return true
}

// SlackString just makes it so SlackBlock can cover all blocks for simplicity
type SlackString string

func (slackString SlackString) validateSlackMessage() bool {
	return true
}

// SlackAttachment attachment struct for Slack messages
// View slack documentation for more info
// https://api.slack.com/docs/message-attachments
type SlackAttachment struct {
	Fallback   string `json:"fallback"`
	Color      string `json:"color,omitempty"`
	Pretext    string `json:"pretext,omitempty"`
	AuthorName string `json:"author_name,omitempty"`
	AuthorLink string `json:"author_link,omitempty"`
	AuthorIcon string `json:"author_icon,omitempty"`
	Title      string `json:"title,omitempty"`
	TitleLink  string `json:"title_link,omitempty"`
	Text       string `json:"text,omitempty"`
	Fields     []struct {
		Title string `json:"title,omitempty"`
		Value string `json:"value,omitempty"`
		Short bool   `json:"short,omitempty"`
	} `json:"fields,omitempty"`
	ImageURL   string `json:"imaue_url,omitempty"`
	ThumbURL   string `json:"thumb_url,omitempty"`
	Footer     string `json:"footer,omitempty"`
	FooterIcon string `json:"footer_icon,omitempty"`
	Timestamp  uint64 `json:"ts,omitempty"`
}

// Slack
func slackSendTicketCreated(msgURL string, ticket *Ticket) {

	responseText := "Ticket \"" + ticket.Title + "\" created by <@" + ticket.User + ">."

	ticketURL := baseURL + "ticket/" + ticket.ID.String()

	var attachments [1]SlackAttachment
	attachments[0] = SlackAttachment{Text: ticketURL}

	response := &SlackMessage{
		Response:    "in_channel",
		Text:        responseText,
		Attachments: attachments[:],
	}

	jsonData, err := json.Marshal(response)

	if err != nil {
		panic(err)
	}

	http.Post(msgURL, "application/json", bytes.NewBuffer(jsonData))
}
