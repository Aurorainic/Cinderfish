# 前端参考重定基：Firefish 原版复原（2026-11-01）

## 决定

1. **放弃 Iceshrimp 作为前端参考。** 此前的计划（`2026-10-31-iceshrimp-architecture-rewrite.md`）和
   iceshrimp 前端/截图参考全部弃用。原因：Iceshrimp.NET 前端是 Blazor WASM 重写，视觉与
   Firefish 血统已经走样；Iceshrimp JS fork 也偏离了 Firefish 的原始密度与布局。
2. **Firefish 上游是唯一前端基准。** 参考源码固定为 `tmp/refcheck/firefish@02c31e0`
   （packages/client，AGPL-3.0，见 `2026-11-01-firefish-port-source-and-license-register.md`）。
   universal 布局（`src/ui/universal.vue`）、组件（`src/components/`）、默认主题
   （`src/themes/_dark.json5`、`_light.json5`、`d-ice.json5` 等）逐项对齐。
3. **后端契约不变。** Misskey `paramDef` / `res` / `api.types.ts` 仍是端点契约来源；
   Cinderfish 的 Go 后端继续按已固定的能力矩阵实现与隐藏端点。
4. **项目定位（最终表述）**：把已停止维护的 Firefish 用 Go 与轻量但保证质量的技术栈重写，
   并兼容最新且必要的 Misskey API。前端是 Firefish 的 Vue 3 + Vite 复原，不是重新设计。
5. **主题系统保留**：现有 token 层、浅/深切换与 `prefers-color-scheme` 自动跟随不变，
   但 token 取值改从 Firefish 默认主题变量推导（panel、panelHeader、navBg、divider、
   accent 等），保证首帧即"看起来就是 Firefish"。

## 复原范围（对照 Firefish）

| Firefish 部分 | Cinderfish 动作 |
|---|---|
| `ui/universal.vue` 三栏（导航 rail / sticky 内容 / widgets）+ 移动端底 tab 与抽屉 | 重排现有壳，对齐结构与断点行为 |
| `themes/*.json5` 变量集 | 映射进 `styles/main.css` token，未支持变量显式置空 |
| `MkTimeline` / `MkNotifications` / `pages/user/*` 页面结构 | 逐页密度与交互对齐，端点缺失部分按能力矩阵隐藏 |
| `MkRenoteButton`、`MkNoteFooter`、`MkPostForm` 交互 | 对齐菜单、tooltip 与乐观更新语义（unrenote 见审计 F-01 裁决） |
| Deck（`ui/deck.vue`） | 保留现有实现，列密度向 Firefish 对齐 |

## 审计 §8 待裁决项的执行决定（本轮起生效）

| # | 项 | 决定 |
|---|---|---|
| 1 | F-01 unrenote | 方案 2：删除「取消转贴」菜单项与 `hasRenotedBefore` 乐观置真；后端 `notes/unrenote` 记入待办 |
| 2 | F-05 readNotification | 方案 1：前端不发 `readNotification`，置读只走显式按钮（已按此实现） |
| 3 | F-03 defaultStore | 已在 Task 4 以 `store/index.ts` shim 落地 |
| 4 | F-11 pin 上限 | 方案 (c)：客户端按 `code` 判断（实现时落地） |
| 5 | F-17 registrationMode | 从 `types.ts` 删除幽灵字段；注册页以 `disableRegistration` 为准 |
| 6 | 通知页自动置读 | 沿用显式按钮（同 #2） |

## 影响的文档

- `docs/superpowers/plans/2026-10-31-iceshrimp-architecture-rewrite.md`：作废（保留历史，不删除）。
- `docs/superpowers/plans/2026-11-01-firefish-frontend-port.md`：仍然有效，Task 5 之后的
  实施以「Firefish 原版复原」为验收口径，参考侧一律指向 firefish 上游。
- 能力矩阵与运行时契约：不因重定基变化。
