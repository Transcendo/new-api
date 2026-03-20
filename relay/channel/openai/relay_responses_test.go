package openai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOaiResponsesHandlerKeepsRawInputTokens(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(`{
			"output": [],
			"usage": {
				"input_tokens": 120,
				"output_tokens": 45,
				"total_tokens": 165,
				"input_tokens_details": {
					"cached_tokens": 30
				}
			}
		}`)),
	}

	usage, err := OaiResponsesHandler(ctx, nil, resp)
	require.Nil(t, err)
	require.Equal(t, 120, usage.InputTokens)
	require.Equal(t, 120, usage.PromptTokens)
	require.Equal(t, 45, usage.OutputTokens)
	require.Equal(t, 45, usage.CompletionTokens)
	require.Equal(t, 165, usage.TotalTokens)
	require.Equal(t, 30, usage.PromptTokensDetails.CachedTokens)
	require.NotNil(t, usage.InputTokensDetails)
	require.Equal(t, 30, usage.InputTokensDetails.CachedTokens)
}

func TestOaiResponsesStreamHandlerKeepsRawInputTokens(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	prevTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 1
	t.Cleanup(func() {
		constant.StreamingTimeout = prevTimeout
	})

	info := &relaycommon.RelayInfo{
		StartTime: time.Now(),
	}

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(strings.Join([]string{
			`data: {"type":"response.completed","response":{"usage":{"input_tokens":120,"output_tokens":45,"total_tokens":165,"input_tokens_details":{"cached_tokens":30}}}}`,
			`data: [DONE]`,
		}, "\n"))),
	}

	usage, err := OaiResponsesStreamHandler(ctx, info, resp)
	require.Nil(t, err)
	require.Equal(t, 120, usage.InputTokens)
	require.Equal(t, 120, usage.PromptTokens)
	require.Equal(t, 45, usage.OutputTokens)
	require.Equal(t, 45, usage.CompletionTokens)
	require.Equal(t, 165, usage.TotalTokens)
	require.Equal(t, 30, usage.PromptTokensDetails.CachedTokens)
	require.NotNil(t, usage.InputTokensDetails)
	require.Equal(t, 30, usage.InputTokensDetails.CachedTokens)
}
