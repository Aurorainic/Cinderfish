# 任务 F3 完成报告：os.ts 命令总线 + PageWindow

日期：2026-10-29 · 提交：`fce5711` `feat(web): command bus (os.ts) and page window overlay`

## 实现内容

### `web/src/os.ts`（新建）

命令式 UI 总线 + 单例弹层容器，全部为纯 TypeScript（渲染用 `h()`，无需额外 SFC 文件，避免越出本任务文件所有权）：

- `openPostForm(opts: { reply?: Note; renote?: Note; quote?: boolean }): void` — **签名已冻结**。当前渲染占位 stub 面板（`PageWindow` 标题「发帖」，正文显示回复/引用上下文与「发帖器尚未接入，此为占位面板（任务 F9 替换为真实组件）」提示），F9 直接替换 stub 内容即可。
- `confirmDialog(message: string): Promise<boolean>` — **本任务即完整可用**：返回 Promise，点「确定」resolve `true`，点「取消」、点遮罩或按 Esc resolve `false`；并发的确认弹窗按 FIFO 排队，逐个应答。
- `OsOverlay` 单例容器组件：挂在 App.vue 根部；无命令时渲染 `null`。内部用模块级 `ref` 状态（`postFormRequest` / `confirmQueue`）替代传统事件回调总线，容器直接订阅响应式状态——语义等价（命令函数=事件发射，容器=监听者），且避免手动解绑。
- Esc 处理在容器层：先应答最上层确认弹窗（false），否则关闭发帖面板；`PageWindow` 自身也监听 Esc 以支持 F5 独立使用，两层行为幂等无冲突。
- 组件卸载时移除 window keydown 监听（`onUnmounted`）。
- 确认弹窗与遮罩样式使用内联 `CSSProperties`（`--panel`/`--border`/`--accent`/`--on-accent`/`--shadow-sm` 等 token），全部留在 os.ts 文件内，避免写他人所有的 main.css。

### `web/src/components/ui/PageWindow.vue`（新建）

- props `{ title: string }`，emits `close`，默认 slot。
- 结构：遮罩（`@click.self` 点遮罩关闭）→ 窗口卡片：标题栏（标题 + `AppIcon name="close"` 关闭按钮，带 `:focus-visible` 描边）+ 滚动正文区。
- 样式：`var(--panel)` + `var(--shadow-sm, 0 4px 16px rgba(0,0,0,.3))`（fallback 字面值：F1 尚未提交 `--shadow-sm` token 时保持可用；F1 落地后自动取 token）。宽屏 720px、高 80vh 居中；≤760px 窄屏全屏（`100dvh`，无圆角无边框）。

### `web/src/App.vue`（修改，最小改动）

+2 行：`import { OsOverlay } from "./os"` 与模板尾部 `<OsOverlay />`。

## 验证证据（RED→GREEN 说明）

本任务简报未列测试步骤（web 无测试套件，package.json 无 test 脚本），故无 TDD 环节；验证以简报指定命令为准：

```
$ cd web && pnpm typecheck   # vue-tsc --noEmit — 0 错误
$ pnpm build                 # vue-tsc --noEmit && vite build — ✓ built in 378ms
```

端到端人工核对点：OsOverlay 挂载且空态渲染 null（不影响现有页面）；openPostForm/confirmDialog 的调用方式符合冻结签名。

## 变更文件

- 新建 `web/src/os.ts`（约 170 行）
- 新建 `web/src/components/ui/PageWindow.vue`
- 修改 `web/src/App.vue`（+2 行）

## 疑虑与备注

1. **F9 接入点**：stub 面板渲染在 `OsOverlay` 的 render 函数内（`os.ts` 底部）；F9 改 os.ts 时把 default slot 换成 `PostForm` 即可，签名与容器结构不变。
2. **`--shadow-sm` fallback**：F1 落地 token 后，PageWindow/confirm 的 fallback 字面值自动失效但保留；如评审要求可删 fallback，需再次触碰我方两个文件。
3. **PageWindow Esc 与遮罩点击均 emit close**：F5 在 deck 模式嵌入时如需阻止遮罩关闭需加 prop（预留问题，未越界实现）。
4. 并发说明：`web/src/lib/mfm.ts` 当时被 F8 修改中，未触碰；仅提交本任务三个所属文件。
