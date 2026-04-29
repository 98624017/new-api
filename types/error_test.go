package types

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestNewAPIErrorToOpenAIErrorMasksBillingAmounts(t *testing.T) {
	t.Parallel()

	err := NewErrorWithStatusCode(
		errors.New("预扣费额度失败, 用户剩余额度: ¥0.056700, 需要预扣费额度: ¥0.069900 (request id req_123)"),
		ErrorCodeInsufficientUserQuota,
		http.StatusForbidden,
	)

	openAIError := err.ToOpenAIError()

	if strings.Contains(openAIError.Message, "0.056700") || strings.Contains(openAIError.Message, "0.069900") {
		t.Fatalf("expected billing amounts to be masked, got %q", openAIError.Message)
	}
	if !strings.Contains(openAIError.Message, "用户剩余额度: ¥***") ||
		!strings.Contains(openAIError.Message, "需要预扣费额度: ¥***") {
		t.Fatalf("expected masked quota labels to remain readable, got %q", openAIError.Message)
	}
	if !strings.Contains(openAIError.Message, "request id req_123") {
		t.Fatalf("expected request id to remain visible, got %q", openAIError.Message)
	}
}

func TestNewAPIErrorToClaudeErrorMasksBillingAmounts(t *testing.T) {
	t.Parallel()

	err := NewErrorWithStatusCode(
		errors.New("token quota is not enough, token remain quota: 120, need quota: 300"),
		ErrorCodePreConsumeTokenQuotaFailed,
		http.StatusForbidden,
	)

	claudeError := err.ToClaudeError()

	if strings.Contains(claudeError.Message, "120") || strings.Contains(claudeError.Message, "300") {
		t.Fatalf("expected billing amounts to be masked, got %q", claudeError.Message)
	}
	if !strings.Contains(claudeError.Message, "token remain quota: ***") ||
		!strings.Contains(claudeError.Message, "need quota: ***") {
		t.Fatalf("expected masked quota labels to remain readable, got %q", claudeError.Message)
	}
}
