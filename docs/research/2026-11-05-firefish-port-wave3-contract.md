# Firefish 前端移植：第三波施工契约（2026-11-05）

本文是第三波所有并行移植任务共用的唯一契约。第二波结果见 `docs/HANDOVER.md` §12R。
参考源码：`tmp/refcheck/firefish-full/packages/client/src`（jclaudan 镜像 `b425f61`）。
后端行为以 `docs/research/api-contract/` 为准，不以 Misskey 文档为准。

## 0. 基线与本波目标

- 分支 `firefish-port`，基线提交 `2b68f7b`，工作树干净。
- 基线门禁已实测通过：`cd web && pnpm exec vue-tsc --noEmit` exit 0；
  `pnpm run check:port`（401 文件）零违规。
- **本波目标**：把「能看」变成「能发、能看人、能设置、能看趋势」：
  发帖链路（composer）、个人页与关注/粉丝列表、设置页骨架与核心子页、
  内容页（收藏/关注请求已发/探索/标签/搜索）、壳层导航与 widgets 栏、deck 列。
- 页面总数 198，当前只有 8 个已实装；本波结束后应达到约 40 个。

## 1. 硬性规则（与第二波相同，违反即返工）

1. 文件头保留归属：
   - `.vue`：`<!-- Ported from firefish@b425f61:packages/client/src/<path> (AGPL-3.0-only) -->`
   - `.ts`：`// Ported from firefish@b425f61:packages/client/src/<path> (AGPL-3.0-only)`
2. **禁止** `$ref` / `$computed` / `$shallowRef`：改成 `ref()` / `computed()` / `shallowRef()`。
3. **禁止**直接 `fetch`/`XHR`（`scripts/upload.ts` 除外，走 `apiForm`）；一律 `import { api } from "@/api/client"`。
4. **禁止**读 `localStorage` 里的 token；会话用 `import { me } from "@/account"`（Firefish 的 `$i`）。
5. 类型一律来自 `@/api/types`，不出现 `firefish-js` / `misskey` 命名空间类型。
6. i18n 用 `t()`（扁平键，键名保留 Firefish 原名，如 `t("_postForm.replyPlaceholder")`）。
   **禁止** `i18n.ts.*` 写法。改完自查 `grep -rn "i18n\.ts" web/src` 必须为空。
7. **不引入任何新依赖**，不改 `web/package.json` / `pnpm-lock.yaml` / `web/src/router/routes.ts` /
   `web/src/i18n/locales/*.json`。缺依赖按下表降级。
8. 不新增全局 CSS 变量；不写 `!important`。样式一律 scoped SCSS，主题变量用 `--panel`、`--fg`、
   `--divider`、`--accent`、`--radius`、`--buttonBg`、`--accentedBg` 等 Firefish 同名变量。
9. 单文件超过约 700 行时照抄结构、删掉不适用分支，不重写架构。
10. 所有 `.vue` 用 `<script setup lang="ts">`。组件名保留 `Mk*`（页面文件名以路由表为准）。
11. 只改「§4 写作用域」里属于你的文件。别人的文件缺失时先按契约接口写，最后整合再对齐；
    发现契约不成立，在最终汇报里写清楚，不要顺手改别人作用域。

## 2. 本波强制降级表

| Firefish 能力 | 本波处理 |
|---|---|
| `unicode-emoji-json` / `emojilib` / `focus-trap-vue` / `textarea-caret` | 不装。表情选择器用内置精选列表（按分类分组，约 200 个）+ `api("emojis")` 自定义表情；光标坐标用 `textarea.selectionStart` 粗略定位，不做像素级 caret 跟随 |
| `insert-text-at-cursor` / `stringz` / `throttle-debounce` / `punycode` | 不装。分别用 `setRangeText` + 手动派发 input、`Array.from(text).length`、`setTimeout` 节流、直接显示原 handle |
| `vue-draggable-plus`（附件拖拽排序） | 不做拖拽排序；附件按选择顺序排列，逐项可删除 |
| `drive/files/update`（改名/敏感标记） | 端点不存在：附件操作只有「移除」 |
| 投票（`notes/create` 的 poll 已支持） | 保留 `MkPollEditor`，提交走 `poll` 参数；不要调用不存在的 `notes/polls/...` 之外的端点 |
| `MkCheatSheetDialog` 的 kaTeX | `$[math]` 按代码显示，不引 katex |
| `s-age`（年龄） | 自己用生日字符串算整数年龄 |
| `MkMiniChart` / `chart.js` | 用户卡片不画迷你图；用数字/文字代替 |
| `MkRipple` | 不移植，点击态用 CSS `:active` |
| 插件系统 `postFormActions` / `plugin` 按钮 | 数组存在但为空；不渲染「插件」按钮 |
| 搜索端点 `notes/search`、`users/search` | 不存在。`Search.vue` 渲染「搜索接口尚未提供」并说明可用探索页标签；不伪造结果 |
| `announcements` 端点 | 不存在。`Announcements.vue` 渲染空态说明 |
| `hashtags/trend` | **存在**，探索页趋势区接它 |
| `i/update` 可写字段 | 只有 `name` / `description` / `avatarId` / `followersVisibility` / `followingVisibility`；其余输入不做假控件 |
| `i/registry/*` | 存在，设置页的客户端偏好可落库；但页面状态仍以 `defaultStore`（localStorage）为主，不双写 |
| `users/following|followers` 隐私 | 403 时显示 `CfInfo` 说明，不当作错误 |
| 自定义 MFM 主题 / 插件 / 声音 | 不实现；声音设置页只保留开关与说明 |

## 3. 跨作用域接口（必须按此实现/调用）

### 3.1 `os.post(props)`（composer-core 提供，composer-wiring / shell 调用）

```ts
// web/src/os/post.ts
export interface PostFormProps {
  reply?: NoteDetailed;
  renote?: NoteDetailed;
  initialText?: string;
  initialVisibility?: NoteVisibility;
  initialFiles?: DriveFile[];
  initialLocalOnly?: boolean;
  initialNote?: NoteDetailed;   // 编辑在 Cinderfish 无端点，忽略
  fixed?: boolean;
  autofocus?: boolean;
}
export function post(props?: PostFormProps): Promise<void>;
```

- 用 `popup(MkPostFormDialog, props, {}, "closed")` 打开；`MkPostFormDialog` 是 `os/` 下的薄包装，
  负责 `CfModal`/`os.popup` 外观并把 `posted`/`cancel` 转成关闭。
- `MkPostForm` 发帖成功 emit `posted`；取消 emit `cancel`。`os.post` 在关闭时 resolve。
- `MkPostForm` 不使用 `inject("modal")`；只用 prop `fixed` 决定是否显示右上角关闭按钮。

### 3.2 `os.openEmojiPicker(target, props, textarea?)`（composer-core 提供）

```ts
export function openEmojiPicker(
  target: HTMLElement,
  props?: { asReactionPicker?: boolean; showPinned?: boolean },
  textarea?: HTMLTextAreaElement | HTMLInputElement | null,
): void;
```

- 选定表情后：有 `textarea` 就插入到光标处（`setRangeText`），否则只 emit 给调用方；
  reaction picker 场景复用 `scripts/reaction-picker.ts` 的最近使用记录。
- `MkEmojiPicker` props：`showPinned?: boolean`、`asReactionPicker?: boolean`、`maxHeight?: number`；
  emits `chosen(emoji: string, ev: MouseEvent)`。

### 3.3 `CfTab` 新契约（tabs 任务重写，所有页面用）

```vue
<CfTab v-model="tab" :tabs="[{ value: 'replies', label: '回复', count: 3, icon: 'ph-chat' }]" style="underline" />
```

- `tabs: { value: string | number; label: string; count?: number; icon?: string }[]`
  （`:key` 用 `value`）。
- 删除 `CfTabItem.vue`；`MkNoteDetailed.vue` 改为数组写法。
- 这是修复第二波遗留缺陷（`<option>` VNode 在 Vue 3 被折叠、标签不显示）。

### 3.4 `MkSuperMenu`（settings-shell 提供）

```vue
<MkSuperMenu :def="[{ title: '基本设置', items: [{ icon: 'ph-user', text: '个人资料', to: '/settings/profile', active: true }] }]" />
```

- 宽容器竖排 sticky，窄容器（<600，用容器宽度判定）收成一行横向滚动胶囊。
- 与第二波旧前端的 `SectionMenu` 同构；不引入 `os` 依赖。

### 3.5 用户卡片（user-profile 提供，follow-pages / widgets 调用）

- `MkUserInfo.vue`：props `{ user: UserLite, detailed?: boolean }`。
- `MkUserCardMini.vue`：props `{ user: UserLite, class?: string }`。
- `MkUserList.vue`：props `{ pagination: Paging }`，内部用 `CfPagination`。
- `getUserMenu(user, ev)` 从 `scripts/get-user-menu.ts` 导出。

### 3.6 壳层（shell-navbar / deck-shell 提供）

- `components/MkNavbar.vue`：Firefish `ui/_common_/navbar.vue` 的移植，右侧/底部发帖按钮调 `os.post()`。
- `ui/layouts/XSidebar.vue` 保留为「rail + MkNavbar」的组合层，Universal/Deck/Visitor 共用。
- `ui/layouts/UniversalLayout.vue` 负责 `<XSidebar /> + 主列 + <MkWidgets /> + 移动底栏`。
- `MkWidgets.vue` 由 widgets 任务提供，props 无；内部读 `defaultStore.state.widgets`。

## 4. 任务与写作用域（互不重叠）

| 任务 | 目标 | 独占文件 |
|---|---|---|
| `tabs` | 重写 CfTab 为数组契约；修 `MkNoteDetailed` 的回复/转发/引用页签 | `web/src/components/ui/CfTab.vue`、删除 `web/src/components/ui/CfTabItem.vue`、`web/src/components/MkNoteDetailed.vue` |
| `composer-core` | 发帖表单、附件、可见性、表情、MFM 速查、自动补全、`os.post`、`os.openEmojiPicker` | `web/src/components/MkPostForm.vue`、`MkPostFormAttaches.vue`、`MkVisibilityPicker.vue`、`MkCheatSheetDialog.vue`、`MkEmojiPicker.vue`、`MkEmojiPicker.section.vue`、`MkEmojiPickerDialog.vue`、`web/src/scripts/autocomplete.ts`、`preprocess.ts`、`katex-macro.ts`、`emojilist.ts`、`web/src/os/post.ts`、`web/src/os/MkPostFormDialog.vue`、`web/src/os/open-emoji-picker.ts`、`web/src/os/index.ts`（只加导出） |
| `composer-wiring` | 笔记回复/引用改走 `os.post`；时间线顶部固定发帖框 | `web/src/components/MkNote.vue`、`web/src/components/MkNoteFooter.vue`（若需新建）、`web/src/pages/Timeline.vue`、`web/src/pages/Note.vue` |
| `user-profile` | 个人页、五个页签、用户卡片组件、用户菜单、`/user-info/:userId` | `web/src/pages/UserProfile.vue`、`web/src/pages/UserInfo.vue`、`web/src/pages/UserIndex.vue`、`web/src/components/MkUserInfo.vue`、`MkUserCardMini.vue`、`MkUserList.vue`、`MkUserOnlineIndicator.vue`、`MkShowMoreButton.vue`、`MkNumber.vue`、`web/src/components/user/*.vue`（新建）、`web/src/scripts/get-user-menu.ts`、`web/src/filters/number.ts` |
| `follow-pages` | 关注/粉丝列表页 | `web/src/pages/UserFollowing.vue`、`web/src/pages/UserFollowers.vue`、`web/src/components/user/FollowList.vue` |
| `settings-shell` | 设置父页、左菜单、表单包装组件、个人资料设置 | `web/src/pages/SettingsIndex.vue`、`web/src/components/MkSuperMenu.vue`、`web/src/components/form/*.vue`、`web/src/pages/SettingsProfile.vue`、`web/src/scripts/langmap.ts` |
| `settings-privacy` | 隐私、通用、账户信息三页 | `web/src/pages/SettingsPrivacy.vue`、`SettingsGeneral.vue`、`SettingsAccountInfo.vue` |
| `settings-misc` | 通知/反应/云盘/安全/其他/导航/声音/deck 八页 | `web/src/pages/SettingsNotifications.vue`、`SettingsReaction.vue`、`SettingsDrive.vue`、`SettingsSecurity.vue`、`SettingsOther.vue`、`SettingsNavbar.vue`、`SettingsSounds.vue`、`SettingsDeck.vue` |
| `content-pages` | 收藏、已发关注请求、探索、标签、搜索 | `web/src/pages/Favorites.vue`、`FollowRequestsSent.vue`、`Explore.vue`、`Tag.vue`、`Search.vue` |
| `static-pages` | 关于、公告、MFM 速查、分享、预览、便签、API 控制台 | `web/src/pages/About.vue`、`Announcements.vue`、`MfmCheatSheet.vue`、`Share.vue`、`Preview.vue`、`Scratchpad.vue`、`ApiConsole.vue` |
| `shell-navbar` | Firefish navbar、三壳布局、移动底栏、访客壳 | `web/src/components/MkNavbar.vue`（新建）、`web/src/ui/layouts/XSidebar.vue`、`UniversalLayout.vue`、`VisitorLayout.vue`、`web/src/ui/UniversalShell.vue`、`DeckShell.vue`（若需） |
| `deck-shell` | Deck 列、列菜单、添加列、列宽 | `web/src/ui/layouts/DeckLayout.vue`、`DeckColumn.vue`、`web/src/store/deck.ts` |
| `widgets` | widgets 栏与 12 个首版 widget | `web/src/components/MkWidgets.vue`（新建）、`web/src/widgets/*.vue`、`web/src/widgets/index.ts`、`web/src/store/widgets.ts` |

## 5. API 事实（避免按 Misskey 文档假设）

- 认证：请求体 `i` 优先于 `Authorization: Bearer`；**禁止**页面直接拼 token。
- `notes/create` 字段：`visibility`、`visibleUserIds`、`text`、`cw`、`localOnly`、`fileIds`、
  `replyId`、`renoteId`、`poll { choices, multiple?, expiredAfter? }`。
  纯转贴只传 `renoteId`；引用传 `renoteId` + `text`。
- 可见性收敛由服务端完成：public 回复 home 笔记会落成 home，回复 followers 笔记会落成 followers；
  composer 不需要自己改写，但 UI 提示可选。
- `users/show` 只接受本地用户名（`username`/`userId`），远程用户返回错误。
- `users/notes` 的 `withReplies` **默认 true**（与上游相反）；个人页时间线要显式传 `withReplies: true`。
- `users/following|followers` 返回 `Following[]`（`{ id, createdAt, followee, follower }`），
  受 `followingVisibility`/`followersVisibility` 约束，403 表示不可见。
- `notes/favorites/create|delete`、`i/favorites` 存在；`i/favorites` 返回 `NoteFavorite[]`（含 `note`）。
- `following/requests/sent` 存在，返回 `FollowRequest[]`（`{ id, follower, followee }`）。
- `hashtags/trend` 无参数，返回 `{ tag, chart: number[7], usersCount }[]`。
- `emojis` 返回 `{ emojis: EmojiSimple[] }`；`EmojiSimple` 字段见 `api/types.ts`。
- HTTP 时间线的 `withRenotes` 参数**无效**，不要依赖。
- 分页统一 `CfPagination` + `usePagination`；`sinceId` 单独使用会翻成 ASC。

## 6. 每个任务的完成标准

1. 页面不再是 `TODO: port` 占位；有真实数据、空态、错误态（`MkLoading` / `MkError` / `CfInfo`）。
2. 所有出网调用经 `@/api/client`；所有会话读取经 `@/account`。
3. `cd web && pnpm exec vue-tsc --noEmit`：你自己的文件无错误。
4. `cd web && pnpm run check:port`：零违规（尤其 `unresolvedComponents`、`missingKeys`）。
5. 报告里写：改了哪些文件、对齐了哪些 Firefish 源文件、降级了哪些能力、还没接的端点、
   以及你实际跑过的命令与退出码。不要声称跑过没跑的命令。

## 7. 整合顺序（Lead 执行，agent 不必做）

1. 先合并 `tabs` / `os` 接口，保证所有页面能编译。
2. 再合并 composer-core → composer-wiring → shell-navbar → widgets → deck-shell。
3. 最后合并设置页与内容页，统一跑 `typecheck` / `check:port` / `build`。
4. 用真实 Go 二进制 + 无头 Chromium 走查：发帖、回复、引用、图片上传、个人页五页签、
   关注/粉丝列表、设置页保存、收藏、探索趋势、widgets、deck 列、移动端底栏。
