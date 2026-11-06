# Firefish 前端移植：第四波施工契约（2026-11-06）

本文是第四波所有并行移植任务共用的唯一契约。第三波结果见 `docs/HANDOVER.md` §12S。
参考源码：`tmp/refcheck/firefish-full/packages/client/src`（jclaudan 镜像 `b425f61`）。
后端行为以 `docs/research/api-contract/` 与 `internal/httpapi` 为准，不以 Misskey 文档为准。

## 0. 基线与本波目标

- 分支 `firefish-port`，基线提交 `714354a`（在第三波之上补了 `i/registry/scopes` 与
  `GET admin/federation/mode`），工作树干净。
- **本波目标**：补齐「管理者能用后台、用户能换主题、客户端设置能自管」的三块：
  1. 管理台（用户、表情、联邦模式、概览）；
  2. 主题体系（剩余 26 个内置调色板、主题选择/安装/管理/编辑器）；
  3. 注册表三页（`/registry/*`）；
  4. 仅靠本地存储即可成立的剩余设置页（自定义 CSS、文字过滤、偏好备份、API）；
  5. 访客与静态收口（欢迎页、404、公开本地时间线、别名页、关注授权页）。
- 不在本波：chat / channels / clips / gallery / pages / antennas / lists / groups、剩余
  widgets、其余 37 个 locale。原因见 `docs/research/2026-11-04-misskey-api-coverage.md` §5
  （首版明确不做）与 §12S.4；这些路由保留占位，但导航不得指向它们（见任务 `nav-trim`）。

## 1. 硬性规则（与二、三波相同，违反即返工）

1. 文件头保留归属：
   - `.vue`：`<!-- Ported from firefish@b425f61:packages/client/src/<path> (AGPL-3.0-only) -->`
   - `.ts`：`// Ported from firefish@b425f61:packages/client/src/<path> (AGPL-3.0-only)`
   - `.json5`：首行 `// Ported from firefish@02c31e0:packages/client/src/themes/<name>.json5 (AGPL-3.0-only)`
   - 完全新写（无对应上游文件）的文件写 `// SPDX-License-Identifier: AGPL-3.0-only`。
2. **禁止** `$ref` / `$computed` / `$shallowRef`：用 `ref()` / `computed()` / `shallowRef()`。
3. **禁止**直接 `fetch`/`XHR`；一律 `import { api } from "@/api/client"`（上传走 `apiForm`）。
4. **禁止**页面读 `localStorage` 里的 token；会话用 `import { me, isAdmin } from "@/account"`。
   主题引擎与设置页读写各自的非 token 键是允许的。
5. 类型来自 `@/api/types`；不出现 `firefish-js` / `misskey` 命名空间类型。
6. i18n：键存在时用 `t("key")`；`zh-CN.json` 缺键时直接写中文文案（第三波先例），
   **不要**为了新文案改 `web/src/i18n/locales/*.json`。
7. **不引入任何新依赖**，不改 `web/package.json` / `pnpm-lock.yaml` / `web/src/router/routes.ts`。
8. 不新增全局 CSS 变量；不写 `!important`。样式 scoped SCSS，主题变量沿用
   `--panel`、`--fg`、`--fgTransparentWeak`、`--divider`、`--accent`、`--radius`、`--buttonBg`、
   `--accentedBg`、`--X4` 等 Firefish 同名变量。
9. 所有 `.vue` 用 `<script setup lang="ts">`。模板里用到的 `Mk*`/`Cf*` 组件必须 import
   （`components/global/` 下的自动注册除外），否则 `check:port` 的 `unresolvedComponents` 会报错。
10. 只改「§4 写作用域」里属于你的文件；跨作用域接口按 §3 写，缺文件时先按接口实现。
11. 每个页面都要有真实数据、空态（`CfInfo` / `_fullinfo`）与错误态（`MkError` 或 `CfInfo warn`），
    不允许保留 `TODO: port` 占位，也不允许用假数据冒充后端。

## 2. 本波强制降级表

| Firefish 能力 | 本波处理 |
|---|---|
| 管理台图表（`charts/*`、queue、AP requests、heatmap、moderators） | 端点不存在。概览只展示能从 `meta`、`admin/show-users`、`admin/federation/mode`、`admin/federation/allowlist` 读到的真实值 |
| `admin/show-users` 的排序/状态/来源筛选与搜索 | 端点只接受 `limit`/`offset`。管理用户页只做分页，不画无效筛选器 |
| `admin/suspend-user` 的 `suspend` 理由/期限 | 端点只接受 `userId`。不做理由输入 |
| 管理台其余页面（queue、files、announcements、database、security、relays、integrations、instance-block、proxy-account、experiments、object-storage、email-settings、hashtags、abuses、settings、custom-css） | 端点不存在或未接线。保留占位，但管理台导航只列本波实装页 |
| `i/export` / `i/import` / `i/delete` / `i/change-password` / 2FA / `i/apps` / `i/webhooks` | 端点不存在。对应设置页保持占位；偏好备份只做浏览器 localStorage 的 JSON 导出/导入 |
| `mute/create` / `block/create`（服务器侧静音/屏蔽） | 端点不存在。`mute-block`、`instance-mute` 保持占位，本波不移植 |
| statusbar / custom-katex / plugin / import-export / theme 的实例级主题 | 不移植或不做假控件；主题只做客户端内置 + 已安装主题 |
| `json5` npm 包 | 不装。安装/编辑主题用 `web/src/themes/json5.ts` 的 `parseJson5` / `parseThemeJson5` |
| 26 个内置主题的上游 UUID | 统一改用文件 slug 作为 `id`（与已移植的 `d-rosepine` / `l-rosepinedawn` 一致），保证可存进 localStorage |
| 欢迎页三种 entrance 变体、`XTimeline`、setup 向导 | 只做一个欢迎页：实例名/简介 + 登录/注册入口 + 最近本地时间线（登录后）；不移植 `welcome.entrance.b/c` |
| `ap/show`（关注授权页对远程 acct） | 端点不存在。远程 acct 显示「远程账号请到对方实例关注本账号」，不伪造关注 |
| `notes/history` | 端点不存在。`NoteHistory.vue` 显示说明与返回笔记链接 |
| `users/search` / `notes/search` | 仍不存在；搜索页保持第三波的说明态 |

## 3. 跨作用域接口（必须按此实现/调用）

### 3.1 主题引擎（`theme-engine` 提供，`theme-pages` 与 `nav-trim` 调用）

`web/src/themes/installed.ts`（新建）：

```ts
export function listInstalledThemes(): Theme[];
export function findInstalledTheme(id: string): Theme | undefined;
export function installTheme(theme: Theme): void;   // id 与内置主题冲突时抛 Error
export function uninstallTheme(id: string): void;
```

- 持久化键 `cinderfish.installedThemes`，值为 `Theme[]` 的 JSON；读取时逐项 `validateTheme`，
  损坏项跳过而不是整表丢弃。
- `web/src/themes/builtin.ts`：`builtinThemes` = `paletteThemes`（`./palettes`，26 个）
  加已存在的 `rosePineDawnTheme` / `rosePineTheme`，按 id 去重；`builtinThemeMap`、`getTheme`、
  `findTheme` 先查内置再查已安装；`getDefaultTheme` 行为不变。

`web/src/composables/use-theme.ts` 的公开面在原基础上增加：

```ts
export const selectedThemeIds: { light: Ref<string | null>; dark: Ref<string | null> };
export function setThemeForMode(mode: ColorSchema, id: string | null): void;
```

- 持久化键改为 `cinderfish.themeId.light` / `cinderfish.themeId.dark`；读到旧的单值
  `cinderfish.themeId` 时按其主题 `themeMode()` 迁进对应槽再删除旧键。
- `activeTheme` = 当前 resolvedMode 槽位的主题，缺失时回落 `getDefaultTheme(resolvedMode)`。
- `setTheme(name)` 保持现有语义：模式名切模式；主题 id 写入该主题的 mode 槽位。
- 现有调用点（`main.ts` 的 `initTheme()`、`MkNavbar` 的主题入口等）不得改签名。

### 3.2 `palettes.ts`（`theme-palettes` 提供）

```ts
// web/src/themes/palettes.ts
export const paletteThemes: Theme[];
```

- 26 个 `.json5` 原样复制到 `web/src/themes/builtin/<slug>.json5`，
  `palettes.ts` 用 `import raw from "./builtin/<slug>.json5?raw"` 加载，`parseJson5(raw)` 解析后
  规范化再断言：`id` 用文件 slug，缺 `author` 时补 `""`（`d-u0` / `l-u0` 上游没有 author），
  最后 `validateTheme()` 不通过就抛出带文件名的错误。`?raw` 已有 `vite/client` 类型，
  不要新建 `.d.ts`。

### 3.3 管理台（`admin-core` / `admin-content` 共用事实）

- 只有 `isAdmin()` 为真才渲染管理页；非管理员显示 `CfInfo warn` 与返回首页链接，不发请求。
- 端点：
  - `admin/show-users`：POST `{ limit?, offset? }` → `UserDetailed[]`（含 `isAdmin` / `isSuspended`）。
  - `admin/accounts/create`：POST `{ username, password }` → `UserDetailed`。
  - `admin/suspend-user` / `admin/unsuspend-user`：POST `{ userId }` → `{}`；不能冻结自己（400）。
  - `admin/emoji/list`：POST `{ query?, limit?, sinceId?, untilId? }` → `AdminEmoji[]`（分页）。
  - `admin/emoji/add`：POST `{ name, fileId, category?, aliases? }` → `AdminEmoji`。
    `name`/`aliases` 必须匹配 `^[a-zA-Z0-9_]+$`；文件用 `selectFiles({ multiple:false })` 选或先上传。
  - `admin/emoji/delete`：POST `{ id }` → `{}`。
  - `admin/federation/mode`：GET → `{ mode: "public"|"allowlist"|"none" }`；POST `{ mode }` → 同形状。
  - `admin/federation/allowlist`：GET → `{ hosts: string[] }`；POST `{ host, remove? }` → `{ hosts }`。
- `AdminIndex.vue` 用 `MkSuperMenu` 做分区导航，只列已实装页：用户、表情、联邦、概览。
  子路由为空时重定向到 `/admin/users`。

### 3.4 注册表（`registry-pages` 使用）

- `i/registry/scopes`（本波新增）：POST `{}` → `string[][]`。
- `i/registry/keys`：POST `{ scope }` → `string[]`。
- `i/registry/get-detail`：POST `{ key, scope }` → `{ updatedAt, value }`。
- `i/registry/get-all`：POST `{ scope }` → `Record<string, unknown>`。
- `i/registry/set`：POST `{ key, scope, value }` → `{}`；
  `i/registry/remove`：POST `{ key, scope }` → `{}`；`i/registry/clear`：POST `{ scope }` → `{}`。
- 路由固定为系统域：`/registry/keys/system/:path(.*)*`，scope = `["system", ...path.split("/")]`；
  值为 JSON，编辑时用 `parseJson5` 解析（允许注释/尾逗号），非法 JSON 就地报错不提交。

### 3.5 自定义 CSS（`settings-local` 提供，`nav-trim` 不改）

- `defaultStore.state.customCss`（新增字段，`where: "account"`，默认 `""`）。
- `web/src/main.ts` 在挂载后调用新文件 `web/src/composables/use-custom-css.ts` 的
  `initCustomCss()`：创建一个 `<style id="cinderfish-custom-css">`，watch 该值并写入；
  空字符串时清空内容。

### 3.6 静态页别名

- `MyFavorites.vue` 直接渲染 `Favorites.vue`，`MyNotifications.vue` 直接渲染 `Notifications.vue`；
  不要在路由表加别名，也不要复制页面实现。

## 4. 任务与写作用域（互不重叠）

| 任务 | 目标 | 独占文件 |
|---|---|---|
| ~~`theme-engine`~~（Lead 已完成） | 已安装主题存储、内置主题统一查找、light/dark 双槽、旧键迁移 | `web/src/themes/installed.ts`、`web/src/themes/builtin.ts`、`web/src/themes/index.ts`、`web/src/composables/use-theme.ts`；`palettes.ts` 已有空 stub，由 `theme-palettes` 填充 |
| `theme-palettes` | 剩余 26 个 Firefish 内置调色板（原样 json5 + `?raw` 加载） | `web/src/themes/palettes.ts`（新建）、`web/src/themes/builtin/*.json5`（26 个新文件） |
| `theme-pages` | 主题设置、安装、管理、编辑器四页 | `web/src/pages/SettingsTheme.vue`、`SettingsThemeInstall.vue`、`SettingsThemeManage.vue`、`ThemeEditor.vue` |
| `admin-core` | 管理台壳、用户管理、概览 | `web/src/pages/AdminIndex.vue`、`AdminUsers.vue`、`AdminOverview.vue`、`web/src/components/admin/*`（新建） |
| `admin-content` | 自定义表情、联邦开关与白名单 | `web/src/pages/AdminEmojis.vue`、`AdminFederation.vue` |
| `registry-pages` | 注册表总览、键列表、值编辑 | `web/src/pages/Registry.vue`、`RegistryKeys.vue`、`RegistryValue.vue` |
| `settings-local` | 自定义 CSS、文字过滤、偏好备份、API 页 | `web/src/pages/SettingsCustomCss.vue`、`SettingsWordMute.vue`、`SettingsPreferencesBackups.vue`、`SettingsApi.vue`、`web/src/composables/use-custom-css.ts`（新建）、`web/src/store/base.ts`（只加 `customCss` 字段）、`web/src/main.ts`（只加 `initCustomCss()`） |
| `static-pages` | 欢迎、404、公开本地时间线、别名、关注授权、笔记历史 | `web/src/pages/Welcome.vue`、`NotFound.vue`、`Empty.vue`、`ErrorPage.vue`、`NoGraze.vue`、`MyFavorites.vue`、`MyNotifications.vue`、`Follow.vue`、`FollowMe.vue`、`NoteHistory.vue` |
| `nav-trim`（合并后） | 导航只指向已实装页；设置菜单补齐新页；管理员入口 | `web/src/components/MkNavbar.vue`、`web/src/pages/SettingsIndex.vue` |

## 5. API 事实（避免按 Misskey 文档假设）

- 所有 `/api` 路由 POST + JSON；`ping`、`admin/federation/allowlist`、`admin/federation/mode`
  的 GET 例外。GET 用 `apiGet` 或 `call`（`GET_ENDPOINTS` 已登记 mode）。
- 管理端点都要求管理员；非管理员返回 403，页面应先判 `isAdmin()` 再请求。
- `admin/emoji/list` 用 `sinceId`/`untilId` 游标；`CfPagination` 可直接传
  `{ endpoint: "admin/emoji/list", limit: 30, params: { query } }`。
- `admin/show-users` 只有 `limit`/`offset`；`CfPagination` 的游标模型不适用，
  用户页用 `offset` 手动分页（上一页/下一页按钮）。
- `i/registry/get` 返回裸值（可能是对象/数组/字符串），`get-detail` 才带 `updatedAt`。
- `i` 响应里的 `isAdmin` 是布尔；`isAdmin()` 是账号层函数，不是响应式布尔。
- 被冻结用户的资料仍可读；冻结只影响登录/发帖。管理用户页要显示 `isSuspended` 标记。
- 主题色值语法：`"#86b300"`、`"@accent"`、`":darken<10<@accent"`、`"$const"`、引号包裹的字面量。
  编辑器只编辑常用 token（`accent`、`bg`、`fg`、`panel`、`divider`、`link`、`warn`、`error`），
  其余 token 从所选 `base` 主题继承。

## 6. 每个任务的完成标准

1. 页面不再是 `TODO: port`；有真实数据、空态、错误态；降级处写明原因。
2. 所有出网调用经 `@/api/client`；会话读取经 `@/account`；不出现直接 `fetch`。
3. `cd web && pnpm exec vue-tsc --noEmit`：你自己的文件无错误（全仓可能有并行任务造成的临时错误，
   报告里注明）。
4. `cd web && pnpm run check:port`：你的文件零硬违规（`forbiddenDeps` / `i18nStyle` /
   `reactivity` / `directFetch` / `tokenRead` / `unresolvedComponents`）。
5. 报告里写：改了哪些文件、对照了哪些 Firefish 源文件、降级了哪些能力、没接的端点、
   实际跑过的命令与退出码。不要声称跑过没跑的命令。

## 7. 整合顺序（Lead 执行，agent 不必做）

1. 先合并 `theme-palettes` → `theme-engine` → `theme-pages`，确认主题切换与安装可用。
2. 合并 `admin-core` / `admin-content` / `registry-pages` / `settings-local` / `static-pages`。
3. 最后做 `nav-trim`（依赖以上页面全部落地），再统一跑 `typecheck` / `check:port` / `build` / Go 测试。
4. 真实 Go 二进制 + 无头 Chromium 走查：管理员登录后台、冻结/解冻、表情增删、联邦模式切换并切回、
   主题切换/安装/编辑器、注册表 CRUD、自定义 CSS、文字过滤、偏好备份、欢迎页、404、公开本地时间线；
   桌面 1440 与移动 390 两档，检查 `scrollWidth == clientWidth` 与 console 异常。
