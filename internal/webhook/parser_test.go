package webhook

import "testing"

func TestParseMessages_TextMessage(t *testing.T) {
	payload := &WebhookPayload{
		Object: "instagram",
		Entry: []Entry{{
			ID:   "123",
			Time: 1700000000,
			Messaging: []Messaging{{
				Sender:    Participant{ID: "sender1"},
				Recipient: Participant{ID: "page1"},
				Timestamp: 1700000000,
				Message: &Message{
					MID:  "mid.123",
					Text: "Hola, quiero info",
				},
			}},
		}},
	}

	msgs := ParseMessages(payload)

	if len(msgs) != 1 {
		t.Fatalf("got %d messages, want 1", len(msgs))
	}

	msg := msgs[0]
	if msg.SenderID != "sender1" {
		t.Errorf("SenderID = %s, want sender1", msg.SenderID)
	}
	if msg.Type != MessageTypeText {
		t.Errorf("Type = %s, want %s", msg.Type, MessageTypeText)
	}
	if msg.Text != "Hola, quiero info" {
		t.Errorf("Text = %q, want %q", msg.Text, "Hola, quiero info")
	}
	if msg.MessageID != "mid.123" {
		t.Errorf("MessageID = %s, want mid.123", msg.MessageID)
	}
}

func TestParseMessages_AudioMessage(t *testing.T) {
	payload := &WebhookPayload{
		Object: "instagram",
		Entry: []Entry{{
			Messaging: []Messaging{{
				Sender:    Participant{ID: "sender1"},
				Recipient: Participant{ID: "page1"},
				Timestamp: 1700000000,
				Message: &Message{
					MID: "mid.456",
					Attachments: []Attachment{{
						Type:    "audio",
						Payload: Payload{URL: "https://cdn.fbsbx.com/audio.mp4"},
					}},
				},
			}},
		}},
	}

	msgs := ParseMessages(payload)

	if len(msgs) != 1 {
		t.Fatalf("got %d messages, want 1", len(msgs))
	}
	if msgs[0].Type != MessageTypeAudio {
		t.Errorf("Type = %s, want %s", msgs[0].Type, MessageTypeAudio)
	}
	if msgs[0].MediaURL != "https://cdn.fbsbx.com/audio.mp4" {
		t.Errorf("MediaURL = %s, want audio URL", msgs[0].MediaURL)
	}
}

func TestParseMessages_ImageMessage(t *testing.T) {
	payload := &WebhookPayload{
		Object: "instagram",
		Entry: []Entry{{
			Messaging: []Messaging{{
				Sender:    Participant{ID: "sender1"},
				Recipient: Participant{ID: "page1"},
				Message: &Message{
					MID: "mid.789",
					Attachments: []Attachment{{
						Type:    "image",
						Payload: Payload{URL: "https://cdn.fbsbx.com/img.jpg"},
					}},
				},
			}},
		}},
	}

	msgs := ParseMessages(payload)

	if len(msgs) != 1 {
		t.Fatalf("got %d messages, want 1", len(msgs))
	}
	if msgs[0].Type != MessageTypeImage {
		t.Errorf("Type = %s, want %s", msgs[0].Type, MessageTypeImage)
	}
}

func TestParseMessages_SkipsEchoMessages(t *testing.T) {
	payload := &WebhookPayload{
		Entry: []Entry{{
			Messaging: []Messaging{{
				Sender:    Participant{ID: "page1"},
				Recipient: Participant{ID: "sender1"},
				Message: &Message{
					MID:    "mid.echo",
					Text:   "echo message",
					IsEcho: true,
				},
			}},
		}},
	}

	msgs := ParseMessages(payload)

	if len(msgs) != 0 {
		t.Errorf("got %d messages, want 0 (echo should be skipped)", len(msgs))
	}
}

func TestParseMessages_SkipsReadReceipts(t *testing.T) {
	payload := &WebhookPayload{
		Entry: []Entry{{
			Messaging: []Messaging{{
				Sender:    Participant{ID: "sender1"},
				Recipient: Participant{ID: "page1"},
				Read:      &Read{Watermark: 1700000000},
			}},
		}},
	}

	msgs := ParseMessages(payload)

	if len(msgs) != 0 {
		t.Errorf("got %d messages, want 0 (read receipt should be skipped)", len(msgs))
	}
}

func TestParseMessages_MultipleMessagesInEntry(t *testing.T) {
	payload := &WebhookPayload{
		Entry: []Entry{{
			Messaging: []Messaging{
				{
					Sender:  Participant{ID: "sender1"},
					Message: &Message{MID: "mid.1", Text: "hola"},
				},
				{
					Sender:  Participant{ID: "sender2"},
					Message: &Message{MID: "mid.2", Text: "buenas"},
				},
			},
		}},
	}

	msgs := ParseMessages(payload)

	if len(msgs) != 2 {
		t.Fatalf("got %d messages, want 2", len(msgs))
	}
	if msgs[0].SenderID != "sender1" || msgs[1].SenderID != "sender2" {
		t.Error("sender IDs don't match expected order")
	}
}

func TestParseMessages_EmptyPayload(t *testing.T) {
	payload := &WebhookPayload{}
	msgs := ParseMessages(payload)

	if len(msgs) != 0 {
		t.Errorf("got %d messages from empty payload, want 0", len(msgs))
	}
}

func TestParseMessages_UnknownAttachmentType(t *testing.T) {
	payload := &WebhookPayload{
		Entry: []Entry{{
			Messaging: []Messaging{{
				Sender: Participant{ID: "sender1"},
				Message: &Message{
					MID: "mid.unk",
					Attachments: []Attachment{{
						Type:    "file",
						Payload: Payload{URL: "https://example.com/doc.pdf"},
					}},
				},
			}},
		}},
	}

	msgs := ParseMessages(payload)

	if len(msgs) != 1 {
		t.Fatalf("got %d messages, want 1", len(msgs))
	}
	if msgs[0].Type != MessageTypeUnknown {
		t.Errorf("Type = %s, want %s", msgs[0].Type, MessageTypeUnknown)
	}
}

func TestHasCommentChanges(t *testing.T) {
	payload := &WebhookPayload{
		Entry: []Entry{{
			Changes: []Change{{Field: "comments"}},
		}},
	}

	if !HasCommentChanges(payload) {
		t.Fatal("expected HasCommentChanges to return true")
	}
}

func TestParseCommentTriggers_MatchesLatestMediaAndKeyword(t *testing.T) {
	payload := &WebhookPayload{
		Entry: []Entry{{
			ID:   "ig_business",
			Time: 1700000000,
			Changes: []Change{{
				Field: "comments",
				Value: ChangeValue{
					ID:   "comment.1",
					Text: "Quiero ser piloto",
					From: Participant{ID: "user123"},
					Media: Media{
						ID: "media.latest",
					},
				},
			}},
		}},
	}

	msgs := ParseCommentTriggers(payload, "media.latest", "piloto")
	if len(msgs) != 1 {
		t.Fatalf("got %d messages, want 1", len(msgs))
	}
	if msgs[0].SenderID != "user123" {
		t.Errorf("SenderID = %s, want user123", msgs[0].SenderID)
	}
	if msgs[0].Text != "Quiero ser piloto" {
		t.Errorf("Text = %q, want original comment text", msgs[0].Text)
	}
}

func TestParseCommentTriggers_SkipsWrongMediaOrKeyword(t *testing.T) {
	payload := &WebhookPayload{
		Entry: []Entry{{
			Changes: []Change{
				{
					Field: "comments",
					Value: ChangeValue{
						ID:    "comment.1",
						Text:  "quiero info",
						From:  Participant{ID: "user1"},
						Media: Media{ID: "media.latest"},
					},
				},
				{
					Field: "comments",
					Value: ChangeValue{
						ID:    "comment.2",
						Text:  "piloto",
						From:  Participant{ID: "user2"},
						Media: Media{ID: "media.old"},
					},
				},
			},
		}},
	}

	msgs := ParseCommentTriggers(payload, "media.latest", "piloto")
	if len(msgs) != 0 {
		t.Fatalf("got %d messages, want 0", len(msgs))
	}
}
