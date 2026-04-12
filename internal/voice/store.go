package voice

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// AudioStore holds generated audio files in memory with auto-expiry.
// Implements domain.AudioStore.
type AudioStore struct {
	mu    sync.RWMutex
	files map[string]*audioEntry
}

type audioEntry struct {
	data      []byte
	createdAt time.Time
}

const audioTTL = 5 * time.Minute

func NewAudioStore() *AudioStore {
	s := &AudioStore{files: make(map[string]*audioEntry)}
	go s.cleanup()
	return s
}

// Put stores audio data and returns a unique ID.
func (s *AudioStore) Put(data []byte) string {
	id := randomID()
	s.mu.Lock()
	s.files[id] = &audioEntry{data: data, createdAt: time.Now()}
	s.mu.Unlock()
	return id
}

// Get retrieves audio data by ID.
func (s *AudioStore) Get(id string) ([]byte, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.files[id]
	if !ok {
		return nil, false
	}
	return entry.data, true
}

func (s *AudioStore) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		s.mu.Lock()
		for id, entry := range s.files {
			if time.Since(entry.createdAt) > audioTTL {
				delete(s.files, id)
			}
		}
		s.mu.Unlock()
	}
}

func randomID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
