package service

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"testing/synctest"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRelayResponseHeaderTimeout(t *testing.T) {
	// 启动配置会修改进程级状态，子进程使用独立环境并关闭日志目录。
	const childEnv = "NEWAPI_TEST_RELAY_RESPONSE_HEADER_TIMEOUT"
	if os.Getenv(childEnv) != "1" {
		cmd := exec.Command(os.Args[0], "-test.run=^TestRelayResponseHeaderTimeout$", "-log-dir=")
		cmd.Env = []string{childEnv + "=1"}
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "%s", output)
		return
	}

	for _, tc := range []struct {
		name       string
		env        string
		timeout    time.Duration
		headerWait time.Duration
		wantErr    bool
	}{
		{"default times out", "", 30 * time.Minute, 31 * time.Minute, true},
		{"late headers time out", "1", time.Second, 2 * time.Second, true},
		{"slow body is not cut off", "1", time.Second, 0, false},
		{"zero allows late headers", "0", 0, 2 * time.Second, false},
		{"negative disables timeout", "-1", 0, 2 * time.Second, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("RELAY_RESPONSE_HEADER_TIMEOUT", tc.env)
			common.InitEnv()
			InitHttpClient()
			t.Cleanup(ResetProxyClientCache)
			for _, proxyURL := range []string{"http://proxy.test", "https://proxy.test", "socks5://proxy.test", "socks5h://proxy.test"} {
				client, err := GetHttpClientWithProxy(proxyURL)
				require.NoError(t, err)
				transport, ok := client.Transport.(*http.Transport)
				require.True(t, ok, "%s transport", proxyURL)
				assert.Equal(t, tc.timeout, transport.ResponseHeaderTimeout, "%s", proxyURL)
			}
			synctest.Test(t, func(t *testing.T) {
				client := GetHttpClient()
				transport := client.Transport.(*http.Transport)
				defer transport.CloseIdleConnections()
				clientConn, upstream := net.Pipe()
				done := make(chan struct{})
				defer func() {
					_ = clientConn.Close()
					<-done
				}()
				transport.DialContext = func(context.Context, string, string) (net.Conn, error) { return clientConn, nil }
				go func() {
					defer close(done)
					defer upstream.Close()
					if _, err := http.ReadRequest(bufio.NewReader(upstream)); err != nil {
						return
					}
					time.Sleep(tc.headerWait)
					if _, err := io.WriteString(upstream, "HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\n"); err != nil {
						return
					}
					time.Sleep(3 * time.Second)
					_, _ = io.WriteString(upstream, "ok")
				}()

				resp, err := client.Get("http://upstream.test")
				if tc.wantErr {
					var timeout net.Error
					require.ErrorAs(t, err, &timeout)
					assert.True(t, timeout.Timeout())
					return
				}
				require.NoError(t, err)
				defer resp.Body.Close()
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				assert.Equal(t, "ok", string(body))
			})
		})
	}
}
