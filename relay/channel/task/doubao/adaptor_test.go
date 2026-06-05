package doubao

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
)

func TestEstimateBillingDoublePriceForWhitelistedOpenAIVideosReferenceVideo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("SEEDANCE_REFERENCE_VIDEO_DOUBLE_PRICE_MODELS", "doubao-seedance-2-0-260128-2")
	ReloadReferenceVideoDoublePriceModelsFromEnv()
	t.Cleanup(ReloadReferenceVideoDoublePriceModelsFromEnv)

	for _, body := range []string{
		`{"model":"doubao-seedance-2-0-260128-2","prompt":"p","input_video":["https://cdn.example.com/ref.mp4"]}`,
		`{"model":"doubao-seedance-2-0-260128-2","prompt":"p","video_url":"https://cdn.example.com/ref.mov"}`,
		`{"model":"doubao-seedance-2-0-260128-2","prompt":"p","reference_video":"https://cdn.example.com/ref.webm"}`,
		`{"model":"doubao-seedance-2-0-260128-2","prompt":"p","files":["https://cdn.example.com/ref-image.jpg","https://cdn.example.com/ref-video.mp4"]}`,
	} {
		ratios := estimateDoubaoBillingForBody(t, body)
		if ratios["video_input"] != 2 {
			t.Fatalf("expected video_input double price for body %s, got %#v", body, ratios)
		}
	}
}

func TestEstimateBillingDoesNotDoublePriceNonVideoFiles(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("SEEDANCE_REFERENCE_VIDEO_DOUBLE_PRICE_MODELS", "doubao-seedance-2-0-260128-2")
	ReloadReferenceVideoDoublePriceModelsFromEnv()
	t.Cleanup(ReloadReferenceVideoDoublePriceModelsFromEnv)

	body := `{
		"model": "doubao-seedance-2-0-260128-2",
		"prompt": "p",
		"files": ["https://cdn.example.com/ref.jpg", "https://cdn.example.com/ref.mp3"]
	}`
	ratios := estimateDoubaoBillingForBody(t, body)
	if _, ok := ratios["video_input"]; ok {
		t.Fatalf("expected no video_input ratio for image/audio files, got %#v", ratios)
	}
}

func TestEstimateBillingDoesNotDoublePriceWhenModelNotWhitelisted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("SEEDANCE_REFERENCE_VIDEO_DOUBLE_PRICE_MODELS", "")
	ReloadReferenceVideoDoublePriceModelsFromEnv()
	t.Cleanup(ReloadReferenceVideoDoublePriceModelsFromEnv)

	body := `{
		"model": "doubao-seedance-2-0-260128-2",
		"prompt": "p",
		"input_video": ["https://cdn.example.com/ref.mp4"]
	}`
	ratios := estimateDoubaoBillingForBody(t, body)
	if _, ok := ratios["video_input"]; ok {
		t.Fatalf("expected no video_input ratio without env whitelist, got %#v", ratios)
	}
}

func TestEstimateBillingDoesNotDoublePriceTopLevelVideoForLegacyVolcEngine(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("SEEDANCE_REFERENCE_VIDEO_DOUBLE_PRICE_MODELS", "doubao-seedance-2-0-260128-2")
	ReloadReferenceVideoDoublePriceModelsFromEnv()
	t.Cleanup(ReloadReferenceVideoDoublePriceModelsFromEnv)

	body := `{
		"model": "doubao-seedance-2-0-260128-2",
		"prompt": "p",
		"input_video": ["https://cdn.example.com/ref.mp4"]
	}`
	ratios := estimateDoubaoBillingForBodyWithChannel(t, body, constant.ChannelTypeVolcEngine)
	if _, ok := ratios["video_input"]; ok {
		t.Fatalf("expected legacy volcengine top-level input_video not to trigger video_input ratio, got %#v", ratios)
	}
}

func TestEstimateBillingKeepsMetadataVideoInputRatio(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("SEEDANCE_REFERENCE_VIDEO_DOUBLE_PRICE_MODELS", "")
	ReloadReferenceVideoDoublePriceModelsFromEnv()
	t.Cleanup(ReloadReferenceVideoDoublePriceModelsFromEnv)

	body := `{
		"model": "doubao-seedance-2-0-260128",
		"prompt": "p",
		"metadata": {
			"content": [
				{"type": "video_url", "video_url": {"url": "https://cdn.example.com/ref.mp4"}}
			]
		}
	}`
	ratios := estimateDoubaoBillingForBody(t, body)
	want, _ := GetVideoInputRatio("doubao-seedance-2-0-260128")
	if ratios["video_input"] != want {
		t.Fatalf("expected metadata video_input ratio %v, got %#v", want, ratios)
	}
}

func TestEstimateBillingDoublePriceForWhitelistedMetadataReferenceVideo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("SEEDANCE_REFERENCE_VIDEO_DOUBLE_PRICE_MODELS", "doubao-seedance-2-0-260128")
	ReloadReferenceVideoDoublePriceModelsFromEnv()
	t.Cleanup(ReloadReferenceVideoDoublePriceModelsFromEnv)

	body := `{
		"model": "doubao-seedance-2-0-260128",
		"prompt": "p",
		"metadata": {
			"content": [
				{"type": "video_url", "video_url": {"url": "https://cdn.example.com/ref.mp4"}}
			]
		}
	}`
	ratios := estimateDoubaoBillingForBody(t, body)
	if ratios["video_input"] != 2 {
		t.Fatalf("expected whitelist metadata video_input double price, got %#v", ratios)
	}
}

func TestBuildRequestBodyKeepsSeedanceOpenAIVideosFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := `{
		"model": "doubao-seedance-2-0-260128-2",
		"prompt": "p",
		"seconds": "4",
		"aspect_ratio": "16:9",
		"files": ["https://cdn.example.com/ref-image.jpg", "https://cdn.example.com/ref-video.mp4"],
		"input_video": ["https://cdn.example.com/ref-video-2.mp4"],
		"audio": ["https://cdn.example.com/ref-audio.mp3"],
		"generate_audio": true,
		"resolution": "480p"
	}`
	c := newDoubaoTaskContext(body)
	info := &relaycommon.RelayInfo{
		TaskRelayInfo: &relaycommon.TaskRelayInfo{},
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeDoubaoVideo,
			ChannelBaseUrl:    "https://seedance.example.com",
			UpstreamModelName: "mapped-seedance-model",
			IsModelMapped:     true,
		},
	}
	adaptor := &TaskAdaptor{}
	adaptor.Init(info)
	if taskErr := adaptor.ValidateRequestAndSetAction(c, info); taskErr != nil {
		t.Fatalf("ValidateRequestAndSetAction returned error: code=%s message=%s", taskErr.Code, taskErr.Message)
	}
	url, err := adaptor.BuildRequestURL(info)
	if err != nil {
		t.Fatal(err)
	}
	if url != "https://seedance.example.com/v1/videos" {
		t.Fatalf("unexpected seedance URL: %s", url)
	}
	reader, err := adaptor.BuildRequestBody(c, info)
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]interface{}
	if err := common.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got["model"] != "mapped-seedance-model" {
		t.Fatalf("expected mapped model, got %v", got["model"])
	}
	if got["duration"] != "4" {
		t.Fatalf("expected seconds to populate duration, got %#v", got["duration"])
	}
	for _, field := range []string{"files", "input_video", "audio", "aspect_ratio", "generate_audio", "resolution"} {
		if _, ok := got[field]; !ok {
			t.Fatalf("expected seedance body to keep %s, got %#v", field, got)
		}
	}
}

func TestBuildRequestBodyKeepsLegacyVolcEngineFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := `{
		"model": "doubao-seedance-2-0-260128",
		"prompt": "p",
		"metadata": {
			"content": [
				{"type": "video_url", "video_url": {"url": "https://cdn.example.com/ref.mp4"}}
			]
		}
	}`
	c := newDoubaoTaskContext(body)
	info := &relaycommon.RelayInfo{
		TaskRelayInfo: &relaycommon.TaskRelayInfo{},
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:    constant.ChannelTypeVolcEngine,
			ChannelBaseUrl: "https://ark.cn-beijing.volces.com",
		},
	}
	adaptor := &TaskAdaptor{}
	adaptor.Init(info)
	if taskErr := adaptor.ValidateRequestAndSetAction(c, info); taskErr != nil {
		t.Fatalf("ValidateRequestAndSetAction returned error: code=%s message=%s", taskErr.Code, taskErr.Message)
	}
	url, err := adaptor.BuildRequestURL(info)
	if err != nil {
		t.Fatal(err)
	}
	if url != "https://ark.cn-beijing.volces.com/api/v3/contents/generations/tasks" {
		t.Fatalf("unexpected legacy URL: %s", url)
	}
	reader, err := adaptor.BuildRequestBody(c, info)
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]interface{}
	if err := common.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if _, ok := got["content"]; !ok {
		t.Fatalf("expected legacy body to contain content, got %#v", got)
	}
	if _, ok := got["files"]; ok {
		t.Fatalf("expected legacy body not to pass files through, got %#v", got)
	}
}

func estimateDoubaoBillingForBody(t *testing.T, body string) map[string]float64 {
	t.Helper()
	return estimateDoubaoBillingForBodyWithChannel(t, body, constant.ChannelTypeDoubaoVideo)
}

func estimateDoubaoBillingForBodyWithChannel(t *testing.T, body string, channelType int) map[string]float64 {
	t.Helper()
	c := newDoubaoTaskContext(body)
	info := &relaycommon.RelayInfo{
		TaskRelayInfo: &relaycommon.TaskRelayInfo{},
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType: channelType,
		},
	}
	adaptor := &TaskAdaptor{}
	adaptor.Init(info)
	if taskErr := adaptor.ValidateRequestAndSetAction(c, info); taskErr != nil {
		t.Fatalf("ValidateRequestAndSetAction returned error: code=%s message=%s", taskErr.Code, taskErr.Message)
	}
	ratios := adaptor.EstimateBilling(c, info)
	if ratios == nil {
		return map[string]float64{}
	}
	return ratios
}

func newDoubaoTaskContext(body string) *gin.Context {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest("POST", "/v1/videos", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	return c
}
