package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestTokenCriticalRateLimitIsIsolatedByTokenID(t *testing.T) {
	previousRedisEnabled := common.RedisEnabled
	previousEnabled := common.CriticalRateLimitEnable
	previousLimit := common.CriticalRateLimitNum
	previousDuration := common.CriticalRateLimitDuration
	common.RedisEnabled = false
	common.CriticalRateLimitEnable = true
	common.CriticalRateLimitNum = 1
	common.CriticalRateLimitDuration = 60
	t.Cleanup(func() {
		common.RedisEnabled = previousRedisEnabled
		common.CriticalRateLimitEnable = previousEnabled
		common.CriticalRateLimitNum = previousLimit
		common.CriticalRateLimitDuration = previousDuration
	})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		tokenID, err := strconv.Atoi(c.GetHeader("X-Test-Token-ID"))
		require.NoError(t, err)
		c.Set("id", 1)
		c.Set("token_id", tokenID)
		c.Next()
	})
	router.GET("/", TokenCriticalRateLimit(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	request := func(tokenID string) int {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Test-Token-ID", tokenID)
		router.ServeHTTP(recorder, req)
		return recorder.Code
	}

	require.Equal(t, http.StatusNoContent, request("91001"))
	require.Equal(t, http.StatusNoContent, request("91002"))
	require.Equal(t, http.StatusTooManyRequests, request("91001"))
}

func TestTokenAuthAttemptRateLimitRunsWhenGlobalLimitIsDisabled(t *testing.T) {
	previousRedisEnabled := common.RedisEnabled
	previousGlobalEnabled := common.GlobalApiRateLimitEnable
	previousLimit := common.TokenAuthRateLimitNum
	previousDuration := common.TokenAuthRateLimitDuration
	common.RedisEnabled = false
	common.GlobalApiRateLimitEnable = false
	common.TokenAuthRateLimitNum = 1
	common.TokenAuthRateLimitDuration = 60
	t.Cleanup(func() {
		common.RedisEnabled = previousRedisEnabled
		common.GlobalApiRateLimitEnable = previousGlobalEnabled
		common.TokenAuthRateLimitNum = previousLimit
		common.TokenAuthRateLimitDuration = previousDuration
	})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	authAttempts := 0
	router.POST("/", TokenAuthAttemptRateLimit(), func(c *gin.Context) {
		authAttempts++
		c.Status(http.StatusUnauthorized)
	})

	request := func(remoteAddr string) int {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.RemoteAddr = remoteAddr
		router.ServeHTTP(recorder, req)
		return recorder.Code
	}

	require.Equal(t, http.StatusUnauthorized, request("192.0.2.10:1234"))
	require.Equal(t, http.StatusTooManyRequests, request("192.0.2.10:5678"))
	require.Equal(t, http.StatusUnauthorized, request("192.0.2.11:1234"))
	require.Equal(t, 2, authAttempts)
}
