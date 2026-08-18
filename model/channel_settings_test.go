package model

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdvancedCustomChannelRequiresModelListRouteOnlyWhenUpdateChecksEnabled(t *testing.T) {
	inferenceRoute := dto.AdvancedCustomRoute{
		IncomingPath: "/v1/chat/completions",
		UpstreamPath: "/v1/chat/completions",
		Converter:    "none",
	}

	tests := []struct {
		name          string
		checksEnabled bool
		routes        []dto.AdvancedCustomRoute
		wantErr       string
	}{
		{
			name:   "legacy channel without discovery route remains valid",
			routes: []dto.AdvancedCustomRoute{inferenceRoute},
		},
		{
			name:          "enabled checks require discovery route",
			checksEnabled: true,
			routes:        []dto.AdvancedCustomRoute{inferenceRoute},
			wantErr:       dto.AdvancedCustomModelListPath,
		},
		{
			name:          "enabled checks accept discovery route",
			checksEnabled: true,
			routes: []dto.AdvancedCustomRoute{
				inferenceRoute,
				{
					IncomingPath: dto.AdvancedCustomModelListPath,
					UpstreamPath: dto.AdvancedCustomModelListPath,
					Converter:    "none",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel := &Channel{Type: constant.ChannelTypeAdvancedCustom}
			channel.SetOtherSettings(dto.ChannelOtherSettings{
				UpstreamModelUpdateCheckEnabled: tt.checksEnabled,
				AdvancedCustom: &dto.AdvancedCustomConfig{
					Routes: tt.routes,
				},
			})

			err := channel.ValidateSettings()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestClaudeInputBillingModeValidation(t *testing.T) {
	tests := []struct {
		name        string
		channelType int
		mode        dto.ClaudeInputBillingMode
		wantError   string
	}{
		{name: "empty mode remains backward compatible", channelType: constant.ChannelTypeAnthropic},
		{name: "explicit upstream", channelType: constant.ChannelTypeAnthropic, mode: dto.ClaudeInputBillingModeUpstream},
		{name: "audit", channelType: constant.ChannelTypeAnthropic, mode: dto.ClaudeInputBillingModeLocalEstimateAudit},
		{name: "apply", channelType: constant.ChannelTypeAnthropic, mode: dto.ClaudeInputBillingModeLocalEstimate},
		{name: "unknown mode", channelType: constant.ChannelTypeAnthropic, mode: "unknown", wantError: "invalid claude_input_billing_mode"},
		{name: "non Anthropic channel", channelType: constant.ChannelTypeOpenAI, mode: dto.ClaudeInputBillingModeLocalEstimate, wantError: "only supported for Anthropic"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel := &Channel{Type: tt.channelType}
			channel.SetOtherSettings(dto.ChannelOtherSettings{ClaudeInputBillingMode: tt.mode})

			err := channel.ValidateSettings()
			if tt.wantError == "" {
				require.NoError(t, err)
				assert.Equal(t, tt.mode, channel.GetOtherSettings().ClaudeInputBillingMode)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantError)
		})
	}
}
