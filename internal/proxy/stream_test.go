package proxy

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestSSEScanner_MultipleDeltas(t *testing.T) {
	data := strings.Join([]string{
		`data: {"choices":[{"delta":{"content":"hel"}}]}`,
		`data: {"choices":[{"delta":{"content":"lo"}}]}`,
		`data: [DONE]`,
	}, "\n") + "\n"
	scanner := NewSSEScanner(bytes.NewBufferString(data))

	var deltas []string
	for scanner.Scan() {
		deltas = append(deltas, scanner.LastDelta())
	}

	if len(deltas) != 2 {
		t.Fatalf("got %d deltas, want 2: %v", len(deltas), deltas)
	}
	if deltas[0] != "hel" || deltas[1] != "lo" {
		t.Errorf("deltas = %v, want [hel lo]", deltas)
	}
	if !scanner.Completed() {
		t.Error("Completed() = false, want true after [DONE]")
	}
}

func TestSSEScanner_ChunkMeta(t *testing.T) {
	line := `data: {"id":"chatcmpl-abc","model":"gpt-4","choices":[{"finish_reason":null,"delta":{"content":"x"}}]}`
	scanner := NewSSEScanner(bytes.NewBufferString(line + "\n"))

	if !scanner.Scan() {
		t.Fatal("Scan() = false, want true")
	}
	if scanner.FirstChunkID() != "chatcmpl-abc" {
		t.Errorf("FirstChunkID() = %q, want chatcmpl-abc", scanner.FirstChunkID())
	}
	if scanner.FirstChunkModel() != "gpt-4" {
		t.Errorf("FirstChunkModel() = %q, want gpt-4", scanner.FirstChunkModel())
	}
}

func TestSSEScanner_FinishReason(t *testing.T) {
	data := strings.Join([]string{
		`data: {"choices":[{"delta":{"content":"done"}}]}`,
		`data: {"choices":[{"finish_reason":"stop","delta":{}}]}`,
		`data: [DONE]`,
	}, "\n") + "\n"
	scanner := NewSSEScanner(bytes.NewBufferString(data))

	for scanner.Scan() {
	}
	if scanner.LastFinishReason() != "stop" {
		t.Errorf("LastFinishReason() = %q, want stop", scanner.LastFinishReason())
	}
}

func TestSSEScanner_SkipsNonDataLines(t *testing.T) {
	data := ": comment\n\nevent: ping\n\ndata: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\ndata: [DONE]\n\n"
	scanner := NewSSEScanner(bytes.NewBufferString(data))

	var deltas []string
	for scanner.Scan() {
		deltas = append(deltas, scanner.LastDelta())
	}
	if len(deltas) != 1 || deltas[0] != "ok" {
		t.Errorf("deltas = %v, want [ok]", deltas)
	}
}

func TestSSEScanner_PartialLineAcrossReads(t *testing.T) {
	payload := []byte("data: {\"choices\":[{\"delta\":{\"content\":\"split\"}}]}\n")
	scanner := NewSSEScanner(&splitReader{parts: [][]byte{payload[:20], payload[20:]}})

	if !scanner.Scan() {
		t.Fatal("Scan() = false on split read")
	}
	if scanner.LastDelta() != "split" {
		t.Errorf("LastDelta() = %q, want split", scanner.LastDelta())
	}
}

type splitReader struct {
	parts [][]byte
	idx   int
}

func (r *splitReader) Read(p []byte) (int, error) {
	if r.idx >= len(r.parts) {
		return 0, nil
	}
	n := copy(p, r.parts[r.idx])
	r.idx++
	return n, nil
}

func TestParseChunkMeta(t *testing.T) {
	payload := []byte(`{"id":"id-1","model":"m1","choices":[{"finish_reason":"length"}]}`)
	id, model, reason := parseChunkMeta(payload)
	if id != "id-1" || model != "m1" || reason != "length" {
		t.Errorf("parseChunkMeta = (%q,%q,%q), want (id-1,m1,length)", id, model, reason)
	}
}

func TestParseChunkMeta_Invalid(t *testing.T) {
	id, model, reason := parseChunkMeta([]byte("not-json"))
	if id != "" || model != "" || reason != "" {
		t.Errorf("parseChunkMeta(invalid) = (%q,%q,%q), want empty", id, model, reason)
	}
}

func TestSSEScanner_RawLine(t *testing.T) {
	line := `data: {"choices":[{"delta":{"content":"x"}}]}`
	scanner := NewSSEScanner(bytes.NewBufferString(line + "\n"))
	scanner.Scan()
	if !bytes.Equal(scanner.RawLine(), []byte(line)) {
		t.Errorf("RawLine() = %q, want %q", scanner.RawLine(), line)
	}
}

func TestSSEScannerWithStart(t *testing.T) {
	data := strings.Join([]string{
		`data: {"choices":[{"delta":{"content":"first"}}]}`,
		`data: {"choices":[{"delta":{"content":"second"}}]}`,
		`data: [DONE]`,
	}, "\n") + "\n"

	start := time.Now().Add(-50 * time.Millisecond)
	scanner := NewSSEScannerWithStart(bytes.NewBufferString(data), start)

	if !scanner.Scan() {
		t.Fatal("Scan() = false on first chunk")
	}
	if scanner.TimeToFirstChunkMs() < 45 {
		t.Errorf("TimeToFirstChunkMs() = %dms, want >= 45ms (should include pre-scanner latency)", scanner.TimeToFirstChunkMs())
	}
	if scanner.LastDelta() != "first" {
		t.Errorf("LastDelta() = %q, want first", scanner.LastDelta())
	}
}

func TestNewSSEScannerBackwardCompat(t *testing.T) {
	data := `data: {"choices":[{"delta":{"content":"test"}}]}` + "\n"
	scanner := NewSSEScanner(bytes.NewBufferString(data))

	if !scanner.Scan() {
		t.Fatal("Scan() = false")
	}
	if scanner.TimeToFirstChunkMs() < 0 {
		t.Errorf("TimeToFirstChunkMs() = %dms, want >= 0", scanner.TimeToFirstChunkMs())
	}
}
