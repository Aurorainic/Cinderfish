# 前端界面重构：参考源码分析（2026-10-29）

背景：Cinderfish 现有前端被评价为「AI 味过重的 Mastodon 克隆」。本文记录为界面设计改造
讨论准备的三份参考源码分析结论。分析通过只读子代理完成，重点覆盖布局、组件组织、主题
系统与视觉语言四个维度。

## 参考源码（稀疏克隆，位于 /tmp/refcheck/，重启后需重建）

| 项目 | 仓库 | 版本 | 稀疏范围 |
|------|------|------|----------|
| Misskey | https://github.com/misskey-dev/misskey.git | `8a6a71a6a7d4` (2026-10-29) | packages/frontend, frontend-shared, frontend-embed, misskey-js |
| Firefish（原版） | https://codeberg.org/firefish/firefish.git（develop 分支） | `02c31e0a2804` (2024-12-11) | packages/client |
| Iceshrimp（Firefish fork） | https://codeberg.org/banaanihillo/iceshrimp.git（原 iceshrimp.dev 仓库的快照 fork；原仓库与 GitHub 镜像均已不可访问） | `1fa829ed10f4` (2025-07-25) | packages/client |

克隆命令（GitHub 需代理 `127.0.0.1:19823`，Codeberg 直连即可）：

```bash
git clone --depth 1 --filter=blob:none --sparse <URL> /tmp/refcheck/<name>
cd /tmp/refcheck/<name> && git sparse-checkout set packages/client
```

注：`git.joinfirefish.org` 与 `code.iceshrimp.dev` 经代理和直连均 TLS 握手失败，不可达。
Firefish 开发已中止，develop 分支停在 2024-12；这仍是「原版 Firefish 体验」的权威来源。

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

## 三份参考各自的要点

### Misskey（API 契约权威，UI 亦最完整）

- 五套 shell（universal/deck/minimum/visitor/zen），deck 列宽 330、可拖拽/堆叠/多 profile（`src/ui/deck.vue`）。
- `MkNote` 纯 UI 编排 + `composables/use-note.ts` 承载逻辑；用 **container query**（非媒体查询）适配
  350px 窄列，是 deck 化的前提（`MkNote.vue` 尾部）。
- `MkPagination` + `MkStreamingNotesTimeline`：分页/无限滚动/下拉刷新/流式前插全部统一。
- 主题：`frontend-shared/themes/_light.json5` 约 70 个 token，支持 `:alpha<0.15<@accent` 派生运算，
  运行时编译为 `--MI_THEME-*` CSS 变量（`src/theme.ts:146-171`）。
- `src/os.ts` 命令式 UI 总线：`os.post()/os.popup()/os.menu()` 统一入口。
- 反应胶囊 42px、自反应 accent 底 + inset 描边；`MkReactionEffect` 飘字特效。

### Firefish（产品定位基准）

- 同样三 shell + deck；独特处：deck 内部导航走 `os.pageWindow` 弹窗而非换页（`ui/deck.vue:217-225`）。
- note 拆成 7 个部件（`components/note/MkNoteHeader|Content|Media|RenoteBar...`），单条 note 有键盘热键 `r/e/q/j/k`。
- 时间线 = `MkPagination` + stream + 「有新笔记」浮标 + `foldNotes` 线程折叠。
- 主题 DSL 最成熟：`scripts/theme.ts` 支持 `:darken/:lighten/:alpha/:hue/:saturate`，27 套内置主题大多
  只需覆盖十几个色值（`d-dark.json5` 仅 15 行）。
- 笔记为无边框平铺行、48px 头像、hover 才显现 footer 操作（`MkNote.vue:409-417`）——去卡片化的第一要素。
- 字体 Atkinson Hyperlegible、图标 Phosphor、默认 accent 青柠绿 `#86b300`。

### Iceshrimp（fork 上的取舍参考）

- 把 Firefish 的 note 9 件套拍平为 `MkNote + MkNoteHeader + MkSubNoteContent` 三件套（978 行），
  是「拆到什么粒度合适」的对照样本。
- 新增 `MkDateSeparatedList` 日期分隔、搜索过滤、Heatmap widget；删掉通知折叠/广告。
- 主题 registry 存服务端（`theme-store.ts`），换设备同步，适合个人实例。
- 反应条横向可滚、mask 渐隐出血；`MkStarButton` 一键默认反应。

## 现有前端可保留的骨架

1. token 化主题 + `main.ts` 首帧前解析（可低成本接入主题编译模型）。
2. 单一 `stores/streaming.ts` 订阅式客户端——deck 每列只需 `subscribe(channel)`。
3. `TimelineFeed` 以 `source` 参数化——天然可复用为 deck 列。
4. MFM 占位符 stash 管线（`lib/mfm.ts:6-36`）——结构正确，可安全扩展函数/emoji/KaTeX。
5. `api/types.ts` 已含 `reactionEmojis/emojis/pinnedNotes` 等 Misskey 字段——自定义 emoji、反应
   picker、置顶笔记可纯前端补齐。
6. 乐观反应切换逻辑（换反应 = 减旧 + 加新，`NoteCard.vue:67-96`）可复用。

## 建议的改造路线（供讨论，未定案）

| 步骤 | 内容 | 参考依据 |
|------|------|----------|
| 1 | 主题系统：token 扩为 Firefish 级语义粒度 + 派生运算 | firefish `scripts/theme.ts` |
| 2 | NoteCard 拆件 + 去卡片化（无边框行 + 分隔线 + hover 显操作） | firefish `components/note/*` |
| 3 | MFM 渲染补全：自定义 emoji、`$[x2]/$[spin]/$[blur]` | misskey `MkMfm` |
| 4 | 反应 picker 单例 + 自定义 emoji + 反应者浮层 | iceshrimp `MkEmojiPickerDialog` |
| 5 | 发帖器改为全局 props 驱动对话框，补 `specified`/草稿/引用 | firefish `os.post` |
| 6 | shell 评估：universal + deck 双模式（先 universal，deck 二期） | 两家 `ui/deck.vue` |

## 待讨论问题

1. 是否做 deck（多列）？最小路径是先保留单列 + `os.pageWindow` 式弹窗导航。
2. 主题粒度：照搬 Firefish ~95 token 还是裁剪到 Cinderfish 实际用到的子集？
3. 是否引入 i18n 层（当前中文硬编码），还是首版只面向中文用户？
4. 字体与图标：是否沿用 Inter + 手写 SVG，还是换 Phosphor/Atkinson？
5. Cinderfish ember 橙强调色与 Firefish 绿并存还是替换？
