package sora

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/abema/go-mp4"
	"github.com/gin-gonic/gin"
)

const (
	referenceVideoTotalSecondsKey        = "sora_reference_video_total_seconds"
	defaultReferenceVideoRangeProbeBytes = 1024 * 1024
)

var referenceVideoDurationProbeTimeout = 30 * time.Second

func setReferenceVideoTotalSeconds(c *gin.Context, seconds float64) {
	c.Set(referenceVideoTotalSecondsKey, seconds)
}

func getReferenceVideoTotalSeconds(c *gin.Context) (float64, bool) {
	v, ok := c.Get(referenceVideoTotalSecondsKey)
	if !ok {
		return 0, false
	}
	seconds, ok := v.(float64)
	return seconds, ok
}

func extractReferenceVideoURLs(content []map[string]any) ([]string, error) {
	urls := make([]string, 0)
	for _, item := range content {
		if item == nil {
			continue
		}
		_, hasVideoURL := item["video_url"]
		if item["type"] != "video_url" && !hasVideoURL {
			continue
		}
		url, err := referenceVideoURLFromItem(item)
		if err != nil {
			return nil, err
		}
		urls = append(urls, url)
	}
	return urls, nil
}

func referenceVideoURLFromItem(item map[string]any) (string, error) {
	raw, ok := item["video_url"]
	if !ok {
		return "", fmt.Errorf("reference video item missing video_url")
	}
	switch v := raw.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return "", fmt.Errorf("reference video url is empty")
		}
		return strings.TrimSpace(v), nil
	case map[string]any:
		url, _ := v["url"].(string)
		if strings.TrimSpace(url) == "" {
			return "", fmt.Errorf("reference video url is empty")
		}
		return strings.TrimSpace(url), nil
	default:
		return "", fmt.Errorf("reference video url has invalid format")
	}
}

func sumReferenceVideoDurationsWithTimeout(ctx context.Context, urls []string) (float64, error) {
	if referenceVideoDurationProbeTimeout <= 0 {
		return sumReferenceVideoDurations(ctx, urls)
	}
	probeCtx, cancel := context.WithTimeout(ctx, referenceVideoDurationProbeTimeout)
	defer cancel()

	totalSeconds, err := sumReferenceVideoDurations(probeCtx, urls)
	if err != nil && probeCtx.Err() == context.DeadlineExceeded {
		return 0, fmt.Errorf("reference video duration detection timed out after %s", referenceVideoDurationProbeTimeout)
	}
	return totalSeconds, err
}

func sumReferenceVideoDurations(ctx context.Context, urls []string) (float64, error) {
	var totalSeconds float64
	for _, videoURL := range urls {
		seconds, err := detectReferenceVideoDuration(ctx, videoURL)
		if err != nil {
			return 0, fmt.Errorf("detect reference video duration failed: %w", err)
		}
		if seconds <= 0 {
			return 0, fmt.Errorf("reference video duration must be positive")
		}
		totalSeconds += seconds
	}
	return totalSeconds, nil
}

func detectReferenceVideoDuration(ctx context.Context, videoURL string) (float64, error) {
	seconds, rangeErr := detectReferenceVideoDurationByRange(ctx, videoURL)
	if rangeErr == nil {
		return seconds, nil
	}
	seconds, fullErr := detectReferenceVideoDurationByFullDownload(ctx, videoURL)
	if fullErr == nil {
		return seconds, nil
	}
	return 0, fmt.Errorf("range probe failed: %v; full download failed: %w", rangeErr, fullErr)
}

func detectReferenceVideoDurationByRange(ctx context.Context, videoURL string) (float64, error) {
	resp, err := doReferenceVideoRequest(ctx, videoURL, fmt.Sprintf("bytes=0-%d", defaultReferenceVideoRangeProbeBytes-1))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return 0, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}
	data, err := readBodyLimit(resp.Body, defaultReferenceVideoRangeProbeBytes, false)
	if err != nil {
		return 0, err
	}
	return parseMP4DurationSeconds(data)
}

func detectReferenceVideoDurationByFullDownload(ctx context.Context, videoURL string) (float64, error) {
	resp, err := doReferenceVideoRequest(ctx, videoURL, "")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return 0, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}
	data, err := readBodyLimit(resp.Body, maxReferenceVideoDownloadBytes(), true)
	if err != nil {
		return 0, err
	}
	return parseMP4DurationSeconds(data)
}

func doReferenceVideoRequest(ctx context.Context, videoURL string, rangeHeader string) (*http.Response, error) {
	if system_setting.EnableWorker() {
		headers := map[string]string{}
		if rangeHeader != "" {
			headers["Range"] = rangeHeader
		}
		return service.DoWorkerRequestWithContext(ctx, &service.WorkerRequest{
			URL:     videoURL,
			Key:     system_setting.WorkerValidKey,
			Headers: headers,
		})
	}

	fetchSetting := system_setting.GetFetchSetting()
	if err := common.ValidateURLWithFetchSetting(videoURL, fetchSetting.EnableSSRFProtection, fetchSetting.AllowPrivateIp, fetchSetting.DomainFilterMode, fetchSetting.IpFilterMode, fetchSetting.DomainList, fetchSetting.IpList, fetchSetting.AllowedPorts, fetchSetting.ApplyIPFilterForDomain); err != nil {
		return nil, fmt.Errorf("request reject: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, videoURL, nil)
	if err != nil {
		return nil, err
	}
	if rangeHeader != "" {
		req.Header.Set("Range", rangeHeader)
	}
	client := service.GetHttpClient()
	if client == nil {
		client = http.DefaultClient
	}
	return client.Do(req)
}

func readBodyLimit(r io.Reader, maxBytes int64, failOnOverflow bool) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		if failOnOverflow {
			return nil, fmt.Errorf("reference video exceeds %d bytes", maxBytes)
		}
		data = data[:maxBytes]
	}
	return data, nil
}

func maxReferenceVideoDownloadBytes() int64 {
	maxMB := constant.MaxFileDownloadMB
	if maxMB <= 0 {
		maxMB = 64
	}
	return int64(maxMB) << 20
}

func parseMP4DurationSeconds(data []byte) (float64, error) {
	info, err := mp4.Probe(bytes.NewReader(data))
	if err != nil {
		return 0, err
	}
	if info.Timescale > 0 && info.Duration > 0 {
		return float64(info.Duration) / float64(info.Timescale), nil
	}
	var maxSeconds float64
	for _, track := range info.Tracks {
		if track.Timescale == 0 || track.Duration == 0 {
			continue
		}
		seconds := float64(track.Duration) / float64(track.Timescale)
		if seconds > maxSeconds {
			maxSeconds = seconds
		}
	}
	if maxSeconds > 0 {
		return maxSeconds, nil
	}
	return 0, fmt.Errorf("mp4 duration not found")
}
