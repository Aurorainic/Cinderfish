# M5 联邦（ActivityPub）源码校准记录

- 日期：2026-10-28
- 上游参考：`/tmp/refcheck/misskey`（`misskey-dev/misskey`，AGPL-3.0-only，浅克隆）
- 对照文件：`packages/backend/src/server/ActivityPubServerService.ts`、`WellKnownServerService.ts`、
  `NodeinfoServerService.ts`、`core/WebfingerService.ts`、`core/activitypub/ApRendererService.ts`、
  `core/activitypub/ApRequestService.ts`、`core/activitypub/ApInboxService.ts`、
  `core/activitypub/misc/contexts.ts`、`core/RemoteUserResolveService.ts`

本文只记录**从源码读出、并且在本仓库实现中被采用或被刻意偏离**的事实。每一条都附上游出处，便于复核。

## 1. 路由与响应类型（ActivityPubServerService.ts）

| 路径 | 方法 | 联邦模式 `none` 时的行为 | 备注 |
|------|------|--------------------------|------|
| `/inbox` | POST | 403 | 共享收件箱，body 上限 64 KiB |
| `/users/:user/inbox` | POST | 403 | 与共享收件箱同一处理器 |
| `/users/:user` | GET | 403 | 按 **id** 查，不是用户名；远程用户 301 跳转到其 `uri` |
| `/@:acct` | GET | 403 | 按 `username@host` 解析，本域 host 归一化为 null |
| `/users/:user/outbox` | GET | 403 | 首页 `OrderedCollection`，`?page=true` 才出条目 |
| `/users/:user/followers` | GET | 403 | 同上；`followersVisibility` 非 public 时 403 + `Cache-Control: public, max-age=30` |
| `/users/:user/following` | GET | 403 | 同上 |
| `/users/:user/collections/featured` | GET | 403 | 置顶笔记；只含 public/home 且非 localOnly |
| `/users/:user/publickey` | GET | 403 | `type: Key`，`owner` 是 actor uri |
| `/notes/:note` | GET | 403 | 只返回 public/home 且非 localOnly；远程笔记 301 跳转 |
| `/notes/:note/activity` | GET | 403 | 与 outbox 条目一致的 Create/Announce |

- **内容协商**：`setResponseType` 在 `application/activity+json` 与
  `application/ld+json; profile="https://www.w3.org/ns/activitystreams"` 间选择，并 `Vary: Accept`。
  `/@:acct` 与 `/users/:id` 还带 `apOrHtml` 路由约束：`Accept` 为 html 时交给前端，否则走 AP。
- **缓存**：actor / collection / note 都带 `Cache-Control: public, max-age=180`；host-meta.json 等为 30/600 秒。
- **CORS**：每一条 AP 路由都设 `Access-Control-Allow-Origin: *`、`Allow-Headers: Accept`、`Allow-Methods: GET, OPTIONS`、`Expose-Headers: Vary`，且 `OPTIONS /.well-known/*` 直接 204。

## 2. WebFinger 与 host-meta（WellKnownServerService.ts）

- `GET /.well-known/webfinger?resource=...`，resource 接受三种形状：
  `acct:user@host`、`user@host`、`${url}/users/<id>` 与 `${url}/@<user>`。
  实现上先 `toLowerCase()` 再分类（源码 `generateQuery(request.query.resource.toLowerCase())`）。
- 查询结果非本域 host 时返回 **422**（不是 404）。
- 成功响应（JRD）：
  ```json
  {
    "subject": "acct:<username>@<host>",
    "links": [
      {"rel":"self","type":"application/activity+json","href":"<url>/users/<id>"},
      {"rel":"http://webfinger.net/rel/profile-page","type":"text/html","href":"<url>/@<username>"},
      {"rel":"http://ostatus.org/schema/1.0/subscribe","template":"<url>/authorize-follow?acct={uri}"}
    ]
  }
  ```
  `Content-Type: application/jrd+json`，`Cache-Control: public, max-age=180`，`Vary: Accept`；
  `Accept` 含 `application/xrd+xml` 时改出 XRD XML（本项目不实现 XML 变体，见 §9）。
- 缺 `resource` 参数 → 400；用户不存在 → 404；模式 `none` → 403。
- `GET /.well-known/host-meta`（XRD）与 `/.well-known/host-meta.json` 都返回 lrdd 模板
  `<url>/.well-known/webfinger?resource={uri}`。
- `/.well-known/nodeinfo` 返回 2.1 与 2.0 两个链接（见 §3）。

## 3. NodeInfo（NodeinfoServerService.ts）

- `/.well-known/nodeinfo`：
  ```json
  {"links":[{"rel":"http://nodeinfo.diaspora.software/ns/schema/2.1","href":"<url>/nodeinfo/2.1"},
            {"rel":"http://nodeinfo.diaspora.software/ns/schema/2.0","href":"<url>/nodeinfo/2.0"}]}
  ```
- `/nodeinfo/2.1` 与 `/nodeinfo/2.0` 同体，差别只有 `version` 字段与 2.0 会删掉 `software.repository`。
- `Content-Type: application/json; profile="http://nodeinfo.diaspora.software/ns/schema/2.1#"`，缓存 600 秒。
- 关键字段：`protocols: ["activitypub"]`、`openRegistrations`、`usage.localPosts`、
  `metadata.maxNoteTextLength`、`metadata.disableRegistration`、`software.name/version`。

## 4. HTTP Signatures（ApRequestService.ts）

- **出站 POST** 签名的头顺序固定：`(request-target) date host digest`。
  - `Digest: SHA-256=<base64(sha256(body))>`
  - `Date` 用 `toUTCString`（RFC 1123），`Content-Type: application/activity+json`
  - `Signature: keyId="...",algorithm="rsa-sha256",headers="(request-target) date host digest",signature="..."`
  - `keyId` 形状：`<url>/users/<id>#main-key`
- **出站 GET** 签名：`(request-target) date host`，无 digest。
- **入站校验顺序**（`ActivityPubServerService.inbox`），任何一步失败即刻拒绝：
  1. 模式 `none` → 403
  2. 解析 `Signature` 头失败 → 401
  3. 签名的 `headers` 不含 `host`，或 `Host` 头与配置 host 不符 → 401
  4. `headers` 不含 `digest` → 401；`Digest` 头格式非法 → 401
  5. digest 算法不是 `SHA-256` → 401；重算不匹配 → 401
  6. body 不是对象或没有 `actor` → 400
  7. 通过后立即回 **202 Accepted**，实际处理异步入队
- 解析器配置为 `headers: ['(request-target)', 'host', 'date'], authorizationHeaderName: 'signature'`，
  即也接受 `Authorization: Signature ...` 这种写法。

## 5. JSON-LD 上下文（misc/contexts.ts）

`@context` 是 `["https://www.w3.org/ns/activitystreams", "https://w3id.org/security/v1", <扩展对象>]`，
扩展对象里关键的键：

| 键 | 值 | 用途 |
|----|----|------|
| `Key` | `sec:Key` | 公钥对象类型 |
| `manuallyApprovesFollowers` | `as:manuallyApprovesFollowers` | 锁推 |
| `sensitive` | `as:sensitive` | 敏感媒体 |
| `Hashtag` | `as:Hashtag` | 标签 |
| `quoteUrl` | `as:quoteUrl` | 引用 |
| `toot` | `http://joinmastodon.org/ns#` | Mastodon 扩展前缀 |
| `Emoji`/`featured`/`discoverable` | `toot:*` | Mastodon 兼容 |
| `schema`/`PropertyValue`/`value` | `http://schema.org#` | 资料字段 |
| `misskey` | `https://misskey-hub.net/ns#` | Misskey 扩展前缀 |
| `_misskey_content`/`_misskey_quote`/`_misskey_reaction`/`_misskey_votes` | `misskey:*` | Misskey 私有扩展 |
| `_misskey_summary`/`_misskey_followedMessage`/`_misskey_license` | `misskey:*` | 同上 |
| `isCat` | `misskey:isCat` | 同上 |
| `vcard` | `http://www.w3.org/2006/vcard/ns#` | 生日/地址 |

## 6. Person 渲染（ApRendererService.renderPerson）

必须字段与形状：

```json
{
  "type": "Person | Service | Application",
  "id": "<url>/users/<id>",
  "inbox": "<id>/inbox", "outbox": "<id>/outbox",
  "followers": "<id>/followers", "following": "<id>/following",
  "featured": "<id>/collections/featured",
  "sharedInbox": "<url>/inbox", "endpoints": {"sharedInbox": "<url>/inbox"},
  "url": "<url>/@<username>",
  "preferredUsername": "<username>",
  "name": "...", "summary": "<HTML>", "_misskey_summary": "<MFM>",
  "icon": {"type":"Image","url":...}, "image": null,
  "tag": [], "manuallyApprovesFollowers": false, "discoverable": true,
  "publicKey": {"id":"<url>/users/<id>#main-key","owner":"<url>/users/<id>","publicKeyPem":"..."},
  "attachment": [{"type":"PropertyValue","name":"...","value":"..."}]
}
```
- `type` 规则：用户名含 `.` → `Application`；`isBot` → `Service`；否则 `Person`。
- `inbox` 等五个集合 URL 全部以 actor id 为前缀（即 `/users/<id>/...`），**不是**以 `/@username` 为前缀。
- `summary` 是 HTML，`_misskey_summary` 保留原始 MFM。

## 7. Note 渲染（ApRendererService.renderNote）

- `id`：本地笔记 `<url>/notes/<id>`，远程笔记用其 `uri`。
- `attributedTo`：作者的 actor uri。
- `content` 为 HTML；同时输出 `_misskey_content`（原始 MFM）与
  `source: {content, mediaType: "text/x.misskeymarkdown"}`。
- 可见性 → `to`/`cc`：

| visibility | to | cc |
|------------|----|----|
| public | `[.../activitystreams#Public]` | `[<actor>/followers, ...mentions]` |
| home | `[<actor>/followers]` | `[...#Public, ...mentions]` |
| followers | `[<actor>/followers]` | `...mentions` |
| specified | `...mentions` | `[]` |

- `tag` = 标签 Hashtag + 被提及用户 Mention + 自定义 emoji Emoji。
- `summary` 仅在 CW 存在时出现；CW 为空串时输出 U+200B。
- 引用笔记另加 `quoteUrl` / `_misskey_quote`，并在 HTML 末尾附 `<span class="quote-inline">`。
- `inReplyTo`：远程或被回复对象的 uri 直接引用；本地则嵌入对象（`dive`）。

## 8. Inbox 活动处理（ApInboxService.ts）

- 顶层按 `type` 分派：`Create`、`Delete`、`Update`、`Follow`、`Accept`、`Reject`、`Add`、`Remove`、
  `Announce`、`Like`、`Undo`、`Block`、`Flag`、`Move`；`Collection`/`OrderedCollection` 逐条处理。
- 每条的返回值是 `ok` 或以 `skip:` 开头的说明串（用于日志与管理员可见的诊断），**不影响 HTTP 状态码**。
- 幂等点：
  - `Create`：按笔记 uri 查重，已存在直接 `skip: note exists`。
  - `Create`：`actor.uri !== note.attributedTo` 拒绝；actor 与笔记 id 的 host 必须一致。
  - `Announce`：按 activity uri 查重。
  - `Like`：重复反应视为 `skip: already reacted`。
- `Follow`：`object` 必须是**本地**用户，否则 skip；通过 `userFollowingService.follow(actor, followee, {requestId})`
  处理，锁推用户进 `follow_requests`。
- `Undo`：支持 `Follow`/`Block`/`Like`/`Announce`/`Accept` 五种内层对象。
- `Delete`：`object` 为 Tombstone 时取 `formerType`；type 不明且 `actor.uri === object` 视为 `Person`，否则 `Note`。
- `Update`：内层是 Actor 则更新资料，是 `Question` 则更新投票。

## 9. 本项目的实现选择与偏离

| 项 | 上游 | 本项目 | 理由 |
|----|------|--------|------|
| XRD（`application/xrd+xml`）WebFinger | 支持 | **不实现** | 现代联邦软件全部用 JRD；Mastodon、GoToSocial、Sharkey 都不请求 XML。记入已知缺口。 |
| JSON-LD 规范化签名（`signatureValue` 的 LinkedDataSignature） | 支持 | **不实现** | 依赖完整的 jsonld 规范化实现；Mastodon 与 GoToSocial 也只用 HTTP Signatures。 |
| Inbox 异步入队 | 立即 202 + 队列 | **同步处理 + 202** | 单机个人实例不需要额外队列；慢速远程在客户端超时前完成。若某活动处理失败仍返回 202，避免远端无限重试。 |
| Relay | 支持 | **不做** | 项目已定案首版不做 Relay。 |
| 活动字段 `_misskey_*` | 全量 | 输出 `_misskey_content`、`_misskey_quote`、`_misskey_reaction`、`_misskey_summary` | 覆盖 Misskey 客户端最常用的四类；其余随功能落地再补。 |
| `/notes/:id/activity` | 有 | 有 | 便宜，直接实现。 |
| `/@:acct` 的 HTML 分支 | 交给前端 | **不实现 html 分支** | M6 前端尚未开始；一律返回 AP 表示。 |

## 10. 联邦三模式的闸门位置

按架构文档 §5.3，模式判断集中在 `federation.Policy`，闸门在四处：

1. **入站 WebFinger / Actor / Note GET**：`none` → 403。
2. **入站 Inbox POST**：`none` → 403；`allowlist` 且来源 host 不在白名单 → 403。
3. **出站投递**：`none` → 直接丢弃；`allowlist` 且目标不在白名单 → 丢弃并记日志。
4. **远程解析（WebFinger 出站查询、signed GET）**：`none` 时不发起；`allowlist` 时只查白名单域。

白名单匹配沿用上游 `isFederationAllowedHost` 的后缀语义：
`.${host}`.endsWith(`.${allowed}`)，即 `allowed = "example.com"` 时匹配
`example.com` 与 `sub.example.com`，但不匹配 `notexample.com`。
