package proxy

import (
	"testing"
)

func TestExtractPromptText(t *testing.T) {
	body := []byte(`{"messages":[{"role":"user","content":"hello"},{"role":"system","content":"be helpful"}]}`)
	got := extractPromptText(body)
	want := "user: hello\nsystem: be helpful\n"
	if got != want {
		t.Errorf("extractPromptText() = %q, want %q", got, want)
	}
}

func TestExtractPromptText_InvalidJSON(t *testing.T) {
	if got := extractPromptText([]byte("not json")); got != "" {
		t.Errorf("extractPromptText(invalid) = %q, want empty", got)
	}
}

func TestExtractPromptText_EmptyMessages(t *testing.T) {
	if got := extractPromptText([]byte(`{"messages":[]}`)); got != "" {
		t.Errorf("extractPromptText(empty) = %q, want empty", got)
	}
}

func TestExtractTokenUsage_Full(t *testing.T) {
	body := []byte(`{"usage":{"prompt_tokens":10,"completion_tokens":20,"total_tokens":30}}`)
	info := extractTokenUsage(body)
	if info == nil {
		t.Fatal("extractTokenUsage returned nil")
	}
	if info.PromptTokens == nil || *info.PromptTokens != 10 {
		t.Errorf("PromptTokens = %v, want 10", info.PromptTokens)
	}
	if info.CompletionTokens == nil || *info.CompletionTokens != 20 {
		t.Errorf("CompletionTokens = %v, want 20", info.CompletionTokens)
	}
	if info.TotalTokens == nil || *info.TotalTokens != 30 {
		t.Errorf("TotalTokens = %v, want 30", info.TotalTokens)
	}
}

func TestExtractTokenUsage_Missing(t *testing.T) {
	tests := []struct {
		name string
		body []byte
	}{
		{"invalid json", []byte("bad")},
		{"zero usage", []byte(`{"usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}`)},
		{"no usage field", []byte(`{"choices":[]}`)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if info := extractTokenUsage(tt.body); info != nil {
				t.Errorf("extractTokenUsage(%s) = %+v, want nil", tt.name, info)
			}
		})
	}
}

func TestCountStreamTokens(t *testing.T) {
	reqBody := []byte(`{"messages":[{"role":"user","content":"Say hi"}]}`)
	info := countStreamTokens("Hello world", reqBody)
	if info == nil {
		t.Skip("tiktoken encoding unavailable")
	}
	if info.CompletionTokens == nil || *info.CompletionTokens <= 0 {
		t.Errorf("CompletionTokens = %v, want > 0", info.CompletionTokens)
	}
	if info.PromptTokens == nil || *info.PromptTokens <= 0 {
		t.Errorf("PromptTokens = %v, want > 0", info.PromptTokens)
	}
	if info.TotalTokens == nil {
		t.Fatal("TotalTokens is nil")
	}
	expected := *info.PromptTokens + *info.CompletionTokens
	if *info.TotalTokens != expected {
		t.Errorf("TotalTokens = %d, want %d", *info.TotalTokens, expected)
	}
}

func TestCountStreamTokens_EmptyCompletion(t *testing.T) {
	reqBody := []byte(`{"messages":[{"role":"user","content":"x"}]}`)
	info := countStreamTokens("", reqBody)
	if info == nil {
		t.Skip("tiktoken encoding unavailable")
	}
	if info.CompletionTokens != nil {
		t.Errorf("CompletionTokens = %v, want nil for empty completion", info.CompletionTokens)
	}
	if info.PromptTokens == nil || *info.PromptTokens <= 0 {
		t.Errorf("PromptTokens = %v, want > 0", info.PromptTokens)
	}
}
