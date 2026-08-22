package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestFilterChannelsByForcedClaudeRoute(t *testing.T) {
	previousEnabled := common.ClaudeForceCFJWL
	previousChannels := channelsIDM
	t.Cleanup(func() {
		common.ClaudeForceCFJWL = previousEnabled
		channelsIDM = previousChannels
	})

	common.ClaudeForceCFJWL = true
	channelsIDM = map[int]*Channel{
		4: {Id: 4, Name: common.ClaudeCFJWLChannelName, Type: constant.ChannelTypeAnthropic},
		5: {Id: 5, Name: "kiro2cc-claude", Type: constant.ChannelTypeAnthropic},
		6: {Id: 6, Name: common.ClaudeCFJWLChannelName, Type: constant.ChannelTypeOpenAI},
	}

	require.Equal(t, []int{4}, filterChannelsByForcedClaudeRoute([]int{4, 5, 6}, "claude-sonnet-5"))
	require.Equal(t, []int{4, 5, 6}, filterChannelsByForcedClaudeRoute([]int{4, 5, 6}, "gpt-5.6-sol"))

	common.ClaudeForceCFJWL = false
	require.Equal(t, []int{4, 5}, filterChannelsByForcedClaudeRoute([]int{4, 5}, "claude-sonnet-5"))
}

func TestGetRandomSatisfiedChannelForcedClaudeFallsBackToCanonicalCandidate(t *testing.T) {
	previousEnabled := common.ClaudeForceCFJWL
	previousMemoryCache := common.MemoryCacheEnabled

	channelSyncLock.Lock()
	previousChannels := channelsIDM
	previousGroups := group2model2channels
	channelsIDM = map[int]*Channel{
		4: {Id: 4, Name: common.ClaudeCFJWLChannelName, Type: constant.ChannelTypeAnthropic},
		5: {Id: 5, Name: "kiro2cc-claude", Type: constant.ChannelTypeAnthropic},
	}
	group2model2channels = map[string]map[string][]int{
		"default": {
			"anthropic/claude-sonnet-5": {5},
			"claude-sonnet-5":           {4},
		},
	}
	channelSyncLock.Unlock()

	t.Cleanup(func() {
		common.ClaudeForceCFJWL = previousEnabled
		common.MemoryCacheEnabled = previousMemoryCache
		channelSyncLock.Lock()
		channelsIDM = previousChannels
		group2model2channels = previousGroups
		channelSyncLock.Unlock()
	})

	common.ClaudeForceCFJWL = true
	common.MemoryCacheEnabled = true
	channel, err := GetRandomSatisfiedChannel("default", "anthropic/claude-sonnet-5", 0, "/v1/messages")
	require.NoError(t, err)
	require.NotNil(t, channel)
	require.Equal(t, 4, channel.Id)
}

func TestGetForcedClaudeChannelIgnoresOtherProviderPriority(t *testing.T) {
	previousDB := DB
	previousType := common.MainDatabaseType()
	previousEnabled := common.ClaudeForceCFJWL
	previousGroupCol := commonGroupCol
	t.Cleanup(func() {
		DB = previousDB
		common.SetMainDatabaseType(previousType)
		common.ClaudeForceCFJWL = previousEnabled
		commonGroupCol = previousGroupCol
	})

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	DB = db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	initCol()
	common.ClaudeForceCFJWL = true
	require.NoError(t, db.AutoMigrate(&Channel{}, &Ability{}))

	cfjwlPriority := int64(0)
	kiroPriority := int64(100)
	channels := []Channel{
		{Id: 4, Name: common.ClaudeCFJWLChannelName, Type: constant.ChannelTypeAnthropic, Status: common.ChannelStatusEnabled},
		{Id: 5, Name: "kiro2cc-claude", Type: constant.ChannelTypeAnthropic, Status: common.ChannelStatusEnabled},
	}
	require.NoError(t, db.Create(&channels).Error)
	require.NoError(t, db.Create(&[]Ability{
		{Group: "default", Model: "claude-sonnet-5", ChannelId: 4, Enabled: true, Priority: &cfjwlPriority},
		{Group: "default", Model: "claude-sonnet-5", ChannelId: 5, Enabled: true, Priority: &kiroPriority},
	}).Error)

	channel, err := GetChannel("default", "claude-sonnet-5", 0, "/v1/messages")
	require.NoError(t, err)
	require.NotNil(t, channel)
	require.Equal(t, 4, channel.Id)

	channel, err = GetChannel("default", "anthropic/claude-sonnet-5", 0, "/v1/messages")
	require.NoError(t, err)
	require.NotNil(t, channel)
	require.Equal(t, 4, channel.Id)
}
