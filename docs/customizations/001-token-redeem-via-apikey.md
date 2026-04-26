# 001-token-redeem-via-apikey

## 1. 背景

上游默认兑换接口为 `POST /api/user/topup`，依赖用户登录 Session。  
在本地业务中，存在通过 API Key 二次分发或由外部工具直接调用兑换接口的需求，仅依赖 Session 不够方便。

## 2. 目标

- 新增一个支持 API Key 认证的兑换接口
- 复用原有兑换逻辑，不重写兑换核心规则
- 尽量缩小对上游代码的侵入范围

不解决：

- 不改变管理员创建兑换码的逻辑
- 不改变原有 `/api/user/topup` 行为
- 不扩展新的权限模型

## 3. 业务规则

- 新接口路径为 `POST /api/token/redeem`
- 认证方式为 `TokenAuthReadOnly()`
- 请求体与原 `TopUp` 接口保持一致
- 兑换成功后，额度进入用户钱包
- 兑换成功后，当前请求使用的 token/key 额度同步增加相同值
- 兑换成功后的充值使用记录需标明兑换到哪个 token/key 名称
- 兑换失败时，返回与原有兑换逻辑一致的错误语义

## 4. 影响范围

- 接口层：新增一个兑换入口
- 控制器层：新增 `TokenRedeem`
- 鉴权层：复用现有 Token 鉴权中间件
- 兑换核心：新增 token 场景下的钱包 + token quota 联动兑换
- 使用记录：token 场景下的充值日志追加 token/key 名称，便于排查额度来源

## 5. 关键文件

- `controller/user.go`
  - 新增 `TokenRedeem`
- `model/redemption.go`
  - 新增 token 场景下的联动兑换逻辑
- `router/api-router.go`
  - 注册 `POST /api/token/redeem`
- `patches/001-token-redeem-via-apikey.patch`
  - 保存本二开的可重放差异

## 6. 数据流

1. 客户端携带 `Bearer sk-xxx` 调用 `POST /api/token/redeem`
2. `TokenAuthReadOnly()` 校验 token 并写入用户上下文
3. `TokenRedeem` 复用原有 `topUpRequest`
4. 调用 token 场景下的兑换逻辑，在同一事务中：
   - 增加用户钱包 quota
   - 增加当前 token 的 remain_quota
   - 读取当前 token 的名称，用于生成充值使用记录
   - 核销兑换码
5. 成功则返回额度结果，失败则返回原有错误

## 7. 风险点

- 上游若调整 `TopUp` 的请求结构或锁逻辑，本地接口也要同步
- 上游若调整 `TokenAuthReadOnly()` 行为，可能影响本接口可用性
- 上游若调整 token quota 字段或更新方式，本地联动补额逻辑也要同步
- 若未来兑换逻辑加入额外风控，本接口需要确认是否同步继承

## 8. 测试方案

建议覆盖：

- `Bearer sk-xxx` 可通过 `TokenAuthReadOnly()` 进入兑换逻辑
- token 鉴权成功时可以进入兑换逻辑
- 兑换成功时用户钱包增加额度
- 兑换成功时当前 token 的 remain_quota 同步增加
- 兑换成功时充值使用记录包含当前 token/key 名称
- 兑换失败时返回预期错误
- 并发兑换时仍受同一用户锁保护

建议最小验证命令：

```bash
go test ./controller -run '^TestTokenRedeem' -v
```

当前已落地的最小回归用例：

- 成功兑换时钱包和 token 额度同时增加
- 成功兑换时充值日志显示兑换到的 token/key 名称
- 无效 token 返回 401
- 无效兑换码返回业务失败

## 9. 升级关注点

上游同步时重点关注：

- `controller/user.go` 中 `TopUp` 附近逻辑是否重构
- `model/redemption.go` 中兑换码核销逻辑是否重构
- `router/api-router.go` 中用户 API 路由组织方式是否变化
- `middleware/auth.go` 中 `TokenAuthReadOnly()` 是否调整
