package proxy

import (
	"encoding/json"
	"strings"

	"github.com/pkoukk/tiktoken-go"
)

type tokenInfo struct {
	PromptTokens     *int
	CompletionTokens *int
	TotalTokens      *int
}

func countStreamTokens(completionText string, reqBody []byte) *tokenInfo {
	enc, err := tiktoken.GetEncoding(tiktoken.MODEL_CL100K_BASE)
	if err != nil {
		return nil
	}

	var ti tokenInfo

	completionCount := len(enc.Encode(completionText, nil, nil))
	if completionCount > 0 {
		ti.CompletionTokens = &completionCount
	}

	promptText := extractPromptText(reqBody)
	promptCount := len(enc.Encode(promptText, nil, nil))
	if promptCount > 0 {
		ti.PromptTokens = &promptCount
	}

	total := promptCount + completionCount
	ti.TotalTokens = &total
	return &ti
}

func extractPromptText(reqBody []byte) string {
	var req struct {
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(reqBody, &req); err != nil {
		return ""
	}
	var sb strings.Builder
	for _, msg := range req.Messages {
		sb.WriteString(msg.Role)
		sb.WriteString(": ")
		sb.WriteString(msg.Content)
		sb.WriteString("\n")
	}
	return sb.String()
}

func extractTokenUsage(body []byte) *tokenInfo {
	var resp struct {
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &resp); err != nil || resp.Usage.PromptTokens == 0 && resp.Usage.TotalTokens == 0 {
		return nil
	}
	var info tokenInfo
	if resp.Usage.PromptTokens > 0 {
		v := resp.Usage.PromptTokens
		info.PromptTokens = &v
	}
	if resp.Usage.CompletionTokens > 0 {
		v := resp.Usage.CompletionTokens
		info.CompletionTokens = &v
	}
	if resp.Usage.TotalTokens > 0 {
		v := resp.Usage.TotalTokens
		info.TotalTokens = &v
	}
	return &info
}
