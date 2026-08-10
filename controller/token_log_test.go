package controller_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeTokenLogRouter() *gin.Engine {
	router := gin.New()
	router.GET("/api/log/token", middleware.TokenAuthAttemptRateLimit(), middleware.TokenAuthReadOnly(), middleware.TokenSearchRateLimit(), controller.GetLogByKey)
	return router
}

func performTokenLogRequest(t *testing.T, router *gin.Engine, auth string, rawQuery string) *httptest.ResponseRecorder {
	t.Helper()
	target := "/api/log/token"
	if rawQuery != "" {
		target += "?" + rawQuery
	}
	req := httptest.NewRequest(http.MethodGet, target, nil)
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestGetLogByKeyFiltersAndPaginatesCurrentToken(t *testing.T) {
	truncateTables(t)

	const userID = 41
	const tokenID = 4101
	const otherTokenID = 4102
	const tokenKey = "logtoken4101"
	seedRedeemUser(t, userID, 0)
	seedRedeemToken(t, tokenID, userID, tokenKey, 1000, 0)
	seedRedeemToken(t, otherTokenID, userID, "logtoken4102", 1000, 0)

	logs := []model.Log{
		{UserId: userID, TokenId: tokenID, CreatedAt: 100, Content: "outside-range"},
		{UserId: userID, TokenId: tokenID, CreatedAt: 200, Content: "page-two"},
		{UserId: userID, TokenId: tokenID, CreatedAt: 300, Content: "page-one"},
		{UserId: userID, TokenId: otherTokenID, CreatedAt: 250, Content: "other-token"},
	}
	require.NoError(t, model.LOG_DB.Create(&logs).Error)

	w := performTokenLogRequest(
		t,
		makeTokenLogRouter(),
		"Bearer sk-"+tokenKey,
		"p=2&page_size=1&start_timestamp=150&end_timestamp=300",
	)

	require.Equal(t, http.StatusOK, w.Code)
	var response struct {
		Success bool        `json:"success"`
		Data    []model.Log `json:"data"`
	}
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Len(t, response.Data, 1)
	assert.Equal(t, "page-two", response.Data[0].Content)
}

func TestGetLogByKeyRejectsUnboundedNegativePageSize(t *testing.T) {
	truncateTables(t)

	const userID = 42
	const tokenID = 4201
	const tokenKey = "logtoken4201"
	seedRedeemUser(t, userID, 0)
	seedRedeemToken(t, tokenID, userID, tokenKey, 1000, 0)

	logs := make([]model.Log, common.ItemsPerPage+1)
	for i := range logs {
		logs[i] = model.Log{UserId: userID, TokenId: tokenID, CreatedAt: int64(i + 1)}
	}
	require.NoError(t, model.LOG_DB.Create(&logs).Error)

	w := performTokenLogRequest(t, makeTokenLogRouter(), "Bearer sk-"+tokenKey, "page_size=-1")

	require.Equal(t, http.StatusOK, w.Code)
	var response struct {
		Success bool        `json:"success"`
		Data    []model.Log `json:"data"`
	}
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &response))
	require.True(t, response.Success)
	assert.Len(t, response.Data, common.ItemsPerPage)
}
