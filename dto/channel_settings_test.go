package dto

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestChannelSettingsTestStreamEnabledDefaultFalse(t *testing.T) {
	var settings ChannelSettings
	err := common.Unmarshal([]byte(`{"proxy":"http://localhost:8080"}`), &settings)
	require.NoError(t, err)
	require.False(t, settings.TestStreamEnabled)
	require.Equal(t, "http://localhost:8080", settings.Proxy)
}
