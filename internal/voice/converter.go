package voice

import (
	"fmt"
	"os"
	"os/exec"
)

// ConvertMP3toM4A uses ffmpeg to convert MP3 bytes to M4A (AAC).
func ConvertMP3toM4A(mp3Data []byte) ([]byte, error) {
	tmpIn, err := os.CreateTemp("", "tts-*.mp3")
	if err != nil {
		return nil, fmt.Errorf("create temp input: %w", err)
	}
	defer os.Remove(tmpIn.Name())

	if _, err := tmpIn.Write(mp3Data); err != nil {
		tmpIn.Close()
		return nil, fmt.Errorf("write temp input: %w", err)
	}
	tmpIn.Close()

	outPath := tmpIn.Name() + ".m4a"
	defer os.Remove(outPath)

	cmd := exec.Command("ffmpeg", "-y", "-i", tmpIn.Name(), "-c:a", "aac", "-b:a", "64k", outPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("ffmpeg: %w: %s", err, string(output))
	}

	return os.ReadFile(outPath)
}
