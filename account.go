package signalmgr

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/DonovanDiamond/signalmgr/signaltypes"
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
	return get[Account_Configuration](fmt.Sprintf("/v1/configuration/%s/settings", a.Number))
}

// Set account specific settings.
func (a *Account) PostConfiguration(data Account_Configuration) (err error) {
	_, err = post[any](fmt.Sprintf("/v1/configuration/%s/settings", a.Number), data)
	return
}

// Links another device to this device. Only works, if this is the master device.
func (a *Account) PostLinkDevice(data struct {
	URI string `json:"uri"`
}) (err error) {
	_, err = post[any](fmt.Sprintf("/v1/devices/%s", a.Number), data)
	return
}

// Register a phone number.
//
// Register a phone number with the signal network.
func (a *Account) PostRegister(captcha string, useVoice bool) error {
	_, err := post[any](fmt.Sprintf("/v1/register/%s", a.Number), struct {
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
	_, err := post[any](fmt.Sprintf("/v1/register/%s/verify/%s", a.Number, token), struct {
		PIN string `json:"pin"`
	}{
		PIN: pin,
	})
	return err
}

// Unregister a phone number.
//
// Disables push support for this device. **WARNING:** If *delete_account* is set to *true*, the account will be deleted from the Signal Server. This cannot be undone without loss.
func (a *Account) PostUnregistert(data struct {
	DeleteAccount   bool `json:"delete_account"`
	DeleteLocalData bool `json:"delete_local_data"`
}) (err error) {
	_, err = post[any](fmt.Sprintf("/v1/unregister/%s", a.Number), data)
	return
}

// Lift rate limit restrictions by solving a captcha.
//
// When running into rate limits, sometimes the limit can be lifted, by solving a CAPTCHA. To get the captcha token, go to https://signalcaptchas.org/challenge/generate.html For the staging environment, use: https://signalcaptchas.org/staging/registration/generate.html. The \"challenge_token\" is the token from the failed send attempt. The \"captcha\" is the captcha result, starting with signalcaptcha://.
func (a *Account) PostRateLimitChallenge(data struct {
	Captcha        string `json:"captcha"`
	ChallengeToken string `json:"challenge_token"`
}) (err error) {
	_, err = post[any](fmt.Sprintf("/v1/accounts/%s/rate-limit-challenge", a.Number), data)
	return
}

// Update the account settings.
//
// Update the account attributes on the signal server.
func (a *Account) PutSettings(data struct {
	DiscoverableByNumber bool `json:"discoverable_by_number"`
	ShareNumber          bool `json:"share_number"`
}) (err error) {
	_, err = put[any](fmt.Sprintf("/v1/accounts/%s/settings", a.Number), data)
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
	return post[Account_PostUsernameResponse](fmt.Sprintf("/v1/accounts/%s/username", a.Number), data)
}

// Remove a username.
//
// Delete the username associated with this account.
func (a *Account) DeleteUsername() (err error) {
	_, err = delete[any](fmt.Sprintf("/v1/accounts/%s/username", a.Number), nil)
	return
}

type Group struct {
	Admins          []string `json:"admins"`
	Blocked         bool     `json:"blocked"`
	ID              string   `json:"id"`
	InternalID      string   `json:"internal_id"`
	InviteLink      string   `json:"invite_link"`
	Members         []string `json:"members"`
	Name            string   `json:"name"`
	PendingInvites  []string `json:"pending_invites"`
	PendingRequests []string `json:"pending_requests"`
}

// List all Signal Groups.
func (a *Account) GetGroups() (groups []Group, err error) {
	return get[[]Group](fmt.Sprintf("/v1/groups/%s", a.Number))
}

// Create a new Signal Group with the specified members.
func (a *Account) PostCreateGroup(data struct {
	Description    string   `json:"description"`
	ExpirationTime int      `json:"expiration_time"`
	GroupLink      string   `json:"group_link"`
	Members        []string `json:"members"`
	Name           string   `json:"name"`
	Permissions    struct {
		AddMembers string `json:"add_members"`
		EditGroup  string `json:"edit_group"`
	} `json:"permissions"`
}) (resp struct {
	ID string `json:"id"`
}, err error) {
	return post[struct {
		ID string `json:"id"`
	}](fmt.Sprintf("/v1/groups/%s", a.Number), data)
}

// List a specific Signal Group.
func (a *Account) GetGroup(groupID string) (group Group, err error) {
	return get[Group](fmt.Sprintf("/v1/groups/%s/%s", a.Number, groupID))
}

// Update the state of a Signal Group.
func (a *Account) PutGroupSettings(groupID string, data struct {
	Base64Avatar string `json:"base64_avatar"`
	Description  string `json:"description"`
	Name         string `json:"name"`
}) (err error) {
	_, err = put[any](fmt.Sprintf("/v1/groups/%s/%s", a.Number, groupID), data)
	return
}

// Delete the specified Signal Group.
func (a *Account) DeleteGroup(groupID string) (err error) {
	_, err = delete[any](fmt.Sprintf("/v1/groups/%s/%s", a.Number, groupID), nil)
	return
}

// Add one or more admins to an existing Signal Group.
func (a *Account) PostGroupAdmins(groupID string, data struct {
	Admins []string `json:"admins"`
}) (err error) {
	_, err = post[any](fmt.Sprintf("/v1/groups/%s/%s/admins", a.Number, groupID), data)
	return
}

// Remove one or more admins from an existing Signal Group.
func (a *Account) DeleteGroupAdmins(groupID string, data struct {
	Admins []string `json:"admins"`
}) (err error) {
	_, err = delete[any](fmt.Sprintf("/v1/groups/%s/%s/admins", a.Number, groupID), data)
	return
}

// Block the specified Signal Group.
func (a *Account) PostBlockGroup(groupID string) (err error) {
	_, err = post[any](fmt.Sprintf("/v1/groups/%s/%s/block", a.Number, groupID), nil)
	return
}

// Join the specified Signal Group.
func (a *Account) PostJoinGroup(groupID string) (err error) {
	_, err = post[any](fmt.Sprintf("/v1/groups/%s/%s/join", a.Number, groupID), nil)
	return
}

// Add one or more members to an existing Signal Group.
func (a *Account) PostGroupMembers(groupID string, data struct {
	Members []string `json:"members"`
}) (err error) {
	_, err = post[any](fmt.Sprintf("/v1/groups/%s/%s/members", a.Number, groupID), nil)
	return
}

// Remove one or more members from an existing Signal Group.
func (a *Account) DeleteGroupMembers(groupID string, data struct {
	Members []string `json:"members"`
}) (err error) {
	_, err = delete[any](fmt.Sprintf("/v1/groups/%s/%s/members", a.Number, groupID), nil)
	return
}

// Quit the specified Signal Group.
func (a *Account) PostQuitGroup(groupID string) (err error) {
	_, err = post[any](fmt.Sprintf("/v1/groups/%s/%s/quit", a.Number, groupID), nil)
	return
}

type MessageResponse struct {
	Envelope signaltypes.MessageEnvelope `json:"envelope"`
	Account  string                      `json:"account"`
	Raw      string                      `json:"raw"`
}

// Receive Signal Messages.
//
// Only works if the signal api is running in `normal` or `native` mode. If you are running in `json-rpc` mode, use `GetMessagesSocket`.
func (a *Account) GetMessages() (messages []MessageResponse, err error) {
	return get[[]MessageResponse](fmt.Sprintf("/v1/receive/%s", a.Number))
}

// Opens a socket to receive Signal Messages and sends them to the `messages` channel.
//
// Will only return if there is an error or the socket closes.
//
// Only works if the signal api is running in `json-rpc` mode. If you are running in `normal` or `native` mode, use `GetMessages`.
func (a *Account) GetMessagesSocket(messages chan<- MessageResponse) (err error) {
	baseURL := strings.ReplaceAll(API_URL, "https://", "wss://")
	baseURL = strings.ReplaceAll(baseURL, "http://", "ws://")
	fullURL := fmt.Sprintf("%s/v1/receive/%s", baseURL, a.Number)

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
		m.Raw = string(raw)
		messages <- m
	}
}

// Show Typing Indicator.
func (a *Account) PutTypingIndicator(data struct {
	Recipient string `json:"recipient"`
}) (err error) {
	_, err = put[any](fmt.Sprintf("/v1/typing-indicator/%s", a.Number), data)
	return
}

// Hide Typing Indicator.
func (a *Account) DeleteTypingIndicator(data struct {
	Recipient string `json:"recipient"`
}) (err error) {
	_, err = delete[any](fmt.Sprintf("/v1/typing-indicator/%s", a.Number), data)
	return
}

// Update Profile.
//
// Set your name and optional an avatar.
func (a *Account) PostProfile(data struct {
	About        string `json:"about"`
	Base64Avatar string `json:"base64_avatar"`
	Name         string `json:"name"`
}) (err error) {
	_, err = put[any](fmt.Sprintf("/v1/profiles/%s", a.Number), data)
	return
}

type Identity struct {
	Added        string `json:"added"`
	Fingerprint  string `json:"fingerprint"`
	Number       string `json:"number"`
	SafetyNumber string `json:"safety_number"`
	Status       string `json:"status"`
}

// List all identities for the given number.
func (a *Account) GetIdentities() (identities []Identity, err error) {
	return get[[]Identity](fmt.Sprintf("/v1/identities/%s", a.Number))
}

// Trust an identity. When 'trust_all_known_keys' is set to 'true', all known keys of this user are trusted. **This is only recommended for testing.**
func (a *Account) PutTrustIdentity(numberToTrust string, data struct {
	TrustAllKnownKeys    bool   `json:"trust_all_known_keys"`
	VerifiedSafetyNumber string `json:"verified_safety_number"`
}) (err error) {
	_, err = put[any](fmt.Sprintf("/v1/identities/%s/trust/%s", a.Number, numberToTrust), data)
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
	_, err = post[any](fmt.Sprintf("/v1/reactions/%s", a.Number), data)
	return
}

// Remove a reaction.
func (a *Account) DeleteReaction(data struct {
	Reaction     string `json:"reaction"`
	Recipient    string `json:"recipient"`
	TargetAuthor string `json:"target_author"`
	Timestamp    int64  `json:"timestamp"`
}) (err error) {
	_, err = delete[any](fmt.Sprintf("/v1/reactions/%s", a.Number), data)
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
	_, err = post[any](fmt.Sprintf("/v1/receipts/%s", a.Number), data)
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
	return get[[]StickerPack](fmt.Sprintf("/v1/sticker-packs/%s", a.Number))
}

// Add Sticker Pack.
//
// In order to add a sticker pack, browse to https://signalstickers.org/ and select the sticker pack you want to add. Then, press the \"Add to Signal\" button. If you look at the address bar in your browser you should see an URL in this format: https://signal.art/addstickers/#pack_id=XXX\u0026pack_key=YYY, where XXX is the pack_id and YYY is the pack_key.
func (a *Account) PostStickerPack(data struct {
	PackID  string `json:"pack_id"`
	PackKey string `json:"pack_key"`
}) (err error) {
	_, err = post[any](fmt.Sprintf("/v1/sticker-packs/%s", a.Number), data)
	return
}

type Contact struct {
	Blocked           bool   `json:"blocked"`
	Color             string `json:"color"`
	MessageExpiration string `json:"message_expiration"`
	Name              string `json:"name"`
	Number            string `json:"number"`
	ProfileName       string `json:"profile_name"`
	Username          string `json:"username"`
	UUID              string `json:"uuid"`
}

// List Contacts.
//
// List all contacts for the given number.
func (a *Account) GetContacts() (contacts []Contact, err error) {
	return get[[]Contact](fmt.Sprintf("/v1/contacts/%s", a.Number))
}

// Updates the info associated to a number on the contact list. If the contact doesn’t exist yet, it will be added.
func (a *Account) PostContact(data struct {
	ExpirationInSeconds int    `json:"expiration_in_seconds"`
	Name                string `json:"name"`
	Recipient           string `json:"recipient"`
}) (contacts []Contact, err error) {
	_, err = post[any](fmt.Sprintf("/v1/contacts/%s", a.Number), data)
	return
}

// Send a synchronization message with the local contacts list to all linked devices. This command should only be used if this is the primary device.
func (a *Account) PutContactsSync() (err error) {
	_, err = put[any](fmt.Sprintf("/v1/contacts/%s/sync", a.Number), nil)
	return
}
