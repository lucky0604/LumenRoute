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
	reader             io.Reader
	buf                []byte
	rawLine            []byte
	lastDelta          string
	completed          bool
	timeToFirstChunkMs int64
	startTime          time.Time
	firstChunkSeen     bool
	pending            [][]byte
	carry              []byte
	firstChunkID       string
	firstChunkModel    string
	lastFinishReason   string
	chunkMetaParsed    bool
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

				chunkID, chunkModel, finishReason := parseChunkMeta(payload)
				if !s.chunkMetaParsed && (chunkID != "" || chunkModel != "") {
					s.firstChunkID = chunkID
					s.firstChunkModel = chunkModel
					s.chunkMetaParsed = true
				}
				if finishReason != "" {
					s.lastFinishReason = finishReason
				}

				if s.lastDelta != "" && !s.firstChunkSeen {
					s.firstChunkSeen = true
					s.timeToFirstChunkMs = time.Since(s.startTime).Milliseconds()
				}
				return true
			}
		}

		n, err := s.reader.Read(s.buf)
		if n == 0 {
			if len(s.carry) > 0 {
				s.pending = append(s.pending, s.carry)
				s.carry = nil
				continue
			}
			return false
		}
		chunk := s.buf[:n]
		if len(s.carry) > 0 {
			chunk = append(s.carry, chunk...)
			s.carry = nil
		}
		lines := bytes.Split(chunk, []byte("\n"))
		if len(lines) > 0 && !bytes.HasSuffix(chunk, []byte("\n")) {
			s.carry = make([]byte, len(lines[len(lines)-1]))
			copy(s.carry, lines[len(lines)-1])
			lines = lines[:len(lines)-1]
		}
		s.pending = append(s.pending, lines...)
		if err != nil {
			if len(s.carry) > 0 {
				s.pending = append(s.pending, s.carry)
				s.carry = nil
			}
			if len(s.pending) > 0 {
				continue
			}
			return false
		}
	}
}

func (s *SSEScanner) RawLine() []byte          { return s.rawLine }
func (s *SSEScanner) LastDelta() string         { return s.lastDelta }
func (s *SSEScanner) Completed() bool           { return s.completed }
func (s *SSEScanner) TimeToFirstChunkMs() int64 { return s.timeToFirstChunkMs }
func (s *SSEScanner) FirstChunkID() string      { return s.firstChunkID }
func (s *SSEScanner) FirstChunkModel() string   { return s.firstChunkModel }
func (s *SSEScanner) LastFinishReason() string  { return s.lastFinishReason }

func parseDelta(payload []byte) string {
	var chunk struct {
		Choices []struct {
			Delta struct {
				Content          string `json:"content"`
				ReasoningContent string `json:"reasoning_content"`
			} `json:"delta"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(payload, &chunk); err != nil {
		return ""
	}
	if len(chunk.Choices) > 0 {
		d := chunk.Choices[0].Delta
		if d.Content != "" {
			return d.Content
		}
		return d.ReasoningContent
	}
	return ""
}

// parseChunkMeta extracts id, model, and finish_reason from an SSE chunk.
func parseChunkMeta(payload []byte) (id, model, finishReason string) {
	var chunk struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Choices []struct {
			FinishReason *string `json:"finish_reason"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(payload, &chunk); err != nil {
		return
	}
	id = chunk.ID
	model = chunk.Model
	if len(chunk.Choices) > 0 && chunk.Choices[0].FinishReason != nil {
		finishReason = *chunk.Choices[0].FinishReason
	}
	return
}
