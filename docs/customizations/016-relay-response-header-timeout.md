# 016：限制 relay 等待上游响应头的时间

来源：[上游 #6949](https://github.com/QuantumNous/new-api/pull/6949)。当请求已写完、上游却不返回响应头时，长期等待会占用连接、goroutine 和请求体内存。

启动时读取 `RELAY_RESPONSE_HEADER_TIMEOUT`，单位秒，默认 `1800`。设为 `0` 可恢复无限等待，负数也按零处理。直连、HTTP(S) 和 SOCKS5/SOCKS5H 渠道代理共用该设置。使用共享客户端的 Worker 下载以及关闭 SSRF 防护后回退到共享客户端的媒体下载也受此限制。独立媒体抓取客户端与 WebSocket 使用各自的拨号设置。

这个超时按单次上游 HTTP 请求计时。收到响应头后，响应体读取仍由 `RELAY_TIMEOUT` 和已有流式超时控制。非流式上游有时等生成结束才发响应头，应为最长的正常生成留出余量。

Ali 异步生图返回任务 ID 后，状态查询复用提交时的渠道代理和响应头超时。轮询首次等待 5 秒，再以 10 秒为间隔查询，最多 20 次，失败的查询也计入次数。下游断开后继续获取已接单任务的终态，以便结算。查询失败耗尽次数或返回失败终态时，网关向调用方返回错误，并跳过整笔生图的重新提交。首次提交等待响应头超时时，任务是否已被上游接收仍然未知。

## 上游冲突时恢复

按序重放 `patches/016-relay-response-header-timeout.patch`。若上游已修改相同文件，逐项比对并补齐以下行为。

1. 在 `common/constants.go` 保留 `time.Duration` 类型的配置。在 `common/init.go` 的 `InitEnv` 读取秒数，按上述默认值和负数规则处理，并在乘以 `time.Second` 前限制最大可表示值。同步更新根目录 `.env.example` 和 `README.md` 的配置说明。
2. 在 `service/http_client.go` 统一设置 relay transport 的 `ResponseHeaderTimeout`。`InitHttpClient` 的普通客户端使用环境代理，`NewProxyHttpClient` 的 HTTP(S) 分支安装对应的代理 URL，SOCKS5/SOCKS5H 分支安装渠道拨号器。三条路径均保留已有的空闲连接、TLS 和总超时设置。
3. 在 `relay/channel/ali/image.go`，让已接单的图片任务通过 `service.GetHttpClientWithProxy(info.ChannelSetting.Proxy)` 查询状态。核对轮询次数、下游断开后结算、已接单任务跳过整笔重试三个分支。
4. 恢复 `service/http_client_timeout_test.go` 和 `relay/channel/ali/image_polling_test.go`，并把两个包的定向用例接回 `scripts/verify_patches.sh`。更新 016 补丁后执行以下验证。

```bash
go test ./service ./relay/channel/ali -run '^(TestRelayResponseHeaderTimeout|TestAsyncImagePolling)$' -count=1
timeout 120s make verify-patches
```
