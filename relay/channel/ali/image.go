package ali

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

func oaiImage2AliImageRequest(info *relaycommon.RelayInfo, request dto.ImageRequest, isSync bool) (*AliImageRequest, error) {
	var imageRequest AliImageRequest
	imageRequest.Model = request.Model
	imageRequest.ResponseFormat = request.ResponseFormat
	if request.Extra != nil {
		if val, ok := request.Extra["parameters"]; ok {
			err := common.Unmarshal(val, &imageRequest.Parameters)
			if err != nil {
				return nil, fmt.Errorf("invalid parameters field: %w", err)
			}
		} else {
			// 兼容没有parameters字段的情况，从openai标准字段中提取参数
			imageRequest.Parameters = AliImageParameters{
				Size:      strings.Replace(request.Size, "x", "*", -1),
				N:         int(lo.FromPtrOr(request.N, uint(1))),
				Watermark: request.Watermark,
			}
		}
		if val, ok := request.Extra["input"]; ok {
			err := common.Unmarshal(val, &imageRequest.Input)
			if err != nil {
				return nil, fmt.Errorf("invalid input field: %w", err)
			}
		}
	}

	if strings.Contains(request.Model, "z-image") {
		// z-image 开启prompt_extend后，按2倍计费
		if imageRequest.Parameters.PromptExtendValue() {
			info.PriceData.AddOtherRatio("prompt_extend", 2)
		}
	}

	// Parameters may come from Extra["parameters"], bypassing the standard
	// top-level n validation; enforce the same bound before it becomes a
	// billing multiplier.
	if imageRequest.Parameters.N < 0 || imageRequest.Parameters.N > dto.MaxImageN {
		return nil, fmt.Errorf("parameters.n must be an integer between 1 and %d", dto.MaxImageN)
	}
	if imageRequest.Parameters.N != 0 {
		info.PriceData.AddOtherRatio("n", float64(imageRequest.Parameters.N))
	}

	// 同步图片模型和异步图片模型请求格式不一样
	if isSync {
		if imageRequest.Input == nil {
			imageRequest.Input = AliImageInput{
				Messages: []AliMessage{
					{
						Role: "user",
						Content: []AliMediaContent{
							{
								Text: request.Prompt,
							},
						},
					},
				},
			}
		}
	} else {
		if imageRequest.Input == nil {
			imageRequest.Input = AliImageInput{
				Prompt: request.Prompt,
			}
		}
	}

	return &imageRequest, nil
}
func getImageBase64sFromForm(c *gin.Context, fieldName string) ([]string, error) {
	mf := c.Request.MultipartForm
	if mf == nil {
		if _, err := c.MultipartForm(); err != nil {
			return nil, fmt.Errorf("failed to parse image edit form request: %w", err)
		}
		mf = c.Request.MultipartForm
	}

	var imageFiles []*multipart.FileHeader
	var exists bool

	// First check for standard "image" field
	if imageFiles, exists = mf.File["image"]; !exists || len(imageFiles) == 0 {
		// If not found, check for "image[]" field
		if imageFiles, exists = mf.File["image[]"]; !exists || len(imageFiles) == 0 {
			// If still not found, iterate through all fields to find any that start with "image["
			foundArrayImages := false
			for fieldName, files := range mf.File {
				if strings.HasPrefix(fieldName, "image[") && len(files) > 0 {
					foundArrayImages = true
					imageFiles = append(imageFiles, files...)
				}
			}

			// If no image fields found at all
			if !foundArrayImages && (len(imageFiles) == 0) {
				return nil, errors.New("image is required")
			}
		}
	}

	if len(imageFiles) == 0 {
		return nil, errors.New("image is required")
	}

	//if len(imageFiles) > 1 {
	//	return nil, errors.New("only one image is supported for qwen edit")
	//}

	// 获取base64编码的图片
	var imageBase64s []string
	for _, file := range imageFiles {
		image, err := file.Open()
		if err != nil {
			return nil, errors.New("failed to open image file")
		}

		// 读取文件内容
		imageData, err := io.ReadAll(image)
		if err != nil {
			return nil, errors.New("failed to read image file")
		}

		// 获取MIME类型
		mimeType := http.DetectContentType(imageData)

		// 编码为base64
		base64Data := base64.StdEncoding.EncodeToString(imageData)

		// 构造data URL格式
		dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, base64Data)
		imageBase64s = append(imageBase64s, dataURL)
		image.Close()
	}
	return imageBase64s, nil
}

func oaiFormEdit2AliImageEdit(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (*AliImageRequest, error) {
	var imageRequest AliImageRequest
	imageRequest.Model = request.Model
	imageRequest.ResponseFormat = request.ResponseFormat

	imageBase64s, err := getImageBase64sFromForm(c, "image")
	if err != nil {
		return nil, fmt.Errorf("get image base64s from form failed: %w", err)
	}
	//dto.MediaContent{}
	mediaContents := make([]AliMediaContent, len(imageBase64s))
	for i, b64 := range imageBase64s {
		mediaContents[i] = AliMediaContent{
			Image: b64,
		}
	}
	mediaContents = append(mediaContents, AliMediaContent{
		Text: request.Prompt,
	})
	imageRequest.Input = AliImageInput{
		Messages: []AliMessage{
			{
				Role:    "user",
				Content: mediaContents,
			},
		},
	}
	imageRequest.Parameters = AliImageParameters{
		N:         int(lo.FromPtrOr(request.N, uint(1))),
		Watermark: request.Watermark,
	}
	return &imageRequest, nil
}

func fetchImageTaskStatus(client *http.Client, req *http.Request) (*AliResponse, []byte, error) {
	resp, err := client.Do(req)
	if err != nil {
		var requestErr *url.Error
		if errors.As(err, &requestErr) {
			err = requestErr.Err
		}
		return nil, nil, fmt.Errorf("send image task status request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("read image task status response: %w", err)
	}
	var task AliResponse
	if err := common.Unmarshal(body, &task); err != nil {
		return nil, nil, fmt.Errorf("decode image task status response: %w", err)
	}
	return &task, body, nil
}

func asyncTaskWait(c *gin.Context, info *relaycommon.RelayInfo, taskID string) (*AliResponse, []byte, error) {
	const maxAttempts = 20
	const pollInterval = 10 * time.Second
	client, err := service.GetHttpClientWithProxy(info.ChannelSetting.Proxy)
	if err != nil {
		return nil, nil, errors.New("invalid Ali image task proxy configuration")
	}
	// 上游已接单，客户端断开后仍需取得终态以完成结算。
	req, err := http.NewRequest(http.MethodGet, info.ChannelBaseUrl+"/api/v1/tasks/"+taskID, nil)
	if err != nil {
		return nil, nil, errors.New("invalid Ali image task status URL")
	}
	req.Header.Set("Authorization", "Bearer "+info.ApiKey)

	delay := 5 * time.Second
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		time.Sleep(delay)
		delay = pollInterval
		logger.LogDebug(c, "Ali image task poll %d/%d", attempt, maxAttempts)
		task, body, err := fetchImageTaskStatus(client, req)
		if err != nil {
			logger.LogWarn(c, fmt.Sprintf("Ali image task poll %d/%d: %s", attempt, maxAttempts, common.MaskSensitiveInfo(err.Error())))
			if attempt == maxAttempts {
				if errors.Is(err, context.DeadlineExceeded) {
					return nil, nil, fmt.Errorf("Ali image task polling timed out after %d attempts: %w", attempt, context.DeadlineExceeded)
				}
				return nil, nil, fmt.Errorf("Ali image task status unavailable after %d attempts", attempt)
			}
			continue
		}
		switch task.Output.TaskStatus {
		case "":
			return &AliResponse{}, body, nil
		case "FAILED", "CANCELED", "SUCCEEDED", "UNKNOWN":
			return task, body, nil
		}
	}
	return nil, nil, errors.New("aliAsyncTaskWait timeout")
}

func responseAli2OpenAIImage(c *gin.Context, response *AliResponse, originBody []byte, info *relaycommon.RelayInfo, responseFormat string) *dto.ImageResponse {
	imageResponse := dto.ImageResponse{
		Created: info.StartTime.Unix(),
	}

	if len(response.Output.Results) > 0 {
		imageResponse.Data = response.Output.ResultToOpenAIImageDate(c, responseFormat)
	} else if len(response.Output.Choices) > 0 {
		imageResponse.Data = response.Output.ChoicesToOpenAIImageDate(c, responseFormat)
	}

	imageResponse.Metadata = originBody
	return &imageResponse
}

func aliImageHandler(a *Adaptor, c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (*types.NewAPIError, *dto.Usage) {
	responseFormat := c.GetString("response_format")

	var aliTaskResponse AliResponse
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError), nil
	}
	service.CloseResponseBodyGracefully(resp)
	err = common.Unmarshal(responseBody, &aliTaskResponse)
	if err != nil {
		return types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError), nil
	}

	if aliTaskResponse.Message != "" {
		logger.LogError(c, "ali_async_task_failed: "+aliTaskResponse.Message)
		return types.NewError(errors.New(aliTaskResponse.Message), types.ErrorCodeBadResponse), nil
	}

	var (
		aliResponse    *AliResponse
		originRespBody []byte
	)

	if a.IsSyncImageModel {
		aliResponse = &aliTaskResponse
		originRespBody = responseBody
	} else {
		// 异步图片模型需要轮询任务结果
		taskID := aliTaskResponse.Output.TaskId
		if taskID == "" {
			return types.NewError(errors.New("missing task ID in Ali image response"), types.ErrorCodeBadResponse), nil
		}
		aliResponse, originRespBody, err = asyncTaskWait(c, info, taskID)
		if err != nil {
			return types.NewError(err, types.ErrorCodeBadResponse, types.ErrOptionWithSkipRetry()), nil
		}
		if aliResponse.Output.TaskStatus != "SUCCEEDED" {
			return types.WithOpenAIError(types.OpenAIError{
				Message: aliResponse.Output.Message,
				Type:    "ali_error",
				Param:   "",
				Code:    aliResponse.Output.Code,
			}, resp.StatusCode, types.ErrOptionWithSkipRetry()), nil
		}
	}

	if a.IsSyncImageModel {
		logger.LogDebug(c, "ali_sync_image_result: %s", originRespBody)
	} else {
		logger.LogDebug(c, "ali_async_image_result: %s", originRespBody)
	}

	imageResponses := responseAli2OpenAIImage(c, aliResponse, originRespBody, info, responseFormat)
	if aliResponse.Usage.ImageCount != 0 {
		info.PriceData.AddOtherRatio("n", float64(aliResponse.Usage.ImageCount))
	} else if len(imageResponses.Data) != 0 {
		info.PriceData.AddOtherRatio("n", float64(len(imageResponses.Data)))
	}
	jsonResponse, err := common.Marshal(imageResponses)
	if err != nil {
		return types.NewError(err, types.ErrorCodeBadResponseBody), nil
	}
	service.IOCopyBytesGracefully(c, resp, jsonResponse)

	return nil, &dto.Usage{}
}
