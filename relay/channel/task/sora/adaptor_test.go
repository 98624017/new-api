package sora

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	commonjson "github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
)

const testReferenceVideoRangeProbeBytes = 1024 * 1024

func TestEstimateBillingDefaultsToDoublePriceWhenEnvConfiguredModelHasReferenceVideo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("SORA_REFERENCE_VIDEO_DOUBLE_PRICE_MODELS", "seedance-2.0")
	t.Setenv("SORA_REFERENCE_VIDEO_DURATION_BILLING_ENABLED", "")
	ReloadReferenceVideoDoublePriceModelsFromEnv()
	t.Cleanup(ReloadReferenceVideoDoublePriceModelsFromEnv)

	ratios := estimateBillingForBody(t, referenceVideoRequestBody("seedance-2.0"))
	if ratios["seconds"] != 10 {
		t.Fatalf("expected generated seconds to stay unchanged, got %#v", ratios)
	}
	if ratios["video_input"] != 2 {
		t.Fatalf("expected default video_input double-price ratio, got %#v", ratios)
	}
}

func TestSeedanceAssetSkipsVideoBillingRatios(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ratios := estimateBillingForBody(t, `{
		"model": "seedance-asset",
		"prompt": "林春芽",
		"seconds": "10",
		"size": "1792x1024",
		"files": ["https://asset.test/person.jpg"]
	}`)
	if len(ratios) != 0 {
		t.Fatalf("expected no ratios for seedance asset, got %#v", ratios)
	}
}

func TestSeedanceAssetRejectsUnsafeURLs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, imageURL := range []string{
		"http://127.0.0.1/person.jpg",
		"http://[fe80::1%25eth0]/person.jpg",
	} {
		taskErr := validateBody(t, fmt.Sprintf(`{
			"model": "seedance-asset",
			"prompt": "林春芽",
			"input_reference": %q
		}`, imageURL))
		if taskErr == nil {
			t.Fatalf("expected unsafe URL %q to be rejected", imageURL)
		}
		if taskErr.Code != "invalid_request" {
			t.Fatalf("code=%s message=%s", taskErr.Code, taskErr.Message)
		}
	}
}

func TestSeedanceAssetAcceptsFilesURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	taskErr := validateBody(t, `{
		"model": "seedance-asset",
		"prompt": "林春芽",
		"files": ["https://asset.test/person.jpg"]
	}`)
	if taskErr != nil {
		t.Fatalf("ValidateRequestAndSetAction returned error: code=%s message=%s", taskErr.Code, taskErr.Message)
	}
}

func TestDoResponsePreservesAssetFieldsWhenReplacingPublicTaskID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	info := &common.RelayInfo{TaskRelayInfo: &common.TaskRelayInfo{PublicTaskID: "task_public"}}
	adaptor := &TaskAdaptor{}
	resp := &http.Response{
		Body: io.NopCloser(strings.NewReader(`{
			"id":"asset_req_1780830000_abcdef123456",
			"task_id":"asset_req_1780830000_abcdef123456",
			"object":"video",
			"model":"seedance-asset",
			"status":"queued",
			"progress":0,
			"asset_id":"asset-123",
			"metadata":{"seedance":{"asset_id":"asset-123","kind":"asset"}}
		}`)),
	}

	upstreamID, taskData, taskErr := adaptor.DoResponse(c, resp, info)
	if taskErr != nil {
		t.Fatalf("DoResponse taskErr = %v", taskErr)
	}
	if upstreamID != "asset_req_1780830000_abcdef123456" {
		t.Fatalf("upstreamID = %q", upstreamID)
	}
	if !strings.Contains(string(taskData), `"asset_id":"asset-123"`) {
		t.Fatalf("taskData lost asset_id: %s", string(taskData))
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("response JSON error = %v body=%s", err, rec.Body.String())
	}
	if out["id"] != "task_public" || out["task_id"] != "task_public" || out["asset_id"] != "asset-123" {
		t.Fatalf("unexpected response: %#v", out)
	}
}

func TestConvertToOpenAIVideoReplacesAssetTaskIDs(t *testing.T) {
	adaptor := &TaskAdaptor{}
	task := &model.Task{
		TaskID: "task_public",
		Data: []byte(`{
			"id":"asset_req_1780830000_abcdef123456",
			"task_id":"asset_req_1780830000_abcdef123456",
			"object":"video",
			"model":"seedance-asset",
			"status":"completed",
			"progress":100,
			"asset_id":"asset-123",
			"metadata":{"seedance":{"asset_id":"asset-123","kind":"asset"}}
		}`),
	}

	body, err := adaptor.ConvertToOpenAIVideo(task)
	if err != nil {
		t.Fatalf("ConvertToOpenAIVideo error = %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("response JSON error = %v body=%s", err, string(body))
	}
	if out["id"] != "task_public" || out["task_id"] != "task_public" || out["asset_id"] != "asset-123" {
		t.Fatalf("unexpected response: %#v", out)
	}
}

func TestEstimateBillingAddsAllReferenceVideoDurationsWhenDurationBillingEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("SORA_REFERENCE_VIDEO_DOUBLE_PRICE_MODELS", "seedance-2.0")
	t.Setenv("SORA_REFERENCE_VIDEO_DURATION_BILLING_ENABLED", "true")
	ReloadReferenceVideoDoublePriceModelsFromEnv()
	t.Cleanup(ReloadReferenceVideoDoublePriceModelsFromEnv)
	allowLocalVideoServers(t)

	firstVideoServer := newRangeVideoServer(t, minimalMP4(t, 6))
	defer firstVideoServer.Close()
	secondVideoServer := newRangeVideoServer(t, minimalMP4(t, 3))
	defer secondVideoServer.Close()

	ratios := estimateBillingForBody(t, referenceVideoRequestBody("seedance-2.0", firstVideoServer.URL, secondVideoServer.URL))
	if ratios["seconds"] != 19 {
		t.Fatalf("expected seconds ratio to include 9s total reference videos, got %#v", ratios)
	}
	if _, ok := ratios["video_input"]; ok {
		t.Fatalf("expected no video_input double-price ratio, got %#v", ratios)
	}
}

func TestEstimateBillingFallsBackToFullDownloadWhenRangeProbeCannotReadDuration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("SORA_REFERENCE_VIDEO_DOUBLE_PRICE_MODELS", "seedance-2.0")
	t.Setenv("SORA_REFERENCE_VIDEO_DURATION_BILLING_ENABLED", "true")
	ReloadReferenceVideoDoublePriceModelsFromEnv()
	t.Cleanup(ReloadReferenceVideoDoublePriceModelsFromEnv)
	allowLocalVideoServers(t)

	var rangeRequests int
	var fullRequests int
	videoBytes := appendMinimalMP4AfterPayload(t, 4, bytes.Repeat([]byte{0}, testReferenceVideoRangeProbeBytes+64))
	videoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Range") != "" {
			rangeRequests++
			w.Header().Set("Content-Range", fmt.Sprintf("bytes 0-%d/%d", testReferenceVideoRangeProbeBytes-1, len(videoBytes)))
			w.WriteHeader(http.StatusPartialContent)
			_, _ = w.Write(videoBytes[:testReferenceVideoRangeProbeBytes])
			return
		}
		fullRequests++
		_, _ = w.Write(videoBytes)
	}))
	defer videoServer.Close()

	ratios := estimateBillingForBody(t, referenceVideoRequestBody("seedance-2.0", videoServer.URL))
	if ratios["seconds"] != 14 {
		t.Fatalf("expected seconds ratio to include 4s reference video, got %#v", ratios)
	}
	if rangeRequests == 0 || fullRequests == 0 {
		t.Fatalf("expected range probe then full fallback, got range=%d full=%d", rangeRequests, fullRequests)
	}
}

func TestValidateRejectsWhenReferenceVideoDurationCannotBeDetected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("SORA_REFERENCE_VIDEO_DOUBLE_PRICE_MODELS", "seedance-2.0")
	t.Setenv("SORA_REFERENCE_VIDEO_DURATION_BILLING_ENABLED", "true")
	ReloadReferenceVideoDoublePriceModelsFromEnv()
	t.Cleanup(ReloadReferenceVideoDoublePriceModelsFromEnv)
	allowLocalVideoServers(t)

	videoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not an mp4"))
	}))
	defer videoServer.Close()

	taskErr := validateBody(t, referenceVideoRequestBody("seedance-2.0", videoServer.URL))
	if taskErr == nil {
		t.Fatal("expected validation to reject unparseable reference video")
	}
	if taskErr.Code != "reference_video_duration_unavailable" {
		t.Fatalf("expected reference video duration error, got code=%s message=%s", taskErr.Code, taskErr.Message)
	}
}

func TestValidateRejectsWhenReferenceVideoDurationTimesOut(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("SORA_REFERENCE_VIDEO_DOUBLE_PRICE_MODELS", "seedance-2.0")
	t.Setenv("SORA_REFERENCE_VIDEO_DURATION_BILLING_ENABLED", "true")
	ReloadReferenceVideoDoublePriceModelsFromEnv()
	t.Cleanup(ReloadReferenceVideoDoublePriceModelsFromEnv)
	allowLocalVideoServers(t)
	setReferenceVideoDurationProbeTimeoutForTest(t, 20*time.Millisecond)

	videoBytes := minimalMP4(t, 1)
	videoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			return
		case <-time.After(200 * time.Millisecond):
			_, _ = w.Write(videoBytes)
		}
	}))
	defer videoServer.Close()

	taskErr := validateBody(t, referenceVideoRequestBody("seedance-2.0", videoServer.URL))
	if taskErr == nil {
		t.Fatal("expected validation to reject timed out reference video")
	}
	if taskErr.Code != "reference_video_duration_unavailable" {
		t.Fatalf("expected reference video duration error, got code=%s message=%s", taskErr.Code, taskErr.Message)
	}
	if !strings.Contains(taskErr.Message, "timed out") {
		t.Fatalf("expected timeout message, got %s", taskErr.Message)
	}
}

func TestEstimateBillingDoesNotProbeReferenceVideoWhenEnvDoesNotListModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("SORA_REFERENCE_VIDEO_DOUBLE_PRICE_MODELS", "")
	t.Setenv("SORA_REFERENCE_VIDEO_DURATION_BILLING_ENABLED", "true")
	ReloadReferenceVideoDoublePriceModelsFromEnv()
	t.Cleanup(ReloadReferenceVideoDoublePriceModelsFromEnv)

	ratios := estimateBillingForBody(t, referenceVideoRequestBody("seedance-2.0"))
	if _, ok := ratios["video_input"]; ok {
		t.Fatalf("expected no video_input ratio without env whitelist, got %#v", ratios)
	}
}

func TestEstimateBillingDoublePriceForSeedanceOpenAIVideosReferenceVideo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("SORA_REFERENCE_VIDEO_DOUBLE_PRICE_MODELS", "doubao-seedance-2-0-260128-2")
	t.Setenv("SORA_REFERENCE_VIDEO_DURATION_BILLING_ENABLED", "")
	ReloadReferenceVideoDoublePriceModelsFromEnv()
	t.Cleanup(ReloadReferenceVideoDoublePriceModelsFromEnv)

	for _, body := range []string{
		`{"model":"doubao-seedance-2-0-260128-2","prompt":"p","input_video":["https://cdn.example.com/ref.mp4"]}`,
		`{"model":"doubao-seedance-2-0-260128-2","prompt":"p","video_url":"https://cdn.example.com/ref.mov"}`,
		`{"model":"doubao-seedance-2-0-260128-2","prompt":"p","reference_video":"https://cdn.example.com/ref.webm"}`,
		`{"model":"doubao-seedance-2-0-260128-2","prompt":"p","files":["https://cdn.example.com/ref-image.jpg","https://cdn.example.com/ref-video.mp4"]}`,
	} {
		ratios := estimateBillingForBody(t, body)
		if ratios["video_input"] != 2 {
			t.Fatalf("expected seedance top-level reference video double-price for body %s, got %#v", body, ratios)
		}
	}
}

func TestEstimateBillingDoesNotDoublePriceSeedanceNonVideoFiles(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("SORA_REFERENCE_VIDEO_DOUBLE_PRICE_MODELS", "doubao-seedance-2-0-260128-2")
	t.Setenv("SORA_REFERENCE_VIDEO_DURATION_BILLING_ENABLED", "")
	ReloadReferenceVideoDoublePriceModelsFromEnv()
	t.Cleanup(ReloadReferenceVideoDoublePriceModelsFromEnv)

	body := `{
		"model": "doubao-seedance-2-0-260128-2",
		"prompt": "p",
		"files": ["https://cdn.example.com/ref.jpg", "https://cdn.example.com/ref.mp3"]
	}`
	ratios := estimateBillingForBody(t, body)
	if _, ok := ratios["video_input"]; ok {
		t.Fatalf("expected no video_input ratio for image/audio files, got %#v", ratios)
	}
}

func TestEstimateBillingDoesNotDoublePriceSeedanceWhenModelNotWhitelisted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("SORA_REFERENCE_VIDEO_DOUBLE_PRICE_MODELS", "")
	t.Setenv("SORA_REFERENCE_VIDEO_DURATION_BILLING_ENABLED", "")
	ReloadReferenceVideoDoublePriceModelsFromEnv()
	t.Cleanup(ReloadReferenceVideoDoublePriceModelsFromEnv)

	body := `{
		"model": "doubao-seedance-2-0-260128-2",
		"prompt": "p",
		"input_video": ["https://cdn.example.com/ref.mp4"]
	}`
	ratios := estimateBillingForBody(t, body)
	if _, ok := ratios["video_input"]; ok {
		t.Fatalf("expected no video_input ratio without seedance env whitelist, got %#v", ratios)
	}
}

func TestReferenceVideoDoublePriceModelsRequireExplicitReload(t *testing.T) {
	t.Setenv("SORA_REFERENCE_VIDEO_DOUBLE_PRICE_MODELS", "")
	t.Setenv("SORA_REFERENCE_VIDEO_DURATION_BILLING_ENABLED", "")
	ReloadReferenceVideoDoublePriceModelsFromEnv()
	t.Setenv("SORA_REFERENCE_VIDEO_DOUBLE_PRICE_MODELS", "seedance-2.0")
	t.Setenv("SORA_REFERENCE_VIDEO_DURATION_BILLING_ENABLED", "true")
	t.Cleanup(ReloadReferenceVideoDoublePriceModelsFromEnv)

	if IsReferenceVideoDoublePriceModel("seedance-2.0") {
		t.Fatal("expected env change not to load until explicit reload")
	}
	if ReferenceVideoDurationBillingEnabled() {
		t.Fatal("expected env change not to load duration billing until explicit reload")
	}

	ReloadReferenceVideoDoublePriceModelsFromEnv()
	if !IsReferenceVideoDoublePriceModel("seedance-2.0") {
		t.Fatal("expected explicit reload to load env whitelist")
	}
	if !ReferenceVideoDurationBillingEnabled() {
		t.Fatal("expected explicit reload to load duration billing toggle")
	}
}

func TestReferenceVideoDoublePriceModelsIncludeSeedanceModelsInSoraEnv(t *testing.T) {
	t.Setenv("SORA_REFERENCE_VIDEO_DOUBLE_PRICE_MODELS", "doubao-seedance-2-0-260128-2")
	t.Setenv("SORA_REFERENCE_VIDEO_DURATION_BILLING_ENABLED", "")
	ReloadReferenceVideoDoublePriceModelsFromEnv()
	t.Cleanup(ReloadReferenceVideoDoublePriceModelsFromEnv)

	if !IsReferenceVideoDoublePriceModel("doubao-seedance-2-0-260128-2") {
		t.Fatal("expected seedance model in sora env whitelist to load into task billing models")
	}
}

func TestBuildRequestBodyKeepsSeedanceOpenAIVideosFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := `{
		"model": "doubao-seedance-2-0-260128-2",
		"prompt": "p",
		"duration": "4",
		"aspect_ratio": "16:9",
		"files": ["https://cdn.example.com/ref-image.jpg", "https://cdn.example.com/ref-video.mp4"],
		"input_video": ["https://cdn.example.com/ref-video-2.mp4"],
		"audio": ["https://cdn.example.com/ref-audio.mp3"],
		"generate_audio": true,
		"resolution": "480p"
	}`
	c := newTaskContext(t, body)
	info := &common.RelayInfo{
		TaskRelayInfo: &common.TaskRelayInfo{},
		ChannelMeta: &common.ChannelMeta{
			UpstreamModelName: "mapped-seedance-model",
		},
	}
	adaptor := &TaskAdaptor{}
	if taskErr := adaptor.ValidateRequestAndSetAction(c, info); taskErr != nil {
		t.Fatalf("ValidateRequestAndSetAction returned error: code=%s message=%s", taskErr.Code, taskErr.Message)
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
	if err := commonjson.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got["model"] != "mapped-seedance-model" {
		t.Fatalf("expected mapped model, got %v", got["model"])
	}
	for _, field := range []string{"files", "input_video", "audio", "aspect_ratio", "generate_audio", "resolution"} {
		if _, ok := got[field]; !ok {
			t.Fatalf("expected sora seedance body to keep %s, got %#v", field, got)
		}
	}
}

func estimateBillingForBody(t *testing.T, body string) map[string]float64 {
	t.Helper()
	c := newTaskContext(t, body)
	info := &common.RelayInfo{TaskRelayInfo: &common.TaskRelayInfo{}}
	adaptor := &TaskAdaptor{}
	if taskErr := adaptor.ValidateRequestAndSetAction(c, info); taskErr != nil {
		t.Fatalf("ValidateRequestAndSetAction returned error: code=%s message=%s", taskErr.Code, taskErr.Message)
	}

	return adaptor.EstimateBilling(c, info)
}

func validateBody(t *testing.T, body string) *dto.TaskError {
	t.Helper()
	c := newTaskContext(t, body)
	info := &common.RelayInfo{TaskRelayInfo: &common.TaskRelayInfo{}}
	adaptor := &TaskAdaptor{}
	return adaptor.ValidateRequestAndSetAction(c, info)
}

func newTaskContext(t *testing.T, body string) *gin.Context {
	t.Helper()
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest("POST", "/v1/videos", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	return c
}

func allowLocalVideoServers(t *testing.T) {
	t.Helper()
	fetchSetting := system_setting.GetFetchSetting()
	original := *fetchSetting
	fetchSetting.EnableSSRFProtection = false
	t.Cleanup(func() {
		*fetchSetting = original
	})
}

func setReferenceVideoDurationProbeTimeoutForTest(t *testing.T, timeout time.Duration) {
	t.Helper()
	original := referenceVideoDurationProbeTimeout
	referenceVideoDurationProbeTimeout = timeout
	t.Cleanup(func() {
		referenceVideoDurationProbeTimeout = original
	})
}

func referenceVideoRequestBody(modelName string, videoURLs ...string) string {
	if len(videoURLs) == 0 {
		videoURLs = []string{"https://example.com/reference.mp4"}
	}
	var items strings.Builder
	for _, videoURL := range videoURLs {
		items.WriteString(`,
			{
				"type": "video_url",
				"role": "reference_video",
				"video_url": {"url": "` + videoURL + `"}
			}`)
	}
	return `{
		"model": "` + modelName + `",
		"duration": 10,
		"generate_audio": true,
		"ratio": "16:9",
		"prompt": "placeholder",
		"content": [
			{"type": "text", "text": "keep identity"}` + items.String() + `
		]
	}`
}

func newRangeVideoServer(t *testing.T, videoBytes []byte) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Range") != "" {
			w.Header().Set("Content-Range", fmt.Sprintf("bytes 0-%d/%d", len(videoBytes)-1, len(videoBytes)))
			w.WriteHeader(http.StatusPartialContent)
		}
		_, _ = w.Write(videoBytes)
	}))
}

func minimalMP4(t *testing.T, durationSeconds uint32) []byte {
	t.Helper()
	var out []byte
	out = append(out, mp4Box("ftyp", []byte("isom\x00\x00\x00\x00isom"))...)
	out = append(out, mp4Box("moov", minimalMvhd(t, 1000, durationSeconds*1000))...)
	return out
}

func appendMinimalMP4AfterPayload(t *testing.T, durationSeconds uint32, payload []byte) []byte {
	t.Helper()
	var out []byte
	out = append(out, mp4Box("ftyp", []byte("isom\x00\x00\x00\x00isom"))...)
	out = append(out, mp4Box("mdat", payload)...)
	out = append(out, mp4Box("moov", minimalMvhd(t, 1000, durationSeconds*1000))...)
	return out
}

func minimalMvhd(t *testing.T, timescale uint32, duration uint32) []byte {
	t.Helper()
	buf := &bytes.Buffer{}
	writeU32(t, buf, 0)
	writeU32(t, buf, 0)
	writeU32(t, buf, 0)
	writeU32(t, buf, timescale)
	writeU32(t, buf, duration)
	writeU32(t, buf, 0x00010000)
	writeU16(t, buf, 0x0100)
	writeU16(t, buf, 0)
	writeU32(t, buf, 0)
	writeU32(t, buf, 0)
	for _, v := range []uint32{0x00010000, 0, 0, 0, 0x00010000, 0, 0, 0, 0x40000000} {
		writeU32(t, buf, v)
	}
	for i := 0; i < 6; i++ {
		writeU32(t, buf, 0)
	}
	writeU32(t, buf, 2)
	return mp4Box("mvhd", buf.Bytes())
}

func mp4Box(name string, payload []byte) []byte {
	buf := &bytes.Buffer{}
	_ = binary.Write(buf, binary.BigEndian, uint32(len(payload)+8))
	_, _ = io.WriteString(buf, name)
	_, _ = buf.Write(payload)
	return buf.Bytes()
}

func writeU32(t *testing.T, buf *bytes.Buffer, v uint32) {
	t.Helper()
	if err := binary.Write(buf, binary.BigEndian, v); err != nil {
		t.Fatal(err)
	}
}

func writeU16(t *testing.T, buf *bytes.Buffer, v uint16) {
	t.Helper()
	if err := binary.Write(buf, binary.BigEndian, v); err != nil {
		t.Fatal(err)
	}
}
