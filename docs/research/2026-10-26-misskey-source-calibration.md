# 源码校准：以 Misskey 真实实现为契约基准

- 记录日期：2026-10-26
- 起因：用户指出"其实可以 pull 源码来抄的"。此前调研只依赖官方文档，源码可得时应以源码为准。
- 参考副本：`misskey-dev/misskey`（AGPL-3.0-only），本地浅克隆置于 `/tmp/refcheck/misskey`（不入库）。

## 1. 许可证约束（关键）

| 项目 | 许可证 | 可借鉴性 |
|------|--------|---------|
| misskey-dev/misskey | **AGPL-3.0-only** | 可读、可参考契约；**逐行翻译其源码到 Go 会使本项目受 AGPL 传染**，需评估 |
| Firefish（`jclaudan/firefish`，2 star 残留镜像） | AGPL-3.0 | 主仓库已删除，仅余归档 |
| YunoHost-Apps/firefish_ynh | AGPL-3.0 | 仅打包脚本 |
| iceshrimp/Iceshrimp.NET | （仓库已不可访问） | 不可用 |
| yitsushi/go-misskey（Go SDK） | GPL-3.0，**已归档** | 客户端 SDK，可读类型定义 |

**结论**：可以 pull 源码来"抄契约"（读实现、照行为、复刻端点语义），这是加速正确性的关键手段。但逐行翻译 Misskey 的 TypeScript 实现到 Go 属于派生作品，会触发 AGPL-3.0 的开源义务。本项目目前的定位是"个人自用实例"，若未来公开分发，需自行评估是否改为 AGPL-3.0 或保持"读实现、自写实现"的边界。

实践准则（写入 AGENTS.md）：**读源码确定行为与字段，不做逐行翻译**；涉及数据结构的语义可照搬（表结构、枚举值、字段名属于事实性契约），涉及表达式的实现自行用 Go 惯用法重写。

## 2. GitHub 上不存在 Go 版 Misskey 服务器

搜索 `misskey language:go` 全部为客户端 SDK 或周边工具（`yitsushi/go-misskey`、`mikuta0407/misskey-cli` 等），无 Go 语言的服务端实现。朋友的项目"Miuno"未发布、仓库不可搜索到。因此 Go 路线没有可抄的同类实现，只能以 Misskey 的 TS 源码为行为基准。

## 3. `/api/meta` 真实契约（校正 M1 实现）

来源：`packages/backend/src/core/entities/MetaEntityService.ts`（pack / packDetailed）。

`meta` 有 `detail` 参数（默认 true）。lite 与 detailed 两套；detailed 额外含：`cacheRemoteFiles`、`cacheRemoteSensitiveFiles`、`requireSetup`、`proxyAccountName`、`features`。

真实 lite 字段（必须实现）：
`maintainerName` `maintainerEmail` `version` `providesTarball` `name` `shortName` `uri` `description` `langs` `tosUrl` `repositoryUrl` `feedbackUrl` `impressumUrl` `privacyPolicyUrl` `inquiryUrl` `disableRegistration` `emailRequiredForSignup` `enableHcaptcha` `hcaptchaSiteKey` `enableMcaptcha` `mcaptchaSiteKey` `mcaptchaInstanceUrl` `enableRecaptcha` `recaptchaSiteKey` `enableTurnstile` `turnstileSiteKey` `enableTestcaptcha` `googleAnalyticsMeasurementId` `swPublickey` `themeColor` `mascotImageUrl` `bannerUrl` `infoImageUrl` `serverErrorImageUrl` `notFoundImageUrl` `iconUrl` `backgroundImageUrl` `logoImageUrl` `maxNoteTextLength` `defaultLightTheme` `defaultDarkTheme` `clientOptions` `ads` `notesPerOneAd` `enableEmail` `enableServiceWorker` `translatorAvailable` `serverRules` `policies` `sentryForFrontend` `mediaProxy` `enableUrlPreview` `noteSearchableScope` `maxFileSize` `federation`

真实 `features`（detailed 才返回）：`localTimeline` `globalTimeline` `registration` `emailRequiredForSignup` `hcaptcha` `recaptcha` `turnstile` `objectStorage` `serviceWorker` `miauth`

**M1 实现的偏差**（已修正）：缺 `uri`/`shortName`/`description`/`langs`/`tosUrl`/`inquiryUrl`/`iconUrl`/`backgroundImageUrl`/`logoImageUrl`/`defaultLightTheme`/`defaultDarkTheme`/`clientOptions`/`enableUrlPreview`/`noteSearchableScope`/`maxFileSize`/`federation`/`cacheRemoteFiles`/`requireSetup`/`proxyAccountName`；`features` 结构写错（用了 `emailRequiredForSignup`/`localTimeline` 混排，且缺 `hcaptcha`/`recaptcha`/`turnstile`/`objectStorage`/`serviceWorker`/`miauth`）；多写了 `providesTranslationKeys`/`translatorAvailable` 语义偏差等。

关键点：`uri` 是实例 URL（`config.url`），不是 `instanceUrl`——**Misskey 用 `uri`**，M1 写成了 `instanceUrl`，属实质错误。

`policies` 来自 `DEFAULT_POLICIES`（RoleService），含 `canUseFederation` 等实例级策略项。

## 4. 错误响应体真实形状

来源：`packages/backend/src/server/api/ApiCallService.ts` 的 `send()` 与 `error.ts`。

```json
{"error":{"message":"...","code":"...","id":"uuid","kind":"client|server|permission"}}
```

- `kind` **必填**（默认 `client`），M1 实现漏了。
- `info` 字段仅在 `err.info` 存在时附加。
- HTTP 状态码：client 类默认 400；permission 类默认 403；401 时带 `WWW-Authenticate: Bearer realm="Misskey"`。
- 校验失败统一为 `INVALID_PARAM`（不是 `INVALID_JSON`），错误 id 固定 `3d81ceae-475f-4600-b2a8-2bc116157532`。
- 认证失败为 `AUTHENTICATION_FAILED`，id `b0a7f5f8-dc2f-4171-b91f-de88ad238e14`，HTTP 401。
- 内部错误 `INTERNAL_ERROR`，id `5d37dbcb-891e-41ca-a3d6-e690c97775ac`，HTTP 500。
- `id` 是 UUID 格式（用于追踪），每个端点的错误 id 固定。

**M1 实现的偏差**：`ping` 的 JSON 解析失败返回了 `INVALID_JSON`（应为 `INVALID_PARAM`）；错误体缺 `kind`。

## 5. `/api/ping` 真实行为

`ping.ts`：无参数（`paramDef.properties` 为空），返回 `{ pong: Date.now() }`——**是服务端当前时间戳，不是回显客户端传入值**。M1 实现回显了客户端 `pong`，且接受参数，与真实契约不符。响应 schema 要求 `pong` 为 number。

## 6. 联邦模式：Misskey 的原生三态可直接映射

`packages/backend/src/models/Meta.ts`：`federation: 'all' | 'specified' | 'none'`，配合 `federationHosts: string[]` 白名单。

判定逻辑（`packages/backend/src/core/UtilityService.ts`）：

```ts
if (meta.federation === 'none') return false;
if (meta.federation === 'specified' && !meta.federationHosts.some(x => `.${host.toLowerCase()}`.endsWith(`.${x}`))) return false;
if (isBlockedHost(meta.blockedHosts, host)) return false;
return true;
```

- 三态与本项目要求的"公开互联 / 白名单 / 完全不互联"**完全对应**（all / specified / none），无需自定义映射。
- 白名单匹配为**后缀匹配**且两侧加点：`.${host}`.endsWith(`.${x}`)。含义是白名单项 `example.com` 匹配 `example.com` 与 `sub.example.com`，但**不匹配** `notexample.com`（因为 `notexample.com` 变 `.notexample.com`，不以 `.example.com` 结尾）。这是安全的关键细节。
- 另有独立的 `blockedHosts` 黑名单，优先级在白名单之后。
- 另有 `silencedHosts` 静默列表（只影响展示，不阻断通信）。

**六个拦截点**（`isFederationAllowedUri` / `isFederationAllowedHost` 调用处）：

| 位置 | 文件 | 作用 |
|------|------|------|
| AP 对象展示 | `server/api/endpoints/ap/show.ts:135` | 阻止获取被禁主机的远程对象 |
| Inbox 收件 | `queue/processors/InboxProcessorService.ts:78`、`:209` | 拒绝入站活动 |
| 出站投递 | `queue/processors/DeliverProcessorService.ts:56` | 阻止发往被禁主机 |
| 笔记解析 | `core/activitypub/models/ApNoteService.ts:358` | 阻止解析被禁主机的笔记 |
| 远程解析 | `core/activitypub/ApResolverService.ts:126` | 阻止解析被禁主机 |
| Inbox 活动处理 | `core/activitypub/ApInboxService.ts:314` | 二次拦截 |

这验证了本项目设计文档"策略必须贯穿发现、收取、发送、远程媒体和对象解析，不能只在发送队列处拦截"的判断——与上游实现一致。`meta.federation` 字段直接对外暴露（`/api/meta` 返回 `federation`），说明客户端可感知实例联邦状态。

## 7. 顺带发现的安全机制：Activity 与 URL 一致性

`packages/backend/src/core/activitypub/misc/check-against-url.ts`（源自 Sharkey 贡献）实现 `assertActivityMatchesUrl`：校验活动 `id` 与请求 URL、最终 URL 的主机一致性，防止跨源伪造与联邦回路攻击。支持 `NonCanonicalId` / `MisalignedOrigin` / `CrossOrigin` 三级 soft-fail 掩码。本项目实现 ActivityPub 收件时必须包含等价检查，否则会成为 SSRF/伪造漏洞入口。

## 8. 后续工作方式

- 每个要实现的 Misskey 端点，先读 `packages/backend/src/server/api/endpoints/<name>.ts` 的 `paramDef`（参数 schema）与 `res`（响应 schema），再读 `packages/misskey-js/src/api.types.ts` 的类型定义，据此写 Go 端实现与契约测试。
- 端点全集在 `packages/backend/src/server/api/endpoints/`（61 个子目录，含 admin/charts/drive/auth 等）。
- `paramDef` 是 JSON Schema，可用 ajv 语义对应到 Go 侧校验；`misskey:id` 格式为 `/^[a-zA-Z0-9]+$/`。
