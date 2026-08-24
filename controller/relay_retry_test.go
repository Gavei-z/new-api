package controller

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestShouldRetryHonorsSpecificChannelForChannelErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	channelErr := types.NewError(errors.New("channel failed"), types.ErrorCodeChannelInvalidKey)

	fixedContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	fixedContext.Set("specific_channel_id", "5")
	require.False(t, shouldRetry(fixedContext, channelErr, 3))

	ordinaryContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	require.True(t, shouldRetry(ordinaryContext, channelErr, 3))
}
