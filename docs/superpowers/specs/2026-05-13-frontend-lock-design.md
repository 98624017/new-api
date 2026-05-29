# Frontend Lock Design

## Goal

为 new-api 前端增加一个可选的弱隐藏门禁：公开域名访问网页时先看到锁屏和项目公告，输入环境变量配置的密码后才能进入原前端页面。

## Scope

- 只保护浏览器加载的前端页面，不改变 `/v1`、`/v1beta`、`/mj`、`/suno` 等 API 调用。
- 不作为真正安全边界。密码会出现在返回给浏览器的 HTML/JS 中，适合阻挡普通访客，不适合抵御有技术能力的访问者。
- 密码通过运行时环境变量 `FRONTEND_LOCK_PASSWORD` 配置；为空时完全关闭门禁。
- 兼容构建时变量 `VITE_FRONTEND_LOCK_PASSWORD`，方便本地前端开发预览。

## Architecture

后端启动时读取 `FRONTEND_LOCK_PASSWORD`，如果非空则向嵌入的 `index.html` 注入：

```html
<script>window.__FRONTEND_LOCK_PASSWORD__="...";</script>
```

前端入口在渲染原有 `PageLayout` 前读取该值。如果值为空，直接渲染原应用；如果值非空且本次会话未解锁，渲染 `FrontendLock`。解锁状态写入 `sessionStorage`，关闭浏览器会话后需要重新输入。

## UI Behavior

锁屏页面包含：

- 清晰的锁定状态和密码输入框。
- 原项目公告内容：优先显示 `/api/notice` 返回的 Markdown 公告；同时读取 `/api/status` 中的系统公告列表，作为页面下方信息。
- 密码错误时显示轻量错误提示，不发送到后端。
- 密码正确时写入 `sessionStorage` 并进入原有前端。

## Files

- `main.go`：新增运行时注入函数，并在 analytics 注入后执行。
- `main_test.go`：覆盖密码为空不注入、密码非空注入、特殊字符安全转义。
- `web/src/index.jsx`：在根渲染处包裹前端门禁。
- `web/src/components/common/FrontendLock.jsx`：新增锁屏组件。
- `web/src/helpers/frontendLock.js`：新增密码读取、会话状态和校验 helper。
- `docs/customizations/006-frontend-lock.md`：记录本地二开背景、风险和验证。
- `patches/006-frontend-lock.patch`：记录该二开补丁。

## Validation

- `go test . -run TestInjectFrontendLockPassword -count=1`
- `cd web && bun run build`
- `make verify-patches`

## Risks

- 该方案不能替代 Cloudflare Access、反代路径限制或后端鉴权。
- `FRONTEND_LOCK_PASSWORD` 变更后需要重启 Go 服务，浏览器刷新后才能拿到新的注入值。
- 已经通过 `sessionStorage` 解锁的浏览器标签页在本会话内不会再次提示。
