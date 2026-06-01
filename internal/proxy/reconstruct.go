package proxy

import (
	"encoding/json"
	"fmt"
	"math/rand"
)

// reconstructStreamResponse builds a standard chat completion JSON from
// streamed SSE chunks, making stream and non-stream captures uniform.
func reconstructStreamResponse(completionText string, sr *StreamResult, scanner *SSEScanner, reqBody []byte) []byte {
	chunkID := scanner.FirstChunkID()
	chunkModel := scanner.FirstChunkModel()

	if chunkID == "" {
		chunkID = fmt.Sprintf("chatcmpl-%x", rand.Uint64())
	}
	if chunkModel == "" {
		var req map[string]interface{}
		json.Unmarshal(reqBody, &req)
		chunkModel, _ = req["model"].(string)
	}

	finishReason := scanner.LastFinishReason()
	if finishReason == "" {
		finishReason = "stop"
	}

	resp := map[string]interface{}{
		"id":     chunkID,
		"object": "chat.completion",
		"model":  chunkModel,
		"choices": []map[string]interface{}{{
			"index": 0,
			"message": map[string]string{
				"role":    "assistant",
				"content": completionText,
			},
			"finish_reason": finishReason,
		}},
	}

	if sr.TotalTokens > 0 {
		resp["usage"] = map[string]int{
			"prompt_tokens":     sr.PromptTokens,
			"completion_tokens": sr.CompletionTokens,
			"total_tokens":      sr.TotalTokens,
		}
	}

	out, _ := json.Marshal(resp)
	return out
}
