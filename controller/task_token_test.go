package controller_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type taskPageResponse struct {
	Total int                `json:"total"`
	Items []taskResponseItem `json:"items"`
}

type taskResponseItem struct {
	TaskID string `json:"task_id"`
	Status string `json:"status"`
}

func makeTokenTaskRouter() *gin.Engine {
	router := gin.New()
	router.GET("/api/task/token/self", middleware.TokenAuthReadOnly(), controller.GetUserTokenTask)
	return router
}

func performTokenTaskRequest(t *testing.T, router *gin.Engine, auth string, rawQuery string) *httptest.ResponseRecorder {
	t.Helper()

	target := "/api/task/token/self"
	if rawQuery != "" {
		target += "?" + rawQuery
	}
	req := httptest.NewRequest(http.MethodGet, target, nil)
	req.Header.Set("Accept-Language", "zh-CN")
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func seedAsyncTask(t *testing.T, taskID string, userID int, legacyTokenID int, status model.TaskStatus, submitTime int64) {
	t.Helper()

	task := &model.Task{
		TaskID:     taskID,
		UserId:     userID,
		Platform:   constant.TaskPlatform("openai"),
		Action:     "generate",
		Status:     status,
		Progress:   "100%",
		SubmitTime: submitTime,
		Properties: model.Properties{
			OriginModelName: "sora-2",
		},
		PrivateData: model.TaskPrivateData{
			TokenId: legacyTokenID,
		},
	}
	require.NoError(t, model.DB.Create(task).Error)
}

func decodeTaskPageResponse(t *testing.T, body []byte) taskPageResponse {
	t.Helper()

	var resp struct {
		Success bool             `json:"success"`
		Message string           `json:"message"`
		Data    taskPageResponse `json:"data"`
	}
	require.NoError(t, common.Unmarshal(body, &resp))
	require.True(t, resp.Success, "unexpected error: %s", resp.Message)
	return resp.Data
}

func getTaskTokenIDForTest(t *testing.T, taskID string) int {
	t.Helper()

	var task model.Task
	require.NoError(t, model.DB.Select("token_id").Where("task_id = ?", taskID).First(&task).Error)
	return task.TokenId
}

func TestGetUserTokenTask_ListOnlyCurrentTokenTasks(t *testing.T) {
	truncateTables(t)

	const userID = 21
	const otherUserID = 22
	const tokenID = 2101
	const otherTokenID = 2102
	const tokenKey = "tasktoken2101"

	seedRedeemUser(t, userID, 0)
	seedRedeemUser(t, otherUserID, 0)
	seedRedeemToken(t, tokenID, userID, tokenKey, 1000, 0)
	seedRedeemToken(t, otherTokenID, userID, "tasktoken2102", 1000, 0)

	// 模拟补丁上线前的老数据：只有 private_data.token_id，没有独立 token_id 列。
	seedAsyncTask(t, "task-token-match-1", userID, tokenID, model.TaskStatusSuccess, 100)
	seedAsyncTask(t, "task-token-other", userID, otherTokenID, model.TaskStatusSuccess, 200)
	seedAsyncTask(t, "task-token-match-2", userID, tokenID, model.TaskStatusInProgress, 300)
	seedAsyncTask(t, "task-other-user", otherUserID, tokenID, model.TaskStatusSuccess, 400)

	w := performTokenTaskRequest(t, makeTokenTaskRouter(), "Bearer sk-"+tokenKey, "p=1&page_size=10")

	require.Equal(t, http.StatusOK, w.Code)
	page := decodeTaskPageResponse(t, w.Body.Bytes())
	require.Equal(t, 2, page.Total)
	require.Len(t, page.Items, 2)
	assert.Equal(t, "task-token-match-2", page.Items[0].TaskID)
	assert.Equal(t, string(model.TaskStatusInProgress), page.Items[0].Status)
	assert.Equal(t, "task-token-match-1", page.Items[1].TaskID)
	assert.Equal(t, string(model.TaskStatusSuccess), page.Items[1].Status)
	assert.Equal(t, tokenID, getTaskTokenIDForTest(t, "task-token-match-1"))
	assert.Equal(t, tokenID, getTaskTokenIDForTest(t, "task-token-match-2"))
	assert.Equal(t, otherTokenID, getTaskTokenIDForTest(t, "task-token-other"))
}

func TestGetUserTokenTask_SupportsTaskIDFilter(t *testing.T) {
	truncateTables(t)

	const userID = 31
	const tokenID = 3101
	const tokenKey = "tasktoken3101"

	seedRedeemUser(t, userID, 0)
	seedRedeemToken(t, tokenID, userID, tokenKey, 1000, 0)
	seedAsyncTask(t, "task-filter-hit", userID, tokenID, model.TaskStatusSuccess, 100)
	seedAsyncTask(t, "task-filter-miss", userID, tokenID, model.TaskStatusFailure, 200)

	w := performTokenTaskRequest(t, makeTokenTaskRouter(), "Bearer sk-"+tokenKey, "p=1&page_size=10&task_id=task-filter-hit")

	require.Equal(t, http.StatusOK, w.Code)
	page := decodeTaskPageResponse(t, w.Body.Bytes())
	require.Equal(t, 1, page.Total)
	require.Len(t, page.Items, 1)
	assert.Equal(t, "task-filter-hit", page.Items[0].TaskID)
	assert.Equal(t, string(model.TaskStatusSuccess), page.Items[0].Status)
}
