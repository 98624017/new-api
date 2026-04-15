package common

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestValidateBasicTaskRequest_MultipartWithMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("prompt", "draw a cat"); err != nil {
		t.Fatalf("WriteField prompt failed: %v", err)
	}
	if err := writer.WriteField("model", "kling-v1"); err != nil {
		t.Fatalf("WriteField model failed: %v", err)
	}
	if err := writer.WriteField("metadata", `{"seed":123,"style":"anime"}`); err != nil {
		t.Fatalf("WriteField metadata failed: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close writer failed: %v", err)
	}

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest("POST", "/v1/tasks", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	c.Request = req

	info := &RelayInfo{TaskRelayInfo: &TaskRelayInfo{}}
	taskErr := ValidateBasicTaskRequest(c, info, "generate")
	if taskErr != nil {
		t.Fatalf("ValidateBasicTaskRequest returned error: code=%s message=%s", taskErr.Code, taskErr.Message)
	}

	taskReq, err := GetTaskRequest(c)
	if err != nil {
		t.Fatalf("GetTaskRequest failed: %v", err)
	}

	if taskReq.Prompt != "draw a cat" {
		t.Fatalf("unexpected prompt: %q", taskReq.Prompt)
	}
	if taskReq.Model != "kling-v1" {
		t.Fatalf("unexpected model: %q", taskReq.Model)
	}

	if taskReq.Metadata["seed"] != float64(123) {
		t.Fatalf("unexpected metadata seed: %#v", taskReq.Metadata["seed"])
	}
	if taskReq.Metadata["style"] != "anime" {
		t.Fatalf("unexpected metadata style: %#v", taskReq.Metadata["style"])
	}
}
