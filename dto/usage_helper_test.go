package dto

import "testing"

func TestBuildUsageFromOpenAIResponses(t *testing.T) {
	upstream := &Usage{
		InputTokens:  120,
		OutputTokens: 45,
		TotalTokens:  165,
		InputTokensDetails: &InputTokenDetails{
			CachedTokens: 30,
			ImageTokens:  5,
			AudioTokens:  2,
		},
		CompletionTokenDetails: OutputTokenDetails{
			ReasoningTokens: 9,
		},
	}

	usage := BuildUsageFromOpenAIResponses(upstream)

	if usage.InputTokens != 120 {
		t.Fatalf("InputTokens = %d, want 120", usage.InputTokens)
	}
	if usage.PromptTokens != 120 {
		t.Fatalf("PromptTokens = %d, want 120", usage.PromptTokens)
	}
	if usage.OutputTokens != 45 {
		t.Fatalf("OutputTokens = %d, want 45", usage.OutputTokens)
	}
	if usage.CompletionTokens != 45 {
		t.Fatalf("CompletionTokens = %d, want 45", usage.CompletionTokens)
	}
	if usage.TotalTokens != 165 {
		t.Fatalf("TotalTokens = %d, want 165", usage.TotalTokens)
	}
	if usage.PromptTokensDetails.CachedTokens != 30 {
		t.Fatalf("CachedTokens = %d, want 30", usage.PromptTokensDetails.CachedTokens)
	}
	if usage.PromptTokensDetails.ImageTokens != 5 {
		t.Fatalf("ImageTokens = %d, want 5", usage.PromptTokensDetails.ImageTokens)
	}
	if usage.PromptTokensDetails.AudioTokens != 2 {
		t.Fatalf("AudioTokens = %d, want 2", usage.PromptTokensDetails.AudioTokens)
	}
	if usage.InputTokensDetails == nil || usage.InputTokensDetails.CachedTokens != 30 {
		t.Fatalf("InputTokensDetails = %#v, want cached_tokens 30", usage.InputTokensDetails)
	}
	if usage.CompletionTokenDetails.ReasoningTokens != 9 {
		t.Fatalf("ReasoningTokens = %d, want 9", usage.CompletionTokenDetails.ReasoningTokens)
	}
}
