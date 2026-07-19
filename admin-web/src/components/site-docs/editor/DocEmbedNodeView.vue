<script setup lang="ts">
import { computed, markRaw, type Component } from 'vue'
import { NodeViewWrapper, nodeViewProps } from '@tiptap/vue-3'
import {
  AlertTriangle,
  CheckCircle2,
  GripVertical,
  HelpCircle,
  Info,
  Link as LinkIcon,
  ListOrdered,
  Plus,
  Table as TableIcon,
  Terminal,
  Trash2,
  Zap,
} from 'lucide-vue-next'
import { Select } from '@/components/ui'
import type { DocBlock } from '@/lib/mock/site-docs'

type EmbedBlock = Exclude<DocBlock, { type: 'richText' }>
type EmbedBlockType = EmbedBlock['type']
type PayloadFor<T extends EmbedBlockType> = Omit<Extract<EmbedBlock, { type: T }>, 'id' | 'type'>
type EmbedPayload = PayloadFor<EmbedBlockType>
type CalloutPayload = PayloadFor<'callout'>
type EndpointPayload = PayloadFor<'endpoint'>
type TablePayload = PayloadFor<'table'>
type CodePayload = PayloadFor<'code'>
type StepsPayload = PayloadFor<'steps'>
type LinksPayload = PayloadFor<'links'>
type FaqPayload = PayloadFor<'faq'>
type TableColumn = TablePayload['columns'][number]

type ModuleMeta = {
  label: string
  accent: string
  icon: Component
}

const props = defineProps(nodeViewProps)

const moduleMeta: Record<EmbedBlockType, ModuleMeta> = {
  callout: { label: '提示信息', accent: 'accent-callout', icon: Info },
  endpoint: { label: '接口地址', accent: 'accent-endpoint', icon: Terminal },
  table: { label: '参数表格', accent: 'accent-table', icon: TableIcon },
  code: { label: '代码示例', accent: 'accent-code', icon: Zap },
  steps: { label: '步骤列表', accent: 'accent-steps', icon: ListOrdered },
  links: { label: '相关链接', accent: 'accent-links', icon: LinkIcon },
  faq: { label: '常见问题', accent: 'accent-faq', icon: HelpCircle },
}

const toneOptions = [
  { value: 'info', label: '信息', icon: markRaw(Info) },
  { value: 'warning', label: '警告', icon: markRaw(AlertTriangle) },
  { value: 'success', label: '成功', icon: markRaw(CheckCircle2) },
]

const methodOptions = [
  { value: 'GET', label: 'GET' },
  { value: 'POST', label: 'POST' },
]

// 文档常用代码语言，覆盖聚合支付对接场景（请求/响应/示例代码）
const codeLangOptions = [
  { value: 'json', label: 'JSON' },
  { value: 'http', label: 'HTTP' },
  { value: 'bash', label: 'Shell / cURL' },
  { value: 'php', label: 'PHP' },
  { value: 'javascript', label: 'JavaScript' },
  { value: 'python', label: 'Python' },
  { value: 'java', label: 'Java' },
  { value: 'go', label: 'Go' },
  { value: 'html', label: 'HTML' },
  { value: 'xml', label: 'XML' },
  { value: 'sql', label: 'SQL' },
  { value: 'text', label: '纯文本' },
]

const columnKindOptions = [
  { value: 'text', label: '文本' },
  { value: 'code', label: '代码' },
  { value: 'status', label: '状态' },
]

const blockType = computed(() => props.node.attrs.blockType as EmbedBlockType)
const meta = computed(() => moduleMeta[blockType.value] ?? moduleMeta.callout)
const payload = computed(() => props.node.attrs.payload as EmbedPayload)

const calloutPayload = computed(() => payload.value as CalloutPayload)
const calloutIcon = computed(() => {
  const map: Record<string, Component> = { info: Info, warning: AlertTriangle, success: CheckCircle2 }
  return map[calloutPayload.value.tone] ?? Info
})
const endpointPayload = computed(() => payload.value as EndpointPayload)
const tablePayload = computed(() => payload.value as TablePayload)
const codePayload = computed(() => payload.value as CodePayload)
const stepsPayload = computed(() => payload.value as StepsPayload)
const linksPayload = computed(() => payload.value as LinksPayload)
const faqPayload = computed(() => payload.value as FaqPayload)

function cloneValue<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T
}

function commitPayload<T extends EmbedPayload>(nextPayload: T) {
  props.updateAttributes({ payload: cloneValue(nextPayload) })
}

function mutatePayload<T extends EmbedPayload>(mutate: (draft: T) => void) {
  const draft = cloneValue(payload.value) as T
  mutate(draft)
  commitPayload(draft)
}

function eventValue(event: Event) {
  return (event.target as HTMLInputElement | HTMLTextAreaElement).value
}

function setCalloutTone(value: string | number) {
  mutatePayload<CalloutPayload>((draft) => {
    draft.tone = value as CalloutPayload['tone']
  })
}

function setCalloutHtml(value: string) {
  mutatePayload<CalloutPayload>((draft) => {
    draft.html = value
  })
}

function setEndpointMethod(value: string | number) {
  mutatePayload<EndpointPayload>((draft) => {
    draft.method = value as EndpointPayload['method']
  })
}

function setEndpointUrl(value: string) {
  mutatePayload<EndpointPayload>((draft) => {
    draft.url = value
  })
}

function setCodeLanguage(value: string) {
  mutatePayload<CodePayload>((draft) => {
    draft.language = value
  })
}

function setCode(value: string) {
  mutatePayload<CodePayload>((draft) => {
    draft.code = value
  })
}

function setStep(index: number, value: string) {
  mutatePayload<StepsPayload>((draft) => {
    draft.items[index] = value
  })
}

function addStep() {
  mutatePayload<StepsPayload>((draft) => {
    draft.items.push('')
  })
}

function removeStep(index: number) {
  mutatePayload<StepsPayload>((draft) => {
    draft.items.splice(index, 1)
  })
}

function setLink(index: number, field: 'label' | 'url', value: string) {
  mutatePayload<LinksPayload>((draft) => {
    const item = draft.items[index]
    if (item) item[field] = value
  })
}

function setLinkPrimary(index: number, primary: boolean) {
  mutatePayload<LinksPayload>((draft) => {
    const item = draft.items[index]
    if (item) item.primary = primary
  })
}

function addLink() {
  mutatePayload<LinksPayload>((draft) => {
    draft.items.push({ label: '', url: '', primary: false })
  })
}

function removeLink(index: number) {
  mutatePayload<LinksPayload>((draft) => {
    draft.items.splice(index, 1)
  })
}

function setFaq(index: number, field: 'q' | 'a', value: string) {
  mutatePayload<FaqPayload>((draft) => {
    const item = draft.items[index]
    if (item) item[field] = value
  })
}

function addFaq() {
  mutatePayload<FaqPayload>((draft) => {
    draft.items.push({ q: '', a: '' })
  })
}

function removeFaq(index: number) {
  mutatePayload<FaqPayload>((draft) => {
    draft.items.splice(index, 1)
  })
}

function uniqueColumnKey(columns: TableColumn[]) {
  let index = columns.length + 1
  let key = `field_${index}`
  while (columns.some((column) => column.key === key)) {
    index += 1
    key = `field_${index}`
  }
  return key
}

function addColumn() {
  mutatePayload<TablePayload>((draft) => {
    const key = uniqueColumnKey(draft.columns)
    draft.columns.push({ key, label: '新列', kind: 'text' })
    draft.rows.forEach((row) => {
      row[key] = ''
    })
  })
}

function removeColumn(index: number) {
  mutatePayload<TablePayload>((draft) => {
    const column = draft.columns[index]
    if (!column) return
    draft.columns.splice(index, 1)
    draft.rows.forEach((row) => {
      delete row[column.key]
    })
  })
}

function setColumn(index: number, field: 'label' | 'width', value: string) {
  mutatePayload<TablePayload>((draft) => {
    const column = draft.columns[index]
    if (!column) return
    if (field === 'width' && value === '') delete column.width
    else column[field] = value
  })
}

function setColumnKind(index: number, value: string | number) {
  mutatePayload<TablePayload>((draft) => {
    const column = draft.columns[index]
    if (column) column.kind = value as NonNullable<TableColumn['kind']>
  })
}

function setColumnKey(index: number, value: string) {
  mutatePayload<TablePayload>((draft) => {
    const column = draft.columns[index]
    if (!column || value === column.key) return
    const oldKey = column.key
    column.key = value
    draft.rows.forEach((row) => {
      row[value] = row[oldKey] ?? ''
      delete row[oldKey]
    })
  })
}

function addRow() {
  mutatePayload<TablePayload>((draft) => {
    const row: Record<string, string> = {}
    draft.columns.forEach((column) => {
      row[column.key] = ''
    })
    draft.rows.push(row)
  })
}

function removeRow(index: number) {
  mutatePayload<TablePayload>((draft) => {
    draft.rows.splice(index, 1)
  })
}

function setCell(rowIndex: number, key: string, value: string) {
  mutatePayload<TablePayload>((draft) => {
    const row = draft.rows[rowIndex]
    if (row) row[key] = value
  })
}
</script>

<template>
  <!-- 统一「工具面板」外壳：深色标题栏 + 白色面板体，7 种模块共用 -->
  <NodeViewWrapper
    class="panel group"
    :class="[meta.accent, { 'is-selected': selected }]"
    contenteditable="false"
    @mousedown.stop
    @click.stop
    @keydown.stop
    @keyup.stop
  >
    <div class="panel-head">
      <button type="button" class="panel-grip" data-drag-handle title="拖动模块" aria-label="拖动模块" @mousedown.stop @click.stop>
        <GripVertical />
      </button>
      <component :is="meta.icon" class="panel-ic" />
      <span class="panel-name">{{ meta.label }}</span>
      <span class="panel-type">{{ blockType }}</span>
      <button type="button" class="panel-del" title="删除模块" aria-label="删除模块" @mousedown.stop @click.stop="deleteNode">
        <Trash2 />
      </button>
    </div>

    <div class="panel-body">
      <!-- 提示信息：图标 + 类型选择 + 内容合为一个彩色容器 -->
      <template v-if="blockType === 'callout'">
        <div class="callout-preview" :class="`callout-preview--${calloutPayload.tone}`">
          <span class="callout-badge"><component :is="calloutIcon" /></span>
          <Select class="callout-tone" :model-value="calloutPayload.tone" :options="toneOptions" @update:model-value="setCalloutTone" />
          <textarea
            class="callout-text"
            rows="1"
            :value="calloutPayload.html"
            placeholder="输入提示内容，可保留简单 HTML 标签"
            @input="setCalloutHtml(eventValue($event))"
          />
        </div>
      </template>

      <!-- 接口地址 -->
      <template v-else-if="blockType === 'endpoint'">
        <div class="frow">
          <span class="flabel">方法</span>
          <Select class="ep-method" :model-value="endpointPayload.method" :options="methodOptions" @update:model-value="setEndpointMethod" />
          <span class="flabel">地址</span>
          <input class="fld ep-url" :value="endpointPayload.url" placeholder="{apiurl}api/pay/create" @input="setEndpointUrl(eventValue($event))" />
        </div>
      </template>

      <!-- 代码示例 -->
      <template v-else-if="blockType === 'code'">
        <div class="code-wrap">
          <div class="code-bar">
            <span class="code-dots"><i /><i /><i /></span>
            <select class="code-lang" :value="codePayload.language || 'text'" title="代码语言" @change="setCodeLanguage(eventValue($event))">
              <option v-for="opt in codeLangOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
            </select>
          </div>
          <textarea class="code-body" :value="codePayload.code" spellcheck="false" placeholder="粘贴代码…" @input="setCode(eventValue($event))" />
        </div>
      </template>

      <!-- 步骤列表 -->
      <template v-else-if="blockType === 'steps'">
        <div class="steps">
          <div v-for="(item, index) in stepsPayload.items" :key="index" class="step">
            <span class="step-num">{{ index + 1 }}</span>
            <textarea class="step-text" rows="1" :value="item" placeholder="步骤说明" @input="setStep(index, eventValue($event))" />
            <button type="button" class="mini-del" title="删除步骤" @click="removeStep(index)"><Trash2 /></button>
          </div>
          <button type="button" class="panel-add" @click="addStep"><Plus />添加步骤</button>
        </div>
      </template>

      <!-- 相关链接 -->
      <template v-else-if="blockType === 'links'">
        <div class="links">
          <div v-for="(item, index) in linksPayload.items" :key="index" class="link">
            <span class="flabel">文字</span>
            <input class="fld link-label" :value="item.label" placeholder="按钮文字" @input="setLink(index, 'label', eventValue($event))" />
            <span class="flabel">地址</span>
            <input class="fld link-url" :value="item.url" placeholder="https:// 或 /docs/" @input="setLink(index, 'url', eventValue($event))" />
            <button type="button" class="link-toggle" :class="{ 'is-on': item.primary }" :title="item.primary ? '主按钮' : '次按钮'" @click="setLinkPrimary(index, !item.primary)">{{ item.primary ? '主' : '次' }}</button>
            <button type="button" class="mini-del" title="删除链接" @click="removeLink(index)"><Trash2 /></button>
          </div>
          <button type="button" class="panel-add" @click="addLink"><Plus />添加链接</button>
        </div>
      </template>

      <!-- 常见问题 -->
      <template v-else-if="blockType === 'faq'">
        <div class="faqs">
          <div v-for="(item, index) in faqPayload.items" :key="index" class="faq">
            <div class="faq-head">
              <span class="faq-q">Q{{ index + 1 }}</span>
              <input class="fld faq-question" :value="item.q" placeholder="输入问题" @input="setFaq(index, 'q', eventValue($event))" />
              <button type="button" class="mini-del" title="删除问答" @click="removeFaq(index)"><Trash2 /></button>
            </div>
            <textarea class="fld faq-answer" rows="2" :value="item.a" placeholder="输入回答" @input="setFaq(index, 'a', eventValue($event))" />
          </div>
          <button type="button" class="panel-add" @click="addFaq"><Plus />添加问答</button>
        </div>
      </template>

      <!-- 参数表格 -->
      <template v-else-if="blockType === 'table'">
        <div v-if="tablePayload.columns.length" class="tbl-wrap">
          <table class="tbl-edit">
            <thead>
              <tr>
                <th v-for="(column, index) in tablePayload.columns" :key="`${column.key}-${index}`" :style="column.width ? { width: column.width } : undefined">
                  <div class="th-cell">
                    <input class="th-name" :value="column.label" placeholder="列名" @input="setColumn(index, 'label', eventValue($event))" />
                    <div class="th-meta">
                      <input class="th-key" :value="column.key" title="字段键" placeholder="field" @change="setColumnKey(index, eventValue($event))" />
                      <select class="th-kind" :value="column.kind ?? 'text'" title="列类型" @change="setColumnKind(index, eventValue($event))">
                        <option v-for="opt in columnKindOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
                      </select>
                      <input class="th-width" :value="column.width ?? ''" title="宽度，如 20%" placeholder="宽" @input="setColumn(index, 'width', eventValue($event))" />
                      <button type="button" class="mini-del" title="删除列" @click="removeColumn(index)"><Trash2 /></button>
                    </div>
                  </div>
                </th>
                <th class="col-add-cell">
                  <button type="button" class="col-add" title="新增列" @click="addColumn"><Plus /></button>
                </th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(row, rowIndex) in tablePayload.rows" :key="rowIndex">
                <td v-for="column in tablePayload.columns" :key="column.key">
                  <input class="cell-input" :value="row[column.key] ?? ''" @input="setCell(rowIndex, column.key, eventValue($event))" />
                </td>
                <td class="row-del-cell">
                  <button type="button" class="mini-del" title="删除行" @click="removeRow(rowIndex)"><Trash2 /></button>
                </td>
              </tr>
              <tr v-if="tablePayload.rows.length === 0">
                <td :colspan="tablePayload.columns.length + 1" class="empty-cell">暂无数据行，点击下方添加</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-else class="empty-state">
          <button type="button" class="panel-add" @click="addColumn"><Plus />添加第一列</button>
        </div>
        <button v-if="tablePayload.columns.length" type="button" class="panel-add" @click="addRow"><Plus />添加数据行</button>
      </template>
    </div>
  </NodeViewWrapper>
</template>

<style scoped>
/* ══ 统一工具面板外壳 ══ */
.panel {
  --embed-accent: oklch(0.53 0.23 262);
  position: relative;
  width: 100%;
  margin: 12px 0;
  border: 1px solid var(--border);
  border-radius: 0.5rem;
  background: var(--card);
  transition: box-shadow 0.15s, border-color 0.15s;
}
.panel.is-selected {
  border-color: color-mix(in oklch, var(--embed-accent) 50%, var(--border));
  box-shadow: 0 0 0 3px color-mix(in oklch, var(--embed-accent) 12%, transparent);
}

.accent-callout { --embed-accent: oklch(0.72 0.15 75); }
.accent-endpoint { --embed-accent: oklch(0.53 0.23 262); }
.accent-table { --embed-accent: oklch(0.6 0.13 180); }
.accent-code { --embed-accent: oklch(0.55 0.03 264); }
.accent-steps { --embed-accent: oklch(0.55 0.2 293); }
.accent-links { --embed-accent: oklch(0.58 0.15 233); }
.accent-faq { --embed-accent: oklch(0.62 0.19 40); }

/* 柔和的 accent 浅色标题栏，和内容区自然过渡，不生硬 */
.panel-head {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 36px;
  padding: 0 8px 0 6px;
  border-radius: 0.5rem 0.5rem 0 0;
  background: color-mix(in oklch, var(--embed-accent) 7%, var(--card));
  border-bottom: 1px solid color-mix(in oklch, var(--embed-accent) 14%, var(--border));
}
.panel-grip {
  display: inline-flex;
  width: 22px;
  height: 24px;
  flex: none;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 4px;
  color: var(--muted-foreground);
  background: transparent;
  cursor: grab;
  opacity: 0;
  transition: opacity 0.12s, color 0.12s, background 0.12s;
}
.panel:hover .panel-grip,
.panel.is-selected .panel-grip { opacity: 1; }
.panel-grip:hover { color: var(--foreground); background: color-mix(in oklch, var(--embed-accent) 12%, transparent); }
.panel-grip:active { cursor: grabbing; }
.panel-grip :deep(svg) { width: 15px; height: 15px; }
.panel-ic { width: 15px; height: 15px; flex: none; color: var(--embed-accent); }
.panel-name { font-size: 12.5px; font-weight: 600; letter-spacing: 0.01em; color: color-mix(in oklch, var(--embed-accent) 70%, var(--foreground)); }
.panel-type {
  padding: 1px 7px;
  border-radius: 999px;
  background: color-mix(in oklch, var(--embed-accent) 12%, transparent);
  color: color-mix(in oklch, var(--embed-accent) 75%, var(--muted-foreground));
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 9.5px;
  font-weight: 600;
  letter-spacing: 0.07em;
  text-transform: uppercase;
}
.panel-del {
  display: inline-flex;
  width: 24px;
  height: 24px;
  flex: none;
  margin-left: auto;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 4px;
  color: var(--muted-foreground);
  background: transparent;
  cursor: pointer;
  opacity: 0;
  transition: color 0.12s, background 0.12s, opacity 0.12s;
}
.panel:hover .panel-del,
.panel.is-selected .panel-del { opacity: 1; }
.panel-del:hover { color: var(--destructive); background: color-mix(in oklch, var(--destructive) 12%, transparent); }
.panel-del :deep(svg) { width: 14px; height: 14px; }

.panel-body { padding: 14px; }

/* 通用字段行与标签 */
.frow { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.frow + .callout-preview { margin-top: 10px; }
.flabel { flex: none; font-size: 11px; font-weight: 600; color: var(--muted-foreground); }
.fld {
  border: 1px solid var(--border);
  border-radius: 6px;
  padding: 5px 9px;
  background: var(--background);
  color: var(--foreground);
  font-size: 0.85rem;
  outline: none;
  transition: border-color 0.12s, box-shadow 0.12s;
}
.fld:hover { border-color: color-mix(in oklch, var(--ring) 40%, var(--border)); }
.fld:focus { border-color: var(--ring); box-shadow: 0 0 0 3px color-mix(in oklch, var(--ring) 12%, transparent); }
.fld::placeholder { color: color-mix(in oklch, var(--muted-foreground) 60%, transparent); }

/* 通用行内删除（列表项用，hover 浮现） */
.mini-del {
  display: inline-flex;
  width: 26px;
  height: 26px;
  flex: none;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 6px;
  color: color-mix(in oklch, var(--muted-foreground) 75%, transparent);
  background: transparent;
  cursor: pointer;
  opacity: 0;
  transition: color 0.12s, background 0.12s, opacity 0.12s;
}
.panel:hover .mini-del,
.panel.is-selected .mini-del { opacity: 1; }
.mini-del:hover { color: var(--destructive); background: color-mix(in oklch, var(--destructive) 10%, transparent); }
.mini-del :deep(svg) { width: 14px; height: 14px; }

/* 通用「添加」按钮 */
.panel-add {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin-top: 10px;
  border: 1px dashed color-mix(in oklch, var(--embed-accent) 40%, var(--border));
  border-radius: 6px;
  padding: 5px 12px;
  color: var(--embed-accent);
  background: transparent;
  font-size: 0.8rem;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.12s;
}
.panel-add:hover { background: color-mix(in oklch, var(--embed-accent) 8%, transparent); }
.panel-add :deep(svg) { width: 14px; height: 14px; }

/* ── 提示信息 ── */
.callout-tone { width: 108px; flex: none; }
.callout-tone :deep(button) { height: 30px; }
/* 1:1 复用 Badge 的 Element UI 淡底描边配色 */
.callout-preview {
  --c: #409eff;
  --c-bg: #ecf5ff;
  --c-bd: #d9ecff;
  display: flex;
  align-items: flex-start;
  gap: 10px;
  border: 1px solid var(--c-bd);
  border-radius: 10px;
  background: var(--c-bg);
  padding: 12px 14px;
}
.callout-preview--info { --c: #409eff; --c-bg: #ecf5ff; --c-bd: #d9ecff; }
.callout-preview--warning { --c: #e6a23c; --c-bg: #fdf6ec; --c-bd: #faecd8; }
.callout-preview--success { --c: #67c23a; --c-bg: #f0f9eb; --c-bd: #e1f3d8; }
.callout-preview--error { --c: #f56c6c; --c-bg: #fef0f0; --c-bd: #fde2e2; }
.callout-badge {
  display: inline-flex;
  flex: none;
  margin-top: 7px;
  color: var(--c);
}
.callout-badge :deep(svg) { width: 17px; height: 17px; stroke-width: 2.2; }
.callout-tone { width: 104px; flex: none; }
/* 只染触发按钮，不碰下拉浮层（浮层保持中性白底） */
.callout-tone :deep(> button) {
  height: 28px;
  border-color: var(--c-bd);
  background: var(--background);
}
.callout-tone :deep(> button > span) { color: var(--c); font-weight: 500; }
.callout-tone :deep(> button > svg:first-child) { color: var(--c); }
.callout-text {
  display: block;
  flex: 1;
  min-width: 0;
  min-height: 28px;
  resize: vertical;
  border: 0;
  padding: 4px 0 0;
  background: transparent;
  color: var(--foreground);
  font-size: 0.9rem;
  line-height: 1.7;
  outline: none;
  overflow-wrap: anywhere;
  word-break: break-word;
}
.callout-text::placeholder { color: color-mix(in oklch, var(--muted-foreground) 60%, transparent); }

/* ── 接口地址 ── */
.ep-method { width: 84px; flex: none; }
.ep-method :deep(button) {
  height: 30px;
  border-color: color-mix(in oklch, var(--primary) 35%, var(--border));
  background: color-mix(in oklch, var(--primary) 10%, var(--background));
}
.ep-method :deep(button > span) { color: var(--primary); font-weight: 700; font-size: 0.78rem; }
.ep-url { flex: 1; min-width: 12rem; font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; }

/* ── 代码示例：整块深色，顶部栏放窗口点 + 语言下拉 ── */
.code-wrap {
  border-radius: 0.5rem;
  border: 1px solid oklch(0.28 0.02 264);
  background: oklch(0.18 0.02 264);
  overflow: hidden;
}
.code-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border-bottom: 1px solid oklch(0.26 0.02 264);
  background: oklch(0.15 0.02 264);
}
.code-dots { display: inline-flex; gap: 6px; }
.code-dots i { width: 10px; height: 10px; border-radius: 999px; background: oklch(0.36 0.02 264); }
.code-dots i:nth-child(1) { background: oklch(0.6 0.16 25); }
.code-dots i:nth-child(2) { background: oklch(0.75 0.14 85); }
.code-dots i:nth-child(3) { background: oklch(0.65 0.15 145); }
.code-lang {
  margin-left: auto;
  height: 24px;
  border: 1px solid oklch(0.32 0.02 264);
  border-radius: 5px;
  padding: 0 6px;
  background: oklch(0.22 0.02 264);
  color: oklch(0.82 0.03 264);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 11px;
  outline: none;
  cursor: pointer;
}
.code-lang:focus { border-color: oklch(0.5 0.1 264); }
.code-body {
  display: block;
  width: 100%;
  min-height: 120px;
  resize: vertical;
  border: 0;
  padding: 0.9rem 1.1rem;
  background: transparent;
  color: oklch(0.9 0.02 264);
  font: 12.5px/1.7 ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  outline: none;
}
.code-body::placeholder { color: oklch(0.55 0.02 264); }

/* ── 步骤列表 ── */
.steps { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 8px; }
.step {
  position: relative;
  display: flex;
  gap: 8px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--background);
  padding: 9px 22px 9px 9px;
}
.step-num {
  display: inline-flex;
  width: 20px;
  height: 20px;
  flex: none;
  margin-top: 1px;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  background: color-mix(in oklch, var(--primary) 12%, transparent);
  color: var(--primary);
  font-size: 11px;
  font-weight: 700;
}
.step-text {
  flex: 1;
  min-width: 0;
  resize: none;
  border: 0;
  padding: 0;
  background: transparent;
  color: var(--foreground);
  font-size: 0.85rem;
  line-height: 1.55;
  outline: none;
}
.step-text::placeholder { color: color-mix(in oklch, var(--muted-foreground) 60%, transparent); }
.step .mini-del { position: absolute; top: 2px; right: 2px; width: 22px; height: 22px; }
.steps .panel-add { grid-column: 1 / -1; margin-top: 0; justify-content: center; }

/* ── 相关链接 ── */
.links { display: flex; flex-direction: column; gap: 7px; align-items: flex-start; }
.links .link { align-self: stretch; }
.links .panel-add, .faqs .panel-add { align-self: flex-start; }
.link { display: flex; align-items: center; gap: 7px; }
.link-label { width: 8rem; flex: none; }
.link-url { flex: 1; min-width: 0; font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; font-size: 0.78rem; }
.link-toggle {
  width: 28px;
  height: 28px;
  flex: none;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--background);
  color: var(--muted-foreground);
  font-size: 12px;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.12s;
}
.link-toggle.is-on { border-color: var(--primary); background: color-mix(in oklch, var(--primary) 12%, transparent); color: var(--primary); }

/* ── 常见问题 ── */
.faqs { display: flex; flex-direction: column; gap: 10px; }
.faq { display: flex; flex-direction: column; gap: 5px; }
.faq-head { display: flex; align-items: center; gap: 7px; }
.faq-q {
  flex: none;
  display: inline-flex;
  align-items: center;
  height: 20px;
  padding: 0 7px;
  border-radius: 4px;
  background: color-mix(in oklch, var(--primary) 12%, transparent);
  color: var(--primary);
  font-size: 11px;
  font-weight: 700;
}
.faq-question { flex: 1; min-width: 0; font-weight: 600; }
.faq-answer { margin-left: 27px; width: calc(100% - 27px); resize: vertical; line-height: 1.6; }

/* ── 参数表格 ── */
.tbl-wrap { overflow-x: auto; overflow-y: hidden; border: 1px solid var(--border); border-radius: 6px; background: var(--background); }
.tbl-edit { width: 100%; min-width: 480px; border-collapse: collapse; font-size: 0.8rem; }
.tbl-edit th {
  padding: 6px;
  border-bottom: 1px solid var(--border);
  border-right: 1px solid color-mix(in oklch, var(--border) 60%, transparent);
  background: color-mix(in oklch, var(--muted) 45%, transparent);
  vertical-align: top;
  text-align: left;
}
.tbl-edit td {
  padding: 0;
  border-bottom: 1px solid var(--border);
  border-right: 1px solid color-mix(in oklch, var(--border) 45%, transparent);
}
.tbl-edit th:last-child, .tbl-edit td:last-child { border-right: 0; }
.tbl-edit tbody tr:last-child td { border-bottom: 0; }
.th-cell { display: flex; flex-direction: column; gap: 5px; }
.th-name {
  border: 1px solid transparent;
  border-radius: 4px;
  padding: 3px 5px;
  background: transparent;
  color: var(--foreground);
  font-size: 0.8rem;
  font-weight: 600;
  outline: none;
}
.th-name:hover { border-color: var(--border); }
.th-name:focus { border-color: var(--ring); background: var(--background); }
.th-meta { display: flex; align-items: center; gap: 3px; opacity: 0; transition: opacity 0.12s; }
.panel:hover .th-meta, .panel.is-selected .th-meta { opacity: 1; }
.th-key {
  width: 56px;
  border: 1px solid var(--border);
  border-radius: 4px;
  padding: 2px 4px;
  background: var(--background);
  color: var(--muted-foreground);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 10px;
  outline: none;
}
.th-kind {
  width: 60px;
  height: 22px;
  flex: none;
  border: 1px solid var(--border);
  border-radius: 4px;
  padding: 0 4px;
  background: var(--background);
  color: var(--foreground);
  font-size: 10px;
  outline: none;
  cursor: pointer;
}
.th-kind:focus { border-color: var(--ring); }
.th-width {
  width: 34px;
  border: 1px solid var(--border);
  border-radius: 4px;
  padding: 2px 4px;
  background: var(--background);
  color: var(--muted-foreground);
  font-size: 10px;
  outline: none;
}
.th-meta .mini-del { width: 22px; height: 22px; opacity: 1; }
.col-add-cell { width: 32px; text-align: center; vertical-align: middle; }
.col-add {
  display: inline-flex;
  width: 22px;
  height: 22px;
  align-items: center;
  justify-content: center;
  border: 1px dashed var(--border);
  border-radius: 5px;
  color: var(--primary);
  background: transparent;
  cursor: pointer;
}
.col-add:hover { background: color-mix(in oklch, var(--primary) 8%, transparent); }
.col-add :deep(svg) { width: 13px; height: 13px; }
.cell-input {
  width: 100%;
  min-width: 88px;
  border: 0;
  padding: 6px 7px;
  background: transparent;
  color: var(--foreground);
  font-size: 0.8rem;
  outline: none;
}
.cell-input:focus { background: color-mix(in oklch, var(--ring) 6%, transparent); box-shadow: inset 0 0 0 1px var(--ring); }
.row-del-cell { width: 32px; text-align: center; }
.row-del-cell .mini-del { opacity: 1; }
.empty-cell { padding: 14px !important; color: var(--muted-foreground); text-align: center; }
.empty-state { padding: 4px 0; }

@media (max-width: 767px) {
  .steps { grid-template-columns: 1fr; }
  .link { flex-wrap: wrap; }
  .link-label, .link-url { width: 100%; flex: 1 1 100%; }
}
</style>
