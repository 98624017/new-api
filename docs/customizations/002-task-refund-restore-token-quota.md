# 002-task-refund-restore-token-quota

## 1. 背景

当前异步视频任务走“提交后预扣费、轮询终态后结算”的链路。  
在用户使用自己的 API Key 再做二次分发时，若视频任务最终失败，仅退款到用户钱包会带来一个业务问题：

- 用户的钱包额度恢复了
- 但触发请求的 token/key 可用额度没有恢复

对于依赖 token/key 进行二次销售或配额管理的用户，这会造成账实不一致，影响后续分发。

## 2. 目标

- 新增一个环境变量开关
- 开关开启时，在异步视频任务失败退款路径上，除了退款到钱包，还恢复该 token/key 对应额度
- 默认关闭，保持现有默认行为可控

不解决：

- 不改动成功任务的结算规则
- 不改动同步接口的计费逻辑
- 不引入新的用户级账本模型

## 3. 建议开关

建议环境变量名：

```bash
TASK_REFUND_RESTORE_TOKEN_QUOTA=false
```

建议默认值：

- 默认关闭

开启后行为：

- 异步视频任务失败时，退回用户资金来源
- 同时恢复该任务绑定 token 的额度

关闭后行为：

- 保持当前默认逻辑

## 4. 业务规则

- 仅对走 `videos` 异步轮询链路的任务生效
- 仅对进入失败终态的任务生效
- 成功终态不触发额外 token 恢复
- 超时若当前系统按失败处理，也应遵循同样规则
- 同一任务只能执行一次终态退款，避免重复补回
- 当任务未绑定 token 时，跳过 token 恢复
- 当 token 已删除或无法查询时，只记录 warning，不阻断钱包退款

## 5. 数据流

1. 用户发起异步视频任务
2. 系统提交任务并预扣额度
3. 轮询任务状态
4. 若任务成功：
   - 按现有链路结算，不触发新增逻辑
5. 若任务失败：
   - 先退款到资金来源
   - 若开关开启且任务关联了 token，则恢复 token 额度
   - 记录退款日志与恢复路径信息

## 6. 预期影响文件

- `common/init.go`
  - 读取环境变量并初始化开关
- `service/task_billing.go`
  - 在失败退款逻辑中按开关决定是否恢复 token 额度
- `service/task_polling.go`
  - 确认失败终态调用路径与幂等约束
- `service/task_billing_test.go`
  - 补充开关开启 / 关闭的测试

## 7. 风险点

- 若失败终态重复进入，可能导致重复退款或重复恢复 token
- 若 token 已删除，恢复动作可能失败
- 若 token 为无限额度，恢复逻辑需要确认是否跳过
- 若未来上游调整异步任务计费链路，patch 可能容易冲突
- 若当前系统实际已经在某些路径恢复 token，新增逻辑必须避免重复叠加

## 8. 测试方案

建议覆盖：

- 开关关闭：失败后只退资金来源
- 开关开启：失败后退资金来源并恢复 token 额度
- token 缺失：钱包退款成功，token 恢复仅记日志
- 重复失败轮询：不会重复退款 / 重复恢复
- 成功终态：不触发新增恢复逻辑

建议最小验证命令：

```bash
go test ./service -run '^(TestRefundTaskQuota|TestCASGuarded)' -v
```

建议补充业务黑盒验收：

```bash
bash scripts/verify_task_refund_restore_token_quota.sh new-api:verify-20260406
```

该脚本会：

- 先用 `scripts/seed_task_refund_fixture.go` 离线生成 SQLite fixture
- 再分别以 `TASK_REFUND_RESTORE_TOKEN_QUOTA=false` 和 `true` 启动容器
- 通过用户登录后的 `GET /api/user/self` 验证钱包退款结果
- 通过 `GET /api/usage/token/` 验证 key/token 剩余额度是否恢复
- 最后回读数据库，确认任务状态已经落为 `FAILURE`

当前黑盒验收基线：

- 初始：`USER_QUOTA=800`，`TOKEN_REMAIN=300`
- 开关关闭：`USER_QUOTA=1000`，`TOKEN_REMAIN=300`
- 开关开启：`USER_QUOTA=1000`，`TOKEN_REMAIN=500`

## 9. 升级关注点

上游同步时重点关注：

- `service/task_billing.go` 中失败退款逻辑是否重构
- `service/task_polling.go` 中终态处理与 CAS 幂等逻辑是否变化
- `service/billing_session.go` 与 `service/quota.go` 中 token quota 调整接口是否变化
- `model/token.go` 中 token quota 增减实现是否变化

## 10. 当前状态

- 需求已确认
- 已新增环境变量 `TASK_REFUND_RESTORE_TOKEN_QUOTA`
- 已将失败退款中的 token 恢复逻辑改为按开关执行
- 已补充 `RefundTaskQuota` 相关测试
- 已补充更接近业务路径的容器黑盒验收脚本：
  - `scripts/seed_task_refund_fixture.go`
  - `scripts/verify_task_refund_restore_token_quota.sh`
- 已生成 `patches/002-task-refund-restore-token-quota.patch`
