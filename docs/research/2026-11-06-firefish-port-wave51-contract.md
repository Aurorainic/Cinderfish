# 第五十一波施工契约（账号迁移：i/move、i/known-as、入站 Move）

日期：2026-11-06
状态：已落地，证据见 `docs/HANDOVER.md` §12BP
上游参考：Firefish `b425f61`：

- `packages/client/src/pages/settings/migration.vue`（迁移页）
- `packages/client/src/components/MkMoved.vue`（迁移横幅）
- `packages/client/src/pages/user/home.vue`（横幅挂载位置）
- `packages/backend/src/server/api/endpoints/i/move.ts`（校验顺序与副作用）
- `packages/backend/src/server/api/endpoints/i/known-as.ts`
- `packages/backend/src/remote/activitypub/kernel/move/index.ts`（入站 Move 校验）
- `packages/backend/src/remote/activitypub/renderer/person.ts`（`movedTo` / `alsoKnownAs` 输出）

## 1. 为什么做这一波

账号迁移是最后一处有联邦影响的设置，也是 `SettingsMigration` 占位页的唯一依赖。
没有它，实例用户换账号只能重建关注关系，且对端无法知道旧账号已废弃。

## 2. 协议与端点契约

### 2.1 `POST /api/i/known-as`

| 项目 | 值 |
|------|-----|
| 鉴权 | 需要凭据 |
| 参数 | `alsoKnownAs: string[]`，最多 10 条，元素可以是 `@user`、`user@host`、`@user@host` |
| 返回 | `MeDetailed` |
| 错误 | `NO_SUCH_USER`、`FORBIDDEN_TO_SET_YOURSELF`、`ALREADY_MOVED` |

- 本地用户名解析为本地 Actor URI；`user@host` 走 WebFinger → Actor 文档 → 落库（`users` 行）
- 存的是 **Actor URI**，与 Firefish 的列语义一致；去重后整体覆盖
- 已迁移的账号拒绝改写别名（与上游一致）

### 2.2 `POST /api/i/move`

| 项目 | 值 |
|------|-----|
| 鉴权 | 需要凭据 |
| 参数 | `moveToAccount: string`（必须是远端 `user@host`） |
| 返回 | `MeDetailed`（含 `movedToUri`） |
| 错误 | `NO_SUCH_MOVE_TARGET`、`NOT_REMOTE`、`REMOTE_ACCOUNT_FORBIDS`、`NOT_ADMIN_FORBIDDEN`、`ALREADY_MOVED`、`URI_NULL` |

执行顺序（照 `i/move.ts`）：

1. 管理员拒绝；已迁移拒绝
2. 无 host → `NOT_REMOTE`（本地账号不是合法目标）
3. WebFinger 解析目标；失败 → `NO_SUCH_MOVE_TARGET`
4. **重新抓取**目标 Actor（不吃缓存），其 `alsoKnownAs` 必须包含本账号 Actor URI，
   否则 `REMOTE_ACCOUNT_FORBIDS`
5. 写 `users.moved_to_uri`，并把目标 URI 追加到 `also_known_as`
6. 向关注者投递 `Move`（`{type,actor,object,target}`，按本账号密钥签名，走投递队列）
7. 本地关注者改关目标账号（本地目标直接落库；远端目标经队列发 `Follow`）

### 2.3 入站 `Move`

`federation.Inbox` 新增 `Move` 分支：

- `object` 若存在必须等于 `actor` 的 URI
- 取 `target`，**强制重新抓取**目标 Actor 文档
- 目标的 `alsoKnownAs` 必须含 `actor.URI`，否则 `skip: target does not list the actor in alsoKnownAs`
- 通过后写旧账号的 `moved_to_uri`，并通过 `OnMoved` 回调交给 HTTP 层迁移本地关注者
  （回调是必要的：投递队列归 HTTP 层持有，Inbox 只有签名客户端）

### 2.4 Actor 文档

`RenderPerson` 输出 `movedTo`（URI 字符串）与 `alsoKnownAs`（URI 数组），空值不输出（与上游一致）。

## 3. 数据模型

迁移 **v19**：

```sql
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS moved_to_uri TEXT,
    ADD COLUMN IF NOT EXISTS also_known_as JSONB NOT NULL DEFAULT '[]'::jsonb;
CREATE INDEX ... ON users (moved_to_uri) WHERE moved_to_uri IS NOT NULL;
```

`account.User` 增加 `MovedToURI`、`AlsoKnownAs`，`scanUser` 的投影统一为 `userColumns` 常量；
六处查询（`GetByID` / `GetByUsername` / `ListLocal` / `admin_users`×2 / `search` / `users_list` /
`tokens`）同步，新增 `GetByURI`。

## 4. API 打包变化

- `UserLite` 新增 `alsoKnownAs: string[] | null` 与 `movedToUri: null`（列表不查库，保持轻量）
- `UserDetailed` 的 `movedToUri` 由 `Server.packMovedTo` 在详情端点按 URI 反查并打包成 UserLite；
  适用的端点：`users/show`、`i`、`i/known-as`、`i/move`
- `users/show` 补 `userIds[]` 批量参数（Misskey 契约），并对单个 `userId` 增加 **URI 回退**：
  `alsoKnownAs` / `movedToUri` 存的是 URI，Firefish 客户端会把它们原样传回

## 5. 前端写作用域

| 文件 | 动作 |
|------|------|
| `web/src/pages/SettingsMigration.vue` | 占位页 → 真页面（迁移到、从旧账号迁移两段） |
| `web/src/components/MkMoved.vue` | 新增（Firefish 同名组件移植） |
| `web/src/components/user/UserHome.vue` | 挂载横幅，删除「movedTo 恒为 null」的降级注释 |
| `web/src/pages/SettingsIndex.vue` | 菜单加入「迁移」，更新陈旧说明 |
| `web/src/api/types.ts` | `movedToUri` / `alsoKnownAs` 类型 |
| `web/src/api/endpoints.ts` | `i/known-as`、`i/move` 端点常量与类型 |

## 6. 有意的限制

- 迁移页按别名逐个调用 `users/show`（单个 `userId`），不用上游的批量 `userIds`：批量响应会改变
  `users/show` 的返回类型，而端点类型表目前是单用户形状。后端两种形式都支持，前端只是选简单的那种。
- 本地关注者改关只处理 `followings` 行；远端关注者的迁移是对端实例的职责（上游同样如此）。
- 没有「撤销迁移」：协议与上游都没有。
- `alsoKnownAs` 不校验条目数以外的语义（不验证对端是否真的在别名里写了本账号，那由对端在
  `Move` 时校验）。

## 7. 验收口径

- `go test ./...`（含数据库）：新增 4 条用例
  `TestIKnownAsLocalAlias`、`TestIMoveRefusesLocalAndAdminTarget`、
  `TestTwoInstanceMove`、`TestInboundMove`
- `vue-tsc --noEmit`、`check:port`（hardTotal 0）、`pnpm build` 全绿
- 双实例用例走真实 HTTP：WebFinger、签名、Actor 抓取、投递队列都不是桩
