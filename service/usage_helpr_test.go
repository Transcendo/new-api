package service

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
)

func TestNormalizeUsageForBilling_OpenAIResponsesSubtractsCachedTokens(t *testing.T) {
	info := &relaycommon.RelayInfo{
		FinalRequestRelayFormat: types.RelayFormatOpenAIResponses,
	}
	usage := &dto.Usage{
		PromptTokens: 120,
		InputTokens:  120,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens: 30,
		},
	}

	normalized := NormalizeUsageForBilling(info, usage)

	if normalized.PromptTokens != 90 {
		t.Fatalf("PromptTokens = %d, want 90", normalized.PromptTokens)
	}
	if normalized.InputTokens != 120 {
		t.Fatalf("InputTokens = %d, want 120", normalized.InputTokens)
	}
	if usage.PromptTokens != 120 {
		t.Fatalf("original PromptTokens mutated to %d", usage.PromptTokens)
	}
}

func TestNormalizeUsageForBilling_NonResponsesKeepsPromptTokens(t *testing.T) {
	info := &relaycommon.RelayInfo{
		FinalRequestRelayFormat: types.RelayFormatOpenAI,
	}
	usage := &dto.Usage{
		PromptTokens: 120,
		InputTokens:  120,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens: 30,
		},
	}

	normalized := NormalizeUsageForBilling(info, usage)

	if normalized.PromptTokens != 120 {
		t.Fatalf("PromptTokens = %d, want 120", normalized.PromptTokens)
	}
}
