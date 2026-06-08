package controller_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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
	router.POST("/api/task/token/asset/delete", middleware.TokenAuthReadOnly(), controller.DeleteUserTokenAsset)
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

func seedAsyncTask(t *testing.T, taskID string, userID int, tokenID int, status model.TaskStatus, submitTime int64) {
	t.Helper()

	task := &model.Task{
		TaskID:     taskID,
		UserId:     userID,
		TokenId:    tokenID,
		Platform:   constant.TaskPlatform("openai"),
		Action:     "generate",
		Status:     status,
		Progress:   "100%",
		SubmitTime: submitTime,
		Properties: model.Properties{
			OriginModelName: "sora-2",
		},
		PrivateData: model.TaskPrivateData{
			TokenId: tokenID,
		},
	}
	require.NoError(t, model.DB.Create(task).Error)
}

func seedAssetTask(t *testing.T, taskID string, upstreamTaskID string, userID int, tokenID int, channelID int, status model.TaskStatus, data map[string]any) {
	t.Helper()

	originModel := "seedance-asset"
	if modelName, ok := data["model"].(string); ok && strings.TrimSpace(modelName) != "" {
		originModel = strings.TrimSpace(modelName)
	}
	task := &model.Task{
		TaskID:    taskID,
		UserId:    userID,
		TokenId:   tokenID,
		ChannelId: channelID,
		Platform:  constant.TaskPlatform("openai"),
		Action:    "generate",
		Status:    status,
		Progress:  "100%",
		Properties: model.Properties{
			OriginModelName: originModel,
		},
		PrivateData: model.TaskPrivateData{
			TokenId:        tokenID,
			UpstreamTaskID: upstreamTaskID,
		},
	}
	task.SetData(data)
	require.NoError(t, model.DB.Create(task).Error)
}

func seedTaskChannel(t *testing.T, id int, baseURL string, key string) {
	t.Helper()

	channel := &model.Channel{
		Id:     id,
		Type:   constant.ChannelTypeSora,
		Key:    key,
		Status: common.ChannelStatusEnabled,
		Name:   "seedance proxy",
	}
	channel.BaseURL = &baseURL
	require.NoError(t, model.DB.Create(channel).Error)
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

func performDeleteAssetRequest(t *testing.T, router *gin.Engine, auth string, taskID string) *httptest.ResponseRecorder {
	t.Helper()

	body, err := common.Marshal(map[string]any{"task_id": taskID})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/api/task/token/asset/delete", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "zh-CN")
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func decodeGenericAPIResponse(t *testing.T, body []byte) struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
} {
	t.Helper()
	var resp struct {
		Success bool            `json:"success"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	require.NoError(t, common.Unmarshal(body, &resp))
	return resp
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

func TestDeleteUserTokenAsset_DeletesCurrentTokenAssetAndMarksTaskData(t *testing.T) {
	truncateTables(t)

	const userID = 41
	const tokenID = 4101
	const tokenKey = "tasktoken4101"
	const channelID = 6101
	var upstreamMethod string
	var upstreamPath string
	var upstreamAuth string
	var upstreamBody map[string]any

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamMethod = r.Method
		upstreamPath = r.URL.Path
		upstreamAuth = r.Header.Get("Authorization")
		require.NoError(t, common.DecodeJson(r.Body, &upstreamBody))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"message":"","data":{"task_id":"asset_req_1780830000_abcdef123456","deleted":true,"deleted_at":1780830000,"resource_id":123}}`))
	}))
	defer upstream.Close()

	seedRedeemUser(t, userID, 0)
	seedRedeemToken(t, tokenID, userID, tokenKey, 1000, 0)
	seedTaskChannel(t, channelID, upstream.URL, "seedance-token")
	seedAssetTask(t, "task_public_delete_ok", "asset_req_1780830000_abcdef123456", userID, tokenID, channelID, model.TaskStatusSuccess, map[string]any{
		"model":     "seedance-asset",
		"asset_id":  "asset-123",
		"asset_uri": "asset://asset-123",
		"metadata": map[string]any{
			"seedance": map[string]any{
				"kind":        "asset",
				"resource_id": float64(123),
				"asset_id":    "asset-123",
				"asset_uri":   "asset://asset-123",
			},
		},
	})

	w := performDeleteAssetRequest(t, makeTokenTaskRouter(), "Bearer sk-"+tokenKey, "task_public_delete_ok")

	require.Equal(t, http.StatusOK, w.Code)
	resp := decodeGenericAPIResponse(t, w.Body.Bytes())
	require.True(t, resp.Success, "unexpected error: %s", resp.Message)
	assert.Equal(t, http.MethodPost, upstreamMethod)
	assert.Equal(t, "/api/task/token/asset/delete", upstreamPath)
	assert.Equal(t, "Bearer seedance-token", upstreamAuth)
	assert.Equal(t, "asset_req_1780830000_abcdef123456", upstreamBody["task_id"])

	var payload map[string]any
	require.NoError(t, common.Unmarshal(resp.Data, &payload))
	assert.Equal(t, "task_public_delete_ok", payload["task_id"])
	assert.Equal(t, true, payload["deleted"])
	assert.Equal(t, "asset-123", payload["asset_id"])
	assert.Equal(t, "asset://asset-123", payload["asset_uri"])

	var task model.Task
	require.NoError(t, model.DB.Where("task_id = ?", "task_public_delete_ok").First(&task).Error)
	var taskData map[string]any
	require.NoError(t, common.Unmarshal(task.Data, &taskData))
	assert.Equal(t, true, taskData["deleted"])
	metadata := taskData["metadata"].(map[string]any)["seedance"].(map[string]any)
	assert.Equal(t, true, metadata["deleted"])
	assert.NotZero(t, taskData["deleted_at"])
	assert.NotZero(t, metadata["deleted_at"])
}

func TestDeleteUserTokenAsset_RejectsOtherTokenTask(t *testing.T) {
	truncateTables(t)

	const userID = 42
	const tokenID = 4201
	const otherTokenID = 4202
	const tokenKey = "tasktoken4201"
	const channelID = 6201
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("upstream should not be called")
	}))
	defer upstream.Close()

	seedRedeemUser(t, userID, 0)
	seedRedeemToken(t, tokenID, userID, tokenKey, 1000, 0)
	seedRedeemToken(t, otherTokenID, userID, "tasktoken4202", 1000, 0)
	seedTaskChannel(t, channelID, upstream.URL, "seedance-token")
	seedAssetTask(t, "asset_req_other_token", "asset_req_1780830000_other", userID, otherTokenID, channelID, model.TaskStatusSuccess, map[string]any{
		"model": "seedance-asset",
		"metadata": map[string]any{
			"seedance": map[string]any{"kind": "asset", "resource_id": float64(123)},
		},
	})

	w := performDeleteAssetRequest(t, makeTokenTaskRouter(), "Bearer sk-"+tokenKey, "asset_req_other_token")

	require.Equal(t, http.StatusOK, w.Code)
	resp := decodeGenericAPIResponse(t, w.Body.Bytes())
	require.False(t, resp.Success)
	assert.Contains(t, resp.Message, "task not found")
}

func TestDeleteUserTokenAsset_RejectsInvalidAssetStates(t *testing.T) {
	tests := []struct {
		name   string
		status model.TaskStatus
		data   map[string]any
		want   string
	}{
		{
			name:   "not asset",
			status: model.TaskStatusSuccess,
			data:   map[string]any{"model": "sora-2"},
			want:   "task is not a seedance asset",
		},
		{
			name:   "not success",
			status: model.TaskStatusInProgress,
			data: map[string]any{
				"model": "seedance-asset",
				"metadata": map[string]any{
					"seedance": map[string]any{"kind": "asset", "resource_id": float64(123)},
				},
			},
			want: "only successful asset tasks can be deleted",
		},
		{
			name:   "already deleted",
			status: model.TaskStatusSuccess,
			data: map[string]any{
				"model":   "seedance-asset",
				"deleted": true,
				"metadata": map[string]any{
					"seedance": map[string]any{"kind": "asset", "resource_id": float64(123)},
				},
			},
			want: "asset already deleted",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			truncateTables(t)

			const userID = 43
			const tokenID = 4301
			const tokenKey = "tasktoken4301"
			const channelID = 6301
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Fatalf("upstream should not be called")
			}))
			defer upstream.Close()

			seedRedeemUser(t, userID, 0)
			seedRedeemToken(t, tokenID, userID, tokenKey, 1000, 0)
			seedTaskChannel(t, channelID, upstream.URL, "seedance-token")
			taskID := "asset_req_invalid_" + strings.ReplaceAll(tc.name, " ", "_")
			seedAssetTask(t, taskID, "asset_req_1780830000_invalid", userID, tokenID, channelID, tc.status, tc.data)

			w := performDeleteAssetRequest(t, makeTokenTaskRouter(), "Bearer sk-"+tokenKey, taskID)

			require.Equal(t, http.StatusOK, w.Code)
			resp := decodeGenericAPIResponse(t, w.Body.Bytes())
			require.False(t, resp.Success)
			assert.Contains(t, resp.Message, tc.want)
		})
	}
}
