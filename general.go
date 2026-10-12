package signalmgr

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type GetAboutResponse struct {
	Build        int                 `json:"build"`
	Capabilities map[string][]string `json:"capabilities"`
	Mode         string              `json:"mode"`
	Version      string              `json:"version"`
	Versions     []string            `json:"versions"`
}

// Get API information.
//
// Returns the build, mode, version and capabilities of the REST API.
func GetAbout() (resp GetAboutResponse, err error) {
	return get[GetAboutResponse]("/v1/about")
}

// API Health Check.
//
// Internally used by the docker container to perform the health check.
func GetHealth() (err error) {
	_, err = getRaw("/v1/health")
	return
}

type Configuration struct {
	Logging struct {
		Level string `json:"Level"`
	} `json:"logging"`
}

// List the REST API configuration.
func GetConfiguration() (resp Configuration, err error) {
	return get[Configuration]("/v1/configuration")
}

// Set the REST API configuration.
func PostConfiguration(data Configuration) (err error) {
	_, err = request(fiber.MethodPost, "/v1/configuration", data)
	return
}

type GetLinkAccountQRCodeOptions struct {
	QRCodeVersion int
}

// Link device and generate QR code.
//
// Returns the QR code as a PNG image.
func GetLinkAccountQRCode(deviceName string, opts GetLinkAccountQRCodeOptions) (png []byte, err error) {
	q := url.Values{"device_name": {deviceName}}
	if opts.QRCodeVersion > 0 {
		q.Set("qrcode_version", strconv.Itoa(opts.QRCodeVersion))
	}
	return getRaw(withQuery("/v1/qrcodelink", q))
}

// Get raw device link URI.
//
// Generate the deviceLinkUri string for linking without scanning a QR code.
func GetLinkAccountURI(deviceName string) (uri string, err error) {
	resp, err := get[struct {
		DeviceLinkURI string `json:"device_link_uri"`
	}](withQuery("/v1/qrcodelink/raw", url.Values{"device_name": {deviceName}}))
	return resp.DeviceLinkURI, err
}

type SendMessageV2_MessageMention struct {
	Start  int64  `json:"start"`
	Length int64  `json:"length"`
	Author string `json:"author"`
}

type SendMessageV2_LinkPreview struct {
	Base64Thumbnail string `json:"base64_thumbnail,omitempty"`
	Description     string `json:"description,omitempty"`
	Title           string `json:"title,omitempty"`
	URL             string `json:"url,omitempty"`
}

type SendMessageV2 struct {
	Number            string                         `json:"number"`
	Recipients        []string                       `json:"recipients"`
	Message           string                         `json:"message"`
	Base64Attachments []string                       `json:"base64_attachments,omitempty" example:"<BASE64 ENCODED DATA>,data:<MIME-TYPE>;base64<comma><BASE64 ENCODED DATA>,data:<MIME-TYPE>;filename=<FILENAME>;base64<comma><BASE64 ENCODED DATA>"`
	Sticker           string                         `json:"sticker,omitempty"`
	Mentions          []SendMessageV2_MessageMention `json:"mentions,omitempty"`
	QuoteTimestamp    *int64                         `json:"quote_timestamp,omitempty"`
	QuoteAuthor       *string                        `json:"quote_author,omitempty"`
	QuoteMessage      *string                        `json:"quote_message,omitempty"`
	QuoteMentions     []SendMessageV2_MessageMention `json:"quote_mentions,omitempty"`
	TextMode          *string                        `json:"text_mode,omitempty" enums:"normal,styled"`
	EditTimestamp     *int64                         `json:"edit_timestamp,omitempty"`
	NotifySelf        *bool                          `json:"notify_self,omitempty"`
	LinkPreview       *SendMessageV2_LinkPreview     `json:"link_preview,omitempty"`
	ViewOnce          *bool                          `json:"view_once,omitempty"`
}

type SendMessageResponse struct {
	Timestamp string `json:"timestamp"`
	Errors    struct {
		Recipients []struct {
			Number   string `json:"number"`
			Reason   string `json:"reason"`
			Username string `json:"username"`
			UUID     string `json:"uuid"`
		} `json:"recipients"`
	} `json:"errors"`
}

// Send a signal message.
//
// Send a signal message. Set the text_mode to 'styled' in case you want to add formatting to your text message. Styling Options: *italic text*, **bold text**, ~strikethrough text~.
func PostSend(data SendMessageV2) (resp SendMessageResponse, err error) {
	return post[SendMessageResponse]("/v2/send", data)
}

type SendMessageV1 struct {
	Base64Attachment string   `json:"base64_attachment,omitempty"`
	IsGroup          bool     `json:"is_group,omitempty"`
	Message          string   `json:"message"`
	Number           string   `json:"number"`
	Recipients       []string `json:"recipients"`
}

// Send a signal message.
//
// Deprecated: use PostSend.
func PostSendV1(data SendMessageV1) (err error) {
	_, err = request(fiber.MethodPost, "/v1/send", data)
	return
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
	return getRaw(fmt.Sprintf("/v1/attachments/%s", url.PathEscape(id)))
}

// Remove attachment.
//
// Remove the attachment with the given id from filesystem.
func DeleteAttachment(id string) (err error) {
	_, err = request(fiber.MethodDelete, fmt.Sprintf("/v1/attachments/%s", url.PathEscape(id)), nil)
	return
}
