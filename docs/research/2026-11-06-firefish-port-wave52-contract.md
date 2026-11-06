# 第五十二波施工契约（设置页收尾：KaTeX 宏、集成说明页）

日期：2026-11-06
状态：已落地，证据见 `docs/HANDOVER.md` §12BQ
上游参考：Firefish `b425f61`：

- `packages/client/src/pages/settings/custom-katex-macro.vue`
- `packages/client/src/pages/settings/integration.vue`
- `packages/client/src/pages/_empty_.vue`

## 1. 这一波做什么

前端剩 6 个「范围内」占位页里的两个纯前端页，外加一个已无引用的占位文件：

1. `SettingsCustomKatexMacro`：宏编辑页。展开逻辑与 `preprocess()` 早已存在
   （`web/src/scripts/katex-macro.ts`、`web/src/scripts/preprocess.ts`、`defaultStore.enableCustomKaTeXMacro`），
   缺的只是写 localStorage 的界面。
2. `SettingsIntegration`：第三方账号绑定，产品范围明确不做（无 OAuth 客户端、无账号关联表），
   按第四十九波的做法改成说明页。
3. `EmptyPage`：`pages/_empty_.vue` 的占位副本与 `Empty.vue` 重复、且无路由引用，删除。

## 2. 落地细节

### 2.1 SettingsCustomKatexMacro

| 项目 | 值 |
|------|-----|
| 存储键 | `customKaTeXMacro`（原文）、`customKaTeXMacroParsed`（`parseKaTeXMacros` 的结果） |
| 开关 | `defaultStore.enableCustomKaTeXMacro`（device 作用域） |
| 写盘时机 | 「保存」按钮（与上游的每次输入即写不同，避免每敲一个字符都写 localStorage） |
| 生效 | 「重新载入」按钮 → 确认对话框 → `location.reload()` |

页面上明确写出降级：Cinderfish 没有 `katex` 依赖，公式在本站以代码显示，但宏展开发生在
**发帖前**（`preprocess()`），所以展开结果会随笔记发出，Misskey / Firefish 客户端能排版显示。

### 2.2 SettingsIntegration

- 顶部 `CfInfo warn`：第三方账号登录未实现
- 「这里已经有」：MiAuth 应用授权（链到 `/settings/apps`）、ActivityPub 关注（不需要绑定）
- 「上游有、这里没有」：Discord / GitHub / Twitter(X) / Matrix 四条
- 设置菜单新增「整合」（`t("integration")`）入口

## 3. 写作用域

| 文件 | 动作 |
|------|------|
| `web/src/pages/SettingsCustomKatexMacro.vue` | 占位页 → 真页面 |
| `web/src/pages/SettingsIntegration.vue` | 占位页 → 说明页 |
| `web/src/pages/SettingsIndex.vue` | 菜单新增「整合」 |
| `web/src/pages/EmptyPage.vue` | 删除（`Empty.vue` 已在用） |

## 4. 有意的限制

- 不引入 `katex`：只为排版公式增加一个 100 KB+ 的依赖，与首版取舍不符；页面上写清后果。
- 宏页不做「实时预览」：预览需要 katex 才能有意义。
- 集成页只说明、不提供跳转外站的按钮：没有可跳转的授权端点。

## 5. 验收口径

- `vue-tsc --noEmit` 通过
- `check:port` hardTotal 0（前端 402 个文件）
- `pnpm build` 成功
- 后端未改动，无需重跑 `go test`（仍以第五十一波的全绿结果为准）
