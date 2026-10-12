package signalmgr

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/DonovanDiamond/signalmgr/signaltypes"
	"github.com/gofiber/fiber/v2"
	"github.com/gorilla/websocket"
)

type Account struct {
	Number string
}

// List all accounts
//
// Lists all of the accounts linked or registered
func GetAccounts() (accounts []Account, err error) {
	list, err := get[[]string]("/v1/accounts")
	if err != nil {
		return
	}
	for _, num := range list {
		accounts = append(accounts, Account{
			Number: num,
		})
	}
	return
}

type Account_Configuration struct {
	TrustMode string `json:"trust_mode"`
}

// List account specific settings.
func (a *Account) GetConfiguration() (resp Account_Configuration, err error) {
	return get[Account_Configuration](fmt.Sprintf("/v1/configuration/%s/settings", url.PathEscape(a.Number)))
}

// Set account specific settings.
func (a *Account) PostConfiguration(data Account_Configuration) (err error) {
	_, err = request(fiber.MethodPost, fmt.Sprintf("/v1/configuration/%s/settings", url.PathEscape(a.Number)), data)
	return
}

// Links another device to this device. Only works, if this is the master device.
func (a *Account) PostLinkDevice(data struct {
	URI string `json:"uri"`
}) (err error) {
	_, err = request(fiber.MethodPost, fmt.Sprintf("/v1/devices/%s", url.PathEscape(a.Number)), data)
	return
}

// Register a phone number.
//
// Register a phone number with the signal network.
func (a *Account) PostRegister(captcha string, useVoice bool) error {
	_, err := request(fiber.MethodPost, fmt.Sprintf("/v1/register/%s", url.PathEscape(a.Number)), struct {
		Captcha  string `json:"captcha"`
		UseVoice bool   `json:"use_voice"`
	}{
		Captcha:  captcha,
		UseVoice: useVoice,
	})
	return err
}

// Verify a registered phone number.
//
// Verify a registered phone number with the signal network.
func (a *Account) PostRegisterVerify(token string, pin string) error {
	_, err := request(fiber.MethodPost, fmt.Sprintf("/v1/register/%s/verify/%s", url.PathEscape(a.Number), url.PathEscape(token)), struct {
		PIN string `json:"pin"`
	}{
		PIN: pin,
	})
	return err
}

// Unregister a phone number.
//
// Disables push support for this device. **WARNING:** If *delete_account* is set to *true*, the account will be deleted from the Signal Server. This cannot be undone without loss.
func (a *Account) PostUnregister(data struct {
	DeleteAccount   bool `json:"delete_account"`
	DeleteLocalData bool `json:"delete_local_data"`
}) (err error) {
	_, err = request(fiber.MethodPost, fmt.Sprintf("/v1/unregister/%s", url.PathEscape(a.Number)), data)
	return
}

// Lift rate limit restrictions by solving a captcha.
//
// When running into rate limits, sometimes the limit can be lifted, by solving a CAPTCHA. To get the captcha token, go to https://signalcaptchas.org/challenge/generate.html For the staging environment, use: https://signalcaptchas.org/staging/registration/generate.html. The \"challenge_token\" is the token from the failed send attempt. The \"captcha\" is the captcha result, starting with signalcaptcha://.
func (a *Account) PostRateLimitChallenge(data struct {
	Captcha        string `json:"captcha"`
	ChallengeToken string `json:"challenge_token"`
}) (err error) {
	_, err = request(fiber.MethodPost, fmt.Sprintf("/v1/accounts/%s/rate-limit-challenge", url.PathEscape(a.Number)), data)
	return
}

// Update the account settings.
//
// Update the account attributes on the signal server.
func (a *Account) PutSettings(data struct {
	DiscoverableByNumber *bool `json:"discoverable_by_number,omitempty"`
	ShareNumber          *bool `json:"share_number,omitempty"`
}) (err error) {
	_, err = request(fiber.MethodPut, fmt.Sprintf("/v1/accounts/%s/settings", url.PathEscape(a.Number)), data)
	return
}

type Account_PostUsernameResponse struct {
	Username     string `json:"username"`
	UsernameLink string `json:"username_link"`
}

// Set a username.
//
// Allows to set the username that should be used for this account. This can either be just the nickname (e.g. test) or the complete username with discriminator (e.g. test.123). Returns the new username with discriminator and the username link.
func (a *Account) PostUsername(data struct {
	Username string `json:"username"`
}) (resp Account_PostUsernameResponse, err error) {
	return post[Account_PostUsernameResponse](fmt.Sprintf("/v1/accounts/%s/username", url.PathEscape(a.Number)), data)
}

// Remove a username.
//
// Delete the username associated with this account.
func (a *Account) DeleteUsername() (err error) {
	_, err = request(fiber.MethodDelete, fmt.Sprintf("/v1/accounts/%s/username", url.PathEscape(a.Number)), nil)
	return
}

type GroupPermissions struct {
	AddMembers   string `json:"add_members"`
	EditGroup    string `json:"edit_group"`
	SendMessages string `json:"send_messages"`
}

type Group struct {
	Admins          []string         `json:"admins"`
	Blocked         bool             `json:"blocked"`
	Description     string           `json:"description"`
	ID              string           `json:"id"`
	InternalID      string           `json:"internal_id"`
	InviteLink      string           `json:"invite_link"`
	Member          bool             `json:"member"`
	Members         []string         `json:"members"`
	Name            string           `json:"name"`
	Permissions     GroupPermissions `json:"permissions"`
	PendingInvites  []string         `json:"pending_invites"`
	PendingRequests []string         `json:"pending_requests"`
}

type GetGroupsOptions struct {
	Expand bool
}

// List all Signal Groups.
func (a *Account) GetGroups(opts GetGroupsOptions) (groups []Group, err error) {
	q := url.Values{}
	if opts.Expand {
		q.Set("expand", "true")
	}
	return get[[]Group](withQuery(fmt.Sprintf("/v1/groups/%s", url.PathEscape(a.Number)), q))
}

// Create a new Signal Group with the specified members.
func (a *Account) PostCreateGroup(data struct {
	Description    string            `json:"description,omitempty"`
	ExpirationTime int               `json:"expiration_time,omitempty"`
	GroupLink      string            `json:"group_link,omitempty"`
	Members        []string          `json:"members"`
	Name           string            `json:"name"`
	Permissions    *GroupPermissions `json:"permissions,omitempty"`
}) (resp struct {
	ID string `json:"id"`
}, err error) {
	return post[struct {
		ID string `json:"id"`
	}](fmt.Sprintf("/v1/groups/%s", url.PathEscape(a.Number)), data)
}

type GetGroupOptions struct {
	Expand bool
}

// List a specific Signal Group.
func (a *Account) GetGroup(groupID string, opts GetGroupOptions) (group Group, err error) {
	q := url.Values{}
	if opts.Expand {
		q.Set("expand", "true")
	}
	return get[Group](withQuery(fmt.Sprintf("/v1/groups/%s/%s", url.PathEscape(a.Number), url.PathEscape(groupID)), q))
}

// Update the state of a Signal Group.
func (a *Account) PutGroupSettings(groupID string, data struct {
	Base64Avatar   *string           `json:"base64_avatar,omitempty"`
	Description    *string           `json:"description,omitempty"`
	ExpirationTime *int              `json:"expiration_time,omitempty"`
	GroupLink      *string           `json:"group_link,omitempty"`
	Name           *string           `json:"name,omitempty"`
	Permissions    *GroupPermissions `json:"permissions,omitempty"`
}) (err error) {
	_, err = request(fiber.MethodPut, fmt.Sprintf("/v1/groups/%s/%s", url.PathEscape(a.Number), url.PathEscape(groupID)), data)
	return
}

// Delete the specified Signal Group.
func (a *Account) DeleteGroup(groupID string) (err error) {
	_, err = request(fiber.MethodDelete, fmt.Sprintf("/v1/groups/%s/%s", url.PathEscape(a.Number), url.PathEscape(groupID)), nil)
	return
}

// Add one or more admins to an existing Signal Group.
func (a *Account) PostGroupAdmins(groupID string, data struct {
	Admins []string `json:"admins"`
}) (err error) {
	_, err = request(fiber.MethodPost, fmt.Sprintf("/v1/groups/%s/%s/admins", url.PathEscape(a.Number), url.PathEscape(groupID)), data)
	return
}

// Remove one or more admins from an existing Signal Group.
func (a *Account) DeleteGroupAdmins(groupID string, data struct {
	Admins []string `json:"admins"`
}) (err error) {
	_, err = request(fiber.MethodDelete, fmt.Sprintf("/v1/groups/%s/%s/admins", url.PathEscape(a.Number), url.PathEscape(groupID)), data)
	return
}

// Block the specified Signal Group.
func (a *Account) PostBlockGroup(groupID string) (err error) {
	_, err = request(fiber.MethodPost, fmt.Sprintf("/v1/groups/%s/%s/block", url.PathEscape(a.Number), url.PathEscape(groupID)), nil)
	return
}

// Join the specified Signal Group.
func (a *Account) PostJoinGroup(groupID string) (err error) {
	_, err = request(fiber.MethodPost, fmt.Sprintf("/v1/groups/%s/%s/join", url.PathEscape(a.Number), url.PathEscape(groupID)), nil)
	return
}

// Add one or more members to an existing Signal Group.
func (a *Account) PostGroupMembers(groupID string, data struct {
	Members []string `json:"members"`
}) (err error) {
	_, err = request(fiber.MethodPost, fmt.Sprintf("/v1/groups/%s/%s/members", url.PathEscape(a.Number), url.PathEscape(groupID)), data)
	return
}

// Remove one or more members from an existing Signal Group.
func (a *Account) DeleteGroupMembers(groupID string, data struct {
	Members []string `json:"members"`
}) (err error) {
	_, err = request(fiber.MethodDelete, fmt.Sprintf("/v1/groups/%s/%s/members", url.PathEscape(a.Number), url.PathEscape(groupID)), data)
	return
}

// Quit the specified Signal Group.
func (a *Account) PostQuitGroup(groupID string) (err error) {
	_, err = request(fiber.MethodPost, fmt.Sprintf("/v1/groups/%s/%s/quit", url.PathEscape(a.Number), url.PathEscape(groupID)), nil)
	return
}

type MessageResponse struct {
	Envelope  signaltypes.MessageEnvelope `json:"envelope"`
	Account   string                      `json:"account"`
	RawFields map[string]any              `json:"raw_fields"`
}

type GetMessagesOptions struct {
	Timeout           int
	IgnoreAttachments bool
	IgnoreStories     bool
	IgnoreAvatars     bool
	IgnoreStickers    bool
	MaxMessages       int
	SendReadReceipts  bool
}

// Receive Signal Messages.
//
// Only works if the signal api is running in `normal` or `native` mode. If you are running in `json-rpc` mode, use `GetMessagesSocket`.
func (a *Account) GetMessages(opts GetMessagesOptions) (messages []MessageResponse, err error) {
	q := url.Values{}
	if opts.Timeout > 0 {
		q.Set("timeout", strconv.Itoa(opts.Timeout))
	}
	if opts.IgnoreAttachments {
		q.Set("ignore_attachments", "true")
	}
	if opts.IgnoreStories {
		q.Set("ignore_stories", "true")
	}
	if opts.IgnoreAvatars {
		q.Set("ignore_avatars", "true")
	}
	if opts.IgnoreStickers {
		q.Set("ignore_stickers", "true")
	}
	if opts.MaxMessages > 0 {
		q.Set("max_messages", strconv.Itoa(opts.MaxMessages))
	}
	if opts.SendReadReceipts {
		q.Set("send_read_receipts", "true")
	}
	return get[[]MessageResponse](withQuery(fmt.Sprintf("/v1/receive/%s", url.PathEscape(a.Number)), q))
}

// Opens a socket to receive Signal Messages and sends them to the `messages` channel.
//
// Will only return if there is an error or the socket closes.
//
// Only works if the signal api is running in `json-rpc` mode. If you are running in `normal` or `native` mode, use `GetMessages`.
func (a *Account) GetMessagesSocket(messages chan<- MessageResponse) (err error) {
	baseURL := strings.TrimSuffix(API_URL, "/")
	if strings.HasPrefix(baseURL, "https://") {
		baseURL = "wss://" + strings.TrimPrefix(baseURL, "https://")
	} else if strings.HasPrefix(baseURL, "http://") {
		baseURL = "ws://" + strings.TrimPrefix(baseURL, "http://")
	}
	fullURL := fmt.Sprintf("%s/v1/receive/%s", baseURL, url.PathEscape(a.Number))

	c, _, err := websocket.DefaultDialer.Dial(fullURL, nil)
	if err != nil {
		return fmt.Errorf("failed to dial websocket: %w", err)
	}

	defer c.Close()

	for {
		// TODO: Probably not the best to unmarshal this 2 times, but I don't currently know a way around this to get the raw json
		var all = make(map[string]any)
		if err := c.ReadJSON(&all); err != nil {
			return fmt.Errorf("error reading from websocket: %w", err)
		}
		raw, err := json.Marshal(all)
		if err != nil {
			return fmt.Errorf("failed to marshal message from websocket: %w", err)
		}
		var m MessageResponse
		if err := json.Unmarshal(raw, &m); err != nil {
			return fmt.Errorf("failed to unmarshal message from websocket: %w", err)
		}
		m.RawFields = all
		messages <- m
	}
}

// Show Typing Indicator.
func (a *Account) PutTypingIndicator(data struct {
	Recipient string `json:"recipient"`
}) (err error) {
	_, err = request(fiber.MethodPut, fmt.Sprintf("/v1/typing-indicator/%s", url.PathEscape(a.Number)), data)
	return
}

// Hide Typing Indicator.
func (a *Account) DeleteTypingIndicator(data struct {
	Recipient string `json:"recipient"`
}) (err error) {
	_, err = request(fiber.MethodDelete, fmt.Sprintf("/v1/typing-indicator/%s", url.PathEscape(a.Number)), data)
	return
}

// Update Profile.
//
// Set your name and optional an avatar.
func (a *Account) PutProfile(data struct {
	About        string `json:"about,omitempty"`
	Base64Avatar string `json:"base64_avatar,omitempty"`
	Name         string `json:"name"`
}) (err error) {
	_, err = request(fiber.MethodPut, fmt.Sprintf("/v1/profiles/%s", url.PathEscape(a.Number)), data)
	return
}

type Identity struct {
	Added        string `json:"added"`
	Fingerprint  string `json:"fingerprint"`
	Number       string `json:"number"`
	SafetyNumber string `json:"safety_number"`
	Status       string `json:"status"`
	UUID         string `json:"uuid"`
}

// List all identities for the given number.
func (a *Account) GetIdentities() (identities []Identity, err error) {
	return get[[]Identity](fmt.Sprintf("/v1/identities/%s", url.PathEscape(a.Number)))
}

// Trust an identity. When 'trust_all_known_keys' is set to 'true', all known keys of this user are trusted. **This is only recommended for testing.**
func (a *Account) PutTrustIdentity(numberToTrust string, data struct {
	TrustAllKnownKeys    bool   `json:"trust_all_known_keys"`
	VerifiedSafetyNumber string `json:"verified_safety_number"`
}) (err error) {
	_, err = request(fiber.MethodPut, fmt.Sprintf("/v1/identities/%s/trust/%s", url.PathEscape(a.Number), url.PathEscape(numberToTrust)), data)
	return
}

// Send a reaction.
//
// React to a message.
func (a *Account) PostReaction(data struct {
	Reaction     string `json:"reaction"`
	Recipient    string `json:"recipient"`
	TargetAuthor string `json:"target_author"`
	Timestamp    int64  `json:"timestamp"`
}) (err error) {
	_, err = request(fiber.MethodPost, fmt.Sprintf("/v1/reactions/%s", url.PathEscape(a.Number)), data)
	return
}

// Remove a reaction.
func (a *Account) DeleteReaction(data struct {
	Reaction     string `json:"reaction,omitempty"`
	Recipient    string `json:"recipient"`
	TargetAuthor string `json:"target_author"`
	Timestamp    int64  `json:"timestamp"`
}) (err error) {
	_, err = request(fiber.MethodDelete, fmt.Sprintf("/v1/reactions/%s", url.PathEscape(a.Number)), data)
	return
}

// Send receipts.
//
// Send read or viewed receipts.
func (a *Account) PostReceipts(data struct {
	ReceiptType string `json:"receipt_type"`
	Recipient   string `json:"recipient"`
	Timestamp   int64  `json:"timestamp"`
}) (err error) {
	_, err = request(fiber.MethodPost, fmt.Sprintf("/v1/receipts/%s", url.PathEscape(a.Number)), data)
	return
}

type StickerPack struct {
	Author    string `json:"author"`
	Installed bool   `json:"installed"`
	PackID    string `json:"pack_id"`
	Title     string `json:"title"`
	URL       string `json:"url"`
}

// List Installed Sticker Packs.
func (a *Account) GetStickerPacks() (packs []StickerPack, err error) {
	return get[[]StickerPack](fmt.Sprintf("/v1/sticker-packs/%s", url.PathEscape(a.Number)))
}

// Add Sticker Pack.
//
// In order to add a sticker pack, browse to https://signalstickers.org/ and select the sticker pack you want to add. Then, press the \"Add to Signal\" button. If you look at the address bar in your browser you should see an URL in this format: https://signal.art/addstickers/#pack_id=XXX\u0026pack_key=YYY, where XXX is the pack_id and YYY is the pack_key.
func (a *Account) PostStickerPack(data struct {
	PackID  string `json:"pack_id"`
	PackKey string `json:"pack_key"`
}) (err error) {
	_, err = request(fiber.MethodPost, fmt.Sprintf("/v1/sticker-packs/%s", url.PathEscape(a.Number)), data)
	return
}

type Contact_Nickname struct {
	FamilyName string `json:"family_name"`
	GivenName  string `json:"given_name"`
	Name       string `json:"name"`
}

type Contact_Profile struct {
	About                string `json:"about"`
	GivenName            string `json:"given_name"`
	HasAvatar            bool   `json:"has_avatar"`
	LastUpdatedTimestamp int64  `json:"last_updated_timestamp"`
	Lastname             string `json:"lastname"`
}

type Contact struct {
	Blocked           bool             `json:"blocked"`
	Color             string           `json:"color"`
	GivenName         string           `json:"given_name"`
	MessageExpiration string           `json:"message_expiration"`
	Name              string           `json:"name"`
	Nickname          Contact_Nickname `json:"nickname"`
	Note              string           `json:"note"`
	Number            string           `json:"number"`
	Profile           Contact_Profile  `json:"profile"`
	ProfileName       string           `json:"profile_name"`
	Username          string           `json:"username"`
	UUID              string           `json:"uuid"`
}

type GetContactsOptions struct {
	AllRecipients bool
}

// List Contacts.
//
// List all contacts for the given number.
func (a *Account) GetContacts(opts GetContactsOptions) (contacts []Contact, err error) {
	q := url.Values{}
	if opts.AllRecipients {
		q.Set("all_recipients", "true")
	}
	return get[[]Contact](withQuery(fmt.Sprintf("/v1/contacts/%s", url.PathEscape(a.Number)), q))
}

type GetContactOptions struct {
	AllRecipients bool
}

// List Contact.
//
// List a specific contact.
func (a *Account) GetContact(uuid string, opts GetContactOptions) (contact Contact, err error) {
	q := url.Values{}
	if opts.AllRecipients {
		q.Set("all_recipients", "true")
	}
	return get[Contact](withQuery(fmt.Sprintf("/v1/contacts/%s/%s", url.PathEscape(a.Number), url.PathEscape(uuid)), q))
}

// Returns the avatar of a contact.
func (a *Account) GetContactAvatar(uuid string) (avatar []byte, err error) {
	return getRaw(fmt.Sprintf("/v1/contacts/%s/%s/avatar", url.PathEscape(a.Number), url.PathEscape(uuid)))
}

// Updates the info associated to a number on the contact list. If the contact doesn’t exist yet, it will be added.
func (a *Account) PutContact(data struct {
	ExpirationInSeconds int    `json:"expiration_in_seconds,omitempty"`
	Name                string `json:"name,omitempty"`
	Recipient           string `json:"recipient"`
}) (err error) {
	_, err = request(fiber.MethodPut, fmt.Sprintf("/v1/contacts/%s", url.PathEscape(a.Number)), data)
	return
}

// Send a synchronization message with the local contacts list to all linked devices. This command should only be used if this is the primary device.
func (a *Account) PostContactsSync() (err error) {
	_, err = request(fiber.MethodPost, fmt.Sprintf("/v1/contacts/%s/sync", url.PathEscape(a.Number)), nil)
	return
}

type SearchResult struct {
	Number     string `json:"number"`
	Registered bool   `json:"registered"`
}

// Check if one or more phone numbers are registered with the Signal Service.
func (a *Account) GetSearch(numbers []string) (results []SearchResult, err error) {
	q := url.Values{"numbers": numbers}
	return get[[]SearchResult](fmt.Sprintf("/v1/search/%s?%s", url.PathEscape(a.Number), q.Encode()))
}

// Set Pin.
//
// Sets a new Signal Pin.
func (a *Account) PostPin(data struct {
	Pin string `json:"pin"`
}) (err error) {
	_, err = request(fiber.MethodPost, fmt.Sprintf("/v1/accounts/%s/pin", url.PathEscape(a.Number)), data)
	return
}

// Remove Pin.
//
// Removes a Signal Pin.
func (a *Account) DeletePin() (err error) {
	_, err = request(fiber.MethodDelete, fmt.Sprintf("/v1/accounts/%s/pin", url.PathEscape(a.Number)), nil)
	return
}

type Device struct {
	CreationTimestamp int64  `json:"creation_timestamp"`
	ID                int64  `json:"id"`
	LastSeenTimestamp int64  `json:"last_seen_timestamp"`
	Name              string `json:"name"`
}

// List linked devices.
//
// List linked devices associated to this device.
func (a *Account) GetDevices() (devices []Device, err error) {
	return get[[]Device](fmt.Sprintf("/v1/devices/%s", url.PathEscape(a.Number)))
}

// Remove linked device.
//
// Remove a linked device from the primary account.
func (a *Account) DeleteDevice(deviceID int64) (err error) {
	_, err = request(fiber.MethodDelete, fmt.Sprintf("/v1/devices/%s/%d", url.PathEscape(a.Number), deviceID), nil)
	return
}

// Delete local account data.
//
// Delete all local data for the specified account. Only use this after unregistering the account or after removing a linked device.
func (a *Account) DeleteLocalData(data struct {
	IgnoreRegistered bool `json:"ignore_registered,omitempty"`
}) (err error) {
	_, err = request(fiber.MethodDelete, fmt.Sprintf("/v1/devices/%s/local-data", url.PathEscape(a.Number)), data)
	return
}

// Returns the avatar of a Signal Group.
func (a *Account) GetGroupAvatar(groupID string) (avatar []byte, err error) {
	return getRaw(fmt.Sprintf("/v1/groups/%s/%s/avatar", url.PathEscape(a.Number), url.PathEscape(groupID)))
}

// Pin a message in a Signal Group.
func (a *Account) PostGroupPinMessage(groupID string, data struct {
	Duration     *int64 `json:"duration,omitempty"`
	TargetAuthor string `json:"target_author"`
	Timestamp    int64  `json:"timestamp"`
}) (err error) {
	_, err = request(fiber.MethodPost, fmt.Sprintf("/v1/groups/%s/%s/pin-message", url.PathEscape(a.Number), url.PathEscape(groupID)), data)
	return
}

// Unpin a message in a Signal Group.
func (a *Account) DeleteGroupPinMessage(groupID string, data struct {
	TargetAuthor string `json:"target_author"`
	Timestamp    int64  `json:"timestamp"`
}) (err error) {
	_, err = request(fiber.MethodDelete, fmt.Sprintf("/v1/groups/%s/%s/pin-message", url.PathEscape(a.Number), url.PathEscape(groupID)), data)
	return
}

type Account_PostPollResponse struct {
	Timestamp string `json:"timestamp"`
}

// Create a new poll.
func (a *Account) PostPoll(data struct {
	AllowMultipleSelections *bool    `json:"allow_multiple_selections,omitempty"`
	Answers                 []string `json:"answers"`
	Question                string   `json:"question"`
	Recipient               string   `json:"recipient"`
}) (resp Account_PostPollResponse, err error) {
	return post[Account_PostPollResponse](fmt.Sprintf("/v1/polls/%s", url.PathEscape(a.Number)), data)
}

// Close a poll.
func (a *Account) DeletePoll(data struct {
	PollTimestamp string `json:"poll_timestamp"`
	Recipient     string `json:"recipient"`
}) (err error) {
	_, err = request(fiber.MethodDelete, fmt.Sprintf("/v1/polls/%s", url.PathEscape(a.Number)), data)
	return
}

// Answer a poll.
func (a *Account) PostPollVote(data struct {
	PollAuthor      string `json:"poll_author"`
	PollTimestamp   string `json:"poll_timestamp"`
	Recipient       string `json:"recipient"`
	SelectedAnswers []int  `json:"selected_answers"`
}) (err error) {
	_, err = request(fiber.MethodPost, fmt.Sprintf("/v1/polls/%s/vote", url.PathEscape(a.Number)), data)
	return
}

type Account_DeleteRemoteMessageResponse struct {
	Timestamp string `json:"timestamp"`
}

// Delete a signal message.
func (a *Account) DeleteRemoteMessage(data struct {
	Recipient string `json:"recipient"`
	Timestamp int64  `json:"timestamp"`
}) (resp Account_DeleteRemoteMessageResponse, err error) {
	return delete[Account_DeleteRemoteMessageResponse](fmt.Sprintf("/v1/remote-delete/%s", url.PathEscape(a.Number)), data)
}
