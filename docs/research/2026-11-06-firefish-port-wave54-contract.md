# 第五十四波施工契约（ap/show 与远程互动页）

日期：2026-11-06
状态：已落地，证据见 `docs/HANDOVER.md` §12BS
上游参考：

- Misskey `ap/show`（`packages/backend/src/server/api/endpoints/ap/show.ts`，契约来自台账与
  `docs/research/api-contract/meta-federation.md`）
- Firefish `packages/client/src/pages/authorize_interaction.vue`（页面语义）

## 1. 为什么做

`/authorize_interaction?uri=…` 是 Fediverse 通用的「在其它实例打开」入口，Mastodon / Misskey /
Firefish 都会生成这种链接。没有 `ap/show` 时这个页面只能展示占位；有了它，用户可以把远端笔记或
账号拉进本实例并直接互动（关注、回复、转发）。

## 2. 端点

`POST /api/ap/show`

| 项目 | 值 |
|------|-----|
| 鉴权 | 需要凭据 |
| 参数 | `uri: string`（必须是可抓取的 ActivityPub 对象地址） |
| 返回 | `{ type: "Note", object: NoteDetailed }` 或 `{ type: "User", object: UserDetailed }` |
| 错误 | 无法解析时 400 `NO_SUCH_OBJECT` |

实现要点：

1. `federation.Inbox.ImportObject(ctx, uri)`：带签名抓取（策略与 allowlist 生效），
   - Actor 文档 → `upsertActor`（复用入站路径）→ `type: User`
   - Note 类对象 → 校验 `attributedTo` 存在、笔记 host 等于作者 host（防伪），
     已存在则返回既有 id，否则 `NoteDataFromAP` + `InsertRemoteNote`
2. HTTP 层按 kind 打包：笔记走 `notePackContext` + `packNoteDetail`（与 `notes/show` 同一包装），
   用户走 `packUserDetailed` + `packMovedTo`
3. 导入的笔记会发布 `notesStream` 事件（复用 `OnNoteCreated`），但不创建通知：该对象并非发给本站的

## 3. 前端

`web/src/pages/AuthorizeInteraction.vue`：

- 无 `uri` → 提示链接不完整
- 加载 → `CfLoading`
- `User` → 头像 / 名称 / 完整 acct / 关注按钮
- `Note` → `MkNoteDetailed`（自带回复、转发、反应等动作）
- 失败 → 说明 + 「在原实例查看」外链（`rel="nofollow noopener"`）

## 4. 写作用域

| 文件 | 动作 |
|------|------|
| `internal/federation/inbox.go` | 新增 `ImportObject` |
| `internal/federation/store.go` | 新增 `NoteIDByAPID` |
| `internal/httpapi/ap_show.go` | 新增处理器 |
| `internal/httpapi/server.go` | 注册 `POST /api/ap/show` |
| `internal/httpapi/ap_show_test.go` | 新增两条用例 |
| `web/src/pages/AuthorizeInteraction.vue` | 占位页 → 真页面 |
| `web/src/api/endpoints.ts` | `ap/show` 类型 |
| `docs/research/2026-11-04-misskey-api-coverage.md` | 141 路由 / 133 同名 / 307 未实现 |

## 5. 有意的限制

- 导入不建通知、不做提及解析：对象不是发给本实例的，伪造通知会被滥用。
- 只接受 Note / Actor 两类对象：其它 ActivityPub 对象（Collection、Article 之外的扩展类型）
  一律按无法解析处理。
- 不做 Mastodon 的 `/authorize_interaction` 语义（那是 Mastodon 兼容层，本项目明确不做）。

## 6. 验收口径

- `go vet ./...` 通过；`CINDERFISH_TEST_DATABASE_URL=… go test ./...` 全绿
- 新用例：
  - `TestAPShowResolvesRemoteObjects`：双实例，笔记 URL → `{type:"Note"}` 且正文一致；
    账号 URL → `{type:"User", username:"betabob"}`
  - `TestAPShowRequiresCredential`：匿名 → 401
- `vue-tsc --noEmit`、`check:port`（hardTotal 0）、`pnpm build` 全绿
