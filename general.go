package signalmgr

import (
	"encoding/json"
	"fmt"
)

type GetAboutResponse struct {
	Build        int                 `json:"build"`
	Capabilities map[string][]string `json:"capabilities"`
	Mode         string              `json:"mode"`
	Version      string              `json:"version"`
	Versions     []string            `json:"versions"`
}

// List all accounts.
//
// Lists all of the accounts linked or registered.
func GetAbout() (resp GetAboutResponse, err error) {
	return get[GetAboutResponse]("/v1/about")
}

// API Health Check.
//
// Internally used by the docker container to perform the health check.
func GetHealth() (resp string, err error) {
	return get[string]("/v1/health")
}

type Configuration struct {
	Logging struct {
		Level string `json:"Level"`
	} `json:"logging"`
}

// List the REST API configuration.
func GetConfiguration() (resp Configuration, err error) {
	return get[Configuration]("/v1/about")
}

// Set the REST API configuration.
func PostConfiguration(data Configuration) (err error) {
	_, err = post[any]("/v1/configuration", data)
	return
}

// Link device and generate QR code.
func GetLinkAccountQRCode(deviceName string) (link string, err error) {
	return get[string]("/v1/qrcodelink?" + encodeParams(params{"device_name": deviceName}))
}

type SendMessageV2_MessageMention struct {
	Start  int64  `json:"start"`
	Length int64  `json:"length"`
	Author string `json:"author"`
}

type SendMessageV2 struct {
	Number            string                         `json:"number"`
	Recipients        []string                       `json:"recipients"`
	Message           string                         `json:"message"`
	Base64Attachments []string                       `json:"base64_attachments" example:"<BASE64 ENCODED DATA>,data:<MIME-TYPE>;base64<comma><BASE64 ENCODED DATA>,data:<MIME-TYPE>;filename=<FILENAME>;base64<comma><BASE64 ENCODED DATA>"`
	Sticker           string                         `json:"sticker"`
	Mentions          []SendMessageV2_MessageMention `json:"mentions"`
	QuoteTimestamp    *int64                         `json:"quote_timestamp"`
	QuoteAuthor       *string                        `json:"quote_author"`
	QuoteMessage      *string                        `json:"quote_message"`
	QuoteMentions     []SendMessageV2_MessageMention `json:"quote_mentions"`
	TextMode          *string                        `json:"text_mode" enums:"normal,styled"`
	EditTimestamp     *int64                         `json:"edit_timestamp"`
	NotifySelf        *bool                          `json:"notify_self"`
}

// Send a signal message.
//
// Send a signal message. Set the text_mode to 'styled' in case you want to add formatting to your text message. Styling Options: *italic text*, **bold text**, ~strikethrough text~.
func PostSend(data SendMessageV2) (resp struct {
	Timestamp string `json:"timestamp"`
}, err error) {
	return post[struct {
		Timestamp string `json:"timestamp"`
	}]("/v2/send", data)
}

// List all attachments.
//
// List all downloaded attachments.
func GetAttachments() (attachments []string, err error) {
	return get[[]string]("/v1/attachments")
}

// Serve Attachment.
//
// Serve the attachment with the given id.
func GetAttachment(id string) (raw []byte, err error) {
	return getRaw(fmt.Sprintf("/v1/attachments/%s", id))
}

// Remove attachment.
//
// Remove the attachment with the given id from filesystem.
func DeleteAttachment(id string) (err error) {
	_, err = delete[any](fmt.Sprintf("/v1/attachments/%s", id), nil)
	return
}

type SearchResult struct {
	Number     string `json:"number"`
	Registered bool   `json:"registered"`
}

// Check if one or more phone numbers are registered with the Signal Service.
func GetSearch(numbers []string) (results []SearchResult, err error) {
	numbersJSON, err := json.Marshal(numbers)
	if err != nil {
		return
	}
	return get[[]SearchResult]("/v1/search?" + encodeParams(params{
		"message": string(numbersJSON),
	}))
}
