package common

import (
	"strings"

	"github.com/QuantumNous/new-api/constant"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"
)

// IsAnthropicDirectPassthrough reports whether this request should keep the
// client's upstream protocol and raw JSON body. The dedicated setting is
// intentionally narrower than pass_through_body_enabled: an Anthropic channel
// can safely forward both Claude Messages and OpenAI-compatible Chat only when
// the upstream exposes those same paths.
func IsAnthropicDirectPassthrough(info *RelayInfo) bool {
	if info == nil ||
		info.ChannelMeta == nil ||
		info.ChannelType != constant.ChannelTypeAnthropic ||
		!info.ChannelOtherSettings.AnthropicDirectPassthroughEnabled {
		return false
	}

	requestPath := strings.SplitN(info.RequestURLPath, "?", 2)[0]
	switch info.RelayFormat {
	case types.RelayFormatClaude:
		return requestPath == "/v1/messages" || requestPath == "/v1/messages/count_tokens"
	case types.RelayFormatOpenAI:
		return info.RelayMode == relayconstant.RelayModeChatCompletions &&
			requestPath == "/v1/chat/completions"
	default:
		return false
	}
}
