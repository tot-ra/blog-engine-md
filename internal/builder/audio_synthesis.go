package builder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"os"
	"os/exec"
	"strings"
	"time"
)

func synthesizeWithEdge(binary, voice, rate, pitch, text, outputPath string) error {
	if strings.TrimSpace(binary) == "" {
		binary = "edge-tts"
	}
	tmp, err := os.CreateTemp("", "blog-audio-*.txt")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.WriteString(text); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	if strings.TrimSpace(rate) == "" {
		rate = "+0%"
	}
	if strings.TrimSpace(pitch) == "" {
		pitch = "+0Hz"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	args := []string{
		"--voice", voice,
		"--rate", rate,
		"--pitch", pitch,
		"--file", tmpPath,
		"--write-media", outputPath,
	}
	cmd := exec.CommandContext(ctx, binary, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}

func synthesizeWithElevenLabs(
	baseURL, apiKey, voiceID, modelID, outputFormat string,
	stability, similarityBoost, style float64,
	speakerBoost bool,
	text, outputPath string,
) error {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = "https://api.elevenlabs.io"
	}
	if strings.TrimSpace(apiKey) == "" {
		return fmt.Errorf("empty elevenlabs api key")
	}
	if strings.TrimSpace(voiceID) == "" {
		return fmt.Errorf("empty elevenlabs voice id")
	}
	if strings.TrimSpace(modelID) == "" {
		modelID = "eleven_multilingual_v2"
	}
	if strings.TrimSpace(outputFormat) == "" {
		outputFormat = "mp3_44100_128"
	}

	const elevenLabsMaxRunesPerRequest = 9500
	chunks := splitTextIntoChunks(text, elevenLabsMaxRunesPerRequest)
	if len(chunks) == 0 {
		return fmt.Errorf("empty text for elevenlabs synthesis")
	}

	endpoint := fmt.Sprintf(
		"%s/v1/text-to-speech/%s?output_format=%s",
		baseURL,
		neturl.PathEscape(strings.TrimSpace(voiceID)),
		neturl.QueryEscape(outputFormat),
	)

	type voiceSettings struct {
		Stability       float64 `json:"stability"`
		SimilarityBoost float64 `json:"similarity_boost"`
		Style           float64 `json:"style,omitempty"`
		SpeakerBoost    bool    `json:"use_speaker_boost"`
	}
	var merged bytes.Buffer
	for idx, chunk := range chunks {
		payload := map[string]any{
			"text":     chunk,
			"model_id": modelID,
			"voice_settings": voiceSettings{
				Stability:       clamp01(stability),
				SimilarityBoost: clamp01(similarityBoost),
				Style:           clamp01(style),
				SpeakerBoost:    speakerBoost,
			},
		}

		audioBytes, err := requestElevenLabsAudio(endpoint, apiKey, payload)
		if err != nil {
			return fmt.Errorf("chunk %d/%d: %w", idx+1, len(chunks), err)
		}
		if _, err := merged.Write(audioBytes); err != nil {
			return err
		}
	}

	if merged.Len() == 0 {
		return fmt.Errorf("empty audio response from elevenlabs")
	}
	return os.WriteFile(outputPath, merged.Bytes(), 0644)
}

func requestElevenLabsAudio(endpoint, apiKey string, payload map[string]any) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("xi-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "audio/mpeg")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("elevenlabs %s: %s", resp.Status, strings.TrimSpace(string(errBody)))
	}

	audioBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if len(audioBytes) == 0 {
		return nil, fmt.Errorf("empty audio response from elevenlabs")
	}
	return audioBytes, nil
}
