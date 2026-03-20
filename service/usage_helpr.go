package service

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

//func GetPromptTokens(textRequest dto.GeneralOpenAIRequest, relayMode int) (int, error) {
//	switch relayMode {
//	case constant.RelayModeChatCompletions:
//		return CountTokenMessages(textRequest.Messages, textRequest.Model)
//	case constant.RelayModeCompletions:
//		return CountTokenInput(textRequest.Prompt, textRequest.Model), nil
//	case constant.RelayModeModerations:
//		return CountTokenInput(textRequest.Input, textRequest.Model), nil
//	}
//	return 0, errors.New("unknown relay mode")
//}

func ResponseText2Usage(c *gin.Context, responseText string, modeName string, promptTokens int) *dto.Usage {
	common.SetContextKey(c, constant.ContextKeyLocalCountTokens, true)
	usage := &dto.Usage{}
	usage.PromptTokens = promptTokens
	usage.CompletionTokens = EstimateTokenByModel(modeName, responseText)
	usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	return usage
}

func ValidUsage(usage *dto.Usage) bool {
	return usage != nil && (usage.PromptTokens != 0 || usage.CompletionTokens != 0)
}

// NormalizeUsageForBilling returns a copy that uses effective prompt tokens for settlement.
// OpenAI Responses-style usage reports input_tokens including cached tokens, so billing paths
// must subtract cached tokens once before applying cache ratios.
func NormalizeUsageForBilling(info *relaycommon.RelayInfo, usage *dto.Usage) *dto.Usage {
	if usage == nil {
		return nil
	}

	normalized := *usage
	if usage.InputTokensDetails != nil {
		details := *usage.InputTokensDetails
		normalized.InputTokensDetails = &details
	}

	if !shouldNormalizeOpenAIResponsesUsage(info) {
		return &normalized
	}

	rawPromptTokens := usage.InputTokens
	if rawPromptTokens == 0 {
		rawPromptTokens = usage.PromptTokens
	}
	if rawPromptTokens == 0 {
		return &normalized
	}

	cachedTokens := usage.PromptTokensDetails.CachedTokens
	if usage.InputTokensDetails != nil && usage.InputTokensDetails.CachedTokens > 0 {
		cachedTokens = usage.InputTokensDetails.CachedTokens
	}
	if cachedTokens <= 0 {
		normalized.InputTokens = rawPromptTokens
		normalized.PromptTokens = rawPromptTokens
		return &normalized
	}
	if rawPromptTokens < cachedTokens {
		rawPromptTokens = cachedTokens
	}

	normalized.InputTokens = rawPromptTokens
	normalized.PromptTokens = rawPromptTokens - cachedTokens
	if normalized.PromptTokens < 0 {
		normalized.PromptTokens = 0
	}
	return &normalized
}

func shouldNormalizeOpenAIResponsesUsage(info *relaycommon.RelayInfo) bool {
	if info == nil {
		return false
	}
	switch info.GetFinalRequestRelayFormat() {
	case types.RelayFormatOpenAIResponses, types.RelayFormatOpenAIResponsesCompaction:
		return true
	default:
		return false
	}
}
