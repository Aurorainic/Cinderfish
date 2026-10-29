# Cinderfish 前端界面重构设计（2026-10-29）

状态：待用户确认。背景与源码分析见 `docs/research/2026-10-29-frontend-redesign-reference-analysis.md`。

## 意图与原因

现有前端被评价为「AI 味过重的 Mastodon 克隆」。本次重构使其贴近 Firefish/Misskey 的
产品语言：扁平密集的信息流、Misskey 式反应系统、MFM 呈现、自适应的壳（deck + 侧边系统）。
产品定位不变：个人或极小规模联邦实例，前端体验接近 Firefish。

## 决策记录（讨论中已确认）

| 决策点 | 结论 | 来源 |
|--------|------|------|
| 主题系统 | 保留现有机制：CSS 变量 token + `data-theme` 双主题 + `localStorage`，不引入 JSON5 主题 DSL；token 词表按新 UI 需要扩充（见下） | 用户明确 |
| UI 框架 | 布局、组件形态、交互模式跟随 Misskey/Firefish | 用户明确 |
| 壳 | 自适应：一个 deck 列容器贯穿两种视口。宽视图默认 universal（可折叠侧栏 + 主列 + 右 widget 栏）；窄视图单列横滑。deck 作为可切换的第二形态保留 | 用户明确 |
| 移动端 deck 的列 | 自适应：列数随视口自动增减（窄=1 列横滑，宽=多列并排），不区分「移动版/桌面版」两套组件 | 用户明确 |
| 范围 | 总体重构，不采用壳先行或内容先行的分期取舍；按里程碑排序推进 | 用户明确 |
| 强调色 | 保留 ember 橙（现有 theme 的一部分，「保留友善」连带配色） | 我的判断 |
| 字体/图标 | 维持 Inter + 手写 SVG 图标集；去 AI 味靠去渐变光斑、去大投影、降圆角实现，不依赖换字体 | 我的判断 |
| 后端范围 | 最小后端打包：把已有数据（notes `files` 从 `fileIds`、用户 `name`/`avatarUrl`/`host`、`pinnedNotes`）填充进现有 API 响应；自定义 emoji 注册表、`notes/reactions` 列表端点等新后端特性推迟 | 用户拍板（推荐档） |

## 目标

1. 壳层自适应：deck 列容器（列数随视口）+ universal 侧边系统（宽视图默认形态）。
2. 笔记呈现 Firefish 化：无边框平铺行 + 1px 分隔线 + 日期分隔 + hover 显操作栏 + renote bar。
3. 反应系统 Misskey 化：完整 emoji picker（unicode + 实例自定义 emoji）、自己的反应置顶、
   反应胶囊视觉对齐 MkReactionsViewer（自反应 accent 底 + inset 描边）。
4. MFM 补全：自定义 emoji 图片渲染 + 常用 `$[...]` 函数（x2/x3/x4、spin、shake、jump、blink、
   blur、rainbow、font、center、small、fg/bg、plain），沿用现有占位符 stash 安全管线。
5. 组件层统一：一个通用分页组件（无限滚动 + 流式前插 + 重连重载）供时间线/通知/探索复用；
   NoteCard 拆为 header/content/media/footer 子组件。
6. 发帖器全局化：props 驱动的全局对话框（`os.post` 模式），时间线顶部固定形态复用同一组件。
7. 全局去 AI 味：去 body 径向渐变光斑、去大投影、圆角降到 12px 级、文案改 Misskey 术语体系
   （笔记/转贴/反应），保留中文界面。

## 非目标（首版不做）

- 自定义 emoji 注册表与 admin 管理、第三方主题导入、主题编辑器——token 词表为将来留接口，机制先不做。
- i18n 层——首版维持中文硬编码。
- antenna、clips、频道、列表等需要新后端 API 的列类型。
- 「谁反应了」浮层——现有后端无 `notes/reactions` 列表端点且本期不加；picker 数据
  来源为本地 unicode 集（自定义 emoji 待后端注册表落地后接入）。
- 后端改动仅限最小打包层（见决策记录）：不新增端点、不改 schema。

## 反目标（出现即失败）

- 界面退回卡片流 / Mastodon 观感。
- 主题机制被替换或引入体积大的新依赖（不引入 mfm-js、UI 框架、虚拟滚动库）。
- 壳在窄视口出现横向滚动条或双列挤压。

## 约束

- 技术栈不变：Vue 3 + TypeScript + Vite + vue-router；现有 `stores/streaming.ts`、
  `api/client.ts`、乐观反应逻辑复用。
- 路由保持 vue-router（不引入 Misskey 的 nirax）。deck 化后 TimelineView 的页内 `source`
  状态改为列配置，路由语义维持 `/`（或 `/deck`）入口。
- 笔记组件在窄列（~350px）与主列（~680px）下都可用：用 container query 适配（参照 Misskey）。
- 可访问性底线：操作按钮 hover 显隐必须保持 focus-visible 可见；Firefish 式单条笔记热键
  （r/e/q/j/k）列为可选增强。

## 方案取舍（已定）

- **壳先行的分期方案**：见效慢，被用户否决。
- **仅换肤（不动组件结构）**：卡片流、单体 NoteCard 的结构性问题不消失，否决。
- **整体重构（本设计）**：采纳。里程碑顺序为执行建议而非硬约束，实现者可按依赖关系调整：
  1. **M1 基础**：主题 token 词表扩充 + 全局样式去 AI 味 + shell 骨架（deck 列容器 + universal）
  2. **M2 内容层**：NoteCard 拆件 + 去卡片化 + 分页组件 + 日期分隔 + renote bar
  3. **M3 交互层**：反应 picker + MFM 补全 + 发帖器全局化
  4. **M4 收尾**：widget 栏、通知页改造、profile 多分区、细节打磨
  M1 先行的理由：NoteCard 的去卡片化和列宽适配都依赖最终壳的度量（列宽、token 值），
  反过来则要二次返工。

## 我已拍板的架构细节（评审轮确定）

- deck 不是独立 URL，而是 shell 偏好（`localStorage`，如 `cinderfish.ui`）。universal 模式走
  正常路由；deck 模式下点笔记/头像采用 Firefish `os.pageWindow` 式弹窗导航，不改全局路由。
- 窄视口导航：底部栏（导航项 + 发帖入口合并），universal 与 deck 的窄形态一致。
- universal 主列顶部保留「首页/本地/社交/全局」四源切换 tab；deck 模式中同一功能在列头菜单。
- widget 栏首版：重样式现有实例卡 + 链接列表，不新增依赖后端数据的 widget。
- 发帖器：全局化 + 引用（renoteId+文本）+ 草稿自动保存；`specified` 可见性提供，但收件人
  用手动 @handle 输入（后端无 users/search，不做补全）。

## 留给实现者的事项

- 新 token 的最终词表与命名（建议参照 Firefish `themes/_dark.json5` 的语义命名裁剪）。
- deck 列类型注册表（首版：home/local/hybrid/global 时间线 + 通知 + 探索），列配置持久化到
  `localStorage`（键名自定），列头菜单与「添加列」入口的交互细节。
- 分页组件的 props 契约与流式合并策略（可参照现有 `TimelineFeed.vue:104-141` 的重连重载逻辑）。
- MFM 函数的完整覆盖清单与降级行为（未知函数按纯文本渲染）。
- 组件文件布局（建议 `components/note/`、`components/deck/`、`components/ui/` 分域）。
- 通知页是否引入类型筛选（现有后端支持范围内）由实现者按工作量判断。

## 验证口径

- `pnpm typecheck` 与 `go vet`/`go test` 全绿。
- 视觉验收：宽视口（≥1100px）呈 universal 三区；窄视口单列横滑无横向滚动条；笔记行无边框、
  hover 显操作；反应 picker 可用且包含实例自定义 emoji。
- 手动检查 `prefers-color-scheme` 未设时默认主题与首帧无闪烁（沿用 `main.ts` 现有逻辑）。
