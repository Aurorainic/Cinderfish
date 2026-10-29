# Task F8 报告：MFM 函数补全

## 实现内容

在 `web/src/lib/mfm.ts` 的渲染管线末尾（`<br />` 转换之后、占位符还原之前）新增了 MFM 函数分发层，`renderMFM(text: string): string` 签名不变。

- **支持的函数**（与 brief 一致）：`$[x2|x3|x4]`、`$[spin]`（含 `left`/`alt`/`alternate` 方向参数、`x`/`y` 轴参数、`speed=` 时长参数）、`$[shake]`、`$[jump]`、`$[blink]`、`$[rainbow]`、`$[blur]`、`$[center]`、`$[small]`、`$[fg.color=hex]`、`$[bg.color=hex]`、`$[font.monospace]`/`$[font.serif]`、`$[plain]`。
- **降级**：未支持的函数名、畸形 token（含非法字符）、无效颜色值（`$[fg.color=zzz]`）一律渲染其内文（内文已被前面的管线渲染为 HTML，符合 Review 口径）。未闭合的 `$[` 保持字面文本。
- **动效**：新建 `web/src/styles/mfm.css`，全部动效用 CSS class + `@keyframes`（`mfm-spin`/`mfm-shake`/`mfm-jump`/`mfm-blink`/`mfm-rainbow`），并包 `@media (prefers-reduced-motion: reduce)` 停用全部动画类（blur/颜色/字体为静态效果保留）。spin 的 `left`/`alternate`/`speed` 通过内联 `animation-direction`/`animation-duration` 表达，speed 值经 `/^\d*\.?\d+(ms|s)?$/` 白名单校验，杜绝 style 注入。
- **嵌套**：函数分发为自顶向下递归（`$[x2 $[blur y]]` 正常渲染），深度上限 40 层防病态输入。
- **占位符 stash 管线未动**：代码块/行内代码的 stash 与 escape 语义保持原样；函数层在所有转义之后运行，产出的 HTML 直接内联（无需 stash），用户文本无法注入标签或属性（`<script>`、引号、`;` 等均已被前序 escape/校验拦截）。
- `:shortcode:` 维持纯文本（mfm-emoji span），与 brief 一致。

## RED → GREEN 证据

Brief 未要求测试步骤（无测试框架配置），验证为 typecheck + build + 手动样例核对。过程中发现并修复一个真实缺陷：

1. **RED**：首版实现中 `findFunctionClose` 深度从 0 起算（漏掉已消费的 `$[`），`pnpm typecheck && pnpm build` 通过但样例核对全部输出字面 `$[x2 doubled]`，嵌套用例 `$[x2 $[blur nested]]` 输出 `<span class="mfm-x2">$[blur nested</span>]`（闭合错位）。
2. **GREEN**：将起始深度改为 1 后，34 个样例用例（x2/x3/x4、spin 各参数、shake/jump/blink/rainbow/blur/center/small、fg/bg 合法与非法 hex、font.monospace/serif、plain、未知函数降级、嵌套、闭合/未闭合、script 注入、URL 含尾随 `]`）全部输出正确 HTML。

## 文件变更

| 文件 | 变更 |
|------|------|
| `web/src/lib/mfm.ts` | 新增函数分发层（renderFunctions/renderFunctionCall/findFunctionClose/sanitizeDuration），URL 尾部标点裁剪加入 `]`（使 `$[x2 https://…/]` 正确闭合），spin 支持 `alt` 别名 |
| `web/src/styles/mfm.css` | 新建：函数样式 + 5 组 keyframes + prefers-reduced-motion 守卫 |
| `web/src/main.ts` | 仅新增 `import "./styles/mfm.css";` |

## 验证命令与结果

- `cd web && pnpm typecheck` → 通过（vue-tsc 无错误）
- `cd web && pnpm build` → 通过（vite build，99 modules，✓ built）
- 手动样例核对：34 用例全对（详见上节；用项目自带 `tsc` 编译 `src/lib/mfm.ts` 至 /tmp 后以 node 运行，未新增任何依赖）

## Concerns

1. **`$[plain]` 的语义近似**：因函数层在所有 transform 之后运行，`$[plain **raw**]` 内文的 bold/链接等已被处理（输出保留 `<strong>`）。Misskey 的 plain 会禁用内文 MFM。完全实现需把函数解析前移到 escape 之前，会改动现有管线顺序，与「保持占位符 stash 安全管线与 escape 语义」的约束冲突，故按近似实现并在报告中说明。
2. **URL 尾部 `]` 裁剪**：为使函数内的 URL 正确闭合，URL 尾部标点裁剪加入 `]`（与 Misskey 的 URL 正则不吸收 `]` 的行为一致）。以 `]` 结尾且确属 URL 一部分的极端 URL 会被裁掉尾字符——影响面极小。
3. 评审者可用上方样例清单直接复核输出。
