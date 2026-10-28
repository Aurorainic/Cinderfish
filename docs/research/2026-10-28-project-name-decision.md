# Cinderfish 命名决策记录

- 日期：2026-10-28
- 结论：项目名定为 **Cinderfish**
- Go module：`github.com/Aurorainic/cinderfish`
- 环境变量前缀：`CINDERFISH_*`

## 为什么叫 Cinderfish

Cinderfish = cinder（余烬）+ fish。这个名字是对 Firefish 的致敬：Firefish 已于 2024 年停止维护，fire → cinder（余烬），火已熄但余烬尚温——本项目取 Firefish 的交互风格，在一个完全独立的 Go 实现中延续。fish 后缀与项目最初的暂用名 MilnFish 保持意象连续。

npm、PyPI、`.social` 域名三处核查均无同名软件，GitHub 检索无同名项目。

## 命名过程

四轮调研的演进脉络：

| 轮次 | 方向 | 结论 |
|------|------|------|
| Round 1 | 鱼群行为词（Shoal）/ 鳍词根（Finlet） | 否决：Shoal 太通用，Finlet 撞金融联想（PyPI 已有 finlet 金融项目） |
| Round 2 | 水域自然词（Fjord、Estuary、Tarn 等） | 全部被 npm/PyPI 占用，无满意者 |
| Round 3 | 自造名（Milune、Tidelet 等），对标朋友项目 miuno 的气质 | Milune 备选，但用户认为 fish 后缀更合适 |
| Round 4 | Firefish 致敬方向（Emberfish / Cinderfish / Tidefish / Lunafish） | **选定 Cinderfish** |

关键转折在 Round 3：本项目不是独立产品，而是自用的 AI 练手小玩意（朋友用 Go 重构 Misskey 的项目叫 miuno，本项目气质与之相近），因此放弃"检索零冲突"目标，改为语感优先。Round 4 在此基础上加入 Firefish 致敬，用户从四个候选中选定 Cinderfish。

## 已执行的改名

- Go module path、`cmd/cinderfish` 入口、`CINDERFISH_*` 环境变量前缀
- 架构文档改名为 `docs/superpowers/specs/2026-10-26-cinderfish-architecture-v0.md`
- `.archify/` 产物目录与文件统一更名
- 测试数据库名改为 `cinderfish_test`
