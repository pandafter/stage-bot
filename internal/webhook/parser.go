package webhook

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
