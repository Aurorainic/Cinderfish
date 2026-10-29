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

## 评审修复（commit 2）

评审发现一个 Important 缺陷并已修复（focused fix commit）：

1. **Important — 函数层在链接生成之后运行导致 href 属性注入**：形如 `https://x.test/a]$[x2]b` 的笔记会让函数层把 `$[x2]` 替换成 span 标记并插入已生成的 `<a href="…">` 属性内部。**修复**：把函数层移到 escape 之后、URL/mention/hashtag 链接生成之前运行；函数产出的标记先内联，后续链接层再把函数内容中的 URL/mention/hashtag 正常链接化。为此 mention/hashtag 的前置字符集加入 `>`（用户输入的 `>` 已被 escape 成 `&gt;`，字面 `>` 只可能来自函数标记，用于函数内容开头处的 @/# 链接化）。同时撤销了上一版为绕过此问题加的 URL 尾部 `]` 裁剪（根因已除，恢复原 URL 语义）。
2. **Minor — HEX_COLOR 过宽**：`/^[0-9a-fA-F]{3,8}$/` 改为只接受 3、4、6、8 位（CSS 合法形式）；`$[fg.color=12345]` 降级为内文。
3. **Minor — `$[plain]` 空壳**：现在真正抑制内文 MFM——plain 内容不递归函数层，并用现有 stash 管线保护其不受后续链接/加粗/emoji pass 影响（内容中的代码块占位符先经 restore 还原再 stash）；空 CSS 规则 `.mfm-plain {}` 已从 mfm.css 删除。

### 回归用例（修复后全部通过）

- `https://x.test/a]$[x2]b` → `<a class="mfm-link" href="https://x.test/a]" …>https://x.test/a]</a><span class="mfm-x2"></span>b`（href 内无 span 标记）
- `https://x.test/$[x2]y`、`https://x.test/a]$[spin.left]b`、`https://x.test/a]$[fg.color=ff0000;style=x]b` → 同样安全，函数标记不会进入 anchor 属性
- `$[x2]b" onmouseover="alert(1)` → 引号已 escape，无属性逃逸
- `$[x2 https://x.test/a]` / `$[x2 @user]` / `$[x2 #tag]` / `$[blur @user https://x.test/a #tag]` / `$[x2 $[blur @user]]` → 函数内容内 URL/mention/hashtag 正常链接化（含嵌套）
- `$[plain untouched **raw** $[x2 hidden]]` → `<span class="mfm-plain">untouched **raw** $[x2 hidden]</span>`（内文无 strong/span）
- `$[fg.color=12345 five]` → 降级；`$[fg.color=abcd four]`、`$[fg.color=12345678 eight]` → 接受
- 原有 34 个样例用例全部保持正确输出。

## 文件变更

| 文件 | 变更 |
|------|------|
| `web/src/lib/mfm.ts` | 函数分发层（renderFunctions/renderFunctionCall/findFunctionClose/sanitizeDuration），评审修复后函数层位于 escape 之后、链接生成之前；spin 支持 `alt` 别名；HEX_COLOR 限定 3/4/6/8 位；`$[plain]` 真实抑制内文 MFM（stash 保护） |
| `web/src/styles/mfm.css` | 新建：函数样式 + 5 组 keyframes + prefers-reduced-motion 守卫（评审后删除空 `.mfm-plain` 规则） |
| `web/src/main.ts` | 仅新增 `import "./styles/mfm.css";` |

## 验证命令与结果

- `cd web && pnpm typecheck` → 通过（vue-tsc 无错误）
- `cd web && pnpm build` → 通过（vite build，✓ built）
- 手动样例核对：初版 34 用例；评审修复后扩至 48 用例（含 anchor 属性注入回归用例）全部正确（用项目自带 `tsc` 编译 `src/lib/mfm.ts` 至 /tmp 后以 node 运行，未新增任何依赖）

## Concerns

1. 函数层现在位于 escape 之后、链接生成之前：这是评审指定的两种方案之一，URL/mention/hashtag 在函数内容内仍正常链接化（用例已覆盖）。函数内容中的代码块占位符随占位符管线在最终一步还原。
2. mention/hashtag 正则前置字符集加入 `>`：用户输入的 `>` 已被 escape（`&gt;`），字面 `>` 只来自本文件生成的标记，不影响纯文本行为。
3. 未闭合的 `$[` 保持字面文本；深度上限 40 层防病态输入。
4. 评审者可用上方用例清单直接复核输出。
