# 第五十三波施工契约（OAuth 交互式同意页）

日期：2026-11-06
状态：已落地，证据见 `docs/HANDOVER.md` §12BR
上游参考：

- `docs/research/api-contract/auth-users.md`（OAuth 一节）
- `internal/httpapi/oauth.go`（本仓库既有实现）
- Firefish `packages/client/src/pages/oauth.vue`（同意页形态；该文件在稀疏快照里缺失，
  按 Misskey OAuth 页面语义重写）

## 1. 问题

`GET /oauth/authorize` 原来只有在请求自身带凭据时才能发码，否则回 401 HTML。
浏览器里的用户根本没有「带凭据」这个概念，于是路由表里的 `/oauth/authorize` 页面
被后端处理器挡住、`web/src/pages/OAuth.vue` 是死代码——第三方应用无法用 RFC OAuth 登录本站。

## 2. 方案

把「发码」与「同意」拆成两步，且不破坏既有服务端调用：

### 2.1 `GET /oauth/authorize`（行为变更，向后兼容）

1. 校验参数（`client_id`、`redirect_uri`、`response_type=code`、`code_challenge`、`S256`），
   失败 400 OAuth 错误体
2. **有凭据**（`Authorization: Bearer`）：发码并 302 到 `redirect_uri`，与之前一致
3. **无凭据且前端构建存在**：返回 SPA 的 `index.html`，浏览器渲染同意页
4. 无凭据且没有前端构建：仍然 401 HTML 提示，API-only 部署行为不变

### 2.2 `POST /api/oauth/authorize`（Cinderfish 自有端点）

同意页调用一次：

| 项目 | 值 |
|------|-----|
| 鉴权 | 需要凭据（body 的 `i`） |
| 参数 | `client_id`、`redirect_uri`、`response_type`、`code_challenge`、`code_challenge_method`、`state?`、`scope?: string[]` |
| 返回 | `{ "redirect": "https://client/callback?code=…&state=…&iss=…" }` |

返回跳转地址而不是 302，因为 SPA 在 JSON 调用里无法跟随后端 302，必须自己 `location.assign`。
参数校验与 GET 共用 `parseOAuthAuthorize`，规则只有一份。

## 3. 前端

`web/src/pages/OAuth.vue`：

- 无 `client_id` / `redirect_uri` / `code_challenge` 等 → 直接显示缺参说明
- 未登录 → 内嵌 `CfSignin`
- 已登录 → 显示应用主机、请求的 scope（read/write/follow/upload 中文说明）、接受 / 取消
- 接受 → `POST /api/oauth/authorize` → `location.assign(redirect)`
- 取消 → 显示已拒绝（不伪造回调，避免客户端以为授权成功）

## 4. 写作用域

| 文件 | 动作 |
|------|------|
| `internal/httpapi/oauth.go` | 抽出 `parseOAuthAuthorize` / `issueOAuthGrant` / `oauthRedirectURL`；GET 分支；新增 POST 处理器 |
| `internal/httpapi/server.go` | 注册 `POST /api/oauth/authorize` |
| `internal/httpapi/oauth_consent_test.go` | 新增三条用例 |
| `web/src/pages/OAuth.vue` | 占位页 → 同意页 |
| `web/src/api/endpoints.ts` | `oauth/authorize` 常量与类型 |
| `docs/research/2026-11-04-misskey-api-coverage.md` | 140 路由 / 8 自有路径 |

## 5. 有意的限制

- 不校验 `client_id` 是不是合法 URL：Misskey 的 `client_id` 就是应用地址，但不是所有客户端都传 URL；
  同意页只在能解析时显示主机名。
- 不提供「记住这个应用」：每次同意都发新码，`oauth_grant_codes` 一次性使用。
- 取消授权不回调客户端（不发送 `error=access_denied`）：客户端会自行超时；伪造错误回调需要额外
  定义客户端可信度，收益不抵风险。
- 同意页没有 scope 的细粒度勾选：本站支持的 scope 是粗粒度四类，按请求原样授予或整体拒绝。

## 6. 验收口径

- `go vet ./...` 通过；`CINDERFISH_TEST_DATABASE_URL=… go test ./...` 全绿
- 新增用例：
  - `TestOAuthAuthorizeConsentAPI`：同意 → 返回 redirect（含 code/state）→ 换 token 成功
  - `TestOAuthAuthorizeConsentAPIRequiresCredential`：匿名 POST → 401
  - `TestOAuthAuthorizeConsentAPIRejectsBadParams`：plain PKCE → 400
  - 既有 `TestOAuthTokenFlow` / `TestOAuthTokenRejectsWrongVerifier` / `TestOAuthAuthorizeRejectsPlainPKCE`
    不受影响
- 前端 `vue-tsc --noEmit`、`check:port`（hardTotal 0）、`pnpm build` 全绿
