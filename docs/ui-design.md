# 0538Pay 管理后台 — UI 设计规范

> 本文档定义 `admin-web` 管理后台的全局设计系统。**开发任何新页面前请先阅读，并严格遵循。**
> 在线对照：启动后访问 `http://localhost:5173/style-guide` 查看所有组件的实时效果。

---

## 一、技术栈

| 项 | 选型 |
|---|---|
| 框架 | Vue 3 + TypeScript + Vite 8 |
| 样式 | Tailwind CSS v4（`@tailwindcss/vite`） |
| 状态 / 路由 | Pinia / vue-router 4 |
| 图标 | lucide-vue-next |
| 图表 | chart.js + vue-chartjs |
| 字体 | @fontsource/inter（本地打包，英文/数字用 Inter，中文用系统黑体） |

启动：`cd admin-web && npm run dev` → `http://localhost:5173`
（Node 需 ≥ 20.19，本机已用 nvm 装 v22.12.0）

---

## 二、核心视觉原则（不可违背）

1. **卡片 = 直角、无边框、无阴影**。纯白底（`bg-card`）直接浮在浅灰内容底上，靠色差区分。
   > ⛔ 不要给卡片加 `rounded-*` / `border` / `ring` / `shadow`。这几点被反复否决过。
2. **主色 = 亮蓝** `--primary`（oklch(0.58 0.2 262)）。侧栏纯白，选中项 = 浅蓝底 + 蓝字（小圆角块）。
3. **紧凑间距**：页面根容器 `space-y-2.5`，并排 grid `gap-2.5`，主内容区外边距 `p-2.5`。
4. **数字统一** `tabular-nums`（等宽对齐）。金额的 `¥` 符号弱化（小号 + `text-muted-foreground`）。

---

## 三、配色令牌

定义在 `src/style.css`，用 Tailwind 语义类引用（`bg-primary` / `text-muted-foreground` 等）：

| 令牌 | 用途 |
|---|---|
| `primary` | 主色亮蓝，主按钮、选中态、链接 |
| `success` / `warning` / `destructive` | 成功绿 / 警告黄 / 危险红（状态、金额涨跌） |
| `foreground` / `muted-foreground` | 主文字 / 次要文字 |
| `card` | 卡片白底 |
| `content`（`--content-bg`） | 内容区浅灰底 |
| `border` | 分隔线、边框 |
| `muted` / `accent` | 弱底 / hover 底 |

亮色 / 暗色主题均已定义，切换由顶栏主题按钮控制。

---

## 四、复用组件

统一从 `@/components/ui` 导入：

```ts
import { PageHeader, Panel, Button, Badge } from '@/components/ui'
```

### PageHeader — 页面标题区
放在页面根容器最上方。
```vue
<PageHeader title="订单管理" subtitle="共 1,234 笔订单">
  <template #actions><Button>导出</Button></template>
</PageHeader>
```

### Panel — 标准内容面板（卡片 + 标题行 + 分隔线 + 内容区）
**页面的每个区块都用 Panel**，不要手写标题行 + 分隔线。
```vue
<Panel title="筛选" subtitle="可选副标题">
  <template #actions><Button variant="ghost" size="sm">重置</Button></template>
  ...内容...
</Panel>

<Panel :no-header="true">纯内容容器，无标题</Panel>
<Panel title="列表" flush>内容区无内边距（整块表格自控 padding 时用）</Panel>
```

⚠️ **列表页的列表 Panel 必守规范**（否则出现孤儿副标题、视觉割裂）：
- 包表格的 Panel **必须带主标题** `title`（如"订单列表 / 结算记录 / 分账列表 / 资金流水"）+ 条数副标题，**不要加 `flush`**（表格走默认 `px-6 py-4` 内边距，和 Orders 一致）。
- **禁止** `<Panel :subtitle="`${total} 条`" flush>` —— 只有裸副标题、没主标题时，标题行会变成一个孤零零的副标题 + 分隔线 + 贴边表格，看起来脱节。
- 有批量操作按钮放 `#actions` 插槽（标题行右侧）。

### Button
- `variant`：`default`(主) / `secondary` / `outline` / `ghost` / `destructive`
- `size`：`default` / `sm` / `lg` / `icon`
- 图标：`<Button><Plus />新建</Button>`（图标自动 16px）

### Badge — 状态标签（浅色描边 + 极淡底 + 小圆角）
- `variant`：`default` / `success` / `warning` / `destructive` / `muted` / `outline`
- 订单状态约定：待支付 `warning`、已支付 `success`、已退款 `destructive`

### Switch — 开关（toggle）
- 用法：`<Switch v-model="enabled" />`，可选 `size="sm"`、`disabled`。
- 样式：开 `bg-primary` / 关 `bg-muted-foreground/30`；圆钮 `bg-white shadow-sm ring-1 ring-black/5`。
- ⚠️ 圆钮**必须带 shadow + ring**，否则在蓝底(开启态)上白钮会"消失"。定位用 `inline-flex items-center px-0.5` + `translate-x-5/0`（默认）或 `translate-x-4/0`（sm），不要用绝对定位 + 任意值位移。
- 所有开关一律用此组件，不要在页面内联手写 `<button role="switch">`。

---

## 五、标准数据表 `.tbl`

写 `<table class="tbl">`，自动获得：左对齐表头（灰）、行分隔线、hover 高亮、等宽数字。

```vue
<div class="overflow-x-auto">
  <table class="tbl">
    <thead>
      <tr><th>日期</th><th>支付宝</th><th>总计</th></tr>
    </thead>
    <tbody>
      <tr v-for="(r, i) in rows" :key="i" :class="i === 0 && 'border-b-2 border-border'">
        <td :class="i === 0 ? 'font-medium' : 'dim'">{{ r.date }}</td>
        <td :class="i === 0 ? 'font-medium' : 'text-foreground/70'">{{ r.a }}</td>
        <td class="font-semibold">{{ r.total }}</td>
      </tr>
    </tbody>
  </table>
</div>
```

约定：
- **默认左对齐**（表头 + 内容），不要右对齐/居中。
- 辅助工具类（写在 `th`/`td` 上）：
  - `.num` — 数字列右对齐等宽
  - `.dim` — 次要文本变灰
  - `.col-center` — **居中列**（状态/操作列）。表头和单元格都加此类，会同时居中并去掉左右 padding，保证表头与内容中心对齐。⚠️ 不要只在单元格加 `text-center`——`.tbl th` 的左对齐优先级更高会导致表头与内容错位。

---

## 五之二、筛选区 / 输入框 / 操作菜单（列表页通用）

这些全局类沉淀自订单页，所有列表页的筛选和行操作直接复用：

### 筛选区 `.filter-bar` + `.filter-item` + `.filter-label`
```vue
<div class="filter-bar">                     <!-- 一行筛选，可多个，自动换行 -->
  <div class="filter-item">                   <!-- 标签+控件 -->
    <label class="filter-label">订单信息</label> <!-- 固定宽右对齐，保证多行控件左边对齐 -->
    <Select v-model="..." :options="..." class="w-32" />
    <input v-model="..." placeholder="搜索内容" class="field-input w-48" />
  </div>
  <div class="filter-item"><label class="text-sm text-muted-foreground">商户号</label>
    <input class="field-input w-40" /></div>
  <div class="ml-auto flex items-center gap-2">  <!-- 按钮组靠右 -->
    <Button size="sm"><Search />搜索</Button>
    <Button variant="outline" size="sm"><RotateCcw />重置</Button>
  </div>
</div>
```
- `.filter-label` 固定宽 4rem + 右对齐。多行筛选时，需要对齐的行首标签都用它。
- 其余标签用 `text-sm text-muted-foreground`。

### 输入框 `.field-input`
标准文本/数字输入框，宽度用 Tailwind `w-*` 叠加：`<input class="field-input w-40" />`。
（下拉用 `Select` 组件，日期用 `DatePicker`/`DateRange` 组件，不用原生控件。）

### 行操作下拉菜单 `.menu-panel` + `.menu-item`
```vue
<tr v-for="(row, si) in pageRows" :key="row.id">
  <td class="relative inline-block">
    <Button variant="ghost" size="sm" @click.stop="toggleMenu(row.id)">操作 <MoreHorizontal /></Button>
    <div
      v-if="openMenu === row.id"
      class="menu-panel absolute right-0 z-20 w-36"
      :class="si >= pageRows.length - 3 && pageRows.length > 3 ? 'bottom-full mb-1.5' : 'top-full mt-1.5'"
      @click.stop
    >
      <div class="menu-sep" />                              <!-- 分隔线 -->
      <button class="menu-item" @click="...">                <!-- 普通项 -->
        <SomeIcon class="size-4 shrink-0 opacity-70" /><span class="flex-1">改未完成</span>
      </button>
      <button class="menu-item menu-item-danger">删除订单</button>  <!-- 危险项(红) -->
    </div>
  </td>
</tr>
```
- 关闭逻辑：`onMounted` 时 `window.addEventListener('click', closeMenu)`，按钮和面板用 `@click.stop` 阻止冒泡。**不要**用 `onClickOutside` 绑 v-for 内的 ref（多元素共享 ref 会失效）。
- ⚠️ **菜单方向必须自适应**：固定 `top-full` 向下弹会让末尾几行的菜单被视口/表格底部截断。v-for 带行索引 `(row, si)`，用上面的 `:class` 让后 3 行改为 `bottom-full` 向上弹。
- **"今日 / 合计"强调行**：用 `border-b-2 border-border` 加粗分隔 + `font-medium`，**不要用背景色块**。合计数值 `font-semibold`（利润类可 `text-success`），不要用蓝色。

---

## 六、布局与新增页面

- **AdminLayout**（`src/layouts/AdminLayout.vue`）：左侧两级折叠菜单（白底，选中浅蓝块）+ 顶栏（面包屑 / 搜索 / 通知抽屉 / 设置 / 主题切换 / 用户菜单）。
- 导航配置：`src/config/nav.ts`（`navMenu` 两级结构 + `allLeaves` 扁平列表）。
- 默认头像：`public/images/avatar-default.png`，引用 `/images/avatar-default.png`。

**新增一个页面的步骤**：
1. `src/config/nav.ts` — 在对应分组的 `children` 加一项 `{ title, to }`。
2. `src/views/` — 新建页面组件（用 `PageHeader` + `Panel` + `.tbl` 组装）。
3. `src/router/index.ts` — 目前非仪表盘路由自动指向占位页；正式开发时把该路径映射到新组件。

**页面骨架范例**：
```vue
<template>
  <div class="space-y-2.5">
    <PageHeader title="订单管理" subtitle="共 1,234 笔">
      <template #actions><Button size="sm"><Download />导出</Button></template>
    </PageHeader>

    <Panel title="筛选"> ...筛选表单... </Panel>

    <Panel title="订单列表" subtitle="1,234 条">
      <div class="overflow-x-auto"><table class="tbl">...</table></div>
    </Panel>
  </div>
</template>
```

---

## 七、踩过的坑（勿重犯）

- 卡片的圆角 / 边框 / 阴影都被否决 → 全部去掉，只留纯白直角底。
- 表格默认**左对齐**，别改右对齐或居中。
- 强调行用分隔线，**别用背景色块**。合计不要用蓝色（除非利润用绿色）。
- 图表刻度文字要够深（tick `rgba(0,0,0,0.7)`）、字号 12；Chart.js 隐藏系列的删除线要用 `generateLabels` 去掉，图例 `padding: 24`。
- `vue-tsc` 严格（`noUnusedLocals`）→ 未使用的 import 必须删干净，否则类型检查报错。
- 用户下拉窄（`w-36`）+ `rounded-md`，图标统一灰色。
