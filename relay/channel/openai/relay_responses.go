package openai

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

func OaiResponsesHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	defer service.CloseResponseBodyGracefully(resp)

	// read response body
	var responsesResponse dto.OpenAIResponsesResponse
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}
	err = common.Unmarshal(responseBody, &responsesResponse)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	if oaiError := responsesResponse.GetOpenAIError(); oaiError != nil && oaiError.Type != "" {
		return nil, types.WithOpenAIError(*oaiError, resp.StatusCode)
	}

	if responsesResponse.HasImageGenerationCall() {
		c.Set("image_generation_call", true)
		c.Set("image_generation_call_quality", responsesResponse.GetQuality())
		c.Set("image_generation_call_size", responsesResponse.GetSize())
	}

	// 写入新的 response body
	service.IOCopyBytesGracefully(c, resp, responseBody)

	// compute usage
	usage := dto.BuildUsageFromOpenAIResponses(responsesResponse.Usage)
	if info == nil || info.ResponsesUsageInfo == nil || info.ResponsesUsageInfo.BuiltInTools == nil {
		return usage, nil
	}
	// 解析 Tools 用量
	for _, tool := range responsesResponse.Tools {
		buildToolinfo, ok := info.ResponsesUsageInfo.BuiltInTools[common.Interface2String(tool["type"])]
		if !ok || buildToolinfo == nil {
			logger.LogError(c, fmt.Sprintf("BuiltInTools not found for tool type: %v", tool["type"]))
			continue
		}
		buildToolinfo.CallCount++
	}
	return usage, nil
}

// OaiResponsesStreamToNonStreamHandler consumes an upstream SSE stream and assembles
// a complete non-stream JSON response to return to the client.
// Used by channels (e.g. Codex) where the upstream only supports streaming.
func OaiResponsesStreamToNonStreamHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	if resp == nil || resp.Body == nil {
		logger.LogError(c, "invalid response or response body")
		return nil, types.NewError(fmt.Errorf("invalid response"), types.ErrorCodeBadResponse)
	}

	// resp.Body is closed by streamScannerHandlerInternal, no need to close here

	var usage = &dto.Usage{}
	var responsesResponse *dto.OpenAIResponsesResponse
	var rawResponseBytes []byte
	var responseTextBuilder strings.Builder

	helper.StreamScannerHandlerRaw(c, resp, info, func(data string) bool {
		var streamResponse dto.ResponsesStreamResponse
		if err := common.UnmarshalJsonStr(data, &streamResponse); err != nil {
			logger.LogError(c, "failed to unmarshal stream response: "+err.Error())
			return true
		}
		switch streamResponse.Type {
		case "response.completed":
			// Extract raw response JSON to preserve all fields (including Codex-specific ones)
			var rawParsed struct {
				Response json.RawMessage `json:"response"`
			}
			if err := common.UnmarshalJsonStr(data, &rawParsed); err == nil && rawParsed.Response != nil {
				rawResponseBytes = []byte(rawParsed.Response)
			}
			// Also parse into typed struct for usage/image/error extraction
			if streamResponse.Response != nil {
				responsesResponse = streamResponse.Response
				if streamResponse.Response.Usage != nil {
					dto.ApplyOpenAIResponsesUsage(usage, streamResponse.Response.Usage)
				}
				if streamResponse.Response.HasImageGenerationCall() {
					c.Set("image_generation_call", true)
					c.Set("image_generation_call_quality", streamResponse.Response.GetQuality())
					c.Set("image_generation_call_size", streamResponse.Response.GetSize())
				}
			}
		case "response.output_text.delta":
			responseTextBuilder.WriteString(streamResponse.Delta)
		case dto.ResponsesOutputTypeItemDone:
			if streamResponse.Item != nil {
				switch streamResponse.Item.Type {
				case dto.BuildInCallWebSearchCall:
					if info != nil && info.ResponsesUsageInfo != nil && info.ResponsesUsageInfo.BuiltInTools != nil {
						if webSearchTool, exists := info.ResponsesUsageInfo.BuiltInTools[dto.BuildInToolWebSearchPreview]; exists && webSearchTool != nil {
							webSearchTool.CallCount++
						}
					}
				}
			}
		}
		return true
	})

	if rawResponseBytes == nil {
		return nil, types.NewError(fmt.Errorf("codex: no response.completed event received from upstream stream"), types.ErrorCodeBadResponse)
	}

	if responsesResponse != nil {
		if oaiError := responsesResponse.GetOpenAIError(); oaiError != nil && oaiError.Type != "" {
			return nil, types.WithOpenAIError(*oaiError, resp.StatusCode)
		}
	}

	// Write JSON response with correct headers (don't copy upstream SSE headers)
	c.Data(http.StatusOK, "application/json; charset=utf-8", rawResponseBytes)

	if usage.CompletionTokens == 0 {
		tempStr := responseTextBuilder.String()
		if len(tempStr) > 0 {
			completionTokens := service.CountTextToken(tempStr, info.UpstreamModelName)
			usage.CompletionTokens = completionTokens
		}
	}

	if usage.PromptTokens == 0 && usage.CompletionTokens != 0 {
		usage.PromptTokens = info.GetEstimatePromptTokens()
	}

	if usage.TotalTokens == 0 {
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	}

	if info == nil || info.ResponsesUsageInfo == nil || info.ResponsesUsageInfo.BuiltInTools == nil {
		return usage, nil
	}
	for _, tool := range responsesResponse.Tools {
		buildToolinfo, ok := info.ResponsesUsageInfo.BuiltInTools[common.Interface2String(tool["type"])]
		if !ok || buildToolinfo == nil {
			logger.LogError(c, fmt.Sprintf("BuiltInTools not found for tool type: %v", tool["type"]))
			continue
		}
		buildToolinfo.CallCount++
	}

	return usage, nil
}

func OaiResponsesStreamHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	if resp == nil || resp.Body == nil {
		logger.LogError(c, "invalid response or response body")
		return nil, types.NewError(fmt.Errorf("invalid response"), types.ErrorCodeBadResponse)
	}

	defer service.CloseResponseBodyGracefully(resp)

	var usage = &dto.Usage{}
	var responseTextBuilder strings.Builder

	helper.StreamScannerHandler(c, resp, info, func(data string) bool {

		// 检查当前数据是否包含 completed 状态和 usage 信息
		var streamResponse dto.ResponsesStreamResponse
		if err := common.UnmarshalJsonStr(data, &streamResponse); err == nil {
			sendResponsesStreamData(c, streamResponse, data)
			switch streamResponse.Type {
			case "response.completed":
				if streamResponse.Response != nil {
					if streamResponse.Response.Usage != nil {
						dto.ApplyOpenAIResponsesUsage(usage, streamResponse.Response.Usage)
					}
					if streamResponse.Response.HasImageGenerationCall() {
						c.Set("image_generation_call", true)
						c.Set("image_generation_call_quality", streamResponse.Response.GetQuality())
						c.Set("image_generation_call_size", streamResponse.Response.GetSize())
					}
				}
			case "response.output_text.delta":
				// 处理输出文本
				responseTextBuilder.WriteString(streamResponse.Delta)
			case dto.ResponsesOutputTypeItemDone:
				// 函数调用处理
				if streamResponse.Item != nil {
					switch streamResponse.Item.Type {
					case dto.BuildInCallWebSearchCall:
						if info != nil && info.ResponsesUsageInfo != nil && info.ResponsesUsageInfo.BuiltInTools != nil {
							if webSearchTool, exists := info.ResponsesUsageInfo.BuiltInTools[dto.BuildInToolWebSearchPreview]; exists && webSearchTool != nil {
								webSearchTool.CallCount++
							}
						}
					}
				}
			}
		} else {
			logger.LogError(c, "failed to unmarshal stream response: "+err.Error())
		}
		return true
	})

	if usage.CompletionTokens == 0 {
		// 计算输出文本的 token 数量
		tempStr := responseTextBuilder.String()
		if len(tempStr) > 0 {
			// 非正常结束，使用输出文本的 token 数量
			completionTokens := service.CountTextToken(tempStr, info.UpstreamModelName)
			usage.CompletionTokens = completionTokens
		}
	}

	if usage.PromptTokens == 0 && usage.CompletionTokens != 0 {
		usage.PromptTokens = info.GetEstimatePromptTokens()
	}

	if usage.TotalTokens == 0 {
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	}

	return usage, nil
}
