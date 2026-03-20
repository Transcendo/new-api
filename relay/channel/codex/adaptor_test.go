package codex

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newCodexInfoForTest() *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		RelayMode: relayconstant.RelayModeResponses,
		ChannelMeta: &relaycommon.ChannelMeta{
			ApiKey: `{"access_token":"access-token","account_id":"account-id"}`,
		},
	}
}

func TestConvertOpenAIResponsesRequestSyncsSessionAndAppliesDefaults(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	ctx.Request.Header.Set("Session_id", "sess-123")

	adaptor := &Adaptor{}
	req := dto.OpenAIResponsesRequest{
		Model: "gpt-5",
		Tools: []byte(`[{"type":"function","name":"shell"}]`),
	}

	convertedAny, err := adaptor.ConvertOpenAIResponsesRequest(ctx, newCodexInfoForTest(), req)
	require.NoError(t, err)

	converted, ok := convertedAny.(dto.OpenAIResponsesRequest)
	require.True(t, ok)
	require.JSONEq(t, `"sess-123"`, string(converted.PromptCacheKey))
	require.JSONEq(t, `""`, string(converted.Instructions))
	require.JSONEq(t, `["reasoning.encrypted_content"]`, string(converted.Include))
	require.JSONEq(t, `"auto"`, string(converted.ToolChoice))
	require.JSONEq(t, `true`, string(converted.ParallelToolCalls))
	require.JSONEq(t, `false`, string(converted.Store))

	promptCacheKeyAny, exists := ctx.Get(codexPromptCacheKeyContextKey)
	require.True(t, exists)
	require.Equal(t, "sess-123", promptCacheKeyAny)
}

func TestConvertOpenAIResponsesRequestPreservesExplicitPromptCacheKey(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	ctx.Request.Header.Set("Session_id", "sess-header")

	adaptor := &Adaptor{}
	req := dto.OpenAIResponsesRequest{
		Model:          "gpt-5",
		PromptCacheKey: []byte(`"cache-body"`),
	}

	convertedAny, err := adaptor.ConvertOpenAIResponsesRequest(ctx, newCodexInfoForTest(), req)
	require.NoError(t, err)

	converted := convertedAny.(dto.OpenAIResponsesRequest)
	require.JSONEq(t, `"cache-body"`, string(converted.PromptCacheKey))
}

func TestSetupRequestHeaderSyncsSessionIDFromPromptCacheKey(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	ctx.Set(codexPromptCacheKeyContextKey, "cache-abc")

	headers := http.Header{}
	adaptor := &Adaptor{}

	err := adaptor.SetupRequestHeader(ctx, &headers, newCodexInfoForTest())
	require.NoError(t, err)
	require.Equal(t, "cache-abc", headers.Get("Session_id"))
	require.Equal(t, "Bearer access-token", headers.Get("Authorization"))
	require.Equal(t, "account-id", headers.Get("Chatgpt-Account-Id"))
}

func TestSetupRequestHeaderKeepsExplicitSessionID(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	ctx.Set(codexPromptCacheKeyContextKey, "cache-body")

	headers := http.Header{}
	headers.Set("Session_id", "sess-explicit")
	adaptor := &Adaptor{}

	err := adaptor.SetupRequestHeader(ctx, &headers, newCodexInfoForTest())
	require.NoError(t, err)
	require.Equal(t, "sess-explicit", headers.Get("Session_id"))
}
