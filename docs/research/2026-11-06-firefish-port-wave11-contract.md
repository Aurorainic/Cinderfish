# 第十一波施工契约：Webhook（2026-11-06）

设置里的 Webhook 一直是占位页。本波按 Firefish/Misskey 的 `i/webhooks/*` 契约补表与 CRUD，
并在发帖链路接入真实投递。

## 1. 数据与端点

迁移 v12：`webhooks(id, user_id, name, url, secret, events TEXT[], active, created_at,
latest_sent_at, latest_status)`。列名用 `events` 而不是 `on`——`ON` 是 PostgreSQL 保留字，
上游靠 TypeORM 加引号才成立，这里避开即可。

| 端点 | 行为 |
|------|------|
| `i/webhooks/create` | `name`(≤100) / `url`(http(s), ≤1024) / `secret`(≤1024) / `on`（非空，枚举内） |
| `i/webhooks/list` | 当前账号的 Webhook，id 倒序 |
| `i/webhooks/show` | `webhookId`；别人的 id 返回 404 |
| `i/webhooks/update` | 全量字段 + `active` |
| `i/webhooks/delete` | `webhookId`，204；别人的 id 也是 204（无副作用） |

实现：`internal/webhook/service.go`（新）、`internal/httpapi/webhooks.go`（新）、
`internal/httpapi/webhooks_dispatch.go`（新）。

## 2. 投递

- 触发点：`notes/create` 成功后调用 `dispatchNoteWebhook`，事件类型按帖子形态取
  `reply` / `renote` / `note`。
- 载荷与请求头沿上游 `webhook-deliver`：
  `{ hookId, userId, eventId, createdAt, type, body }`；
  `X-Firefish-Hook-Id`、`X-Firefish-Hook-Secret`、`User-Agent: Cinderfish-Hooks`。
- 异步：每条 hook 一个 goroutine，5 秒超时；结果回写 `latest_sent_at` / `latest_status`
  （网络错误记 1），失败不影响发帖。
- 只投递 `events` 数组包含该事件且 `active` 的 hook。

## 3. 有意的降级

1. **只产生 note / reply / renote 三种事件**：`follow`、`reaction`、`mention` 等枚举仍被 API 接受，
   但不会触发；设置页只给这三种勾选框并写明原因。
2. **没有 `i/webhooks/test`**：参考快照里没有该端点，不造一个假按钮。
3. **没有重试队列**：失败只记录状态，不重投；个人实例先接受这一点。
4. **没有 `X-Firefish-Host` 头**：上游带头部主机名，这里省略，避免额外配置耦合。

## 4. 前端

| 文件 | 变化 |
|------|------|
| `web/src/components/WebhookEditor.vue`（新） | 新建/编辑共用表单：名称、地址、密钥、三个事件勾选、启用开关 |
| `web/src/pages/SettingsWebhook.vue` | 占位页改为列表：启用徽标、URL、事件、最近投递与 HTTP 状态、编辑/删除 |
| `web/src/pages/SettingsWebhookNew.vue` | 用编辑器创建后返回列表 |
| `web/src/pages/SettingsWebhookEdit.vue` | 加载 show、保存 update、删除并返回 |
| `web/src/pages/SettingsIndex.vue` | 客户端设置组新增 Webhook |
| `web/src/api/endpoints.ts`、`types.ts` | 五个端点与 `Webhook` 类型 |

## 5. 验收

`go vet`、`go test ./...`（新增 3 个用例：CRUD/越权、校验、真实投递）、`vue-tsc`、`check:port`、
`pnpm build` 全绿；真实二进制上用本地接收器验证设置页创建 → 发帖 → 列表显示 HTTP 200 →
停用 → 删除。证据见 HANDOVER §12AA.3。
