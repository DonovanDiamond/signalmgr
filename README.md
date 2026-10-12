# Signal Manager

[![GoDoc](https://pkg.go.dev/badge/github.com/DonovanDiamond/signalmgr.svg)](https://pkg.go.dev/github.com/DonovanDiamond/signalmgr)

`signalmgr` is a Go library designed to interact with the [bbernhard/signal-cli-rest-api](https://github.com/bbernhard/signal-cli-rest-api) to manage Signal accounts more easily. This library provides a wrapper for the various signal-cli-rest-api endpoints covered in [signal-cli-rest-api swagger docs](https://bbernhard.github.io/signal-cli-rest-api/).

## Installation

You can install the `signalmgr` library using `go get`:

```bash
go get github.com/DonovanDiamond/signalmgr
```

## Usage

```go
package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/DonovanDiamond/signalmgr"
)

func main() {
	signalmgr.API_URL = "http://localhost:8080"

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("Enter the number to register to signal: ")
	scanner.Scan()
	number := scanner.Text()

	account := signalmgr.Account{
		Number: number,
	}

	fmt.Print("Enter 'voice' if you want to use a phone call for verification (note this will take longer than SMS verification): ")
	scanner.Scan()
	voice := strings.ToLower(scanner.Text())

	fmt.Println("Go to https://signalcaptchas.org/registration/generate.html, complete the captcha, open the development console and find the line that looks like: 'Prevented navigation to “signalcaptcha://{captcha value}” due to an unknown protocol.' and copy the entire captcha value.")
	fmt.Print("Enter the captcha value including “signalcaptcha://”: ")
	scanner.Scan()
	captcha := scanner.Text()

	if voice == "voice" {
		err := account.PostRegister(captcha, false)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Waiting 61 seconds before requesting voice call...")
		time.Sleep(time.Second * 61)

		fmt.Println("Go to https://signalcaptchas.org/registration/generate.html and complete a new captcha like before.")
		fmt.Print("Enter the captcha value including “signalcaptcha://”: ")
		scanner.Scan()
		captcha := scanner.Text()
	
		err = account.PostRegister(captcha, true)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		err := account.PostRegister(captcha, voice == "voice")
		if err != nil {
			log.Fatal(err)
		}
	}

	fmt.Print("Please enter the token sent via SMS/call to the number provided:")
	scanner.Scan()
	token := scanner.Text()
	fmt.Print("Please enter the PIN for signal (if you have one) or leave this blank:")
	scanner.Scan()
	pin := scanner.Text()

	err := account.PostRegisterVerify(token, pin)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Successfully registered!")

	fmt.Print("Enter a number to send a test message to (formatted +123456789):")
	scanner.Scan()
	numberTo := scanner.Text()
	fmt.Print("Enter a message to send:")
	scanner.Scan()
	message := scanner.Text()

	if _, err := signalmgr.PostSend(signalmgr.SendMessageV2{
		Number:     account.Number,
		Recipients: []string{numberTo},
		Message:    message,
	}); err != nil {
		log.Fatal(err)
	}

	fmt.Print("Send a message to this number and press enter to receive it:")
	scanner.Scan()

	messages, err := account.GetMessages(signalmgr.GetMessagesOptions{})
	if err != nil {
		log.Fatal(err)
	}

	for _, m := range messages {
		fmt.Printf(" - %s: %s\n", m.Envelope.SourceNumber, m.Envelope.DataMessage.Message)
	}
}
```

## Methods

Each method is based on the methods in [signal-cli-rest-api swagger docs](https://bbernhard.github.io/signal-cli-rest-api/). The targeted API version is signal-cli-rest-api 0.101. Methods marked `(a)` are on `*Account`; the rest are package-level. A summary:

### General

- `GetAbout() (GetAboutResponse, error)`: API version and build information.
- `GetHealth() error`: Check the health of the Signal API.
- `GetConfiguration() (Configuration, error)`: Retrieve the current API configuration.
- `PostConfiguration(data Configuration) error`: Update the API configuration.
- `GetLinkAccountQRCode(deviceName string, opts GetLinkAccountQRCodeOptions) ([]byte, error)`: Generate a QR code (PNG) to link a new device. `opts.QRCodeVersion` 0 uses the server default.
- `GetLinkAccountURI(deviceName string) (string, error)`: Get the raw device-link URI.
- `PostSendV1(data SendMessageV1) error`: Deprecated v1 send, use `PostSend`.

### Account Management

- `GetAccounts() ([]Account, error)`: Lists all Signal accounts linked to the service.
- (a) `GetConfiguration() (Account_Configuration, error)`: Retrieve the account configuration.
- (a) `PostConfiguration(data Account_Configuration) error`: Update the account configuration.
- (a) `PostRegister(captcha string, useVoice bool) error`: Register a phone number with the Signal network.
- (a) `PostRegisterVerify(token string, pin string) error`: Verify a registered phone number.
- (a) `PostUnregister(data struct{ DeleteAccount bool; DeleteLocalData bool }) error`: Unregister a phone number and optionally delete its data.
- (a) `PostRateLimitChallenge(data struct{ Captcha string; ChallengeToken string }) error`: Lift rate limit restrictions by solving a captcha.
- (a) `PutSettings(data struct{ DiscoverableByNumber *bool; ShareNumber *bool }) error`: Update the account settings.
- (a) `PostUsername(data struct{ Username string }) (Account_PostUsernameResponse, error)`: Set a username for the account.
- (a) `DeleteUsername() error`: Remove the username.
- (a) `PutProfile(data struct{ About string; Base64Avatar string; Name string }) error`: Update the profile name, about text and avatar.
- (a) `PostPin(data struct{ Pin string }) error`: Set a registration lock PIN.
- (a) `DeletePin() error`: Remove the registration lock PIN.
- (a) `GetIdentities() ([]Identity, error)`: List known identities.
- (a) `PutTrustIdentity(numberToTrust string, data struct{ TrustAllKnownKeys bool; VerifiedSafetyNumber string }) error`: Trust an identity.

### Devices

- (a) `GetDevices() ([]Device, error)`: List devices linked to the account.
- (a) `PostLinkDevice(data struct{ URI string }) error`: Link another device to this account.
- (a) `DeleteDevice(deviceID int64) error`: Unlink a device.
- (a) `DeleteLocalData(data struct{ IgnoreRegistered bool }) error`: Delete the locally stored account data.

### Messaging

- `PostSend(data SendMessageV2) (SendMessageResponse, error)`: Send a message (supports text, mentions, attachments, etc.).
- (a) `GetMessages(opts GetMessagesOptions) ([]MessageResponse, error)`: Receive Signal Messages, when [bbernhard/signal-cli-rest-api](https://github.com/bbernhard/signal-cli-rest-api) is running in `normal` or `native` mode.
- (a) `GetMessagesSocket(messages chan<- MessageResponse) error`: Opens a socket to receive Signal Messages and sends them to the `messages` channel, when [bbernhard/signal-cli-rest-api](https://github.com/bbernhard/signal-cli-rest-api) is running in `json-rpc` mode.
- (a) `PostReaction(data struct{ Reaction string; Recipient string; TargetAuthor string; Timestamp int64 }) error`: Send a reaction to a message.
- (a) `DeleteReaction(data struct{ Reaction string; Recipient string; TargetAuthor string; Timestamp int64 }) error`: Remove a reaction from a message.
- (a) `PostReceipts(data struct{ ReceiptType string; Recipient string; Timestamp int64 }) error`: Send a read/viewed receipt for a message.
- (a) `DeleteRemoteMessage(data struct{ Recipient string; Timestamp int64 }) (Account_DeleteRemoteMessageResponse, error)`: Remote-delete a sent message.
- (a) `PutTypingIndicator(data struct{ Recipient string }) error`: Show the typing indicator.
- (a) `DeleteTypingIndicator(data struct{ Recipient string }) error`: Hide the typing indicator.

### Polls

- (a) `PostPoll(data struct{ AllowMultipleSelections *bool; Answers []string; Question string; Recipient string }) (Account_PostPollResponse, error)`: Create a poll.
- (a) `DeletePoll(data struct{ PollTimestamp string; Recipient string }) error`: Close a poll.
- (a) `PostPollVote(data struct{ PollAuthor string; PollTimestamp string; Recipient string; SelectedAnswers []int }) error`: Vote in a poll.

### Contacts

- (a) `GetContacts(opts GetContactsOptions) ([]Contact, error)`: List all contacts for the account. `opts.AllRecipients` includes all known recipients.
- (a) `GetContact(uuid string, opts GetContactOptions) (Contact, error)`: Get a specific contact.
- (a) `GetContactAvatar(uuid string) ([]byte, error)`: Get a contact's avatar.
- (a) `PutContact(data struct{ ExpirationInSeconds int; Name string; Recipient string }) error`: Add or update contact information.
- (a) `PostContactsSync() error`: Sync contacts across devices.
- (a) `GetSearch(numbers []string) ([]SearchResult, error)`: Check whether phone numbers are registered with Signal.

### Groups

- (a) `GetGroups(opts GetGroupsOptions) ([]Group, error)`: List all Signal groups associated with the account. `opts.Expand` returns expanded group details.
- (a) `PostCreateGroup(data struct{ Description string; ExpirationTime int; GroupLink string; Members []string; Name string; Permissions *GroupPermissions }) (struct{ ID string }, error)`: Create a new group with specified members.
- (a) `GetGroup(groupID string, opts GetGroupOptions) (Group, error)`: Get a specific group. `opts.Expand` returns expanded group details.
- (a) `GetGroupAvatar(groupID string) ([]byte, error)`: Get a group's avatar.
- (a) `PutGroupSettings(groupID string, data struct{ Base64Avatar *string; Description *string; ExpirationTime *int; GroupLink *string; Name *string; Permissions *GroupPermissions }) error`: Update a group.
- (a) `DeleteGroup(groupID string) error`: Delete a group.
- (a) `PostGroupAdmins(groupID string, data struct{ Admins []string }) error`: Add admins to a group.
- (a) `DeleteGroupAdmins(groupID string, data struct{ Admins []string }) error`: Remove admins from a group.
- (a) `PostGroupMembers(groupID string, data struct{ Members []string }) error`: Add members to a group.
- (a) `DeleteGroupMembers(groupID string, data struct{ Members []string }) error`: Remove members from a group.
- (a) `PostGroupPinMessage(groupID string, data struct{ Duration *int64; TargetAuthor string; Timestamp int64 }) error`: Pin a message in a group.
- (a) `DeleteGroupPinMessage(groupID string, data struct{ TargetAuthor string; Timestamp int64 }) error`: Unpin a message in a group.
- (a) `PostBlockGroup(groupID string) error`: Block a group.
- (a) `PostJoinGroup(groupID string) error`: Join a group.
- (a) `PostQuitGroup(groupID string) error`: Quit a group.

### Stickers

- (a) `GetStickerPacks() ([]StickerPack, error)`: List installed sticker packs.
- (a) `PostStickerPack(data struct{ PackID string; PackKey string }) error`: Install a sticker pack.

### Attachments

- `GetAttachments() ([]string, error)`: List all attachments stored in the system.
- `GetAttachment(id string) ([]byte, error)`: Serve an attachment by its ID.
- `DeleteAttachment(id string) error`: Delete an attachment by its ID.

### Errors

Non-2xx responses are returned as `*signalmgr.APIError`:

```go
var apiErr *signalmgr.APIError
if errors.As(err, &apiErr) { fmt.Println(apiErr.Status, apiErr.ChallengeTokens) }
```

## Releasing

The library and the Signal Gateway in `gateway/` share one version: release `vX.Y.Z` is the tag `vX.Y.Z` (library) plus `gateway/vX.Y.Z` (gateway, which requires the library at that same version). Git tags are the only source of truth. `make release VERSION=vX.Y.Z` checks that you are on a clean `main` and that the version is new and the highest, tags and pushes the library, pins `gateway/go.mod` to it, then commits, tags and pushes the gateway. `make -n release VERSION=vX.Y.Z` shows the steps without running them. Day to day, the committed `go.work` makes `gateway/` build against the local library source.

## Docker image

`make image` builds the gateway image as `$(IMAGE):<version>` (default `IMAGE=signal-gateway`), where the version comes from `git describe` on the `gateway/v*` tags, e.g. `v0.4.0` on a tagged clean commit and `v0.4.0-3-gabc123` otherwise. `:latest` is added only for an exact release. Docker runs through `$(DOCKER)` (default `sudo docker`). `make version` prints the computed version; `make help` lists all targets.

## Related Projects

- [signal-cli-rest-api](https://github.com/bbernhard/signal-cli-rest-api) - Signal CLI REST API that this library interacts with.
- [signal-cli](https://github.com/AsamK/signal-cli) - A command-line interface for Signal.

## Acknowledgements

Thanks to the contributors of [signal-cli](https://github.com/AsamK/signal-cli) and [signal-cli-rest-api](https://github.com/bbernhard/signal-cli-rest-api) for providing the underlying API and documentation. This project does not replace any functionality of these projects, and is just a wrapper for easy implementation in Go.
