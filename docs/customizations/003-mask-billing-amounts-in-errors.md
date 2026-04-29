# 003-mask-billing-amounts-in-errors

## 1. 背景

部分上游或本地计费链路在错误信息中会带出具体金额或额度数值。  
例如预扣费失败时，客户端可能看到：

```text
预扣费额度失败, 用户剩余额度: ¥0.056700, 需要预扣费额度: ¥0.069900
```

这类数值可能暴露本地成本价、预扣费策略或上游额度细节，不适合透传给下游客户。

## 2. 目标

- 保留原始错误语义，方便客户知道是额度不足或预扣费失败
- 只脱敏具体金额 / 额度数值
- 保留 `status_code`、`request id` 等排查信息
- 覆盖同步 relay 响应、Claude 响应和异步任务错误响应

不解决：

- 不统一改写所有错误文案
- 不改变计费、预扣费、退款逻辑
- 不新增管理后台开关

## 3. 业务规则

- `¥0.056700`、`$0.069900` 等货币金额展示为 `¥***`、`$***`
- `token remain quota: 120`、`need quota: 300` 等额度标签后的裸数字展示为 `***`
- `need=69900` 这类上游或订阅额度不足文案展示为 `need=***`
- `status_code=403`、`request id req_123` 等非计费字段不脱敏
- 内部原始 error 对象不改写，只在面向客户端的 message 上脱敏

## 4. 影响范围

- `common/str.go`
  - 新增客户端计费金额脱敏函数
- `types/error.go`
  - OpenAI / Claude 风格错误响应增加金额脱敏
- `service/error.go`
  - 异步任务错误响应增加金额脱敏
- 测试文件
  - 覆盖中文预扣费文案、英文额度文案、订阅 `need=` 文案

## 5. 风险点

- 如果上游后续使用新的金额格式，可能需要扩展正则
- 当前策略会脱敏带常见货币符号的金额，以及常见计费标签后的数字
- 为避免误伤，未做“所有数字”全局脱敏

## 6. 测试方案

最小验证命令：

```bash
go test ./common -run TestMaskBillingAmountsForClient -count=1
go test ./types -run TestNewAPIErrorTo -count=1
go test ./service -run 'Test(TaskError.*MasksBillingAmounts|ResetStatusCode)' -count=1
```

完整二开校验：

```bash
make verify-patches
```

## 7. 升级关注点

上游同步时重点关注：

- `types/error.go` 中 `ToOpenAIError` / `ToClaudeError` 是否重构
- `service/error.go` 中异步任务错误包装是否重构
- `common/str.go` 中敏感信息脱敏函数是否重构

## 8. 当前状态

- 已实现客户端错误金额脱敏
- 已补充同步 relay 与异步任务错误测试
- 已生成 `patches/003-mask-billing-amounts-in-errors.patch`
