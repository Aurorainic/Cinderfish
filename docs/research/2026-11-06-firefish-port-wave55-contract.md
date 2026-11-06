# 第五十五波施工契约（第三方应用授权：app/* 与 auth/session/*）

日期：2026-11-06
状态：已落地，证据见 `docs/HANDOVER.md` §12BT
上游参考（**以 Misskey 源码为准**，本轮把 Misskey `4ff2edf` 的
`packages/backend/src` 稀疏克隆到 `tmp/refcheck/misskey`，先前的 `/tmp` 抓取只是同一份文件的副本）：

- `packages/backend/src/server/api/endpoints/app/create.ts`
- `packages/backend/src/server/api/endpoints/app/show.ts`
- `packages/backend/src/server/api/endpoints/auth/accept.ts`
- `packages/backend/src/server/api/endpoints/auth/session/{generate,show,userkey}.ts`
- `packages/backend/src/core/entities/{AppEntityService,AuthSessionEntityService}.ts`
- `packages/frontend/src/pages/auth.vue`、`auth.form.vue`（同意页）

## 1. 流程

1. 客户端 `POST /api/app/create {name, description, permission[], callbackUrl?}`（**无需凭据**）→
   返回 App，含 `secret`
2. 客户端 `POST /api/auth/session/generate {appSecret}` → `{token, url: https://<domain>/auth/<token>}`
3. 用户在浏览器打开 `url`，页面 `POST /api/auth/session/show {token}` →
   `{id, app: {id,name,callbackUrl,permission,isAuthorized}, token}`
4. 用户同意 → `POST /api/auth/accept {token}`（**需要凭据**）→ 204，同时按 `(app, user)` 发放访问令牌
5. 客户端轮询 `POST /api/auth/session/userkey {appSecret, token}` →
   `{accessToken, user}`；成功后会话被删除（只能取一次）；未批准时 `PENDING_SESSION`

`isAuthorized` 为真时同意页不再询问，直接 `auth/accept` 并跳转（Misskey 行为）。

## 2. 数据模型（迁移 v20）

```sql
CREATE TABLE apps (
    id TEXT PRIMARY KEY,
    user_id TEXT REFERENCES users(id) ON DELETE CASCADE,  -- 匿名注册为 NULL
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    permission JSONB NOT NULL DEFAULT '[]',
    callback_url TEXT,
    secret TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE auth_sessions (
    id TEXT PRIMARY KEY,
    app_id TEXT NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    token TEXT NOT NULL UNIQUE,
    user_id TEXT REFERENCES users(id) ON DELETE CASCADE,  -- 批准后写入
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
ALTER TABLE access_tokens ADD COLUMN app_id TEXT REFERENCES apps(id) ON DELETE CASCADE;
```

`access_tokens.app_id` 让 `i/apps` 能按 Misskey 的规则回退到应用名称/权限（`token.name` 优先）。

## 3. 端点契约细节

| 端点 | 凭据 | 参数 | 返回 |
|------|------|------|------|
| `app/create` | 否 | `name`（必填）、`description`、`permission[]`、`callbackUrl?` | App（含 `secret`） |
| `app/show` | 否 | `appId` | App；`secret` 仅当“用原生 token 登录且是应用所有者” |
| `auth/session/generate` | 否 | `appSecret` | `{token, url}` |
| `auth/session/show` | 否 | `token` | `{id, app, token}`（不含 secret） |
| `auth/accept` | 是 | `token` | 204 |
| `auth/session/userkey` | 否 | `appSecret`、`token` | `{accessToken, user: UserDetailed}` |

错误码与 id 全部照抄 Misskey 源码（`NO_SUCH_APP`、`NO_SUCH_SESSION`、`PENDING_SESSION`）。

## 4. 前端

`web/src/pages/Auth.vue`：未登录内嵌 `CfSignin`；已登录读取会话、展示权限、接受/取消；
接受后若应用有 `callbackUrl` 则带着 `?token=<session token>` 跳回（协议白名单拒绝
`javascript:`/`data:` 等）；无回调则提示返回应用。

同时补齐了 `_auth.*` 的三个缺失 i18n 键（`pleaseLogin`、`accepted`、`alreadyAuthorized`），
zh-CN 与 ja-JP 同步。

## 5. 写作用域

| 文件 | 动作 |
|------|------|
| `internal/db/db.go` | 迁移 v20 |
| `internal/account/apps.go` | App / AuthSession 存储与令牌发放 |
| `internal/account/tokens.go` | `i/apps` 关联应用列 |
| `internal/httpapi/app_auth.go` | 六个处理器与 `packApp` |
| `internal/httpapi/server.go` | 六条路由 |
| `internal/httpapi/app_auth_test.go` | 两条用例 |
| `web/src/pages/Auth.vue` | 占位页 → 同意页 |
| `web/src/api/endpoints.ts` | 六个端点与 `AppEntity` 类型 |
| `web/src/i18n/locales/{zh-CN,ja-JP}.json` | 三个 `_auth.*` 键 |

## 6. 有意的限制

- 应用密钥不做过期与轮换：Misskey 也没有，密钥泄露只能删除应用行。
- 同一 `(app, user)` 只有一条访问令牌；重复批准不会新建（与 Misskey 一致）。
- 不校验 `callbackUrl` 的域名所有权，只在跳转时做协议白名单。
- `permission` 不做取值校验（Misskey 也不校验），仅用于展示与记录。

## 7. 验收口径

- `go vet ./...` 通过；`CINDERFISH_TEST_DATABASE_URL=… go test ./...` 全绿
- `TestAppSessionFlow`：注册 → 生成 → show（无 secret、未授权）→ userkey 待批 → 批准 →
  取回可用 token → `/api/i` 生效 → 第二次取回 404 → `i/apps` 名称与权限来自应用
- `TestAuthSessionGenerateRejectsUnknownSecret`：错误密钥 404 `NO_SUCH_APP`
- `vue-tsc --noEmit`、`check:port`（missingKeys 0、hardTotal 0）、`pnpm build` 全绿
