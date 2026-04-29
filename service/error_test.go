package service

import (
	"errors"
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/require"
)

func TestResetStatusCode(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		statusCode       int
		statusCodeConfig string
		expectedCode     int
	}{
		{
			name:             "map string value",
			statusCode:       429,
			statusCodeConfig: `{"429":"503"}`,
			expectedCode:     503,
		},
		{
			name:             "map int value",
			statusCode:       429,
			statusCodeConfig: `{"429":503}`,
			expectedCode:     503,
		},
		{
			name:             "skip invalid string value",
			statusCode:       429,
			statusCodeConfig: `{"429":"bad-code"}`,
			expectedCode:     429,
		},
		{
			name:             "skip status code 200",
			statusCode:       200,
			statusCodeConfig: `{"200":503}`,
			expectedCode:     200,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			newAPIError := &types.NewAPIError{
				StatusCode: tc.statusCode,
			}
			ResetStatusCode(newAPIError, tc.statusCodeConfig)
			require.Equal(t, tc.expectedCode, newAPIError.StatusCode)
		})
	}
}

func TestTaskErrorWrapperMasksBillingAmounts(t *testing.T) {
	t.Parallel()

	taskErr := TaskErrorWrapper(
		errors.New("token quota is not enough, token remain quota: 120, need quota: 300"),
		"insufficient_quota",
		http.StatusForbidden,
	)

	require.NotContains(t, taskErr.Message, "120")
	require.NotContains(t, taskErr.Message, "300")
	require.Contains(t, taskErr.Message, "token remain quota: ***")
	require.Contains(t, taskErr.Message, "need quota: ***")
	require.Equal(t, http.StatusForbidden, taskErr.StatusCode)
}

func TestTaskErrorFromAPIErrorMasksBillingAmounts(t *testing.T) {
	t.Parallel()

	apiErr := types.NewErrorWithStatusCode(
		errors.New("预扣费额度失败, 用户剩余额度: ¥0.056700, 需要预扣费额度: ¥0.069900"),
		types.ErrorCodeInsufficientUserQuota,
		http.StatusForbidden,
	)

	taskErr := TaskErrorFromAPIError(apiErr)

	require.NotContains(t, taskErr.Message, "0.056700")
	require.NotContains(t, taskErr.Message, "0.069900")
	require.Contains(t, taskErr.Message, "用户剩余额度: ¥***")
	require.Contains(t, taskErr.Message, "需要预扣费额度: ¥***")
	require.Equal(t, http.StatusForbidden, taskErr.StatusCode)
}
