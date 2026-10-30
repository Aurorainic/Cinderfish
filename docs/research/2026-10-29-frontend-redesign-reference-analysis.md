# 前端界面重构：参考源码分析（2026-10-29）

背景：Cinderfish 现有前端被评价为「AI 味过重的 Mastodon 克隆」。本文记录为界面设计改造
讨论准备的三份参考源码分析结论。分析通过只读子代理完成，重点覆盖布局、组件组织、主题
系统与视觉语言四个维度。

## 参考边界（2026-10-30 用户校正）

三个项目的角色互不重叠，混用会导致实现跑偏：

| 项目 | 角色 | 允许用作 | 禁止用作 |
|------|------|----------|----------|
| Misskey | **后端 API 契约的权威来源** | 端点 `paramDef`/`res` schema、字段名、枚举值、错误信封、Streaming 帧格式 | 界面、组件结构、布局、视觉语言 |
| Firefish | **前端体验的主要参考** | 壳形态、笔记组件拆分、deck、交互模式、主题语义粒度 | 后端行为 |
| Iceshrimp | **前端体验的补充参考** | 紧凑布局、断点取舍、组件粒度、日期分隔列表 | 后端行为 |

一句话：**抄 Misskey 的后端契约，做 Firefish/Iceshrimp 的前端。** 前端的验收标准是
Firefish/Iceshrimp 的信息密度与交互，不是 Misskey 的文件结构或皮肤。Cinderfish 的
ember 橙、Inter 字体、手写 SVG 图标、自有 CSS token 与双主题机制都是保留项，不算偏离。

已有执行包（`docs/superpowers/plans/2026-10-29-frontend-redesign/`）里的任务简报与报告是
在该校正之前写的，其中「Misskey 风 / Misskey-style」一律指**反应系统与术语体系这类契约层
内容**，不指界面移植。阅读历史文档时按此口径理解，不必回头改它们。

## 参考源码（稀疏克隆，位于 tmp/refcheck/，被 .gitignore 忽略，重启不丢）

| 项目 | 仓库 | 版本 | 稀疏范围 |
|------|------|------|----------|
| Misskey | https://github.com/misskey-dev/misskey.git | `c05e844` (2026-10-29) | packages/frontend/src, packages/misskey-js/src |
| Firefish（原版） | https://codeberg.org/firefish/firefish.git | `02c31e0` (2024-12-11) | packages/client/src, packages/sw/src |
| Iceshrimp（Firefish fork，Vue） | https://codeberg.org/banaanihillo/iceshrimp.git（原 iceshrimp.dev 仓库的快照 fork；原仓库与 GitHub 镜像均已不可访问） | `1fa829ed10f4` (2025-07-25) | packages/client/src, packages/sw/src |
| Iceshrimp.NET（Blazor 重写） | https://codeberg.org/iceshrimp/Iceshrimp.NET.git（`dev` 分支） | `2c61359` (2026-10-27) | Iceshrimp.Frontend |

克隆命令（GitHub 需代理 `127.0.0.1:19823`，Codeberg 直连即可）。目标目录放在仓库内
`tmp/refcheck/`，`.gitignore` 已忽略 `tmp/`，重启不丢：

```bash
export https_proxy=http://127.0.0.1:19823 http_proxy=http://127.0.0.1:19823   # 仅 Misskey 需要
git clone --depth 1 --filter=blob:none --sparse <URL> tmp/refcheck/<name>
cd tmp/refcheck/<name> && git sparse-checkout set <range>   # 见上表
```

注：`git.joinfirefish.org` 与 `code.iceshrimp.dev` 经代理和直连均 TLS 握手失败，不可达。
Firefish 开发已中止（develop 停在 2024-12），但仍是「原版 Firefish 体验」的权威来源。
Iceshrimp 存在两条并行血脉：Vue 的 Firefish fork 与 Blazor 重写的 Iceshrimp.NET；
前者可复用组件结构，后者只能参考布局与交互取舍。
Codeberg 偶发 504，克隆失败时重试即可。

## 「AI 味 Mastodon」的具体成因

1. **卡片流**：每条 note 是 20px 圆角独立卡片 + 12px 间隙（`web/src/components/NoteCard.vue:241-247`），
   而 Firefish/Misskey 是同一面板内的密集行，靠 1px 分隔线与日期分隔区分。
2. **Mastodon 三栏壳**：固定 232/1fr/300 grid、右栏是「关于本站」信息卡，无 deck、无 widget、
   无可配置导航（`web/src/layouts/AppShell.vue:48-118`）。
3. **反应是「点赞」而非 Misskey 反应**：底部心形按钮 + 硬编码 8 个 emoji（`NoteCard.vue:32`）；
   Misskey 系是完整 emoji picker + 自定义 emoji + 反应者浮层 + 自己的反应置顶。
4. **MFM 退化成 Markdown**：`web/src/lib/mfm.ts` 仅 73 行正则（code/link/mention/hashtag/粗斜体），
   `:name:` 自定义 emoji 不渲染、无 `$[...]` 函数，Misskey 内容视觉上塌缩成纯文本。
5. **通用模板审美**：Inter 字体 + radial-gradient 光斑背景 + 大投影 + 全胶囊按钮（`main.css:59-64,140-150`），
   缺 Misskey/Firefish 的扁平面板、12px 圆角、14px 字号的密集感。
6. **发帖器与文案贫瘠**：原生 `<select>` 选可见性、无引用/投票/补全/草稿/预览；
   文案是「现在在想什么？」而非 Misskey 术语体系。

## 参考各项的分工要点

### Misskey（只作后端 API 契约来源）

- 端点契约：`packages/backend/src/server/api/endpoints/*/paramDef.ts|res.ts` 与
  `packages/misskey-js/src/autogen/types.ts`；错误信封、分页游标、`i` 认证字段都以源码为准。
- Streaming：`packages/backend/src/server/api/stream/` 的频道名与帧格式。
- 主题实现可**读取思路但不移植**：它把 JSON5 主题编译成 `--MI_THEME-*` 内联变量，
  Cinderfish 保留自己的 CSS token 机制，不做 JSON5 DSL。
- 明确不作为参考的：五套 shell 的组件划分、`Mk*` 命名体系、nirax 路由、`os.ts` 命令式总线。
  这些是 Misskey 前端的实现细节，照搬会把项目拖进它的整体架构。

### Firefish（前端体验的主要参考）

- 同样三 shell + deck；独特处：deck 内部导航走 `os.pageWindow` 弹窗而非换页（`ui/deck.vue:217-225`）。
- note 拆成 7 个部件（`components/note/MkNoteHeader|Content|Media|RenoteBar...`），单条 note 有键盘热键 `r/e/q/j/k`。
- 时间线 = `MkPagination` + stream + 「有新笔记」浮标 + `foldNotes` 线程折叠。
- 主题语义粒度最细：`scripts/theme.ts` 支持 `:darken/:lighten/:alpha/:hue/:saturate`，27 套内置主题大多
  只需覆盖十几个色值（`d-dark.json5` 仅 15 行）。**借它的 token 词表与语义分组，不借 JSON5 机制。**
- 笔记为无边框平铺行、48px 头像、hover 才显现 footer 操作（`MkNote.vue:409-417`）——去卡片化的第一要素。
- 断点：universal 移动 `<=500px`（smartphone UA 优先）、widget 栏脚本阈值 1100 / 样式隐藏 1090、
  侧栏 `<=1279px` 收成图标栏；note 用容器查询 500/350/300px（`ui/universal.vue:236,676`、`ui/_common_/navbar.vue:179`）。
- 字体 Atkinson Hyperlegible、图标 Phosphor、默认 accent 青柠绿 `#86b300`——这三项不跟，保留 Inter/手写 SVG/ember 橙。

### Iceshrimp（前端体验的补充参考）

Vue fork 版：
- 把 Firefish 的 note 9 件套拍平为 `MkNote + MkNoteHeader + MkSubNoteContent` 三件套，
  是「拆到什么粒度合适」的对照样本——Cinderfish 取中间粒度（header/content/media/footer）。
- 新增 `MkDateSeparatedList` 日期分隔、搜索过滤、Heatmap widget；删掉通知折叠/广告。
- 主题 registry 存服务端（`theme-store.ts`），换设备同步，适合个人实例——列为将来项。
- 反应条横向可滚、mask 渐隐出血；`MkStarButton` 一键默认反应。

Iceshrimp.NET（Blazor）版，只借布局取舍、不借代码：
- 主布局 `Layout/MainLayout.razor:23`：左导航 + 中内容 + 右 widget；`MainLayout.razor.css:1` 在
  <840px 隐藏 widget 并收起操作文字，<600px 为主内容预留底部导航空间——比 Firefish 更早撤退次要内容。
- 时间线把刷新/回顶/时间线选择放在页顶操作区，选择器列出 home/local/social/bubble/global/remote/tag。
- 通知按动作类型给图标与文案，关联 note 内嵌，已读项降不透明度、关联 note 加边框。

## 现有前端可保留的骨架

1. token 化主题 + `main.ts` 首帧前解析（可低成本接入主题编译模型）。
2. 单一 `stores/streaming.ts` 订阅式客户端——deck 每列只需 `subscribe(channel)`。
3. `TimelineFeed` 以 `source` 参数化——天然可复用为 deck 列。
4. MFM 占位符 stash 管线（`lib/mfm.ts:6-36`）——结构正确，可安全扩展函数/emoji/KaTeX。
5. `api/types.ts` 已含 `reactionEmojis/emojis/pinnedNotes` 等 Misskey 字段——自定义 emoji、反应
   picker、置顶笔记可纯前端补齐。
6. 乐观反应切换逻辑（换反应 = 减旧 + 加新，`NoteCard.vue:67-96`）可复用。

## 建议的改造路线（已被设计文档采纳并执行）

| 步骤 | 内容 | 前端参考依据 |
|------|------|--------------|
| 1 | 主题系统：token 扩为 Firefish 级语义粒度 | firefish `scripts/theme.ts` + `themes/_dark.json5` |
| 2 | NoteCard 拆件 + 去卡片化（无边框行 + 分隔线 + hover 显操作） | firefish `components/note/*` |
| 3 | MFM 渲染补全：自定义 emoji、`$[x2]/$[spin]/$[blur]` | firefish `MkMfm`（语法由 Misskey 后端定义） |
| 4 | 反应 picker 单例 + 自己的反应置顶 | iceshrimp `MkEmojiPickerDialog` + firefish `MkReactionIcon` |
| 5 | 发帖器改为全局 props 驱动对话框，补 `specified`/草稿/引用 | firefish `os.post` |
| 6 | shell：universal + deck 双模式，自适应 | 两家 `ui/deck.vue`、`ui/universal.vue` |

## 待讨论问题（2026-10-30 已全部结案）

| 问题 | 结论 |
|------|------|
| 是否做 deck（多列）？ | 做。deck 是壳偏好而非独立 URL，宽度足够时多列并排、窄视口单列横滑。 |
| 主题粒度：照搬 Firefish ~95 token 还是裁剪？ | 裁剪到 Cinderfish 实际用到的子集，按 Firefish 的语义分组命名，不引入 JSON5。 |
| 是否引入 i18n 层？ | 首版不做，中文硬编码。 |
| 字体与图标：Inter + 手写 SVG 还是 Phosphor/Atkinson？ | 保留 Inter + 手写 SVG。 |
| ember 橙与 Firefish 绿并存还是替换？ | 保留 ember 橙。 |
| Misskey 前端是否作为 UI 参考？ | **不。Misskey 只提供后端 API 契约。** UI 参考限于 Firefish 与 Iceshrimp。 |
