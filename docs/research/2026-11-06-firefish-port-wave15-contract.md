# 第十五波施工契约：MiAuth 第三方授权（2026-11-06）

`/miauth/:session` 一直是占位页，第三方客户端也无法按 MiAuth 规范取回 token——后端缺
`miauth/{session}/check`，`access_tokens` 也没有「已被应用取走」的标记。本波补齐这条链路。

## 1. 数据

迁移 v13：`access_tokens` 增加 `fetched BOOLEAN NOT NULL DEFAULT FALSE`。语义：token 已被请求方
通过 check 端点取走；第二次轮询不再返回 token。

## 2. 端点

| 端点 | 权限 | 行为 |
|------|------|------|
| `POST /api/miauth/gen-token` | 登录 | 已存在：批准会话并签发 token（session 作为关联键） |
| `POST /api/miauth/{session}/check`（新） | 无 | 首次轮询返回 `{ok:true, token, user}` 并标记 fetched；之后与未知 session 都返回 `{ok:false}` |

实现：`internal/httpapi/miauth.go`（新）、`server.go` 注册；`account.TokenService` 新增
`FindUnfetchedBySession` 与 `MarkFetched`。

## 3. 前端

`MiAuth.vue` 从占位页改为复用账号层已有的 `CfMiauth`：未登录先显示登录表单，登录后展示应用名与
权限、接受/取消；接受后调用 `miauth/gen-token`，有 `callback` 时带 `session` 参数跳回应用，
否则显示「请返回至应用」。

同时补上缺失的 `_permissions.read:notes` 文案（zh-CN / ja-JP）。

## 4. 有意的限制

1. **scope 不强制**：`permission` 只是展示，服务端除 Streaming 的 `read:account` 外不校验 scope。
2. **没有拒绝回执**：取消只是不签发 token，应用轮询会一直得到 `{ok:false}`，没有显式 denied 状态。
3. **没有授权页过期**：会话可由应用一直轮询；token 一旦签发即长期有效（与现有 token 生命周期一致）。
4. **OAuth 同意页未做**：`GET /oauth/authorize` 仍在校验通过后直接签码，本波只做 MiAuth。

## 5. 验收

`go vet`、`go test ./...`（新增 MiAuth check 用例：一次取走、二次 false、未知 session）、`vue-tsc`、
`check:port`、`pnpm build` 全绿；真实二进制走查授权页与轮询。证据见 HANDOVER §12AE.3。
