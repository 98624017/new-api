# 006-frontend-lock

## 目标

增加一个可选的前端弱隐藏门禁。部署者设置 `FRONTEND_LOCK_PASSWORD` 后，浏览器访问 new-api 前端页面会先看到锁屏和公告；输入正确密码后，当前浏览器会缓存解锁状态并继续访问原前端。

## 背景

部分部署只希望对授权人员开放内部项目服务，不希望普通访客打开同一域名时直接看到 new-api 网页入口。完整安全隔离应通过 Cloudflare Access、反向代理路径限制、源站防火墙或后端鉴权实现。本补丁只提供轻量弱隐藏，降低普通访客误入概率。

## 行为

- `FRONTEND_LOCK_PASSWORD` 为空：前端不显示锁屏，行为与原项目一致。
- `FRONTEND_LOCK_PASSWORD` 非空：Go 服务启动时把密码注入到 `index.html` 的 `window.__FRONTEND_LOCK_PASSWORD__`。
- 前端加载后，如果当前浏览器尚未解锁、解锁缓存已过期，或解锁缓存对应的密码已变化，则显示锁屏页。
- 密码正确后写入 `localStorage`，解锁状态在当前浏览器缓存 2592000 秒（30 天），与后端用户登录 session cookie 的 `MaxAge` 保持一致。
- 后端服务请求路径不受影响，因为非浏览器客户端不会加载前端页面。
- 本地 Vite 开发可使用 `VITE_FRONTEND_LOCK_PASSWORD` 预览锁屏。

## 风险

这不是安全边界。密码会出现在浏览器可见的 HTML/JS 中，具备前端调试能力的人可以找到它。该功能只能防普通访客，不能替代：

- Cloudflare Access
- 反向代理按域名/路径限制
- 源站防火墙
- 后端鉴权

变更 `FRONTEND_LOCK_PASSWORD` 后需要重启服务。浏览器刷新拿到新注入密码后，旧解锁缓存会因密码指纹不匹配而失效，需要重新输入密码。

## 涉及文件

- `main.go`
- `main_test.go`
- `web/src/index.jsx`
- `web/src/components/common/FrontendLock.jsx`
- `web/src/helpers/frontendLock.js`

## 验证

```bash
go test . -run TestInjectFrontendLockPassword -count=1
cd web && bun run build
make verify-patches
```
