# M4 Streaming API 源码校准与实施记录

- 日期：2026-10-28
- 上游源码：`/tmp/refcheck/misskey`（misskey-dev/misskey，浅克隆，重启丢失，重建命令见 AGENTS.md）
- 读取范围：`packages/backend/src/server/api/stream/`、`packages/backend/src/server/api/StreamingApiServerService.ts`、`packages/backend/src/core/{GlobalEventService,NoteCreateService,ReactionService,NoteDeleteService,NotificationService}.ts`、`packages/misskey-js/src/{streaming,streaming.types}.ts`

## 1. 端点与握手

- 路径：`GET /streaming`，HTTP Upgrade 升级为 WebSocket。
- 鉴权（`StreamingApiServerService.attach`）：
  1. token 取值优先级：`Authorization: Bearer <token>` → 否则 query `?i=<token>`。
  2. 无 token 是**合法**的匿名连接（`user = null`），不会被拒绝。只有 token 存在但无效才 401。
  3. 401 响应是**裸 HTTP 行**（不经过 HTTP 框架）：
     ```
     HTTP/1.1 401 Unauthorized
     WWW-Authenticate: Bearer realm="Misskey", error="invalid_token", error_description="Failed to authenticate"
     ```
     然后直接 `socket.destroy()`。
  4. access token 必须含 `read:account` 权限，否则当作认证失败（401）。
  5. 已封禁用户返回 `HTTP/1.1 403 Forbidden`。
  6. `request.url == null` 时 400。
- 连接后**不主动**给客户端发消息（除非客户端 `connect` 时带 `pong: true`）。

## 2. 客户端 → 服务端消息

JSON 对象 `{"type": string, "body": ...}`，parse 失败直接忽略。

| type | 别名 | body | 行为 |
|------|------|------|------|
| `connect` | — | `{channel, id, params?, pong?}` | 订阅频道 |
| `disconnect` | — | `{id}` | 退订频道 |
| `channel` | `ch` | `{id, type, body}` | 向频道发消息（本里程碑的频道均无 onMessage） |
| `subNote` | `s`, `sr` | `{id}` | 订阅单条笔记的更新 |
| `unsubNote` | `un` | `{id}` | 退订单条笔记 |
| `readNotification` | — | — | 全部通知标记已读；body 被忽略 |

`connect` 参数校验（`onChannelConnectRequested`）：`id` 必须 string，`channel` 必须 string，`pong` 只允许 boolean/undefined/null，`params` 只允许对象/undefined。任一条不满足就**静默丢弃**。

## 3. 服务端 → 客户端消息

| type | body | 触发 |
|------|------|------|
| `channel` | `{id, type, body}` | 频道事件。`id` 是客户端给的连接 id |
| `connected` | `{id}` | 仅在 `connect` 带 `pong: true` 时回 |
| `noteUpdated` | `{id, type, body}` | `subNote` 的笔记更新 |
| `broadcast`（展开） | 直接 `{type, body}` | emojiAdded 等全员广播 |

## 4. Connection 关键规则（`stream/Connection.ts`）

- 常量：`MAX_CHANNELS_PER_CONNECTION = 32`；`MAX_SUBSCRIBING_NOTES_PER_CONNECTION = 1536`；`MAX_BUFFERED_AMOUNT_PER_CONNECTION = 8 MiB`。
- 消息串行处理：`promiseLimit<void>(1)`，保证 connect/disconnect 不会交错。
- `connectChannel`：
  1. 同 id 已存在 → 先断开旧的。
  2. 频道数已达 32 → **静默返回**。
  3. 未知频道 → `getChannelConstructor` 抛错 `no such channel: X`。异常在 message 队列里被吞掉，客户端**收不到任何反馈**（只会在服务端日志里）。
  4. `requireCredential && user == null` → 静默返回。
  5. access token 的 kind 检查：`token.kind` 存在时要求 permission 含该 kind；无 kind 但 requireCredential 时同样要求。**原生 token（16 位）不适用此检查**，因为 `this.token` 为 null。
  6. `shouldShare` 的频道（只有 main）重复连接 → 静默忽略。
  7. `init()` 返回 `false` → 断开并从 map 删除（**但不会通知客户端**）。
- 缓冲保护：`sendMessageToWs` 前检查 `bufferedAmount + len(message) > 8 MiB` → 直接 `terminate()`。
- 心跳：服务端每 60s 对每条连接 `ping()`；上次活跃超过 120s 未收到 `pong` → `terminate()`。`message` 也会刷新活跃时间吗？——**不会**，只有 `pong` 和建连会刷新。
- 每 10s 刷新一次用户的 following / muting 关系缓存（`userFollowingsCache` 等）。

## 5. 频道定义（`stream/channel.ts` 与 `channels/`）

| channel | requireCredential | kind | shouldShare | params |
|---------|-------------------|------|-------------|--------|
| `main` | true | `read:account` | **true** | 无 |
| `homeTimeline` | true | `read:account` | false | withRenotes(默认 true), withFiles(默认 false) |
| `localTimeline` | false | — | false | withRenotes, withReplies, withFiles（均默认 false） |
| `hybridTimeline` | true | `read:account` | false | withRenotes, withReplies, withFiles |
| `globalTimeline` | false | — | false | withRenotes, withFiles |

未实现频道（M4 不做）：`userList`、`clip`、`hashtag`、`roleTimeline`、`antenna`、`channel`、`drive`、`serverStats`、`queueStats`、`admin`、`chatUser`、`chatRoom`、`reversi`、`reversiGame`。连接它们的正确行为是**静默丢弃**（上游是抛错然后被吞）。

### 5.1 频道基类的把关逻辑

`isNoteVisibleForMe`（必须与 `QueryService.generateVisibilityQuery` 同步）：
- `specified`：匿名 → false；作者本人 → true；否则看 `visibleUserIds`。
- `followers`：匿名 → false；作者本人 → true；是回复目标 → true；被 @ → true；否则看是否关注作者。
- 其他（public/home）→ true。

`isNoteMutedOrBlocked`：实例屏蔽、用户屏蔽/静默、renote 静默、频道静默。**MilnFish 目前没有 mute/block 数据**，此函数恒为 false，但保留接口位置。

### 5.2 各频道过滤（已实现的部分）

- **main**：只推送 `mainStream:<userId>` 事件。notification 事件里若 note 的 `isHidden`（CW 隐藏，需要详情重新 pack）则以 detail 重新 pack；mention 事件做可见性 + 屏蔽检查。**其他事件原样透传**（reply/renote/follow/followed/unfollow/meUpdated/readAllNotifications/unreadNotification/notificationFlushed/signin/driveFileCreated/announcementCreated/...）。
- **homeTimeline**：`notesStream` → withFiles 过滤 → 只留 `isMe || following 里有作者` → 可见性 → reply 过滤（`following[author].withReplies` 分支：如果开了带回复，则只挡「作者的 followers-only 笔记的回复，而我不关注回复目标且不是我」；否则只留「回复我 / 我发出的回复 / 作者自回复」）→ 纯 renote 且 withRenotes=false 则丢弃（含被 renote 内容的 reply 可见性检查）→ 屏蔽 → `noteStreamingHidingService.filter`（M4 先恒真）→ 输出带 `myReaction`。
- **localTimeline**：`ltlAvailable` 策略（MilnFish 一律 true）→ withFiles → 只留 `user.host == null` 且 `visibility == public` 且 `channelId == null` → 回复过滤（匿名时不过滤）→ renote 过滤 → 屏蔽 → 输出。
- **hybridTimeline**：需登录。无 `channelId` 时要求 `isMe || 关注作者 || (本地 && public)`；有 `channelId` 时要求关注该频道（M4 无频道功能 → 恒 false）→ 可见性 → 屏蔽 → 回复过滤（`following.author.withReplies || params.withReplies`）→ renote 过滤 → 输出。
- **globalTimeline**：只留 `visibility == public` 且 `channelId == null` → renote 过滤 → 屏蔽 → 输出。

### 5.3 回复过滤的三段式（易错点）

```
if (following[noteAuthor]?.withReplies || params.withReplies) {
    // 宽松：「反抗」只能挡掉：回复目标是 followers-only 笔记的作者，而我不关注他且不是我自己
    if (reply.visibility === 'followers' && !following[reply.userId] && reply.userId !== me) return;
} else {
    // 严格：只留「回复我」「我发的回复」「作者自回复」
    if (reply.userId !== me && !isMe && reply.userId !== note.userId) return;
}
```
注意严格分支里 `!isMe` 的判断：`reply.userId !== me && !isMe && reply.userId !== note.userId`，即**我自己的笔记一定保留**。

## 6. 事件源（`GlobalEventService` 的 publish 调用点）

Redis pub/sub 的 `channel` 字段充当「订阅键」，MilnFish 用同样的字符串做进程内 bus 的 key，未来接 Redis 时语义一致。

| publish 方法 | 频道键 | type | body 形状 | 调用点 |
|--------------|--------|------|-----------|--------|
| publishNotesStream | `notesStream` | —（整个 payload 就是 note） | Packed Note | `NoteCreateService` 创建成功后（非 silent） |
| publishMainStream | `mainStream:<uid>` | `notification` | Packed Notification | `NotificationService.createNotification` |
| 同上 | 同上 | `unreadNotification` | Packed Notification | 通知创建 2s 后仍未读 |
| 同上 | 同上 | `readAllNotifications` | null | 标记全部已读 |
| 同上 | 同上 | `notificationFlushed` | null | 清空通知 |
| 同上 | 同上 | `mention` | **detail** Packed Note | 被 @ 的本地用户 |
| 同上 | 同上 | `reply` | Packed Note | 被回复的本地用户（非线程静默） |
| 同上 | 同上 | `renote` | Packed Note | 被 renote 的本地用户（非自己的 renote） |
| 同上 | 同上 | `follow`/`followed`/`unfollow` | Packed User | 关注关系变化 |
| 同上 | 同上 | `meUpdated` | MeDetailed | 自己资料更新 |
| publishNoteStream | `noteStream:<noteId>` | `reacted` | `{reaction, emoji?, userId}` | 添加反应 |
| 同上 | 同上 | `unreacted` | `{reaction, userId}` | 撤销反应 |
| 同上 | 同上 | `deleted` | `{deletedAt}` | 删除笔记 |
| 同上 | 同上 | `pollVoted` | `{choice, userId}` | 投票 |

`publishNoteStream` 的外层信封固定为：
```json
{"id": "<noteId>", "userId": "<authorId>", "visibility": "...", "visibleUserIds": [...], "body": {...}}
```
`subNote` 分发前检查（`onNoteStreamMessage`）：如果不是我自己的笔记，则 specified 要包含我、followers 要我关注作者。**注意这里用的是笔记作者，不是事件里的 userId**——事件里 userId 就是作者。

## 7. 与 MilnFish 现状的差距

1. **笔记创建**：`notes.Create` 有 `notifyMentioned`，但没有 reply/renote 的 main 事件。M4 补齐。
2. **只存 mentions 数组**：`Mentions` 目前恒为空数组（`Create` 不解析文本中的 @）。因此 mention 事件暂时不会触发。**这是已知缺口**，记录在此，不在 M4 修（属于笔记解析，和 M5 的远程解析一起做）。
3. **通知没有 isRead 之外的已读时间**，`readNotification` 靠 `MarkAllRead` 即可。
4. **没有 follow 端点**（`following/create` 未实现），`follow/followed` 事件暂无来源。频道里依赖 followings 表的过滤逻辑照常实现，等 M5 端点落地即生效。
5. **没有 mute/block 数据**：屏蔽过滤按恒 false 实现。
6. **笔记 packer 在 httpapi 包**：streaming 需要用同一个 packer。为避免 streaming → httpapi 反向依赖，packer 留在 httpapi，由 httpapi 在发布事件时就把实体打包成 JSON 再交给总线。

## 8. 实施决定（落地版本）

- **依赖方向**：`httpapi → streaming`，绝不反向。`streaming` 只声明会话需要的少量接口（`Authenticator`、`ViewerLoader`），实现全部在 `httpapi`。笔记与通知的打包只在 `httpapi` 做（`packNoteFor` / `packNotification`），`streaming` 只转发已编码的 map。
- **事件总线 `internal/streaming.Bus`**：
  - `Subscribe(Handler) cancel` / `Publish(subject, Message)`，单会话多订阅。
  - 进程内 fanout 是**唯一**保证单实例正确性的通道。
  - 可选 `SetPublisher(*Publisher)`：每次 Publish 后异步 `pg_notify`，另一进程 `Listen` 收到后回灌本地 bus。这为多进程部署预留，不需要 Redis。
  - NOTIFY payload 上限 8000 字节；超限时自动降级为只带 subject 的唤醒信号（不丢数据，因为弱化只影响跨进程的场景）。
- **连接**：`Connection` 持有频道表、subNote 表（引用计数 + LRU，上限 1536）与缓存的 following 图。`notesStream` 在连接层面**只订阅一次**再分发到各时间线频道，比上游"每个频道各订阅一次"的 N 倍投递更省。
- **Go 嵌入的坑（务必记住）**：`timelineChannel` 内嵌在具体频道里时，**内嵌类型的同名方法会遮蔽外层方法**，通过接口调用 `HandleNote` 命中的是内嵌的实现。因此不能靠"外层覆盖 onNote"做多态，改为把过滤函数存成字段（`timelineChannel.filter`），由各频道 `Init` 时赋值。这个 bug 第一版真实发生过：所有时间线频道都收不到笔记，单元测试暴露后才修正。
- **上下文刷新**：每 10s 重新加载 following，与上游一致；MilnFish 没有 Redis 缓存，直接查库。
- **心跳**：60s ping / 120s 无 pong 断开，与上游一致。
- **发送缓冲**：每连接 256 条发送队列，满了直接断连（对应上游的 bufferedAmount 8 MiB 阈值，语义一致：宁可断开重连也不无限积压）。
- **依赖**：`github.com/gorilla/websocket v1.5.3`（直接依赖）。

## 9. 验证记录（2026-10-28）

| 层次 | 测试 | 结果 |
|------|------|------|
| 频道过滤矩阵 | `internal/streaming/channel_test.go`：五频道的可见性 / withRenotes / withReplies / withFiles / specified / followers / 回复过滤 | ✅ |
| 频道生命周期 | connect / disconnect / 32 频道上限 / main 去重 / pong→connected | ✅ |
| subNote | 引用计数、LRU 上限 1536、noteUpdated 可见性 | ✅ |
| 中继 | `TestStreamingCrossProcessRelay`：A bus → pg_notify → B bus | ✅ |
| 端到端 | `internal/httpapi/streaming_e2e_*.go`：401/403/匿名、笔记投递、反应触发通知 + noteUpdated、specified/followers 不外泄、readNotification 生效 | ✅ |
| 真实二进制 | 起进程 → 注册 → WS 连 main + localTimeline → 发笔记 → 收到 `{"type":"channel","body":{"type":"note",...}}` | ✅ |
| 竞态 | `go test -race ./internal/streaming/ ./internal/httpapi/` | ✅ |

**未做**：用 misskey-js / Misskey Web 前端实际联调；双进程真实部署联调。
