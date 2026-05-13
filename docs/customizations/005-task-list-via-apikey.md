# 005-task-list-via-apikey

## 1. 背景

当前项目已经支持异步视频任务提交后，通过 `GET /v1/videos/{task_id}` 或 `GET /v1/video/generations/{task_id}` 携带 API Key 查询单个任务状态。  
但如果下游客户端只保存了 API Key，没有保存提交返回的 `task_id`，就无法免登录找回“这个 key 自己创建过哪些异步视频任务”。

这在 API Key 二次分发、外部控制台或轮询工具场景下不够用。

## 2. 目标

- 新增一个支持 API Key 免登录的异步任务列表接口
- 只返回“当前请求使用的 token/key 创建的任务”
- 保持与现有 `/api/task/self` 一致的分页和筛选参数风格
- 兼容历史任务：老数据即使只有 `private_data.token_id`，也要能查到

不解决：

- 不改动原有 `/api/task/self` 的 Session 登录模式
- 不新增新的后台页面
- 不开放跨用户、跨 token 的任务检索能力

## 3. 业务规则

- 新接口路径：`GET /api/task/token/self`
- 认证方式：`TokenAuthReadOnly()`
- 支持的查询参数与现有用户任务列表保持一致：
  - `p`
  - `page_size` / `size`
  - `task_id`
  - `status`
  - `action`
  - `platform`
  - `start_timestamp`
  - `end_timestamp`
- 只返回当前 token 创建的任务，不返回同一用户其他 token 创建的任务
- 对于补丁上线前创建的老任务：
  - 若任务表独立 `token_id` 列为空或为 `0`
  - 则在首次查询时按用户批量回填 `private_data.token_id -> token_id`
  - 回填完成后，列表与总数查询都只走独立 `token_id` 列，保持数据库分页

## 4. 影响范围

- `router/api-router.go`
  - 注册 `GET /api/task/token/self`
- `controller/task.go`
  - 新增 `GetUserTokenTask`
- `model/task.go`
  - 任务表新增独立 `token_id` 字段
  - 新增按 token 查询用户任务的方法
  - 新增老任务 `private_data.token_id` 的批量回填逻辑
- `controller/relay.go`
  - 新建异步任务时同步写入独立 `token_id`
- `controller/task_token_test.go`
  - 覆盖当前 token 过滤和 `task_id` 筛选
- `controller/user_token_redeem_test.go`
  - 调整测试 helper，避免多用户场景下唯一索引冲突

## 5. 风险点

- 老任务首次按 token 查询时，会扫描当前用户的 `token_id=0` 任务做一次批量回填，历史数据很多时首查会比新数据慢
- 若上游未来调整任务表结构或移除 `private_data.token_id`，兼容回退逻辑需要同步调整
- 当前接口仍复用任务 DTO，不额外暴露 token 详情，若下游以后需要展示“来自哪个 key”，需要另行扩展响应结构

## 6. 测试方案

最小验证命令：

```bash
go test ./controller -run '^(TestTokenRedeem|TestGetUserTokenTask)' -count=1
```

完整二开校验：

```bash
make verify-patches
```

## 7. 升级关注点

上游同步时重点关注：

- `middleware/auth.go` 中 `TokenAuthReadOnly()` 是否调整
- `controller/task.go` 中用户任务列表入参 / 返回格式是否重构
- `model/task.go` 中任务表字段和查询函数是否重构
- `controller/relay.go` 中异步任务落库逻辑是否调整

## 8. 当前状态

- 已实现 `GET /api/task/token/self`
- 已实现“当前 token 任务列表”分页查询
- 已实现首次查询时按用户批量回填 legacy `token_id`
- 已补充控制器测试
- 已生成 `patches/005-task-list-via-apikey.patch`
