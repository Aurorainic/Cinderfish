# Firefish 前端移植：第二波（Note 树 + 时间线）施工契约（2026-11-05）

本文是第二波所有并行移植任务共用的唯一契约。第一波的骨架见 HANDOVER §12Q。
参考源码：`tmp/refcheck/firefish-full/packages/client/src`（jclaudan 镜像 b425f61）。
冲突时以本文 + `docs/research/api-contract/` 为准，不以 Misskey 文档为准。

## 0. 已经由 Lead 落地、不要重复改的东西

- 依赖已加：`mfm-js@0.23.3`（Firefish 同款 MFM 解析器）、`@phosphor-icons/web@2.0.3`
  （图标字体，bold + fill 两套）。**不要再改 `web/package.json` / `pnpm-lock.yaml`。**
- 本波不做 PostForm / UserProfile / Drive 以外的页面；页面文件只改自己名下的。
- 所有 `web/src/pages/*.vue` 占位页面文件名由 `web/src/router/routes.ts` 决定，**不要改路由表**。

## 1. 命名与目录约定

| 类别 | 约定 |
|------|------|
| 从 Firefish 移植的组件 | 保留 Firefish 文件名与组件名，放到 `web/src/components/`，例如 `MkNote.vue`、`MkTimeline.vue` |
| 全局组件 | `web/src/components/global/`，由 `web/src/components/index.ts` 用 `import.meta.glob` 自动注册，模板里可直接写 `<MkA>` `<Mfm>` 等，不要 import |
| 第一波已有原子组件 | 继续用 `web/src/components/ui/Cf*`，见 §4 映射 |
| 页面 | `web/src/pages/<路由表里的名字>.vue` |
| 脚本/过滤器 | `web/src/scripts/*.ts`、`web/src/filters/*.ts`，签名见 §6 |
| 样式 | 一律 scoped SCSS，主题变量名与 Firefish 相同（`--panel`、`--fg`、`--divider`、`--accent`、`--radius`…） |

文件头必须写归属（AGPL-3.0）：
- `.vue`：`<!-- Ported from firefish@b425f61:packages/client/src/components/MkNote.vue (AGPL-3.0-only) -->`
- `.ts`：`// Ported from firefish@b425f61:packages/client/src/scripts/array.ts (AGPL-3.0-only)`

## 2. 硬性移植规则

1. **禁止** `$ref` / `$computed`（Firefish 开了 reactivity transform，我们没开）：改成 `ref()` / `computed()`。
2. **禁止**直接 `fetch`/`XHR`（`scripts/upload.ts` 除外，走 `apiForm`）：
   一律 `import { api } from "@/api/client"`。
3. **禁止**读 `localStorage` 里的 token；会话用 `import { me } from "@/account"`（Firefish 的 `$i`）。
4. 类型一律来自 `@/api/types`，**不要**出现 `firefish-js` / `misskey` 命名空间类型。
5. i18n 调用全部改写成 `t()`：
   - `i18n.ts._ago.justNow` → `t("_ago.justNow")`
   - `i18n.tsx._time.hoursAgo({ n })` → `t("_time.hoursAgo", { n })`
   键名**保留前导下划线**（locale JSON 里就是 `"_ago.justNow"`）。`import { t } from "@/i18n"`。
   **改完自查：`grep -rn "i18n\.ts" web/src` 必须为空。**
6. 不引入新依赖。以下 Firefish 依赖**不装**，按 §5 降级：
   `swiper`、`photoswipe`、`vue-plyr`、`katex`、`blurhash`、`vue-draggable-plus`、`chart.js`、
   `date-fns`、`tinycolor2`、`insert-text-at-cursor`、`stringz`、`throttle-debounce`、`punycode`。
7. 单个文件超过约 700 行时优先照抄结构、删掉不适用分支，**不要重写架构**。
8. 不新增全局 CSS 变量；不写 `!important`。
9. 所有 `.vue` 使用 `<script setup lang="ts">`（与第一波一致），除非原文件用 Options API 且必须保留。

## 3. 模块映射（Firefish → Cinderfish）

| Firefish | Cinderfish |
|---|---|
| `@/os`（`os.popupMenu` `os.popup` `os.alert` `os.confirm` `os.inputText` `os.select` `os.toast` `os.apiWithDialog`） | `@/os` 同名导出，签名兼容（`popupMenu(items, ev)` 收 `MouseEvent\|{x,y}`；`confirm({...})` 返回 `{canceled, result}`） |
| `@/store` 的 `defaultStore` | `@/store/base` 的 `defaultStore`（与 `@/store/store` 同一个实例，foundation-core 负责合并） |
| `defaultStore.reactiveState.x.value` | 同左（pizzax 有 `reactiveState`） |
| `$i` / `@/account` | `import { me } from "@/account"`（`me.value`，模板里 `me` 自动解包；判空用 `me.value == null`） |
| `@/stream` 的 `stream` | `import { stream } from "@/stream"`（shim 到 `useStream()`；支持 `stream.on/useChannel/send/subscribeNote`） |
| `@/instance` 的 `instance` | `import { instance } from "@/instance"`（由 `meta` 填充的 reactive 对象） |
| `@/config` 的 `host`/`url` | `import { host, url } from "@/config"` |
| `@/const` 的 `FILE_TYPE_BROWSERSAFE` 等 | `import { FILE_TYPE_BROWSERSAFE } from "@/const"` |
| `@/scripts/please-login` `pleaseLogin()` | 同名 shim（未登录跳 `/` 并提示） |
| `@/scripts/page-metadata` `definePageMetadata` | 同名 shim（写入 `document.title`） |
| `@/scripts/copy-to-clipboard` | 同名 shim（`navigator.clipboard`） |
| `@/scripts/clone` `deepClone` | 同名 |
| `@/scripts/array` `concat/sum/erase/unique/chunk` | 同名 |
| `@/filters/user` `userPage/acct`、`@/filters/note` `notePage` | 同名 |
| `@/scripts/get-note-summary` `getNoteSummary` | 同名 |
| `@/scripts/check-word-mute` `getWordSoftMute` | 同名（无后端词过滤时返回 null） |
| `@/scripts/get-static-image-url` | 同名 |
| `@/scripts/upload` `uploadFile/uploads`、`@/scripts/select-file` `selectFiles` | drive 任务实现 |

## 4. 原子组件映射（Firefish 模板 → 我们的组件）

`MkButton`→`CfButton`、`MkInput`→`CfInput`、`MkTextarea`→`CfTextarea`、`MkSelect`→`CfSelect`、
`MkSwitch`→`CfSwitch`、`MkRadio`→`CfRadio`、`MkRadios`→`CfRadios`、`MkInfo`→`CfInfo`、
`MkKeyValue`→`CfKeyValue`、`MkFolder`→`CfFolder`、`MkPagination`→`CfPagination`（`pagination` prop +
`#default="{ items }"` + `#empty` 兼容）、`MkDateSeparatedList`→`CfDateSeparatedList`、
`MkTab`→`CfTab`、`MkModal`→`CfModal`（`web/src/os/CfModal.vue`）、`MkLoading`/`MkError` 用全局同名组件。

全局组件（`web/src/components/global/`，模板直接用、不要 import）：
`Mfm`、`MkA`、`MkAcct`、`MkAvatar`、`MkEmoji`、`MkUserName`、`MkEllipsis`、`MkTime`、`MkUrl`、
`MkLoading`、`MkError`、`MkPageHeader`、`MkSpacer`、`MkStickyContainer`。

## 5. 后端能力缺口 → 强制降级

| 能力 | 处理 |
|---|---|
| 自定义表情 URL 映射 | `note.emojis`/`reactionEmojis` 恒 `[]`；正文 `:shortcode:` 用 `api("emojis")` 注册表（`{emojis: EmojiSimple[]}`）按名字匹配，匹配不到就渲染纯文本 |
| UrlPreview | 无端点，`MkUrlPreview` 移植为渲染纯链接（或空） |
| kaTeX 公式 | 无 katex 依赖：`$[math]`/`\\(\\)` 按纯文本/代码渲染，不引库 |
| PhotoSwipe / 图片查看器 | 用 `os.popup` 自建简单查看或新标签打开；不做手势缩放 |
| blurhash / 缩略图 | `thumbnailUrl` 恒 null：图片直接用 `url`，无占位图 |
| 视频播放器 (vue-plyr) | 原生 `<video controls>` |
| 广告 / Sparkle 之外的重型 MFM | 广告直接不渲染 |
| Swiper 手势切时间线 | 只渲染当前 tab 的一个 `MkTimeline`，不做滑动 |
| 推荐时间线 / 列表 / 天线 / 频道时间线 | 无端点：不进入 tab 列表 |
| 通知类型 | 后端只有 `mention/reply/renote/quote/reaction`，其它分支删除 |
| `users/relation`、`mute/block`、`following/requests` | 无端点：相关菜单项与按钮不渲染 |

## 6. 本波各任务的写作用域（互不重叠）

| 任务 | 写作用域 |
|---|---|
| foundation-core | `web/src/store/**`、`web/src/config.ts`、`web/src/instance.ts`、`web/src/stream.ts`、`web/src/const.ts`、`web/src/components/index.ts`、`web/src/main.ts`、`web/src/scripts/{array,clone,copy-to-clipboard,please-login,reduced-motion,focus,use-tooltip,format-time-string,extract-mentions,index}.ts`、`web/src/filters/{user,note}.ts`、`web/src/directives/**` |
| mfm-globals | `web/src/components/global/**`、`web/src/components/{MkLink,MkCode,MkFormula,MkSparkle,MkGoogle}.vue`、`web/src/styles/mfm.scss`、`web/src/styles/index.scss` |
| note-core | `web/src/components/{MkNote,MkNoteHeader,MkNoteSub,MkSubNoteContent,MkNoteSimple,MkNotePreview,MkNoteDetailed,MkNotes,MkCwButton,MkPoll,MkPollEditor}.vue`、`web/src/scripts/{get-note-menu,get-note-summary,check-word-mute,extract-mfm,reaction-picker}.ts` |
| note-media | `web/src/components/{MkMedia,MkMediaList,MkMediaBanner,MkMediaCaption,MkReactionsViewer,MkReactionsViewer.reaction,MkReactionsViewer.details,MkReactionIcon,MkReactionTooltip,MkStarButton,MkStarButtonNoEmoji,MkRenoteButton,MkQuoteButton,MkVisibility,MkInstanceTicker,MkAvatars,MkUrlPreview,MkImgWithBlurhash,MkFileTypeIcon}.vue`、`web/src/scripts/get-static-image-url.ts` |
| timeline | `web/src/components/MkTimeline.vue`、`web/src/pages/{Timeline,Note,IndexRoute,UserListTimeline,AntennaTimeline}.vue`、`web/src/scripts/{sound,scroll,page-metadata,device-kind}.ts` |
| notifications | `web/src/components/{MkNotification,MkNotificationSettingWindow,MkNotifications,MkFollowButton}.vue`、`web/src/pages/{Notifications,FollowRequests}.vue` |
| drive | `web/src/components/MkDrive*.vue`、`web/src/pages/{Drive,AttachedFiles}.vue`、`web/src/scripts/{upload,select-file}.ts`、`web/src/filters/bytes.ts` |

跨域依赖只用契约里的公开接口；需要的文件不存在时先写代码、最后再回头对齐，
**不要**去改别人作用域里的文件，发现契约不成立就在任务评论/消息里说明。

## 7. API 类型要点（后端契约，见 docs/research/api-contract/）

- `NoteDetailed`/`Note`：`reactionEmojis`、`emojis` 恒 `[]`；`files: DriveFile[]`；`poll?`；
  `myReaction?`；`hasPoll?: true`；`reply?/renote?` 只在详情端点出现。
- `UserLite`：`bannerUrl/description/location` 等为 `null`，不要假设有值；
  `avatarBlurhash`、`instance` 恒 `null`。
- 分页：`untilId` 向下、`sinceId` 单独用会翻成 ASC（`api/pagination.ts` 已封装）。
- 反应：`notes/reactions/create` `{noteId, reaction}`、`delete` `{noteId, reaction?}`；
  反应名单 `notes/reactions` → `NoteReaction[]`。
- 通知：`POST /api/notifications`（`notifications` 常量）。`notifications/mark-all-as-read`。
- 表情注册表：`api("emojis") → { emojis: EmojiSimple[] }`，字段 `name/url/category/aliases/isSensitive`。
- 时间线：`notes/timeline`(home)、`notes/local-timeline`、`notes/hybrid-timeline`(social)、
  `notes/global-timeline`；频道 `homeTimeline/localTimeline/hybridTimeline/globalTimeline`。

## 8. 验收（每个任务收尾必须自己跑）

```bash
cd web && pnpm typecheck   # vue-tsc --noEmit，最后必须 exit 0（只看自己文件，别人未完成的忽略）
cd web && pnpm build       # 构建成功
```

- `vue-tsc` 报错若在别人的文件里：忽略，但在自己的最终汇报里列出。
- 构建通过不代表运行时正确；浏览器走查由 Lead 统一做。
- 完成后向 Lead 汇报：改了哪些文件、新增了哪些公开接口、已知降级点、`vue-tsc` 状态。
