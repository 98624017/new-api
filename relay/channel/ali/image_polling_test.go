package ali

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/synctest"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type imageTaskRoundTrip func(*http.Request) (*http.Response, error)

func (f imageTaskRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestAsyncImagePolling(t *testing.T) {
	const proxyURL = "http://image-poll-proxy.test"
	const result = `{"output":{"task_status":"SUCCEEDED"}}`
	service.InitHttpClient()
	t.Cleanup(service.ResetProxyClientCache)

	for _, tc := range []struct {
		name         string
		proxy        string
		failures     int
		disconnected bool
		wantAttempts int
		wantTimeout  bool
	}{
		{name: "configured proxy retries and returns result", proxy: proxyURL, failures: 1, wantAttempts: 2},
		{name: "direct timeouts exhaust retries", failures: 20, wantAttempts: 20, wantTimeout: true},
		{name: "accepted task settles after disconnect", proxy: proxyURL, disconnected: true, wantAttempts: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				recorder := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(recorder)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				c.Request = httptest.NewRequestWithContext(ctx, http.MethodPost, "/v1/images/generations", nil)
				if tc.disconnected {
					cancel()
				}
				info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{
					ChannelSetting: dto.ChannelSettings{Proxy: tc.proxy},
					ChannelBaseUrl: "http://127.0.0.1:0",
					ApiKey:         "test-key",
				}}
				client, err := service.GetHttpClientWithProxy(tc.proxy)
				require.NoError(t, err)
				original := client.Transport
				defer func() { client.Transport = original }()
				attempts := 0
				client.Transport = imageTaskRoundTrip(func(r *http.Request) (*http.Response, error) {
					attempts++
					assert.Equal(t, http.MethodGet, r.Method)
					assert.Equal(t, "/api/v1/tasks/task-1", r.URL.Path)
					assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
					require.NoError(t, r.Context().Err())
					if attempts <= tc.failures {
						return nil, context.DeadlineExceeded
					}
					return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(result))}, nil
				})

				accepted := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"output":{"task_id":"task-1"}}`))}
				relayErr, usage := aliImageHandler(&Adaptor{}, c, accepted, info)
				assert.Equal(t, tc.wantAttempts, attempts)
				if tc.wantTimeout {
					require.ErrorIs(t, relayErr, context.DeadlineExceeded)
					assert.ErrorContains(t, relayErr, "after 20 attempts")
					assert.True(t, types.IsSkipRetryError(relayErr))
					return
				}
				require.Nil(t, relayErr)
				require.NotNil(t, usage)
				var response dto.ImageResponse
				require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
				assert.JSONEq(t, result, string(response.Metadata))
			})
		})
	}
	t.Run("missing task ID keeps submission retry policy", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		resp := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"output":{}}`))}
		relayErr, _ := aliImageHandler(&Adaptor{}, c, resp, &relaycommon.RelayInfo{})
		require.NotNil(t, relayErr)
		assert.False(t, types.IsSkipRetryError(relayErr))
	})
}
