package webhook

import "strings"

// ParseMessages extracts internal IncomingMessage structs from a webhook payload.
// It filters out echo messages, read receipts, and other non-message events.
func ParseMessages(payload *WebhookPayload) []IncomingMessage {
	var messages []IncomingMessage

	for _, entry := range payload.Entry {
		for _, m := range entry.Messaging {
			// Skip non-message events
			if m.Message == nil {
				continue
			}
			// Skip echo messages (messages sent by the page itself)
			if m.Message.IsEcho {
				continue
			}

			msg := IncomingMessage{
				SenderID:    m.Sender.ID,
				RecipientID: m.Recipient.ID,
				MessageID:   m.Message.MID,
				Timestamp:   m.Timestamp,
			}

			// Determine message type and extract content
			if m.Message.Text != "" {
				msg.Type = MessageTypeText
				msg.Text = m.Message.Text
			} else if len(m.Message.Attachments) > 0 {
				att := m.Message.Attachments[0]
				msg.MediaURL = att.Payload.URL
				switch att.Type {
				case "audio":
					msg.Type = MessageTypeAudio
				case "image":
					msg.Type = MessageTypeImage
				case "video":
					msg.Type = MessageTypeVideo
				default:
					msg.Type = MessageTypeUnknown
				}
			} else {
				msg.Type = MessageTypeUnknown
			}

			messages = append(messages, msg)
		}
	}

	return messages
}

func HasCommentChanges(payload *WebhookPayload) bool {
	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			if change.Field == "comments" {
				return true
			}
		}
	}
	return false
}

func ParseCommentTriggers(payload *WebhookPayload, latestMediaID string, keyword string) []IncomingMessage {
	var messages []IncomingMessage
	if latestMediaID == "" || strings.TrimSpace(keyword) == "" {
		return messages
	}

	keyword = strings.ToLower(strings.TrimSpace(keyword))

	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			if change.Field != "comments" {
				continue
			}
			if change.Value.Media.ID == "" || change.Value.Media.ID != latestMediaID {
				continue
			}
			text := strings.TrimSpace(change.Value.Text)
			if text == "" {
				continue
			}
			if !containsWord(text, keyword) {
				continue
			}

			msg := IncomingMessage{
				SenderID:    change.Value.From.ID,
				RecipientID: entry.ID,
				MessageID:   change.Value.ID,
				Timestamp:   entry.Time,
				Type:        MessageTypeText,
				Text:        text,
			}

			messages = append(messages, msg)
		}
	}

	return messages
}

func containsWord(text, word string) bool {
	for _, token := range strings.Fields(strings.ToLower(text)) {
		normalized := strings.Trim(token, ".,;:!?¡¿\"'()[]{}")
		if normalized == word {
			return true
		}
	}
	return false
}
