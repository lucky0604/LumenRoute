package proxy

import (
	"bytes"
	"encoding/json"
	"io"
	"time"
)

// StreamResult holds the outcome of a single stream request.
type StreamResult struct {
	LatencyMs          int64
	TimeToFirstChunkMs int64
	Completed          bool
	ErrorCode          string
	ErrorMessage       string
	UpstreamStatusCode int
	CompletionTokens   int
	CompletionText     string
	PromptTokens       int
	TotalTokens        int
}

// SSEScanner reads server-sent events from an io.Reader and extracts
// content deltas from OpenAI-compatible streaming responses.
type SSEScanner struct {
	reader            io.Reader
	buf               []byte
	rawLine           []byte
	lastDelta         string
	completed         bool
	timeToFirstChunkMs int64
	startTime         time.Time
	firstChunkSeen    bool
	pending           [][]byte
}

func NewSSEScanner(r io.Reader) *SSEScanner {
	return &SSEScanner{reader: r, buf: make([]byte, 4096), startTime: time.Now()}
}

func (s *SSEScanner) Scan() bool {
	for {
		// Drain pending lines from previous reads first.
		for len(s.pending) > 0 {
			line := s.pending[0]
			s.pending = s.pending[1:]
			if bytes.HasPrefix(line, []byte("data: ")) {
				payload := bytes.TrimPrefix(line, []byte("data: "))
				if bytes.Equal(payload, []byte("[DONE]")) {
					s.completed = true
					return false
				}
				s.rawLine = line
				s.lastDelta = parseDelta(payload)
				if s.lastDelta != "" && !s.firstChunkSeen {
					s.firstChunkSeen = true
					s.timeToFirstChunkMs = time.Since(s.startTime).Milliseconds()
				}
				return true
			}
		}

		n, err := s.reader.Read(s.buf)
		if n == 0 {
			return false
		}
		lines := bytes.Split(s.buf[:n], []byte("\n"))
		s.pending = append(s.pending, lines...)
		if err != nil {
			// Continue draining pending lines on EOF.
			if len(s.pending) > 0 {
				continue
			}
			return false
		}
	}
}

func (s *SSEScanner) RawLine() []byte   { return s.rawLine }
func (s *SSEScanner) LastDelta() string  { return s.lastDelta }
func (s *SSEScanner) Completed() bool    { return s.completed }
func (s *SSEScanner) TimeToFirstChunkMs() int64 { return s.timeToFirstChunkMs }

func parseDelta(payload []byte) string {
	var chunk struct {
		Choices []struct {
			Delta struct {
				Content string `json:"content"`
			} `json:"delta"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(payload, &chunk); err != nil {
		return ""
	}
	if len(chunk.Choices) > 0 {
		return chunk.Choices[0].Delta.Content
	}
	return ""
}
