# Task F1 完成报告：主题 token 词表扩充 + 全局去 AI 味

日期：2026-10-29
提交：`ff1ee8f` — `style(web): expand theme tokens, flatten global aesthetics`
改动文件：`web/src/styles/main.css`（唯一所有权文件，+50/-9）

## 实现内容

### Token 词表（`:root` 暗色 + `:root[data-theme="light"]` 亮色，两套都有值）

保留原词表全部变量名不动：`--accent` `--accent-soft` `--accent-strong` `--on-accent` `--bg` `--bg-raised` `--panel` `--panel-2` `--border` `--text` `--text-strong` `--muted` `--danger` `--success` `--link` `--radius-sm` `--radius` `--font`，以及布局 token `--rail-width` `--side-width` `--content-max`。

新增语义 token（命名裁剪自 Firefish `themes/_dark.json5` 的语义）：

| Token | 暗色 | 亮色 | 用途 |
|-------|------|------|------|
| `--divider` | `rgba(255,255,255,0.08)` | `rgba(0,0,0,0.08)` | 1px 分隔线（行流用） |
| `--panel-header` | `#221f2b` | `#f4f1ea` | 列头/面板头背景（比 panel 微抬） |
| `--nav-bg` | `#17151d` | `#ffffff` | 侧栏背景 |
| `--renote` | `#4fd1c5` | `#0e8a7b` | 转贴强调色（青绿） |
| `--cw-bg` / `--cw-fg` | `#2a2634` / `#ece9f3` | `#f0ebe1` / `#241f1a` | CW 横幅 |
| `--mention` | `#7fb4ff` | `#1668b8` | 提及语义色 |
| `--hashtag` | `#ffb35c` | `#a35a00` | 话题标签语义色 |
| `--shadow-sm` | `0 2px 10px rgba(0,0,0,0.4)` | `0 2px 10px rgba(60,40,20,0.14)` | 弹层弱阴影 |

### 兼容性处理

- `--radius-lg`：20px → **12px**（变量名保留，`NoteCard.vue`、`ProfileView.vue`、`NotificationsView.vue`、`NoteView.vue`、`TimelineFeed.vue`、`NoteComposer.vue` 等引用处无需改动，卡片/组件整体圆角随 token 收敛）。
- `--shadow`：**未删除变量名**，改为 `var(--shadow-sm)` 的弱值别名。原因：`NoteCard.vue:499`（反应弹层 `.palette`）在波次 2 由 F6 重写前仍引用 `var(--shadow)`；控制器约束为「全套现有变量名不动」。若直接删除，该弹层阴影静默失效。现状是：大投影（0 8px 30px）已从全局消失，弹层引用自动落到弱阴影，与设计意图（弹层用 `--shadow-sm`）一致，且 F6 重写时可直接换成 `--shadow-sm`。

### 去 AI 味全局调整

- **body 背景双层 radial-gradient 删除**，改为纯色 `var(--bg)`（59-64 行按要求）。
- **`.button` 圆角 999px → `var(--radius)`**（14px），全胶囊按钮收敛；小圆片 `.icon-button` 保留 50% 正圆。
- 新增全局工具类：
  - `.panel`：`background: var(--panel)` + `border: 1px solid var(--border)` + `border-radius: var(--radius-lg)`（面板底+12px 圆角+边框，无 padding，组件可叠加）；
  - `.note-row`：无边框行基类（上下 12px padding、无圆角），供 F6 去卡片化笔记行使用；
  - `.divider-h`：1px 水平分隔线（`height:1px; background: var(--divider); border:0; margin:0`），`<div>` 与 `<hr>` 皆可用。

### 未动（按所有权与简报范围）

- `.auth-aside` 的两处 radial-gradient、`.brand-mark` 的渐变底与光晕阴影未改——简报逐条清单只点名 body 梯度、按钮胶囊、大投影；这两处留给 F12 全仓 sweep 收尾（见 concerns）。
- 组件 `<style scoped>` 一律未动。

## 验证证据（无 TDD 步骤，按简报验证命令）

```
$ pnpm typecheck          # vue-tsc --noEmit — 通过，无输出
$ pnpm build              # vue-tsc --noEmit && vite build — 通过
  ✓ 94 modules transformed，built in 523ms
```

构建产物抽查：`dist/assets/index-*.css` 中确认 `--divider`、`--panel-header`、`--renote`、`--cw-bg`、`--shadow-sm` 明暗两套值均已落入 bundle（grep 见 `#2a2634`/`#f0ebe1`、`#4fd1c5`/`#0e8a7b` 等）。

提交内容核对：`git status` 显示仅 `web/src/styles/main.css` 被本次提交修改（另有并发 agent 的未跟踪目录 `web/src/components/ui/`，未纳入提交）。

## Files changed

- `web/src/styles/main.css`（唯一文件；token 段重写 + body 去渐变 + `.panel`/`.note-row`/`.divider-h` 新增 + `.button` 去胶囊）

## Concerns

1. **`--shadow` 以弱值别名保留**（未按简报字面删除）：为满足「全套现有变量名不动」且不让 `NoteCard.vue:499` 的弹层阴影在 F6 重写前静默失效。建议评审确认此取舍；若要求字面删除，删掉该行后 NoteCard 弹层阴影会退化（build 不受影响）。
2. **残余 AI 味渐变**：`.auth-aside`（登录页右侧）仍有两处 radial-gradient，`.brand-mark` 仍有渐变底与光晕阴影，不在本简报点名清单内，留给 F12 sweep 或后续指派。
3. 新 token 的**最终取值**属「留给实现者」事项，本次按现有暖橙调色板自定；F2/F5/F6 落地后若观感不匹配可只调 token 值，不动结构。
