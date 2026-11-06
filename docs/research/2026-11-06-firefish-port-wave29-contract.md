# 第二十九波施工契约：关于页（2026-11-06）

`/about-firefish` 是 `TODO: port` 占位，而关于页已经链接过去。本波写成 Cinderfish 自己的
软件说明页；没有后端改动。

## 1. 页面内容

- 实例名与版本：来自 `meta`（`instance.name`、`instance.version`，当前后端版本 `0.1.0`）；
- 简介：Cinderfish 的定位（Misskey API 契约、ActivityPub 联邦、Go 后端、Firefish 结构前端）；
- 链接：源码仓库（仅当实例设置 `repositoryUrl` 非空时显示）、Misskey、Firefish、Iceshrimp；
- 实现说明：技术栈、接口契约来源、联邦模式；
- 许可证：AGPL-3.0-only，参考实现同为 AGPL，来源记录在 `docs/HANDOVER.md`。

## 2. 与 Firefish 的差异

Firefish 的 `about-firefish.vue` 宣传 Firefish 项目、捐赠与贡献者名单。本移植**不冒充
Firefish**：页面只描述 Cinderfish，并致谢三个参考实现；没有捐赠入口、没有虚构贡献者。

## 3. 验收

`vue-tsc --noEmit`、`check:port`、`pnpm build` 通过。按用户 2026-11-06 的指示未做浏览器走查。
