package controller_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)

	common.UsingSQLite = true
	common.RedisEnabled = false
	common.BatchUpdateEnabled = false
	common.LogConsumeEnabled = true
	common.IsMasterNode = true
	common.SQLitePath = filepath.Join(os.TempDir(), "new-api-controller-token-redeem-test.db")

	if err := os.Remove(common.SQLitePath); err != nil && !os.IsNotExist(err) {
		panic("failed to clean test db: " + err.Error())
	}
	if err := model.InitDB(); err != nil {
		panic("failed to init test db: " + err.Error())
	}
	if err := model.InitLogDB(); err != nil {
		panic("failed to init test log db: " + err.Error())
	}
	if err := i18n.Init(); err != nil {
		panic("failed to init i18n: " + err.Error())
	}

	code := m.Run()
	_ = model.CloseDB()
	_ = os.Remove(common.SQLitePath)
	os.Exit(code)
}

func truncateTables(t *testing.T) {
	t.Helper()
	require.NoError(t, model.DB.Exec("DELETE FROM redemptions").Error)
	require.NoError(t, model.DB.Exec("DELETE FROM tokens").Error)
	require.NoError(t, model.DB.Exec("DELETE FROM users").Error)
	require.NoError(t, model.DB.Exec("DELETE FROM logs").Error)
}

func seedRedeemUser(t *testing.T, id int, quota int) {
	t.Helper()
	user := &model.User{
		Id:       id,
		Username: "redeem_user",
		Password: "password123",
		Quota:    quota,
		Status:   common.UserStatusEnabled,
		Group:    "default",
	}
	require.NoError(t, model.DB.Create(user).Error)
}

func seedRedeemToken(t *testing.T, id int, userID int, key string, remainQuota int, usedQuota int) {
	t.Helper()
	seedRedeemTokenWithName(t, id, userID, key, "redeem_token", remainQuota, usedQuota)
}

func seedRedeemTokenWithName(t *testing.T, id int, userID int, key string, name string, remainQuota int, usedQuota int) {
	t.Helper()
	token := &model.Token{
		Id:          id,
		UserId:      userID,
		Key:         key,
		Name:        name,
		Status:      common.TokenStatusEnabled,
		RemainQuota: remainQuota,
		UsedQuota:   usedQuota,
		ExpiredTime: -1,
	}
	require.NoError(t, model.DB.Create(token).Error)
}

func seedRedemptionCode(t *testing.T, id int, key string, quota int) {
	t.Helper()
	redemption := &model.Redemption{
		Id:     id,
		Key:    key,
		Name:   "redeem_code",
		Quota:  quota,
		Status: common.RedemptionCodeStatusEnabled,
	}
	require.NoError(t, model.DB.Create(redemption).Error)
}

func makeTokenRedeemRouter() *gin.Engine {
	router := gin.New()
	router.POST("/api/token/redeem", middleware.TokenAuthReadOnly(), controller.TokenRedeem)
	return router
}

func performTokenRedeemRequest(t *testing.T, router *gin.Engine, auth string, redeemKey string) *httptest.ResponseRecorder {
	t.Helper()
	body, err := common.Marshal(map[string]any{"key": redeemKey})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/token/redeem", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "zh-CN")
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func getUserQuotaForTest(t *testing.T, userID int) int {
	t.Helper()
	var user model.User
	require.NoError(t, model.DB.Select("quota").Where("id = ?", userID).First(&user).Error)
	return user.Quota
}

func getTokenRemainQuotaForTest(t *testing.T, tokenID int) int {
	t.Helper()
	var token model.Token
	require.NoError(t, model.DB.Select("remain_quota").Where("id = ?", tokenID).First(&token).Error)
	return token.RemainQuota
}

func getTokenUsedQuotaForTest(t *testing.T, tokenID int) int {
	t.Helper()
	var token model.Token
	require.NoError(t, model.DB.Select("used_quota").Where("id = ?", tokenID).First(&token).Error)
	return token.UsedQuota
}

func getRedemptionForTest(t *testing.T, key string) *model.Redemption {
	t.Helper()
	var redemption model.Redemption
	require.NoError(t, model.DB.Where("key = ?", key).First(&redemption).Error)
	return &redemption
}

func getLatestTopupLogForTest(t *testing.T, userID int) *model.Log {
	t.Helper()
	var log model.Log
	require.NoError(t, model.LOG_DB.Where("user_id = ? AND type = ?", userID, model.LogTypeTopup).Order("id desc").First(&log).Error)
	return &log
}

func TestTokenRedeem_Success(t *testing.T) {
	truncateTables(t)

	const userID = 1
	const tokenID = 1
	const initQuota = 100
	const redeemQuota = 250
	const initTokenRemain = 1000
	const initTokenUsed = 400
	const tokenKey = "tokenredeem1"
	const redeemKey = "redeem-code-001"

	seedRedeemUser(t, userID, initQuota)
	seedRedeemToken(t, tokenID, userID, tokenKey, initTokenRemain, initTokenUsed)
	seedRedemptionCode(t, 1, redeemKey, redeemQuota)

	w := performTokenRedeemRequest(t, makeTokenRedeemRouter(), "Bearer sk-"+tokenKey, redeemKey)

	require.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    int    `json:"data"`
	}
	require.NoError(t, common.Unmarshal([]byte(w.Body.String()), &resp))
	assert.True(t, resp.Success)
	assert.Equal(t, "", resp.Message)
	assert.Equal(t, redeemQuota, resp.Data)
	assert.Equal(t, initQuota+redeemQuota, getUserQuotaForTest(t, userID))
	assert.Equal(t, initTokenRemain+redeemQuota, getTokenRemainQuotaForTest(t, tokenID))
	assert.Equal(t, initTokenUsed, getTokenUsedQuotaForTest(t, tokenID))
	assert.Equal(t, initTokenRemain+initTokenUsed+redeemQuota, getTokenRemainQuotaForTest(t, tokenID)+getTokenUsedQuotaForTest(t, tokenID))

	redemption := getRedemptionForTest(t, redeemKey)
	assert.Equal(t, common.RedemptionCodeStatusUsed, redemption.Status)
	assert.Equal(t, userID, redemption.UsedUserId)
}

func TestTokenRedeem_LogIncludesTokenName(t *testing.T) {
	truncateTables(t)

	const userID = 11
	const tokenID = 11
	const tokenKey = "tokenredeem11"
	const tokenName = "Claude 专用令牌"
	const redeemKey = "redeem-code-011"

	seedRedeemUser(t, userID, 0)
	seedRedeemTokenWithName(t, tokenID, userID, tokenKey, tokenName, 0, 0)
	seedRedemptionCode(t, 11, redeemKey, 500000)

	w := performTokenRedeemRequest(t, makeTokenRedeemRouter(), "Bearer sk-"+tokenKey, redeemKey)

	require.Equal(t, http.StatusOK, w.Code)
	log := getLatestTopupLogForTest(t, userID)
	assert.Contains(t, log.Content, tokenName)
	assert.Contains(t, log.Content, "兑换到令牌")
}

func TestTokenRedeem_InvalidToken(t *testing.T) {
	truncateTables(t)

	w := performTokenRedeemRequest(t, makeTokenRedeemRouter(), "Bearer sk-invalidtoken", "redeem-code-001")

	require.Equal(t, http.StatusUnauthorized, w.Code)
	var resp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	require.NoError(t, common.Unmarshal([]byte(w.Body.String()), &resp))
	assert.False(t, resp.Success)
	assert.Equal(t, "无效的令牌", resp.Message)
}

func TestTokenRedeem_InvalidRedemption(t *testing.T) {
	truncateTables(t)

	const userID = 2
	const tokenID = 2
	const initQuota = 300
	const tokenKey = "tokenredeem2"

	seedRedeemUser(t, userID, initQuota)
	seedRedeemToken(t, tokenID, userID, tokenKey, 1000, 0)

	w := performTokenRedeemRequest(t, makeTokenRedeemRouter(), "Bearer sk-"+tokenKey, "missing-redemption")

	require.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	require.NoError(t, common.Unmarshal([]byte(w.Body.String()), &resp))
	assert.False(t, resp.Success)
	assert.NotEmpty(t, resp.Message)
	assert.Equal(t, initQuota, getUserQuotaForTest(t, userID))
}
