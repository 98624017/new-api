package controller

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

// UpdateTaskBulk 薄入口，实际轮询逻辑在 service 层
func UpdateTaskBulk() {
	service.TaskPollingLoop()
}

func GetAllTask(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)

	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	// 解析其他查询参数
	queryParams := model.SyncTaskQueryParams{
		Platform:       constant.TaskPlatform(c.Query("platform")),
		TaskID:         c.Query("task_id"),
		Status:         c.Query("status"),
		Action:         c.Query("action"),
		StartTimestamp: startTimestamp,
		EndTimestamp:   endTimestamp,
		ChannelID:      c.Query("channel_id"),
	}

	items := model.TaskGetAllTasks(pageInfo.GetStartIdx(), pageInfo.GetPageSize(), queryParams)
	total := model.TaskCountAllTasks(queryParams)
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(tasksToDto(items, true))
	common.ApiSuccess(c, pageInfo)
}

func GetUserTask(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)

	userId := c.GetInt("id")

	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)

	queryParams := model.SyncTaskQueryParams{
		Platform:       constant.TaskPlatform(c.Query("platform")),
		TaskID:         c.Query("task_id"),
		Status:         c.Query("status"),
		Action:         c.Query("action"),
		StartTimestamp: startTimestamp,
		EndTimestamp:   endTimestamp,
	}

	items := model.TaskGetAllUserTask(userId, pageInfo.GetStartIdx(), pageInfo.GetPageSize(), queryParams)
	total := model.TaskCountAllUserTask(userId, queryParams)
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(tasksToDto(items, false))
	common.ApiSuccess(c, pageInfo)
}

func GetUserTokenTask(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)

	userId := c.GetInt("id")
	tokenId := c.GetInt("token_id")

	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)

	queryParams := model.SyncTaskQueryParams{
		Platform:       constant.TaskPlatform(c.Query("platform")),
		TaskID:         c.Query("task_id"),
		Status:         c.Query("status"),
		Action:         c.Query("action"),
		StartTimestamp: startTimestamp,
		EndTimestamp:   endTimestamp,
	}

	items := model.TaskGetAllUserTokenTask(userId, tokenId, pageInfo.GetStartIdx(), pageInfo.GetPageSize(), queryParams)
	total := model.TaskCountAllUserTokenTask(userId, tokenId, queryParams)
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(tasksToDto(items, false))
	common.ApiSuccess(c, pageInfo)
}

type deleteUserTokenAssetRequest struct {
	TaskID string `json:"task_id"`
}

type seedanceAssetTaskData struct {
	Model     string `json:"model,omitempty"`
	AssetID   string `json:"asset_id,omitempty"`
	AssetURI  string `json:"asset_uri,omitempty"`
	Deleted   bool   `json:"deleted,omitempty"`
	DeletedAt int64  `json:"deleted_at,omitempty"`
	Metadata  struct {
		Seedance struct {
			Kind      string `json:"kind,omitempty"`
			AssetID   string `json:"asset_id,omitempty"`
			AssetURI  string `json:"asset_uri,omitempty"`
			Deleted   bool   `json:"deleted,omitempty"`
			DeletedAt int64  `json:"deleted_at,omitempty"`
		} `json:"seedance,omitempty"`
	} `json:"metadata,omitempty"`
}

func DeleteUserTokenAsset(c *gin.Context) {
	var req deleteUserTokenAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, fmt.Errorf("invalid request body"))
		return
	}
	req.TaskID = strings.TrimSpace(req.TaskID)
	if req.TaskID == "" {
		common.ApiError(c, fmt.Errorf("task_id is required"))
		return
	}

	userId := c.GetInt("id")
	tokenId := c.GetInt("token_id")
	task, exist, err := model.GetByUserTokenTaskId(userId, tokenId, req.TaskID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if !exist {
		common.ApiError(c, fmt.Errorf("task not found"))
		return
	}

	assetData, err := validateDeletableSeedanceAsset(task)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := deleteUpstreamSeedanceAsset(task); err != nil {
		common.ApiError(c, err)
		return
	}

	deletedAt := time.Now().Unix()
	if err := markSeedanceAssetDeleted(task, deletedAt); err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, gin.H{
		"task_id":    task.TaskID,
		"deleted":    true,
		"deleted_at": deletedAt,
		"asset_id":   firstNonEmpty(assetData.AssetID, assetData.Metadata.Seedance.AssetID),
		"asset_uri":  firstNonEmpty(assetData.AssetURI, assetData.Metadata.Seedance.AssetURI),
	})
}

func validateDeletableSeedanceAsset(task *model.Task) (*seedanceAssetTaskData, error) {
	if task.Status != model.TaskStatusSuccess {
		return nil, fmt.Errorf("only successful asset tasks can be deleted")
	}
	var data seedanceAssetTaskData
	if len(task.Data) > 0 {
		if err := common.Unmarshal(task.Data, &data); err != nil {
			return nil, fmt.Errorf("invalid task data")
		}
	}
	if data.Deleted || data.Metadata.Seedance.Deleted {
		return nil, fmt.Errorf("asset already deleted")
	}
	if !isSeedanceAssetTask(task, &data) {
		return nil, fmt.Errorf("task is not a seedance asset")
	}
	return &data, nil
}

func isSeedanceAssetTask(task *model.Task, data *seedanceAssetTaskData) bool {
	return strings.TrimSpace(data.Model) == "seedance-asset" ||
		strings.TrimSpace(task.Properties.OriginModelName) == "seedance-asset" ||
		strings.TrimSpace(data.Metadata.Seedance.Kind) == "asset"
}

func deleteUpstreamSeedanceAsset(task *model.Task) error {
	channel, err := model.GetChannelById(task.ChannelId, true)
	if err != nil {
		return fmt.Errorf("get channel failed: %w", err)
	}
	key, _, newAPIError := channel.GetNextEnabledKey()
	if newAPIError != nil {
		return fmt.Errorf("get channel key failed: %w", newAPIError)
	}
	baseURL := strings.TrimRight(channel.GetBaseURL(), "/")
	if baseURL == "" {
		return fmt.Errorf("channel base_url is empty")
	}
	upstreamTaskID := strings.TrimSpace(task.GetUpstreamTaskID())
	if upstreamTaskID == "" {
		return fmt.Errorf("upstream task_id is missing")
	}
	body, err := common.Marshal(map[string]any{"task_id": upstreamTaskID})
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/task/token/asset/delete", baseURL), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")

	client, err := service.GetHttpClientWithProxy(channel.GetSetting().Proxy)
	if err != nil {
		return err
	}
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("seedance asset delete failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func markSeedanceAssetDeleted(task *model.Task, deletedAt int64) error {
	var raw map[string]any
	if len(task.Data) > 0 {
		if err := common.Unmarshal(task.Data, &raw); err != nil {
			return fmt.Errorf("invalid task data")
		}
	}
	if raw == nil {
		raw = map[string]any{}
	}
	raw["deleted"] = true
	raw["deleted_at"] = deletedAt

	metadata := ensureMap(raw, "metadata")
	seedance := ensureMap(metadata, "seedance")
	seedance["deleted"] = true
	seedance["deleted_at"] = deletedAt

	body, err := common.Marshal(raw)
	if err != nil {
		return err
	}
	task.Data = bytes.TrimSpace(body)
	return task.Update()
}

func ensureMap(parent map[string]any, key string) map[string]any {
	if existing, ok := parent[key].(map[string]any); ok {
		return existing
	}
	next := map[string]any{}
	parent[key] = next
	return next
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func tasksToDto(tasks []*model.Task, fillUser bool) []*dto.TaskDto {
	var userIdMap map[int]*model.UserBase
	if fillUser {
		userIdMap = make(map[int]*model.UserBase)
		userIds := types.NewSet[int]()
		for _, task := range tasks {
			userIds.Add(task.UserId)
		}
		for _, userId := range userIds.Items() {
			cacheUser, err := model.GetUserCache(userId)
			if err == nil {
				userIdMap[userId] = cacheUser
			}
		}
	}
	result := make([]*dto.TaskDto, len(tasks))
	for i, task := range tasks {
		if fillUser {
			if user, ok := userIdMap[task.UserId]; ok {
				task.Username = user.Username
			}
		}
		result[i] = relay.TaskModel2Dto(task)
	}
	return result
}
