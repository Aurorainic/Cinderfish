# 移动端导航与浮层体验修复设计（2026-11-08）

## 意图

修复公网生产环境移动视图中发现的设置导航、二级菜单、底部导航、发帖框窄屏控件与工具提示问题。前端是 Cinderfish 正式组成部分，本仓库允许持续开发前后端；本批次不触碰公网生产环境。

## 目标

- 在移动端打开 `/settings` 默认进入 `/settings/profile`。
- 设置子页切换时，窄屏设置导航保持可见，不要求先返回菜单。
- 二级弹出菜单的图标、文字与点击目标在触屏尺寸下清晰易操作。
- 底部导航链接 hover/focus/active 不显示装饰性下划线；可见键盘焦点保持明确。
- 发帖框在 320–412 CSS px 范围内，头部按钮和输入控件不互相覆盖；底部工具按钮能够换行或自适应。
- 发帖框工具提示绘制在发帖框/模态内容之上，并在视口边缘正确避让。

## 非目标

- 不重设计整套 Firefish 风格客户端或桌面导航。
- 不改变后端 API、发帖行为、弹层关闭语义或设置路由。
- 不在生产网站直接操作或部署。

## 当前证据

- `web/src/pages/SettingsIndex.vue` 窄屏隐藏子页菜单，且 `redirectIfNeeded()` 对窄屏主动跳过 `/settings` 重定向。
- `web/src/os/CfPopupMenu.vue` 菜单字号为 `0.95em`，行 padding 为 9×12px。
- `web/src/styles/_reset.scss` 对所有锚点 hover 都启用文字装饰色，底部导航 `CfA` 因此出现下划线。
- `web/src/components/MkPostForm.vue` 头部使用固定尺寸/间距，footer 按钮有最小 44–48px 尺寸；需以窄宽度验证真实溢出来源并针对断点修复。
- `web/src/components/CfTooltip.vue` 使用独立 popup 的 z-index；需核对 popup/modal z-index 生成顺序及定位尺寸，解决模态中的提示被遮挡问题。
- `web/` 没有匹配到现成的 test/spec 文件；现有验收命令为 `pnpm typecheck`、`pnpm run check:port`、`pnpm build`。

## 方案

考虑过局部放大/覆盖样式，与修复共享菜单、tooltip、通用链接样式。选择共享组件与状态根因修复：保留桌面当前结构，窄屏设置页同时展示可横向滚动的设置标签与当前子页面；`/settings` 按窄宽度导航到资料页；菜单对小屏增强图标/字级和触控尺寸；链接装饰仅对按钮/导航类链接关闭；发帖框使用容器断点和可收缩布局；tooltip 的层级高于其来源弹层，定位时测量实际尺寸并夹取到 viewport。每个 CSS/逻辑改动以项目现有构建、移植门禁和 320/360/390/412px 浏览器检查验证，避免无必要依赖与通用重构。

## 实现责任与验收

- 更新 `AGENTS.md`：删除 2026-11-04 “前端仅后端”范围锁定，保留当前 Firefish 移植基线、来源/许可规则与 API 边界；使规则明确允许前后端并行迭代。
- 设置页：`web/src/pages/SettingsIndex.vue`。验收：窄屏访问 `/settings` 最终到 `/settings/profile`；打开任一已列设置子页时标签仍可见，点另一标签直接切换；桌面行为不变。
- 通用菜单与导航：`web/src/os/CfPopupMenu.vue`、`web/src/styles/_reset.scss`（必要时 `UniversalLayout.vue`）。验收：移动端菜单文字/图标清楚、每行有足够触控高度；底部导航 hover 不加下划线且键盘焦点清晰。
- 发帖框：`web/src/components/MkPostForm.vue` 及必要的 `web/src/os/MkPostFormDialog.vue`。验收：在 320/360/390/412px 下计数、隐私、预览、发送按钮互不遮挡，输入区域宽度不溢出，工具栏不覆盖正文。
- 工具提示：`web/src/components/CfTooltip.vue`、`web/src/os/state.ts`（仅在查明层级竞态时）及必要的 popup 容器。验收：鼠标与键盘提示均在 modal/form 上方且不被裁切，四边靠近视口时仍完整可读；不改变 tooltip 延时与关闭交互。
- 验收命令：在 `web/` 执行 `pnpm typecheck`、`pnpm run check:port`、`pnpm build`；随后用 320、360、390、412 CSS px 移动视口走查上述交互，并确认构建没有留下不需要的新产物。

## 约束与默认决定

- 沿用 Vue 3、TypeScript、Vite、现有 SCSS 与 Phosphor 图标，不加依赖。
- 可访问性优先于移除全部视觉反馈：只去掉装饰性下划线，保留 `:focus-visible`。
- 桌面版设置布局及其它桌面导航外观不变。
- 代码实现遵循当前 AGPL-3.0 来源头与 `docs/HANDOVER.md` 引用规则；只有改写/移植来源文件时才补具体 source entry。
