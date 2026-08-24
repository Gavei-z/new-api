package common

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/require"
)

func TestIsAnthropicDirectPassthrough(t *testing.T) {
	tests := []struct {
		name       string
		channel    int
		format     types.RelayFormat
		mode       int
		path       string
		enabled    bool
		wantDirect bool
	}{
		{name: "native messages", channel: constant.ChannelTypeAnthropic, format: types.RelayFormatClaude, path: "/v1/messages", enabled: true, wantDirect: true},
		{name: "native messages beta query", channel: constant.ChannelTypeAnthropic, format: types.RelayFormatClaude, path: "/v1/messages?beta=true", enabled: true, wantDirect: true},
		{name: "native count tokens", channel: constant.ChannelTypeAnthropic, format: types.RelayFormatClaude, path: "/v1/messages/count_tokens", enabled: true, wantDirect: true},
		{name: "compatible chat", channel: constant.ChannelTypeAnthropic, format: types.RelayFormatOpenAI, mode: relayconstant.RelayModeChatCompletions, path: "/v1/chat/completions", enabled: true, wantDirect: true},
		{name: "legacy completions excluded", channel: constant.ChannelTypeAnthropic, format: types.RelayFormatOpenAI, path: "/v1/completions", enabled: true},
		{name: "responses excluded", channel: constant.ChannelTypeAnthropic, format: types.RelayFormatOpenAIResponses, path: "/v1/responses", enabled: true},
		{name: "wrong channel excluded", channel: constant.ChannelTypeOpenAI, format: types.RelayFormatOpenAI, mode: relayconstant.RelayModeChatCompletions, path: "/v1/chat/completions", enabled: true},
		{name: "disabled", channel: constant.ChannelTypeAnthropic, format: types.RelayFormatClaude, path: "/v1/messages"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			info := &RelayInfo{
				RelayFormat:    test.format,
				RelayMode:      test.mode,
				RequestURLPath: test.path,
				ChannelMeta:    &ChannelMeta{ChannelType: test.channel},
			}
			info.ChannelOtherSettings.AnthropicDirectPassthroughEnabled = test.enabled
			require.Equal(t, test.wantDirect, IsAnthropicDirectPassthrough(info))
		})
	}
}
