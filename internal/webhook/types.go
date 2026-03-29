package webhook

// Instagram Webhook payload types
// Reference: https://developers.facebook.com/docs/messenger-platform/instagram/features/webhook

type WebhookPayload struct {
	Object string  `json:"object"`
	Entry  []Entry `json:"entry"`
}

type Entry struct {
	ID        string      `json:"id"`
	Time      int64       `json:"time"`
	Messaging []Messaging `json:"messaging"`
}

type Messaging struct {
	Sender    Participant `json:"sender"`
	Recipient Participant `json:"recipient"`
	Timestamp int64       `json:"timestamp"`
	Message   *Message    `json:"message,omitempty"`
	Read      *Read       `json:"read,omitempty"`
	Postback  *Postback   `json:"postback,omitempty"`
}

type Participant struct {
	ID string `json:"id"`
}

type Message struct {
	MID         string       `json:"mid"`
	Text        string       `json:"text,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
	QuickReply  *QuickReply  `json:"quick_reply,omitempty"`
	IsEcho      bool         `json:"is_echo,omitempty"`
	ReplyTo     *ReplyTo     `json:"reply_to,omitempty"`
}

type Attachment struct {
	Type    string  `json:"type"` // image, audio, video, file
	Payload Payload `json:"payload"`
}

type Payload struct {
	URL string `json:"url,omitempty"`
}

type QuickReply struct {
	Payload string `json:"payload"`
}

type Read struct {
	Watermark int64 `json:"watermark"`
}

type Postback struct {
	Title   string `json:"title"`
	Payload string `json:"payload"`
}

type ReplyTo struct {
	MID string `json:"mid"`
}

// Internal message representation
type IncomingMessage struct {
	SenderID    string
	RecipientID string
	MessageID   string
	Timestamp   int64
	Type        MessageType
	Text        string
	MediaURL    string
}

type MessageType string

const (
	MessageTypeText    MessageType = "text"
	MessageTypeAudio   MessageType = "audio"
	MessageTypeImage   MessageType = "image"
	MessageTypeVideo   MessageType = "video"
	MessageTypeSticker MessageType = "sticker"
	MessageTypeUnknown MessageType = "unknown"
)
