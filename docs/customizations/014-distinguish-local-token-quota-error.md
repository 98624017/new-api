# 014-distinguish-local-token-quota-error

适配上游基线：`7c28993f6bd9e92616f3f578212577f8b7c40b45`。

## 1. 背景

本地 API Key 的 token 额度不足发生在请求发往上游之前，现有文案可能与上游渠道 key 余额不足的响应混淆。

## 2. 目标

- 默认保持现有本地错误文案，避免已有客户端行为变化。
- 通过 `DISTINGUISH_LOCAL_TOKEN_QUOTA_ERROR=true` 时，将本地 token/key 预扣失败提示改为“本平台 API Key 额度不足，请充值或更换 Key”。
- 上游渠道返回的错误文案、错误码、类型和状态码完全不处理。

## 3. 实现范围

- `common/constants.go` / `common/init.go`：读取默认关闭的环境变量。
- `service/billing_session.go`：统一处理预扣 token 失败文案，覆盖首次预扣和补充预扣路径。
- `service/error_test.go`：覆盖开关关闭、开启以及错误码和状态码保持不变。

## 4. 兼容性与风险

- 环境变量未设置或解析失败时保持关闭。
- 只替换本地预扣 token 失败的错误文本，不影响用户钱包/订阅额度错误，也不影响上游错误处理。

## 5. 回归验证

```bash
go test ./service ./common ./types
make verify-patches
```
