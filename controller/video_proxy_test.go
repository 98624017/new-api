package controller_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestVideoProxyUsesStoredResultURLForSoraTask(t *testing.T) {
	truncateTables(t)
	allowControllerLocalVideoServers(t)
	service.InitHttpClient()

	const userID = 9101
	const channelID = 9102

	var resultHit bool
	resultServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resultHit = true
		w.Header().Set("Content-Type", "video/mp4")
		_, _ = w.Write([]byte("video-bytes"))
	}))
	defer resultServer.Close()

	var upstreamHit bool
	upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamHit = true
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer upstreamServer.Close()

	channel := &model.Channel{
		Id:     channelID,
		Type:   constant.ChannelTypeSora,
		Key:    "sora-key",
		Status: common.ChannelStatusEnabled,
		Name:   "sora result url test",
	}
	channel.BaseURL = &upstreamServer.URL
	require.NoError(t, model.DB.Create(channel).Error)

	task := &model.Task{
		TaskID:    "task_sora_result_url",
		UserId:    userID,
		ChannelId: channelID,
		Platform:  constant.TaskPlatform("openai"),
		Action:    "generate",
		Status:    model.TaskStatusSuccess,
		Progress:  "100%",
		PrivateData: model.TaskPrivateData{
			UpstreamTaskID: "video_upstream",
			ResultURL:      resultServer.URL + "/result.mp4",
		},
	}
	task.SetData(map[string]any{
		"id":       "video_upstream",
		"object":   "video",
		"model":    "seedance",
		"status":   "unknown",
		"progress": 100,
	})
	require.NoError(t, model.DB.Create(task).Error)

	router := gin.New()
	router.GET("/v1/videos/:task_id/content", func(c *gin.Context) {
		c.Set("id", userID)
		controller.VideoProxy(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/videos/task_sora_result_url/content", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "video-bytes", w.Body.String())
	require.True(t, resultHit, "expected stored result url to be fetched")
	require.False(t, upstreamHit, "expected upstream /content not to be fetched when result url is stored")
}

func allowControllerLocalVideoServers(t *testing.T) {
	t.Helper()
	fetchSetting := system_setting.GetFetchSetting()
	original := *fetchSetting
	fetchSetting.EnableSSRFProtection = false
	t.Cleanup(func() {
		*fetchSetting = original
	})
}
