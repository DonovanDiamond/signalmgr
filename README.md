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

	messages, err := account.GetMessages()
	if err != nil {
		log.Fatal(err)
	}

	for _, m := range messages {
		fmt.Printf(" - %s: %s\n", m.Envelope.SourceNumber, m.Envelope.DataMessage.Message)
	}
}
```

## Methods

Each method is based on the methods in [signal-cli-rest-api swagger docs](https://bbernhard.github.io/signal-cli-rest-api/), below is a summary of them:

### Account Management

- `GetAccounts() ([]Account, error)`: Lists all Signal accounts linked to the service.
- `PostRegister(captcha string, useVoice bool) error`: Register a phone number with the Signal network.
- `PostRegisterVerify(token string, pin string) error`: Verify a registered phone number.
- `PostUnregister(data struct{ DeleteAccount bool; DeleteLocalData bool }) error`: Unregister a phone number and optionally delete its data.
- `PostUsername(data struct{ Username string }) (Account_PostUsernameResponse, error)`: Set a username for the account.

### Messaging

- `GetMessages() ([]MessageResponse, error)`: Receive Signal Messages, when [bbernhard/signal-cli-rest-api](https://github.com/bbernhard/signal-cli-rest-api) is running in `normal` or `native` mode.
- `GetMessagesSocket(messages chan<- MessageResponse) error`: Opens a socket to receive Signal Messages and sends them to the `messages` channel, when [bbernhard/signal-cli-rest-api](https://github.com/bbernhard/signal-cli-rest-api) is running in `json-rpc` mode.
- `PostSend(data SendMessageV2) (struct{ Timestamp string }, error)`: Send a message (supports text, mentions, attachments, etc.).
- `PostReaction(data struct{ Reaction string; Recipient string; TargetAuthor string; Timestamp int64 }) error`: Send a reaction to a message.
- `DeleteReaction(data struct{ Reaction string; Recipient string; TargetAuthor string; Timestamp int64 }) error`: Remove a reaction from a message.
- `PostReceipts(data struct{ ReceiptType string; Recipient string; Timestamp int64 }) error`: Send a read/viewed receipt for a message.

### Contacts

- `GetContacts() ([]Contact, error)`: List all contacts for the account.
- `PostContact(data struct{ ExpirationInSeconds int; Name string; Recipient string }) ([]Contact, error)`: Add or update contact information.
- `PutContactsSync() error`: Sync contacts across devices.

### Groups

- `GetGroups() ([]Group, error)`: List all Signal groups associated with the account.
- `PostCreateGroup(data struct{ Description string; ExpirationTime int; GroupLink string; Members []string; Name string; Permissions struct{ AddMembers string; EditGroup string } }) (struct{ ID string }, error)`: Create a new group with specified members.
- `PostGroupAdmins(groupID string, data struct{ Admins []string }) error`: Add admins to a group.
- `DeleteGroupAdmins(groupID string, data struct{ Admins []string }) error`: Remove admins from a group.
- `PostGroupMembers(groupID string, data struct{ Members []string }) error`: Add members to a group.

### Attachments

- `GetAttachments() ([]string, error)`: List all attachments stored in the system.
- `GetAttachment(id string) ([]byte, error)`: Serve an attachment by its ID.
- `DeleteAttachment(id string) error`: Delete an attachment by its ID.

### Device Linking

- `GetLinkAccountQRCode(deviceName string) (string, error)`: Generate a QR code to link a new device.

### Configuration & Health

- `GetHealth() (string, error)`: Check the health of the Signal API.
- `GetConfiguration() (Configuration, error)`: Retrieve the current API configuration.
- `PostConfiguration(data Configuration) error`: Update the API configuration.

## Related Projects

- [signal-cli-rest-api](https://github.com/bbernhard/signal-cli-rest-api) - Signal CLI REST API that this library interacts with.
- [signal-cli](https://github.com/AsamK/signal-cli) - A command-line interface for Signal.

## Acknowledgements

Thanks to the contributors of [signal-cli](https://github.com/AsamK/signal-cli) and [signal-cli-rest-api](https://github.com/bbernhard/signal-cli-rest-api) for providing the underlying API and documentation. This project does not replace any functionality of these projects, and is just a wrapper for easy implementation in Go.
