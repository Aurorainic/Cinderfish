# 第四十四波施工契约：实例 CSS 与邮件设置页（2026-11-06）

两个管理台占位页转真实实现，其中一个带真正的功能（实例级样式），另一个是诚实的状态页。

## 1. 实例级自定义 CSS

| 层 | 变化 |
|----|------|
| 存储 | `instance_meta.customCss`（`instanceSettings.CustomCSS`，`instanceMetaKeys["customCss"]`） |
| 写入 | `admin/update-meta` 接受 `customCss`，上限 100 KiB（字节），空串即清除 |
| 下发 | `meta` 与 `admin/meta` 都返回 `customCss` |
| 前端 | `composables/use-custom-css.ts` 注入 `<style id="cinderfish-instance-custom-css">` |
| 页面 | `pages/AdminCustomCss.vue`（编辑器）+ `/admin/custom-css` 路由 + 管理台菜单项 |

注入顺序：实例样式先写、账号样式后写。两者都追加在 `<head>` 末尾，所以账号级规则天然覆盖
实例级规则，不需要 `!important`。`instance` 从 localStorage 缓存读到的旧值在首屏就会应用，
`meta` 到达后再校正。

## 2. 邮件设置页

`admin/meta` 新增 `emailServer`（SMTP 主机名，未配置为空串），`meta`/`admin/meta` 的
`enableEmail` 改为按 `[mail] host` 计算，不再是常量 `false`。

`pages/AdminEmailSettings.vue` 显示：是否已配置、SMTP 主机、三种邮件流程（验证、重置、通知
——通知邮件四类在第四十八波已实现，页面文案于 2026-11-07 修正）、以及可用环境变量。
**不渲染可编辑字段**：端口、账号、密码、发件人、TLS 都在 `config.toml`，改完要重启，
做成输入框只会静默失效。

## 3. 有意的限制

1. 实例 CSS 没有语法校验（浏览器会静默忽略坏规则），也没有版本历史。
2. 实例 CSS 对未登录访客同样生效——这是特性，页面里已警示。
3. 邮件设置只能看不能改；没有运行期 SMTP 重载。
4. 未新增后端路由（台账仍是 136/129/311）。

## 4. 验收

```
gofmt -l internal/ cmd/                 # 空
go vet ./...                            # exit 0
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # hardTotal 0
cd web && pnpm build                    # exit 0
```

浏览器走查于 2026-11-07 补齐（`TestInstanceCustomCSS` + Helium 走查，见 HANDOVER §12BH.4）。
