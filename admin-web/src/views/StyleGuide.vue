<script setup lang="ts">
import { ref } from 'vue'
import { Panel, PageHeader, Button, Badge, Switch } from '@/components/ui'
import { Plus, Download, Trash2 } from 'lucide-vue-next'

const sw1 = ref(true)
const sw2 = ref(false)
const sw3 = ref(true)

const colors = [
  { name: 'primary', cls: 'bg-primary', label: '主色（亮蓝）' },
  { name: 'success', cls: 'bg-success', label: '成功（绿）' },
  { name: 'warning', cls: 'bg-warning', label: '警告（黄）' },
  { name: 'destructive', cls: 'bg-destructive', label: '危险（红）' },
  { name: 'foreground', cls: 'bg-foreground', label: '前景文字' },
  { name: 'muted', cls: 'bg-muted', label: '弱底' },
  { name: 'border', cls: 'bg-border', label: '边框线' },
  { name: 'content', cls: 'bg-content', label: '内容底灰' },
]

const rows = [
  { date: '今日', a: 59.8, b: 54.2, total: 142.38 },
  { date: '07-12', a: 56.4, b: 51.0, total: 134.28 },
  { date: '07-11', a: 51.0, b: 46.2, total: 121.54 },
]
</script>

<template>
  <div class="space-y-2.5">
    <PageHeader title="样式规范 Style Guide" subtitle="全局设计系统 · 新页面开发请对照此页">
      <template #actions>
        <Button variant="outline" size="sm"><Download />导出</Button>
        <Button size="sm"><Plus />新建</Button>
      </template>
    </PageHeader>

    <!-- 颜色 -->
    <Panel title="配色 Colors">
      <div class="grid grid-cols-2 gap-4 sm:grid-cols-4 lg:grid-cols-8">
        <div v-for="c in colors" :key="c.name" class="space-y-1.5">
          <div :class="['h-14 w-full rounded', c.cls]" />
          <div class="text-xs font-medium">{{ c.name }}</div>
          <div class="text-[11px] text-muted-foreground">{{ c.label }}</div>
        </div>
      </div>
    </Panel>

    <!-- 按钮 -->
    <Panel title="按钮 Button">
      <div class="flex flex-wrap items-center gap-3">
        <Button>默认按钮</Button>
        <Button variant="secondary">次要</Button>
        <Button variant="outline">描边</Button>
        <Button variant="ghost">幽灵</Button>
        <Button variant="destructive"><Trash2 />删除</Button>
        <Button size="sm">小号</Button>
        <Button size="lg">大号</Button>
      </div>
    </Panel>

    <!-- 标签 -->
    <Panel title="状态标签 Badge">
      <div class="flex flex-wrap items-center gap-3">
        <Badge>默认</Badge>
        <Badge variant="success">已支付</Badge>
        <Badge variant="warning">待支付</Badge>
        <Badge variant="destructive">已退款</Badge>
        <Badge variant="muted">已关闭</Badge>
        <Badge variant="outline">描边</Badge>
      </div>
    </Panel>

    <!-- 开关 -->
    <Panel title="开关 Switch" subtitle="启用/禁用切换，v-model 绑 boolean">
      <div class="flex flex-wrap items-center gap-8">
        <div class="flex items-center gap-2">
          <Switch v-model="sw1" />
          <span class="text-sm text-muted-foreground">默认尺寸（{{ sw1 ? '开' : '关' }}）</span>
        </div>
        <div class="flex items-center gap-2">
          <Switch v-model="sw2" size="sm" />
          <span class="text-sm text-muted-foreground">小尺寸 size="sm"</span>
        </div>
        <div class="flex items-center gap-2">
          <Switch v-model="sw3" disabled />
          <span class="text-sm text-muted-foreground">禁用 disabled</span>
        </div>
      </div>
    </Panel>

    <!-- 文字层级 -->
    <Panel title="文字层级 Typography">
      <div class="space-y-2">
        <h1 class="text-xl font-semibold">页面大标题 text-xl / semibold</h1>
        <h3 class="text-[15px] font-semibold">面板标题 15px / semibold</h3>
        <p class="text-sm">正文 text-sm / normal</p>
        <p class="text-sm text-muted-foreground">次要说明 text-sm / muted-foreground</p>
        <p class="text-xs text-muted-foreground">辅助信息 text-xs / muted-foreground</p>
        <p class="text-2xl font-normal tabular-nums">1,234,567.89 数字 tabular-nums</p>
      </div>
    </Panel>

    <!-- 标准表格 -->
    <Panel title="标准数据表 .tbl" subtitle="左对齐 / 等宽数字 / 合计行加粗">
      <div class="overflow-x-auto">
        <table class="tbl">
          <thead>
            <tr>
              <th>日期</th>
              <th>支付宝</th>
              <th>微信</th>
              <th>总计</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="(r, i) in rows"
              :key="i"
              :class="i === 0 && 'border-b-2 border-border'"
            >
              <td :class="i === 0 ? 'font-medium' : 'dim'">{{ r.date }}</td>
              <td :class="i === 0 ? 'font-medium' : 'text-foreground/70'">{{ r.a.toFixed(2) }}</td>
              <td :class="i === 0 ? 'font-medium' : 'text-foreground/70'">{{ r.b.toFixed(2) }}</td>
              <td class="font-semibold">{{ r.total.toFixed(2) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </Panel>

    <!-- Panel 用法 -->
    <div class="grid grid-cols-1 gap-2.5 xl:grid-cols-2">
      <Panel title="面板 A" subtitle="带副标题">面板内容区（px-6 py-4）</Panel>
      <Panel title="面板 B">
        <template #actions><Button variant="ghost" size="sm">操作</Button></template>
        右侧带操作按钮
      </Panel>
    </div>
  </div>
</template>
