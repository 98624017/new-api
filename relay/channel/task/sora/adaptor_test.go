package sora

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/dto"
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
