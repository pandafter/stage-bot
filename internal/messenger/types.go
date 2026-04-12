package messenger

// Send API types for Instagram Graph API
// Reference: https://developers.facebook.com/docs/instagram-messaging/send-api

type SendRequest struct {
	Recipient Recipient `json:"recipient"`
	Message   SendMsg   `json:"message"`
}

type Recipient struct {
	ID string `json:"id"`
}

type SendMsg struct {
	Text         string       `json:"text,omitempty"`
	Attachment   *SendAttach  `json:"attachment,omitempty"`
	QuickReplies []QuickReply `json:"quick_replies,omitempty"`
}

type SendAttach struct {
	Type    string      `json:"type"`
	Payload SendPayload `json:"payload"`
}

type SendPayload struct {
	URL        string `json:"url,omitempty"`
	IsReusable bool   `json:"is_reusable,omitempty"`
}

type QuickReply struct {
	ContentType string `json:"content_type"`
	Title       string `json:"title"`
	Payload     string `json:"payload"`
}

type SenderActionRequest struct {
	Recipient    Recipient `json:"recipient"`
	SenderAction string    `json:"sender_action"`
}

type SendResponse struct {
	RecipientID string `json:"recipient_id"`
	MessageID   string `json:"message_id"`
}

type APIError struct {
	Error struct {
		Message   string `json:"message"`
		Type      string `json:"type"`
		Code      int    `json:"code"`
		FBTraceID string `json:"fbtrace_id"`
	} `json:"error"`
}
