package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func TestResolveChannelTestStream(t *testing.T) {
	t.Run("uses channel default when query is missing", func(t *testing.T) {
		channel := &model.Channel{}
		channel.SetSetting(dto.ChannelSettings{TestStreamEnabled: true})
		require.True(t, resolveChannelTestStream(channel, ""))
	})

	t.Run("query true overrides channel default", func(t *testing.T) {
		channel := &model.Channel{}
		channel.SetSetting(dto.ChannelSettings{TestStreamEnabled: false})
		require.True(t, resolveChannelTestStream(channel, "true"))
	})

	t.Run("query false overrides channel default", func(t *testing.T) {
		channel := &model.Channel{}
		channel.SetSetting(dto.ChannelSettings{TestStreamEnabled: true})
		require.False(t, resolveChannelTestStream(channel, "false"))
	})

	t.Run("nil channel defaults to false", func(t *testing.T) {
		require.False(t, resolveChannelTestStream(nil, ""))
	})
}

func TestResolveChannelTestStreamWithLegacySettings(t *testing.T) {
	raw := `{"proxy":"http://localhost:8080"}`
	channel := &model.Channel{Setting: common.GetPointer(raw)}
	require.False(t, resolveChannelTestStream(channel, ""))
}
