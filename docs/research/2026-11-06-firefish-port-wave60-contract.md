# 第六十波施工契约（范围内占位页清零：说明页）

日期：2026-11-06
状态：已落地，证据见 `docs/HANDOVER.md` §12BY
上游参考：Firefish `b425f61`：

- `packages/client/src/pages/signup-complete.vue`
- `packages/client/src/pages/settings/plugin.vue`
- `packages/client/src/pages/settings/plugin.install.vue`
- `packages/client/src/scripts/get-user-menu.ts`（插件 hook 的用途说明）

## 1. 这一波做什么

首版范围内最后三个占位页（`SignupComplete`、`SettingsPlugin`、`SettingsPluginInstall`）的依赖在
首版不可能补齐：

| 页面 | 依赖 | 为什么不做 |
|------|------|------------|
| `SignupComplete` | `emailRequiredForSignup` 注册模式 + 待确认账号表 + `signup-pending` 端点 | 产品范围只要求开放 / 邀请码 / 管理员建号三种注册模式 |
| `SettingsPlugin` | AiScript 运行时 | 需要引入解释器依赖并实现整套插件 API，与首版取舍不符 |
| `SettingsPluginInstall` | 插件表 + 文件上传 + 解释器 | 同上 |

按第四十九波的做法，它们变成说明页：说清现状、已有替代手段与缺口，而不是留着 `TODO` 占位。

## 2. 实现

新增共享组件 `web/src/components/CfFeatureGap.vue`（与 `AdminUnavailable` 同构，但面向用户设置页）：

```
props: title, summary, available[{label, note?, to?}], missing[]
```

三个页面只用数据填充：

- **SignupComplete**：说明本站注册模式、这个链接不会属于本站；链到登录/注册；
  缺口：邮箱确认注册、待确认表与有效期、确认后自动登录
- **SettingsPlugin**：说明插件是 AiScript 脚本、本站没有解释器；链到自定义 CSS、KaTeX 宏、API；
  缺口：安装、启停列表、AiScript 运行时
- **SettingsPluginInstall**：说明不接受 `.aiscript` 上传；链到插件说明与自定义 CSS；
  缺口：上传校验、插件表、解释器

## 3. 写作用域

| 文件 | 动作 |
|------|------|
| `web/src/components/CfFeatureGap.vue` | 新增 |
| `web/src/pages/SignupComplete.vue` | 占位 → 说明页 |
| `web/src/pages/SettingsPlugin.vue` | 占位 → 说明页 |
| `web/src/pages/SettingsPluginInstall.vue` | 占位 → 说明页 |

## 4. 结果

占位页从 30 个降到 **24 个**，剩下的全部是用户已裁决的首版范围外页面
（channels / clips / gallery / messaging / antennas / lists / groups / pages 等）。
**首版范围内占位页 = 0。**

## 5. 验收口径

- `vue-tsc --noEmit` 通过
- `check:port` hardTotal 0、`unresolvedComponents` 与 `missingKeys` 均为空
- `pnpm build` 成功
- 本波只改前端，后端测试沿用第五十九波的全绿结果
