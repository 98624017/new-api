package sora

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
)

func TestEstimateBillingDoublesWhenEnvConfiguredModelHasReferenceVideo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("SORA_REFERENCE_VIDEO_DOUBLE_PRICE_MODELS", "seedance-2.0")
	ReloadReferenceVideoDoublePriceModelsFromEnv()
	t.Cleanup(ReloadReferenceVideoDoublePriceModelsFromEnv)

	ratios := estimateBillingForBody(t, referenceVideoRequestBody("seedance-2.0"))
	if ratios["video_input"] != 2 {
		t.Fatalf("expected video_input ratio 2, got %#v", ratios)
	}
}

func TestEstimateBillingDoesNotDoubleWhenEnvDoesNotListModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("SORA_REFERENCE_VIDEO_DOUBLE_PRICE_MODELS", "")
	ReloadReferenceVideoDoublePriceModelsFromEnv()
	t.Cleanup(ReloadReferenceVideoDoublePriceModelsFromEnv)

	ratios := estimateBillingForBody(t, referenceVideoRequestBody("seedance-2.0"))
	if _, ok := ratios["video_input"]; ok {
		t.Fatalf("expected no video_input ratio without env whitelist, got %#v", ratios)
	}
}

func TestReferenceVideoDoublePriceModelsRequireExplicitReload(t *testing.T) {
	t.Setenv("SORA_REFERENCE_VIDEO_DOUBLE_PRICE_MODELS", "")
	ReloadReferenceVideoDoublePriceModelsFromEnv()
	t.Setenv("SORA_REFERENCE_VIDEO_DOUBLE_PRICE_MODELS", "seedance-2.0")
	t.Cleanup(ReloadReferenceVideoDoublePriceModelsFromEnv)

	if IsReferenceVideoDoublePriceModel("seedance-2.0") {
		t.Fatal("expected env change not to load until explicit reload")
	}

	ReloadReferenceVideoDoublePriceModelsFromEnv()
	if !IsReferenceVideoDoublePriceModel("seedance-2.0") {
		t.Fatal("expected explicit reload to load env whitelist")
	}
}

func estimateBillingForBody(t *testing.T, body string) map[string]float64 {
	t.Helper()
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest("POST", "/v1/videos", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	info := &common.RelayInfo{TaskRelayInfo: &common.TaskRelayInfo{}}
	adaptor := &TaskAdaptor{}
	if taskErr := adaptor.ValidateRequestAndSetAction(c, info); taskErr != nil {
		t.Fatalf("ValidateRequestAndSetAction returned error: code=%s message=%s", taskErr.Code, taskErr.Message)
	}

	return adaptor.EstimateBilling(c, info)
}

func referenceVideoRequestBody(modelName string) string {
	return `{
		"model": "` + modelName + `",
		"duration": 10,
		"generate_audio": true,
		"ratio": "16:9",
		"prompt": "placeholder",
		"content": [
			{"type": "text", "text": "keep identity"},
			{
				"type": "video_url",
				"role": "reference_video",
				"video_url": {"url": "https://example.com/reference.mp4"}
			}
		]
	}`
}
