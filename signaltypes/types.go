package signaltypes

type Payment struct {
	Note    string `json:"note,omitempty"`
	Receipt []byte `json:"receipt,omitempty"`
}

type StoryContext struct {
	AuthorNumber  string `json:"authorNumber,omitempty"`
	AuthorUuid    string `json:"authorUuid,omitempty"`
	SentTimestamp int64  `json:"sentTimestamp,omitempty"`
}

type GroupInfo struct {
	GroupId   string `json:"groupId,omitempty"`
	GroupName string `json:"groupName,omitempty"`
	Revision  int    `json:"revision,omitempty"`
	Type      string `json:"type,omitempty"`
}

type TextStyle struct {
	Style  string `json:"style,omitempty"`
	Start  int    `json:"start,omitempty"`
	Length int    `json:"length,omitempty"`
}

type ContactName struct {
	Nickname string `json:"nickname,omitempty"`
	Given    string `json:"given,omitempty"`
	Family   string `json:"family,omitempty"`
	Prefix   string `json:"prefix,omitempty"`
	Suffix   string `json:"suffix,omitempty"`
	Middle   string `json:"middle,omitempty"`
}

type ContactAddress struct {
	Type         string `json:"type,omitempty"`
	Label        string `json:"label,omitempty"`
	Street       string `json:"street,omitempty"`
	Pobox        string `json:"pobox,omitempty"`
	Neighborhood string `json:"neighborhood,omitempty"`
	City         string `json:"city,omitempty"`
	Region       string `json:"region,omitempty"`
	Postcode     string `json:"postcode,omitempty"`
	Country      string `json:"country,omitempty"`
}

type ContactEmail struct {
	Value string `json:"value,omitempty"`
	Type  string `json:"type,omitempty"`
	Label string `json:"label,omitempty"`
}

type ContactPhone struct {
	Value string `json:"value,omitempty"`
	Type  string `json:"type,omitempty"`
	Label string `json:"label,omitempty"`
}

type ContactAvatar struct {
	Attachment Attachment `json:"attachment,omitempty"`
	IsProfile  bool       `json:"isProfile,omitempty"`
}

type SharedContact struct {
	Name         ContactName      `json:"name,omitempty"`
	Avatar       ContactAvatar    `json:"avatar,omitempty"`
	Phone        []ContactPhone   `json:"phone,omitempty"`
	Email        []ContactEmail   `json:"email,omitempty"`
	Address      []ContactAddress `json:"address,omitempty"`
	Organization string           `json:"organization,omitempty"`
}

type RemoteDelete struct {
	Timestamp int64 `json:"timestamp,omitempty"`
}

type Sticker struct {
	PackId    string `json:"packId,omitempty"`
	StickerId int    `json:"stickerId,omitempty"`
}

type QuotedAttachment struct {
	ContentType string     `json:"contentType,omitempty"`
	Filename    string     `json:"filename,omitempty"`
	Thumbnail   Attachment `json:"thumbnail,omitempty"`
}

type Attachment struct {
	ContentType     string `json:"contentType,omitempty"`
	Filename        string `json:"filename,omitempty"`
	Id              string `json:"id,omitempty"`
	Size            int64  `json:"size,omitempty"`
	Width           int    `json:"width,omitempty"`
	Height          int    `json:"height,omitempty"`
	Caption         string `json:"caption,omitempty"`
	UploadTimestamp int64  `json:"uploadTimestamp,omitempty"`
}

type Preview struct {
	Url         string     `json:"url,omitempty"`
	Title       string     `json:"title,omitempty"`
	Description string     `json:"description,omitempty"`
	Image       Attachment `json:"image,omitempty"`
}

type Mention struct {
	// Deprecated: Use Number and Uuid instead
	Name   string `json:"name,omitempty"`
	Number string `json:"number,omitempty"`
	Uuid   string `json:"uuid,omitempty"`
	Start  int    `json:"start,omitempty"`
	Length int    `json:"length,omitempty"`
}

type Quote struct {
	Id int64 `json:"id,omitempty"`
	// Deprecated: Use AuthorNumber and AuthorUuid instead
	Author       string             `json:"author,omitempty"`
	AuthorNumber string             `json:"authorNumber,omitempty"`
	AuthorUuid   string             `json:"authorUuid,omitempty"`
	Text         string             `json:"text,omitempty"`
	Mentions     []Mention          `json:"mentions,omitempty"`
	Attachments  []QuotedAttachment `json:"attachments,omitempty"`
	TextStyles   []TextStyle        `json:"textStyles,omitempty"`
}

type Reaction struct {
	Emoji string `json:"emoji,omitempty"`
	// Deprecated: Use TargetAuthorNumber and TargetAuthorUuid instead
	TargetAuthor        string `json:"targetAuthor,omitempty"`
	TargetAuthorNumber  string `json:"targetAuthorNumber,omitempty"`
	TargetAuthorUuid    string `json:"targetAuthorUuid,omitempty"`
	TargetSentTimestamp int64  `json:"targetSentTimestamp,omitempty"`
	IsRemove            bool   `json:"isRemove,omitempty"`
}

type DataMessage struct {
	Timestamp        int64           `json:"timestamp,omitempty"`
	Message          string          `json:"message,omitempty"`
	ExpiresInSeconds int             `json:"expiresInSeconds,omitempty"`
	ViewOnce         bool            `json:"viewOnce,omitempty"`
	Reaction         Reaction        `json:"reaction,omitempty"`
	Quote            Quote           `json:"quote,omitempty"`
	Payment          Payment         `json:"payment,omitempty"`
	Mentions         []Mention       `json:"mentions,omitempty"`
	Previews         []Preview       `json:"previews,omitempty"`
	Attachments      []Attachment    `json:"attachments,omitempty"`
	Sticker          Sticker         `json:"sticker,omitempty"`
	RemoteDelete     RemoteDelete    `json:"remoteDelete,omitempty"`
	Contacts         []SharedContact `json:"contacts,omitempty"`
	TextStyles       []TextStyle     `json:"textStyles,omitempty"`
	GroupInfo        GroupInfo       `json:"groupInfo,omitempty"`
	StoryContext     StoryContext    `json:"storyContext,omitempty"`
}

type EditMessage struct {
	TargetSentTimestamp int64       `json:"targetSentTimestamp,omitempty"`
	DataMessage         DataMessage `json:"dataMessage,omitempty"`
}

type StoryMessage struct {
	AllowsReplies  bool       `json:"allowsReplies,omitempty"`
	GroupId        string     `json:"groupId,omitempty"`
	FileAttachment Attachment `json:"fileAttachment,omitempty"`
	TextAttachment struct {
		Text                string  `json:"text,omitempty"`
		Style               string  `json:"style,omitempty"`
		TextForegroundColor string  `json:"textForegroundColor,omitempty"`
		TextBackgroundColor string  `json:"textBackgroundColor,omitempty"`
		Preview             Preview `json:"preview,omitempty"`
		BackgroundGradient  struct {
			StartColor string    `json:"startColor,omitempty"`
			EndColor   string    `json:"endColor,omitempty"`
			Colors     []string  `json:"colors,omitempty"`
			Positions  []float64 `json:"positions,omitempty"`
			Angle      int       `json:"angle,omitempty"`
		} `json:"backgroundGradient,omitempty"`
		BackgroundColor string `json:"backgroundColor,omitempty"`
	} `json:"textAttachment,omitempty"`
}

type SyncDataMessage struct {
	// Deprecated: Use DestinationNumber and DestinationUuid instead
	Destination       string      `json:"destination,omitempty"`
	DestinationNumber string      `json:"destinationNumber,omitempty"`
	DestinationUuid   string      `json:"destinationUuid,omitempty"`
	EditMessage       EditMessage `json:"editMessage,omitempty"`
	DataMessage       DataMessage `json:"dataMessage,omitempty"`
}

type SyncStoryMessage struct {
	DestinationNumber string       `json:"destinationNumber,omitempty"`
	DestinationUuid   string       `json:"destinationUuid,omitempty"`
	DataMessage       StoryMessage `json:"dataMessage,omitempty"`
}

type SyncReadMessage struct {
	// Deprecated: Use SenderNumber and SenderUuid instead
	Sender       string `json:"sender,omitempty"`
	SenderNumber string `json:"senderNumber,omitempty"`
	SenderUuid   string `json:"senderUuid,omitempty"`
	Timestamp    int64  `json:"timestamp,omitempty"`
}

type SyncMessage struct {
	// TODO: This had to be changed from SyncDataMessage, find out if this is fine.
	SentMessage      DataMessage       `json:"sentMessage,omitempty"`
	SentStoryMessage SyncStoryMessage  `json:"sentStoryMessage,omitempty"`
	BlockedNumbers   []string          `json:"blockedNumbers,omitempty"`
	BlockedGroupIds  []string          `json:"blockedGroupIds,omitempty"`
	ReadMessages     []SyncReadMessage `json:"readMessages,omitempty"`
	// 0 = CONTACTS_SYNC, 1 = GROUPS_SYNC, 3 = REQUEST_SYNC
	Type int `json:"type,omitempty"`
}

type CallMessage struct {
	OfferMessage struct {
		Id     int64  `json:"id,omitempty"`
		Type   string `json:"type,omitempty"`
		Opaque string `json:"opaque,omitempty"`
	} `json:"offerMessage,omitempty"`
	AnswerMessage struct {
		Id     int64  `json:"id,omitempty"`
		Opaque string `json:"opaque,omitempty"`
	} `json:"answerMessage,omitempty"`
	BusyMessage struct {
		Id int64 `json:"id,omitempty"`
	} `json:"busyMessage,omitempty"`
	HangupMessage struct {
		Id       int64  `json:"id,omitempty"`
		Type     string `json:"type,omitempty"`
		DeviceId int    `json:"DeviceId,omitempty"`
	} `json:"hangupMessage,omitempty"`
	IceUpdateMessages []struct {
		Id     int64  `json:"id,omitempty"`
		Opaque string `json:"opaque,omitempty"`
	} `json:"iceUpdateMessages,omitempty"`
}

type ReceiptMessage struct {
	When       int64   `json:"when,omitempty"`
	IsDelivery bool    `json:"isDelivery,omitempty"`
	IsRead     bool    `json:"isRead,omitempty"`
	IsViewed   bool    `json:"isViewed,omitempty"`
	Timestamps []int64 `json:"timestamps,omitempty"`
}

type TypingMessage struct {
	Action    string `json:"action,omitempty"`
	Timestamp int64  `json:"timestamp,omitempty"`
	GroupId   string `json:"groupId,omitempty"`
}

type MessageEnvelope struct {
	// Deprecated: Use SourceNumber, SourceUuid, SourceName and SourceDevice instead
	Source                   string         `json:"source,omitempty"`
	SourceNumber             string         `json:"sourceNumber,omitempty"`
	SourceUuid               string         `json:"sourceUuid,omitempty"`
	SourceName               string         `json:"sourceName,omitempty"`
	SourceDevice             int            `json:"sourceDevice,omitempty"`
	Timestamp                int64          `json:"timestamp,omitempty"`
	ServerReceivedTimestamp  int64          `json:"serverReceivedTimestamp,omitempty"`
	ServerDeliveredTimestamp int64          `json:"serverDeliveredTimestamp,omitempty"`
	DataMessage              DataMessage    `json:"dataMessage,omitempty"`
	EditMessage              EditMessage    `json:"editMessage,omitempty"`
	StoryMessage             StoryMessage   `json:"storyMessage,omitempty"`
	SyncMessage              SyncMessage    `json:"syncMessage,omitempty"`
	CallMessage              CallMessage    `json:"callMessage,omitempty"`
	ReceiptMessage           ReceiptMessage `json:"receiptMessage,omitempty"`
	TypingMessag             TypingMessage  `json:"typingMessage,omitempty"`
}
