# Firefish 前端移植：运行时适配契约（2026-11-01）

用途：固定 Firefish 客户端依赖的每一个运行时成员在 Cinderfish 里的归属与处理方式（`shim` / `rewrite` / `drop`），
使移植任务不需要每次重新决定「这个 `os.xxx` 我们自己写还是删」。

配套文档：`2026-11-01-firefish-port-source-and-license-register.md`（来源与许可证）、
`2026-11-01-firefish-port-capability-matrix.md`（页面到端点）。
参考基线：`tmp/refcheck/firefish@02c31e0:packages/client/src`。

## 1. 三种处理方式

| 决策 | 定义 | 判定条件 |
|------|------|----------|
| **shim** | 在 Cinderfish 提供同名同签名的实现，移植过来的组件**不改调用点** | 调用点数量大、语义与后端无关、或本身是纯 UI 机制 |
| **rewrite** | 改名或改结构，调用点跟着改 | Firefish 的实现绑定了它的路由/状态/后端契约 |
| **drop** | 不提供，相关 UI 一并移除 | 依赖的能力不在白名单内（见能力矩阵 §5），或根本不是本项目目标 |

实测调用点数量（`grep -rho "os.<成员>(" packages/client/src`）：

| 成员 | 调用点 | 成员 | 调用点 |
|------|--------|------|--------|
| `api` | 329 | `inputText` | 51 |
| `apiWithDialog` | 108 | `popupMenu` | 36 |
| `alert` | 109 | `pageWindow` | 26 |
| `confirm` | 56 | `select` | 16 |
| `popup` | 56 | `post` | 14 |
| `contextMenu` | 13 | `claimZIndex` | 13 |
| `form` | 11 | `selectUser` | 9 |
| `apiGet` | 8 | `yesno` | 3 |
| `selectLocalUser` | 3 | `selectDriveFile` | 3 |
| `selectDriveFolder` | 2 | `openEmojiPicker` | 2 |
| `selectInstance` | 1 | `modalPageWindow` | 1 |
| `toast` | 0 | | |

其他规模量级（同一份源码，命令见下）：

```bash
FF=tmp/refcheck/firefish/packages/client/src
grep -rho "i18n\.ts\.[A-Za-z]" $FF | wc -l                      # 1852 处属性访问
grep -rhoE "(^|[^a-zA-Z0-9_$.])me\.[a-zA-Z]" $FF | wc -l         # 112
grep -rhoE "me!\.[a-zA-Z]" $FF | wc -l                           # 39
grep -rho "me?\." $FF | wc -l                                    # 52
grep -rhoE 'ph-[a-z0-9-]+' --include=*.vue --include=*.ts $FF | sort -u | wc -l   # 251
```

注意 `me` 的统计必须带左边界：直接 `grep "me\."` 会把 `name.`、`theme.`、`time.` 一起算进去（得到 300+），
这是本契约第一版写错的地方。除 `i18n` 的 1852 处外，`me.` / `me!.` / `me?.` 三项合计 **203** 处真实属性访问，
仍然足够多，所以 `me` 仍按 shim 处理。
这些数字决定了下面的取舍：调用点越多，越应该 shim 而不是改调用点。

## 2. 总览

| Firefish 成员 | 定义位置 | Cinderfish 归属 | 决策 |
|---|---|---|---|
| `api` | `os.ts:25-71` | `web/src/os.ts` 导出 `api`，委托 `web/src/api/client.ts` 新增的通用 `request()` | **shim** |
| `apiGet` | `os.ts:72-115` | —（Go 没有 GET API 语义） | **rewrite** → `api` |
| `pendingApiRequestsCount` | `os.ts:19` | `web/src/api/client.ts` 的 `ref` | **shim** |
| `apiWithDialog` | `os.ts:117-131` | `api` + 错误提示 | **shim** |
| `alert` | `os.ts:300-321` | 新建 `components/ui/DialogHost.vue` | **shim**（组件需新建） |
| `confirm` | `os.ts:323-346` | 新建 DialogHost，内部接 `os.ts:86 confirmDialog` | **shim** |
| `yesno` | `os.ts:348-374` | DialogHost 的「是/否」变体 | **shim** |
| `inputText` | `os.ts:376-…` | 新建 DialogHost 的输入模式 | **shim** |
| `select` | `os.ts:563-659` | DialogHost 的单选模式 | **shim** |
| `form` | `os.ts:661-681` | — | **drop**（表单 DSL，见 §3.15） |
| `selectUser` / `selectLocalUser` | `os.ts:683-720` | — | **drop**（依赖不存在的用户搜索） |
| `selectInstance` | `os.ts:721-738` | — | **drop**（联邦实例浏览是隐藏功能） |
| `selectDriveFile` / `selectDriveFolder` | `os.ts:740-844` | — | **drop**（Drive 文件夹整体移除） |
| `openEmojiPicker` | `os.ts:846-912` | `components/ui/EmojiPicker.vue` + `os.ts` 的 `openReactionPicker` | **rewrite** |
| `popup` | `os.ts:224-257` | `OsOverlay` 扩成通用层栈 | **rewrite**（保留签名） |
| `pageWindow` / `modalPageWindow` | `os.ts:259-287` | `components/ui/PageWindow.vue` + `components/deck/deck-page.ts` | **rewrite**（见 §3.15） |
| `popupMenu` | `os.ts:914-956` | 新建 `components/ui/PopupMenu.vue` | **shim** |
| `contextMenu` | `os.ts:957-985` | popupMenu + 光标定位 | **shim** |
| `post` | `os.ts:986-1030` | `web/src/os.ts:46 openPostForm` | **shim**（补 onClosed） |
| `toast` | `os.ts:289-298` | — | **drop**（0 调用点） |
| `claimZIndex` | `os.ts:196-204` | 新建 `web/src/lib/z-index.ts` | **shim** |
| `me` / `isSignedIn` / `isAdmin` | `me.ts`、`account.ts` | `web/src/stores/session.ts` | **shim**（reactive 代理） |
| `i18n` | `i18n.ts` | 新建 `web/src/i18n/`（类 + zh-CN 词典） | **shim** |
| `instance` / `getInstanceInfo()` | `instance.ts` | `web/src/stores/instance.ts` | **shim** |
| 路由 | `router.ts`、`nirax.ts` | `web/src/router/index.ts`（vue-router 4） | **rewrite** |
| 图标 | `<i class="ph-house">`（Phosphor webfont，全客户端 251 个不同类） | `web/src/components/AppIcon.vue`（28 个路径） | **rewrite** + 映射表 |
| 流式 | `stream.ts`（`firefish-js` 的 `Stream`） | `web/src/stores/streaming.ts` | **rewrite** |
| Service Worker | `packages/sw/src/sw.ts` | — | **drop** |
| 多账户 | `account.ts:53-66`、`pages/settings/accounts.vue` | — | **drop** |
| 插件 / AiScript | `plugin.ts`、`cold-store.ts`、`widgets/aiscript.vue` | — | **drop** |
| `pizzax` 状态同步 | `pizzax.ts`（走 `i/registry/get-all` + `registryUpdated`） | `stores/*` 的 localStorage 持久化 | **rewrite** |
| 主题 DSL | `theme-store.ts` + `themes/*.json5` | `stores/theme.ts` + CSS token | **drop**（语义映射，见能力矩阵 §3.11） |

## 3. 逐项契约

### 3.1 `api`

Firefish（`os.ts:25-77`）：

```ts
api(endpoint: string, data = {}, token?: string | null, useToken = true): Promise<T>
```

行为：`POST ${apiUrl}/${endpoint}`，`Content-Type: application/json`，`credentials: "omit"`、`cache: "no-cache"`；
`endpoint` 含 `://` 时当绝对 URL；请求头带 `authorization: Bearer <token ?? me.token>`（`useToken` 为 false 时不带）；
`200` 解析 JSON，`204` 解析为 `undefined`，其它状态 reject `body.error`（Misskey 错误信封）。

Cinderfish 现状（`web/src/api/client.ts:133-152`）：内部 `post(path, body, sendToken)` 同时发送
`i` 字段与 `Authorization: Bearer`，非 2xx 抛 `ApiError`（含 `message`/`code`/`id`/`kind`/`status`），并对响应做
两种 viewer 处理：`/api/i` 等记 viewer，其它响应按 `userId === viewer.id` 覆盖作者名与头像（`stampViewer`，
`:104-131`）。

契约：

- 新增 `request<T>(endpoint: string, data: Record<string, unknown> = {}, opts?: { token?: string | null; useToken?: boolean })`，
  复用 `post()` 的 token、信封与 stamping 逻辑；`useToken: false` 时不发送 `i` 与 header。
- `endpoint` 允许绝对 URL（Firefish 有跨实例调用点），绝对 URL 时跳过 stamping。
- `204` 返回 `undefined`（保持 Firefish 语义，避免调用点判空逻辑失效）。
- 错误统一为 `ApiError`；`apiWithDialog` 的文案依赖 `message` 与 `id` 两个字段。
- `pendingApiRequestsCount` 用 `ref<number>` 实现，供全局 loading 指示器使用。

验收：`api("i")`、`api("notes/show", {noteId})`、`api("meta", {detail: true})`（无 token）三个调用在未登录与登录态都返回与
`api.i()` / `api.showNote()` 一致的对象。

### 3.2 `apiGet` → `rewrite`

Firefish 有 8 处 `apiGet`（`os.ts:80-115`），走 `GET ${apiUrl}/${endpoint}?<query>`。
本项目 `internal/httpapi/server.go` 只注册了两条 GET API 路由（`/api/ping`、`/api/admin/federation/allowlist`），
其余全部是 RPC 风格 POST。

决策：**不实现 `apiGet`**。移植时把这 8 处改成 `api`（POST），并在迁移记录里列出位置；
不为 Firefish 引入第二套 HTTP 语义（设计文档 §4 反目标：不维护两套 API 面）。

### 3.3 `apiWithDialog`

Firefish（`os.ts:117-131`）：包一层 `api`，失败时 `alert({type:"error", text: `${err.message}\n${err.id}`})`，
并用 `promiseDialog` 显示等待层。

契约：**shim**。等待层用 `pendingApiRequestsCount` 的全局面板，失败提示复用 `alert`；
文本拼接保持 `message + "\n" + id`（108 个调用点依赖「失败可见」而不是静默）。

### 3.4 `alert` / `confirm` / `inputText`

| 成员 | Firefish 签名 | 返回 | 现成实现 |
|---|---|---|---|
| `alert` | `os.ts:300-321` `{type, title?, text?, isPlaintext?}`，`text` 为空且 `type==="error"` 时填 `i18n.ts.somethingHappened` | `Promise<void>` | 无 |
| `confirm` | `os.ts:323-350` 同上 + `okText`/`cancelText`/`showCancelButton` | `Promise<{canceled: boolean}>` | `web/src/os.ts:86 confirmDialog(message): Promise<boolean>` |
| `inputText` | `os.ts:376-430` `{type, title?, text?, placeholder?, default?, minLength?, maxLength?}` | `Promise<{canceled:true} \| {canceled:false, result:string}>` | 无 |

契约：新建 `components/ui/DialogHost.vue`（模态 + 遮罩 + Esc + 焦点陷阱），提供三种模式；
`confirm` 内部接到既有的 `confirmDialog`，把 `boolean` 转成 `{canceled}`——**不改 `confirmDialog` 的既有签名**，
它是本项目 `components/` 里已在用的 API。

注意：Firefish 的 `confirm` 返回对象而不是布尔，移植组件里大量写 `if (!({canceled} = await os.confirm(…)).canceled)`；
`alert` 没有返回值依赖，但**必须等用户关闭后再继续**（109 个调用点里有一部分在 `await` 之后清状态）。
`inputText` 的 `canceled`  discriminated union 要原样保留，否则 `result` 的类型收窄会失效。

### 3.5 `popup` → `rewrite`（保留签名）

Firefish（`os.ts:224-272`）：

```ts
popup<Props, Emits>(component, props: CanUseRef<Props>, events = {}, disposeEvent?): Promise<{ dispose: () => void }>
```

它是 `alert`/`confirm`/`popupMenu`/`toast` 的底层：把一个组件挂到全局 `popups` 数组上，按 `disposeEvent` 关闭，
`claimZIndex` 决定层级。56 个直接调用点。

本项目现状：`web/src/os.ts` 只有**具名命令**（`openPostForm` / `openReactionPicker` / `confirmDialog`）+
一个 `OsOverlay` 组件（`:194`），没有通用层。

决策：**rewrite**——在 `os.ts` 里加通用层栈（`openLayer`/`closeLayer`），签名与 Firefish 保持一致
（返回 `Promise<{dispose}>`），`popup` 作为它的薄包装。理由：56 个调用点不可能逐个换成具名命令，
而「通用层栈」本身就是 `OsOverlay` 的自然扩展，不需要引入 Firefish 的 `popups` 全局数组结构。

约束：层栈必须支持键盘 Esc、点击遮罩、`disposeEvent`、以及 `claimZIndex` 的层级；
不允许组件直接操作 `document.body`（本项目已有 `PageWindow.vue` 的弹窗语义可复用）。

### 3.6 `popupMenu` / `contextMenu`

Firefish：`os.ts:914-956`（`popupMenu(items, src?, options?)`）与 `os.ts:957-985`（`contextMenu(items, ev)`，
内部 `ev.preventDefault()`）。`MenuItem` 类型来自 `types/menu.ts`（`label` / `button` / `a` / `link` / `user` /
`switch` / `radio` / `parent`）。

决策：**shim**。新建 `components/ui/PopupMenu.vue` + `contextMenu` 包装（光标定位 + `preventDefault`）。
`MenuItem` 类型**照抄 Firefish 的判别联合**（它描述的是 UI，不是后端契约），放在 `web/src/types/menu.ts`；
`switch`/`radio` 项要回写 `ref`，这是 Firefish 菜单在设置页里的主要用法。

### 3.7 `post`

Firefish：`os.ts:986-1030`，`post(props: MkPostFormDialog["$props"] = {}, onClosed?)`，返回 `Promise<void>`，
关闭后 resolve。

本项目：`web/src/os.ts:46 openPostForm(opts: {reply?, renote?}): void`（未登录会跳登录页）。

决策：**shim**。`post()` 把 Firefish 的 props 子集映射过去：`reply`、`renote`、`initialText`、`initialCw`、
`initialVisibility`、`initialFiles`、`instant`；其余 props（投票、引用、定时、插件钩子）丢弃。
`openPostForm` 需要新增可选的 `onClosed` 回调，让 `post()` 能返回 Promise；这是本 shim 对现有代码唯一的签名扩展。

### 3.8 `claimZIndex`

Firefish（`os.ts:196-204`）：`low/middle/high` 三档，初值 1000000/2000000/3000000，每次 +100。

决策：**shim**，放 `web/src/lib/z-index.ts`，常量与步进与 Firefish 保持一致（本项目样式表里的固定层级不受影响，
只有移植过来的浮层用它）。不要把它接到 CSS 变量体系里：数值语义是「插入顺序」，不是主题 token。

### 3.9 `me`

Firefish：`me.ts` 顶层 `reactive(JSON.parse(localStorage.account))`，`isSignedIn` / `isModerator` / `isEmojiMod` /
`isAdmin` 派生；`account.ts` 的 `updateAccount()` 直接改这个对象，多账户用 `getAccounts/addAccount/removeAccount`。

本项目：`web/src/stores/session.ts:8` 的 `session`（`reactive`）+ `isLoggedIn`/`isAdmin`（`:15-16`）+ `refreshSession()`。

决策：**shim**（`me.` 112 + `me!.` 39 + `me?.` 52 = 203 处属性访问，绝不能靠批量改名）。

契约：

- 新建 `web/src/me.ts`：`export const me` 是 `reactive` 对象，字段由 `session.me` 同步过来（`watch` 或
  `Object.assign` 在 `refreshSession`/`signin`/`signout` 之后），**truthy 判断**保持 Firefish 语义：
  未登录时为 `null`。
- `isSignedIn(me)`、`isAdmin`、`isModerator` 从 `session` 派生。
- **单账户**：`addAccount`/`removeAccount`/`getAccounts` 不实现，账户切换菜单与 `pages/settings/accounts.vue` 一并 drop。
- 唯一真值仍是 `session.me`；`me` 是投影，任何写入必须走 `session`，禁止在 `me` 上就地修改
  （否则会出现两份不同步的当前用户，正是设计文档 §4 反目标里的「页面功能分裂」）。
- `updateAccount(patch)` 保留为 shim，内部调用 `session` 的更新路径。

### 3.10 `i18n`

Firefish：`i18n.ts` 是一个 40 行的 `I18n` 类（`t(key, args)` 支持点分路径与 `{arg}` 替换，缺失时
`console.warn` 并回退 key），实例导出的 `i18n.ts` 同时被当作**词典对象**直接属性访问（`i18n.ts.signinFailed`）。

数量：1852 处 `i18n.ts.<key>`、其余为 `i18n.t(...)`。

本项目：`web/src/` 目前没有 i18n 层（文案为中文直写）。

决策：**shim**，但词典另起。

- 新建 `web/src/i18n/index.ts`：照抄 `I18n` 类（generic、与后端无关，AGPL 允许，文件头保留来源注释）。
- 新建 `web/src/i18n/zh-CN.ts`：**只收录白名单页面用到的 key**，按 Firefish 的 key 命名（`signinFailed`、
  `note.` 前缀等）原样保留，避免移植组件里的 `i18n.ts.x` 失效。
- 缺失 key 的行为保持「回退为 key 本身 + warn」，让漏翻在最早期暴露，而不是静默显示英文。
- 不引入 Firefish 的 30+ 语言包；语言切换在首批只有 zh-CN（Firefish 的 `general.lang` 设置项随之隐藏）。

验收：`grep -c "i18n.ts." ` 在移植后的白名单页面里全部能解析到 zh-CN 词典中的真实条目（无 warn）。

### 3.11 `instance`

Firefish（`instance.ts`）：`getInstanceInfo()` 返回一个**完整的** `DetailedInstanceMetadata` 形状（含
`maxNoteTextLength`、`disableRegistration`、`emojiCategories` 等），启动时从 idb-keyval 读缓存，再 `api("meta", {detail:true})` 刷新。

本项目：`web/src/stores/instance.ts` 的 `instance`（`reactive`）+ `loadInstance()`/`reloadInstance()`，
数据来自 `POST /api/meta`（`internal/httpapi/meta.go`，已含 `disableRegistration`、`emailRequiredForSignup`、
`maxNoteTextLength`；**不含 `emojis`**）。

决策：**shim**。

- 新建 `web/src/instance.ts`：以 `stores/instance.ts` 为源，按 Firefish 的字段名导出（`getInstanceInfo()`），
  缺字段给**明确的空值**（`emojis: []`、`features.serviceWorker: false`），不伪造能力。
  `emojis: []` 是投影层补的空值，不是 `meta` 的真实字段——两者不要混写。
- 缓存从 `idb-keyval` 改为本项目已有机制（`stores/instance.ts`），不新增 IndexedDB 依赖。
- `emojiCategories`/`emojiTags` 返回空数组（`POST /api/meta` **没有 `emojis` 字段**，见能力矩阵 §11；`users_pack.go` 里的
  `emojis: []` 是 User 的字段，不是实例目录）；emoji picker 因此退化为 Unicode。这是字段缺失，不要写成「返回空数组」。

### 3.12 路由（`rewrite`）

Firefish 自带 `nirax`（`nirax.ts` + `router.ts`）：`page(() => import(...))`、`definePage`、`router.push/replace`、
`useRouter()`、滚动位置恢复（`router.ts:708-736` 自己操作 `history`）。

本项目：vue-router 4（`web/src/router/index.ts`），路由名与 `views/` 一一对应，守卫在 `:133-138`。

决策：**rewrite**。规则：

| Firefish | Cinderfish |
|---|---|
| `router.push(path)` / `replace(path)` | 同名 vue-router 方法（签名兼容，直接改名 import） |
| `useRouter()` | `useRouter()` from `vue-router` |
| `page(() => import("./pages/x.vue"))` | 路由记录里的 `component: () => import(…)` |
| `definePage({ … })` / `transition` / `keepAlive` 元信息 | 忽略；需要时写成 `meta` |
| `nirax` 的 `Router` 接口（`current`, `events`） | 不提供；用到 `router.current`/事件总线的组件改写 |
| 滚动位置恢复 | 保留本项目现有行为，不移植 Firefish 的 `history.state.key` 方案 |

路由路径保持本项目的 `/`、`/notes/:id`、`/@:handle`、`/notifications`、`/settings/*`；
Firefish 的 `/my/*`、`/user/:acct`、`/share`、`/miauth/:session` 等不注册（见能力矩阵 §5）。

### 3.13 图标（`rewrite` + 映射表）

Firefish 用 Phosphor webfont 类名（`<i class="ph-house">`），客户端共 251 个不同的 `ph-*` 类（全 `src` 去重：
`grep -rhoE 'ph-[a-z0-9-]+' --include=*.vue --include=*.ts … | sort -u | wc -l`）；
本项目 `components/AppIcon.vue` 是**手写 SVG 路径**（`paths` 里 28 个语义名，Firefish/Misskey 风格描边），
不依赖图标字体——这是要保留的视觉与体积决策。

决策：**rewrite**。移植任务交付一张 `ph-* → AppIcon name` 映射表，并扩展 `AppIcon.vue` 的 `paths` 到覆盖表中全部
语义名（多个 `ph-*` 映射到同一语义名，例如 `ph-heart` 与 `ph-heart-fill` 都映射 `heart`）。
**不引入 `@phosphor-icons/web`**：新增依赖、改变视觉语言，且与「图标系统保持自绘」的既有决定冲突。

验收：移植后的组件里不存在 `class="ph-`；`AppIcon` 的 `name` 类型仍是字面量联合（拼错在 typecheck 阶段报错）。

### 3.14 流式（`rewrite`）

Firefish（`stream.ts`）：`firefish-js` 的 `Stream`，`useChannel(channel, params?, name?)` 返回带 `on/off/dispose/send`
的连接对象；心跳 60s 发 `ping`。

本项目（`web/src/stores/streaming.ts:1-13`）：单 WebSocket 共享，`subscribe(channel): token` 引用计数，
`emit(channel, type, body)` 路由到订阅者，`main` 频道由 store 持有。

决策：**rewrite**——新建一个 `useChannel` 形状的适配器（返回 `{on, off, dispose}`），内部用 `subscribe`/`release`；
`send` 只在频道需要时提供。参数对象（`{listId}`、`{channelId}` 等）对应本项目不支持的频道，一律 drop。
频道与事件的可用性见能力矩阵 §6。

### 3.15 对话框族与页面窗口（brief 的 15 个成员之外，但首批页面会在运行期调用）

brief 只点名了 15 个成员，但 Firefish 的对话框/选择器是同一个 `popup` 家族的多个入口，首批页面实际会调到它们。
不给决策，移植时就会各写一套：

| 成员 | 签名要点（`os.ts`） | 调用点 | 决策 | 理由 |
|---|---|---|---|---|
| `yesno` | `{type,title?,text?,isPlaintext?}` → `Promise<{canceled}>` | `:348-374` | **shim** | 与 `confirm` 同构，只是按钮文案不同；复用 DialogHost |
| `select` | `{title?,text?,default?,items:[{value,text}] \| groupedItems}` → `Promise<{canceled:true} \| {canceled:false,result:C}>` | `:563-659` | **shim** | 单选弹窗；DialogHost 加一个列表模式 |
| `form` | `(title, form: Form)` → `Promise<{result?} \| {canceled?}>`，依赖 `types/form` 的字段 DSL | `:661-681` | **drop** | Firefish 用 `Form` DSL 生成设置表单；本项目设置页是具体模板，保留 DSL 只会多一层抽象。被隐藏的设置页（webhook、plugin、迁移）本来就不移植 |
| `selectUser` / `selectLocalUser` | `()` → `Promise<UserDetailed>`，内部用 `users/search-*` | `:683-720` | **drop** | 依赖不存在的用户搜索端点；需要选人的 UI（列表/群组/天线）已整体隐藏 |
| `selectInstance` | `()` → `Promise<Instance>` | `:721-738` | **drop** | 联邦实例浏览属隐藏功能 |
| `selectDriveFile` / `selectDriveFolder` | `(multiple)` → Drive 选择器 | `:740-844` | **drop** | Drive 文件夹与「从 Drive 选图」流程不在首批；附件只走本地上传 |
| `pageWindow` / `modalPageWindow` | `(path: string)` → 在 deck 里开页面窗口 | `:259-287` | **rewrite** | 本项目已有 `components/ui/PageWindow.vue` + `components/deck/deck-page.ts`，但入口是**路由名 + 参数**而不是路径字符串；移植时把 `path` 映射成 `router.resolve(path)` 后的 location，再交给 PageWindow。`modalPageWindow` 用同一实现（`modal` 标志） |
| `openEmojiPicker` | `(src, opts, initialTextarea)`，把 emoji 插进光标位置 | `:846-912` | **rewrite** | 本项目是 `components/ui/EmojiPicker.vue` + `os.ts:61 openReactionPicker`；emoji 目录为空（见 §3.11），选择器只提供 Unicode |

`pageWindow` 是这组里唯一必须保留行为的：Deck 的「列内打开详情」完全靠它，26 个调用点分布在
时间线与通知的各种「打开笔记/用户」入口。其余四项的 drop 都伴随页面隐藏，不会留下不可达入口。

## 4. drop 清单与理由

| 项 | 位置 | 理由 |
|---|---|---|
| Service Worker | `packages/sw/src/sw.ts`、`init.ts` 的注册逻辑 | 本项目不提供推送/离线缓存；`meta` 里 `features.serviceWorker: false` |
| 多账户 | `account.ts:53-66`、`pages/settings/accounts.vue`、账户切换菜单 | 单账户产品（设计文档 §7 架构里只有一份 session） |
| 插件 / AiScript | `plugin.ts`、`cold-store.ts`、`pages/scratchpad.vue`、`widgets/aiscript.vue`、`widgets/button.vue` | 依赖 `miauth/gen-token` 运行时与插件存储；非目标 |
| `pizzax` | `pizzax.ts`（`i/registry/get-all` + `registryUpdated` 事件） | 端点与流事件都不存在；deck/导航/主题改用 localStorage |
| 主题 DSL | `theme-store.ts`、`themes/*.json5`、`pages/theme-editor.vue` | JSON5 + 派生表达式是 Firefish 的实现细节；本项目 token 语义映射 |
| `toast` | `os.ts:289-298` | 0 个调用点：定义还在，但客户端从不调用，没有移植的理由 |
| 消息 / 天线 / 列表 / 频道 / 剪辑 / 群组 / 画廊 / 页面 | 见能力矩阵 §5 | 端点全缺，UI 一并移除，不留空壳 |
| 通行密钥 / TOTP | `MkSignin.vue:224,254` | 后端 `signin` 恒 `finished: true`，无二次因子 |

## 5. 目标文件布局（shim 落地位置）

```
web/src/os.ts                      # 现有命令 + Firefish os.* shim；api/apiWithDialog 委托
                                   #   web/src/api/client.ts 的 request()（pageWindow 复用 PageWindow）
web/src/api/client.ts              # 新增通用 request()、pendingApiRequestsCount；唯一 token 存取
                                   #   （TOKEN_KEY = "cinderfish.token"、getToken()/setToken()）
web/src/me.ts                      # 基于 stores/session.ts 的 reactive 投影
web/src/instance.ts                # 基于 stores/instance.ts 的 Firefish 形状
web/src/i18n/index.ts              # I18n 类
web/src/i18n/zh-CN.ts              # 白名单页面的词条
web/src/lib/z-index.ts             # claimZIndex
web/src/types/menu.ts              # MenuItem 判别联合
web/src/components/ui/DialogHost.vue   # alert / confirm / yesno / inputText / select 五种模式
web/src/components/ui/PopupMenu.vue    # popupMenu + contextMenu
web/src/components/AppIcon.vue     # 扩充 ph-* 映射所需的路径
```

原则：shim 集中在这 11 个文件里，**移植过来的页面/组件不得直接 `fetch`、不得直接读 `localStorage` 的 token、
不得自己拼 RPC 路径**（设计文档 §7 的边界要求）。`os.ts` 里的 `api`/`apiWithDialog` 只做 Firefish 形状的包装，
所有 HTTP 细节（token、错误信封、viewer stamping）都在 `api/client.ts` 的 `request()` 里；token 也只有
`api/client.ts` 读写（`TOKEN_KEY`、`getToken`/`setToken`），`stores/session.ts` 与 `stores/instance.ts` 只是调用方。
`pageWindow` 不新增文件：它映射到既有的 `components/ui/PageWindow.vue` 与 `components/deck/deck-page.ts`。

## 6. 验收

| # | 检查 | 判定 |
|---|------|------|
| 1 | 未登录调用 `api("meta")` | 不带 token，返回实例信息 |
| 2 | 登录后调用 `api("i")` | 返回 MeDetailed，`me` 投影同步 |
| 3 | 失败调用（如 `notes/show` 传不存在的 id） | `ApiError` 带 `message` 与 `id`，`apiWithDialog` 弹出可关闭的错误框 |
| 4 | `await alert(...)` | 用户关闭后才继续 |
| 5 | `const {canceled, result} = await inputText(...)` | 取消时 `canceled: true`，确认时 `result` 为输入值 |
| 6 | `post({reply})` | 打开发帖框并带回复目标，关闭后 Promise resolve |
| 7 | `popupMenu` 的 `switch` 项 | 切换后回写传入的 `ref` |
| 8 | `claimZIndex` 连续调用 | 单调递增，低/中/高三档不交叉 |
| 9 | 登出后 `me` | 变为 `null`，所有 `isSignedIn(me)` 分支走未登录 |
| 10 | 移植组件里 `grep -r "ph-"` | 无图标字体类名残留 |
| 11 | `await select({items})` 与 `await yesno(...)` | 分别返回 `{canceled:false,result}` 与 `{canceled}`，Esc 视为取消 |
| 12 | `pageWindow("/notes/<id>")` | 在 deck 列内打开笔记页，标记为 modal 的模式不可拖出 |
| 13 | `pnpm typecheck` | 通过（`me`、`i18n`、`MenuItem` 的类型不能是 `any`） |
