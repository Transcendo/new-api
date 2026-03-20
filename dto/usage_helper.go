package dto

func ApplyOpenAIResponsesUsage(dst *Usage, upstream *Usage) {
	if dst == nil || upstream == nil {
		return
	}

	inputTokens := upstream.InputTokens
	if inputTokens == 0 {
		inputTokens = upstream.PromptTokens
	}
	if inputTokens != 0 {
		dst.InputTokens = inputTokens
		dst.PromptTokens = inputTokens
	}

	outputTokens := upstream.OutputTokens
	if outputTokens == 0 {
		outputTokens = upstream.CompletionTokens
	}
	if outputTokens != 0 {
		dst.OutputTokens = outputTokens
		dst.CompletionTokens = outputTokens
	}

	if upstream.TotalTokens != 0 {
		dst.TotalTokens = upstream.TotalTokens
	} else if inputTokens != 0 || outputTokens != 0 {
		dst.TotalTokens = inputTokens + outputTokens
	}

	if upstream.InputTokensDetails != nil {
		details := *upstream.InputTokensDetails
		dst.InputTokensDetails = &details
		dst.PromptTokensDetails = details
	}

	dst.CompletionTokenDetails = upstream.CompletionTokenDetails
}

func BuildUsageFromOpenAIResponses(upstream *Usage) *Usage {
	usage := &Usage{}
	ApplyOpenAIResponsesUsage(usage, upstream)
	return usage
}
