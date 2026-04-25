package domain

// Messenger sends messages to users via a messaging platform.
type Messenger interface {
	SendText(recipientID, text string) error
	SendAudio(recipientID, audioURL string) error
	MarkSeen(recipientID string) error
	SetTypingOn(recipientID string) error
	DownloadMedia(mediaURL string) ([]byte, error)
}

// AIEngine processes user messages and returns AI-generated responses.
// Reply.AudioScript, when non-empty, instructs the transport layer to
// synthesize and send an audio message before the text reply.
type AIEngine interface {
	Process(senderID, text string) (Reply, error)
}

// Reply bundles the AI's textual response with an optional audio script.
type Reply struct {
	Text        string
	AudioScript string
}

// VoiceService handles speech-to-text and text-to-speech operations.
type VoiceService interface {
	Enabled() bool
	SpeechToText(audioData []byte) (string, error)
	TextToSpeech(text string) ([]byte, error)
}

// KnowledgeBase provides contextual business information for AI prompts.
type KnowledgeBase interface {
	Enabled() bool
	FormatContext() string
}

// AudioStore manages temporary audio file storage with auto-expiry.
type AudioStore interface {
	Put(data []byte) string
	Get(id string) ([]byte, bool)
}
