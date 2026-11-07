# 2026-11-07 前端界面视觉审计与修复

对 `web/`(Firefish 整体移植的 Vue 3 客户端)做了一次基于真实浏览器渲染的视觉与交互审计:
后端跑在 `127.0.0.1:3000`(联邦 none、本地存储、Rosé Pine Dawn/Rosé Pine 内置主题),
用专用账号 `uiaudit`/`uiauditadmin` 登录,在 ZCode 内置浏览器里逐页截图巡检,
覆盖**暗色/亮色主题 × 桌面(1440×900)/移动(390×844)** 四象限,
页面包括:欢迎页、时间线(首页/本地/社交)、通知、发现(用户/热门签/本地)、搜索、
设置(基本设置/静音屏蔽等)、个人主页、网盘,并全程收集 console 错误(全程零错误)。

## 修复的问题(一项一个文件,均已构建后浏览器复验)

1. **发现页"关注"按钮被卡片标题面板遮挡,只剩 C 形边框残影**(视觉上最扎眼的问题)。
   `MkUserInfo.vue` 的 `.buttons` 容器(z-index 3)绝对定位在卡片右上角,而 `.title`
   面板(z-index 4、不透明 `--panel` 背景、圆角)画在其上方,把圆形按钮的大部分盖住,
   只有超出圆角的一小段边框可见,形似破损的圆弧。初看极似图标字体缺字形,
   逐层排查(computed style、canvas 字形墨迹、注入对照元素)后定位为层叠遮挡。
   修复:`.buttons` z-index 提到 5。
2. **两个 Phosphor 图标类名不存在,对应位置图标完全空白**。全项目 191 个 `ph-*`
   图标类与 `@phosphor-icons/web@2.0.3` 的 bold/fill/regular 样式表交叉比对,
   仅 `ph-apps`(Auth.vue 的应用图标)与 `ph-server`(设置页"服务器静音")不在包内。
   修复:改为 `ph-app-window` 与 `ph-stack`(均为包内有效类,语义一致)。
3. **无头像占位渲染为空白色块**。`CfAvatar.vue` 的 no-avatar 分支是一个
   `ph-user` 图标 span,但没有任何字号/颜色/居中处理,视觉上只是块底色,像渲染 bug。
   修复:与 `global/MkAvatar.vue` 统一——`noAvatar` 时以 `static-assets/avatar.svg`
   人形轮廓为伪元素背景(panelHighlight 底 + divider 边),两种头像组件现在呈现一致。
4. **小部件列重复的"编辑小部件"入口**。`MkWidgets.vue` 空状态面板里已有一个
   "编辑小部件"按钮,面板下方的 footer 文字按钮又无条件渲染同一动作,新实例的
   右栏因此出现两个相同入口。修复:footer 仅在列表非空或编辑模式下渲染。
5. **标签页标题重复实例名**。`page-metadata.ts` 无条件拼接 `"<页名> | <实例名>"`,
   欢迎页的页名就是实例名,得到 "Cinderfish UI 审计实例 | Cinderfish UI 审计实例"。
   修复:两者相同(或实例未命名)时不再追加后缀。

## 排查过、确认不是问题的现象

- **图标字体系统性缺字形**:发现页弧形残影初判为 Phosphor 字体缺字形。
  canvas `fillText` 实测 5 个代表码位(U+E9xx–EDxx)全部有墨迹,
  `document.fonts` 报告 Phosphor-Bold/Fill 均 loaded,同屏注入的对照元素正常渲染——
  字体与 CSS 码位自洽,弧形残影的真正原因即上面第 1 条的遮挡。
- **`/notifications`、`/drive` 直链 404**:路由表与侧栏链接一致使用
  Misskey 风格的 `/my/*` 前缀,直接输入短路径属于输入问题而非坏链。
- **个人页首次进入短暂空白**:fade transition + v-if 重渲染的瞬间态,内容
  (DOM 快照验证)完整,约一帧后正常,不构成缺陷。
- **大头像人形占位**:`global/MkAvatar.vue` 的 `avatar.svg` 占位是设计好的
  (200 可访问),本次把 `CfAvatar` 对齐到同一视觉语言,而非 favicon。

## 验证

`pnpm typecheck`(vue-tsc)干净、`pnpm build` 成功;修复后在浏览器中复验:
发现页按钮完整、设置页"服务器静音"有图标、侧栏/笔记/卡片头像统一为人形占位、
小部件单入口、标题拼接正常;暗色/亮色 × 桌面/390px 各截屏确认无回归,console 零错误。

## 留待后续(非缺陷,体验观察)

- 左下"帖子"按钮为 accent 渐变,与侧栏其余项风格差异较大(上游 Firefish 同款设计,保留)。
- 右栏空状态文案"还没有小部件"与面板按钮的组合可再精简,但现有单入口已不误导。
- 三个 shell(Deck/Universal/Visitor/Zen)仅巡检了 Universal,Deck 布局未在本次范围内。
