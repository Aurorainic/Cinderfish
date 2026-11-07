# 第五十波施工契约：状态栏（2026-11-06）

`/settings/statusbar` 一直是占位页，`defaultStore.state.statusbars` 这个键在仓库里没有任何使用
者——存得进去、没有东西渲染。本波实现状态栏本体与它的一个可用条目类型。

## 1. 为什么只有一种类型

Firefish 的状态栏条目有三个类型：

| 类型 | 需要什么 | 这里 |
|------|----------|------|
| `federation` | `federation/instances`（第四波已有） | **做** |
| `rss` | 媒体/RSS 代理抓取外部 feed | 不做：没有代理，浏览器直连会被 CORS 挡 |
| `userList` | `users/lists/*` | 不做：首版范围外 |

设置页把后两种列成禁用项并注明原因，而不是让用户选一个渲染不出东西的类型。

> **2026-11-07 更新（第一百六十三波，§12GA）**：`fetch-rss` 在第一百三十九波（§12FC）落地后，
> `rss` 已经没有缺的后端了——服务端抓取绕开 CORS 并带 SSRF 防护。状态栏的 RSS 条目现已实现，
> 只剩 `userList` 仍按首版范围禁用。本节与 §4 的第 1 条是书写当时的决定，保留以便追溯。

## 2. 组件

| 文件 | 职责 |
|------|------|
| `components/statusbar/CfStatusbarFederation.vue` | 拉 `federation/instances`（`sort:+lastCommunicatedAt`、30 条），跑马灯滚动主机名；支持刷新间隔、滚动时长、反向、着色 |
| `components/statusbar/CfStatusbars.vue` | 按 `statusbars` 渲染多条，处理五档高度、深色底、名称；未知类型就地说明 |
| `pages/SettingsStatusbar.vue` | 增删、上下移、名称/类型/高度/深色/联邦参数，改动即时写入 store |

跑马灯用 CSS keyframes 平移一份**复制过的列表**，所以首尾能无缝衔接；`prefers-reduced-motion`
下关闭动画。着色模式没有主题色可用（接口不返回），色相由主机名哈希派生——同一实例永远同色。

## 3. 挂载

`ui/layouts/UniversalLayout.vue`：有配置时在底部固定一条 `CfStatusbars`（`position: fixed`），
同时给主内容加 `padding-block-end` 让最后一行不被遮住。`statusbars` 为空时整条不出现。
`SettingsIndex` 增加「状态栏」入口。

## 4. 有意的限制

1. 只支持 `federation`；rss 与用户列表按上表原因不做。
2. 条目没有拖拽排序，只有上移/下移按钮。
3. 只有 Universal 布局挂载状态栏；Deck / Zen 布局不显示（它们自己的列结构还没接）。
4. 全局状态栏是 `deviceAccount` 作用域：同一设备同一账号共享，不跨设备同步。
5. 测试见 §12BN.4（走查；2026-11-07 补）。

## 5. 验收

```
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # hardTotal 0（400 文件）
cd web && pnpm build                    # exit 0
```

后端未改动，台账不变（137/130/310）。浏览器走查于 2026-11-07 补齐，并修复 `block-end` 为 `inset-block-end`（见 HANDOVER §12BN.4）。
