<script setup lang="ts">
import { type Component } from 'vue'
import { AlertTriangle, CheckCircle2, Info } from 'lucide-vue-next'
import type { DocBlock, DocPage } from '@/lib/mock/site-docs'
import { safeDocUrl, sanitizeDocHtml } from '@/lib/site-docs'

const calloutIcons: Record<string, Component> = {
  info: Info,
  warning: AlertTriangle,
  success: CheckCircle2,
}

withDefaults(defineProps<{
  page: DocPage
  compact?: boolean
}>(), {
  compact: false,
})

interface SafeLinkAttributes {
  href: string
  target?: '_blank'
  rel?: 'noopener noreferrer'
}

function isExternalUrl(url: string): boolean {
  return /^https?:\/\//i.test(url)
}

function safeLinkAttributes(url: string): SafeLinkAttributes {
  const href = safeDocUrl(url)
  return isExternalUrl(href)
    ? { href, target: '_blank', rel: 'noopener noreferrer' }
    : { href }
}

function sanitizedBlockHtml(block: DocBlock): string {
  if (block.type !== 'richText' && block.type !== 'callout') return ''

  const html = sanitizeDocHtml(block.html)
  if (typeof document === 'undefined') return html

  const template = document.createElement('template')
  template.innerHTML = html

  template.content.querySelectorAll('a').forEach((link) => {
    const attributes = safeLinkAttributes(link.getAttribute('href') ?? '')
    link.setAttribute('href', attributes.href)

    if (attributes.target && attributes.rel) {
      link.setAttribute('target', attributes.target)
      link.setAttribute('rel', attributes.rel)
    } else {
      link.removeAttribute('target')
      link.removeAttribute('rel')
    }
  })

  template.content.querySelectorAll('img').forEach((image) => {
    const src = safeDocUrl(image.getAttribute('src') ?? '')
    if (src === '#') image.remove()
    else image.setAttribute('src', src)
  })

  return template.innerHTML
}

function tableCellClass(kind: 'text' | 'code' | 'status' | undefined, value: string): string | undefined {
  if (kind === 'code') return 'doc-table-code'
  if (kind !== 'status') return undefined

  const normalized = value.trim()
  if (normalized === '0' || normalized === '1') return 'doc-table-status-success'
  if (/^-\d+(?:\.\d+)?$/.test(normalized)) return 'doc-table-status-error'
  return undefined
}
</script>

<template>
  <div class="doc-page-content" :class="{ 'doc-page-content--compact': compact }">
    <section
      v-for="(section, sectionIndex) in page.sections"
      :key="section.id"
      class="doc-section"
      :id="section.title ? undefined : section.anchor"
    >
      <h2 v-if="section.title && sectionIndex === 0" :id="section.anchor" class="doc-h2">
        {{ section.title }}
      </h2>
      <h3 v-else-if="section.title" :id="section.anchor" class="doc-h3">
        {{ section.title }}
      </h3>

      <template v-for="block in section.blocks" :key="block.id">
        <div
          v-if="block.type === 'richText'"
          class="doc-rich"
          v-html="sanitizedBlockHtml(block)"
        />

        <div v-else-if="block.type === 'endpoint'" class="doc-meta">
          <span class="doc-method">{{ block.method }}</span>
          <code class="doc-url">{{ block.url }}</code>
        </div>

        <div v-else-if="block.type === 'table'" class="doc-table-wrap">
          <table class="tbl doc-table">
            <colgroup>
              <col
                v-for="column in block.columns"
                :key="column.key"
                :style="column.width ? { width: column.width } : undefined"
              />
            </colgroup>
            <thead>
              <tr>
                <th v-for="column in block.columns" :key="column.key">
                  {{ column.label }}
                </th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(row, rowIndex) in block.rows" :key="rowIndex">
                <td
                  v-for="column in block.columns"
                  :key="column.key"
                  :class="tableCellClass(column.kind, row[column.key] ?? '')"
                >
                  {{ row[column.key] ?? '' }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <div v-else-if="block.type === 'code'" class="doc-code">
          <div class="doc-code-bar">
            <span class="doc-code-dots"><i /><i /><i /></span>
            <span v-if="block.language" class="doc-code-lang">{{ block.language }}</span>
          </div>
          <pre class="doc-code-body"><code>{{ block.code }}</code></pre>
        </div>

        <div
          v-else-if="block.type === 'callout'"
          class="doc-callout"
          :class="`doc-callout--${block.tone}`"
        >
          <span class="doc-callout-badge"><component :is="calloutIcons[block.tone] ?? Info" /></span>
          <div class="doc-callout-body doc-rich" v-html="sanitizedBlockHtml(block)" />
        </div>

        <div v-else-if="block.type === 'steps'" class="doc-steps">
          <div v-for="(item, itemIndex) in block.items" :key="itemIndex" class="doc-step">
            <span class="doc-step-number">{{ itemIndex + 1 }}</span>
            <span>{{ item }}</span>
          </div>
        </div>

        <div v-else-if="block.type === 'links'" class="doc-links">
          <a
            v-for="(item, itemIndex) in block.items"
            :key="itemIndex"
            v-bind="safeLinkAttributes(item.url)"
            class="doc-link"
            :class="item.primary ? 'doc-link--primary' : 'doc-link--secondary'"
          >
            {{ item.label }}
          </a>
        </div>

        <div v-else-if="block.type === 'faq'" class="doc-faq-list">
          <div v-for="(item, itemIndex) in block.items" :key="itemIndex" class="doc-faq-item">
            <h4 class="doc-faq-question">
              <span class="doc-faq-number">Q{{ itemIndex + 1 }}.</span>
              <span>{{ item.q }}</span>
            </h4>
            <p class="doc-p doc-faq-answer">{{ item.a }}</p>
          </div>
        </div>
      </template>
    </section>
  </div>
</template>

<style scoped>
.doc-page-content {
  color: var(--foreground);
}

.doc-section + .doc-section {
  margin-top: 2.5rem;
}

/* A 方向：统一 block 垂直节奏，替代各块零散 margin-top */
.doc-section > .doc-meta,
.doc-section > .doc-table-wrap,
.doc-section > .doc-code,
.doc-section > .doc-callout,
.doc-section > .doc-steps,
.doc-section > .doc-links,
.doc-section > .doc-faq-list,
.doc-section > .doc-rich {
  margin-top: 1.1rem;
}
.doc-section > :first-child {
  margin-top: 0;
}

.doc-h2 {
  font-size: 1.35rem;
  font-weight: 700;
  letter-spacing: -0.01em;
  scroll-margin-top: 5rem;
}

.doc-h3 {
  margin-top: 1.5rem;
  margin-bottom: 0.25rem;
  font-size: 0.95rem;
  font-weight: 600;
  scroll-margin-top: 5rem;
}

.doc-p,
.doc-rich :deep(p) {
  margin-top: 0.75rem;
  font-size: 0.9rem;
  line-height: 1.75;
  color: var(--muted-foreground);
}

.doc-rich :deep(p:first-child) {
  margin-top: 0.75rem;
}

.doc-rich :deep(strong),
.doc-rich :deep(b) {
  font-weight: 600;
  color: var(--foreground);
}

.doc-rich :deep(em),
.doc-rich :deep(i) {
  font-style: italic;
}

.doc-rich :deep(s) {
  text-decoration: line-through;
}

.doc-rich :deep(u) {
  text-decoration: underline;
  text-underline-offset: 0.15em;
}

.doc-rich :deep(h2),
.doc-rich :deep(h3) {
  margin-top: 1.5rem;
  margin-bottom: 0.25rem;
  font-weight: 600;
  color: var(--foreground);
}

.doc-rich :deep(h2) {
  font-size: 1.1rem;
}

.doc-rich :deep(h3) {
  font-size: 0.95rem;
}

.doc-rich :deep(ul),
.doc-rich :deep(ol) {
  margin-top: 0.75rem;
  padding-left: 1.35rem;
  font-size: 0.9rem;
  line-height: 1.75;
  color: var(--muted-foreground);
}

.doc-rich :deep(ul) {
  list-style: disc;
}

.doc-rich :deep(ol) {
  list-style: decimal;
}

.doc-rich :deep(li + li) {
  margin-top: 0.2rem;
}

.doc-rich :deep(blockquote) {
  margin-top: 0.75rem;
  border-left: 3px solid color-mix(in oklch, var(--primary) 45%, transparent);
  padding: 0.5rem 0 0.5rem 1rem;
  color: var(--muted-foreground);
}

.doc-rich :deep(a) {
  color: var(--primary);
  text-decoration: underline;
  text-decoration-color: color-mix(in oklch, var(--primary) 35%, transparent);
  text-underline-offset: 0.2em;
}

.doc-rich :deep(a:hover) {
  text-decoration-color: var(--primary);
}

.doc-rich :deep(code) {
  border-radius: 0.25rem;
  background: var(--muted);
  padding: 0.1rem 0.4rem;
  font-family: ui-monospace, "SFMono-Regular", Menlo, Consolas, monospace;
  font-size: 0.82em;
  color: var(--primary);
}

.doc-rich :deep(pre) {
  margin-top: 0.75rem;
  overflow-x: auto;
  border-radius: 0.5rem;
  border: 1px solid var(--border);
  background: oklch(0.18 0.02 264);
  padding: 1rem 1.25rem;
  font-family: ui-monospace, "SFMono-Regular", Menlo, Consolas, monospace;
  font-size: 12.5px;
  line-height: 1.7;
  color: oklch(0.9 0.02 264);
}

.doc-rich :deep(pre code) {
  border-radius: 0;
  background: transparent;
  padding: 0;
  font-size: inherit;
  color: inherit;
}

.doc-rich :deep(img) {
  display: block;
  max-width: 100%;
  height: auto;
  margin-top: 0.75rem;
  border-radius: 0.5rem;
  border: 1px solid var(--border);
}

.doc-rich :deep(hr) {
  margin: 1.5rem 0;
  border: 0;
  border-top: 1px solid var(--border);
}

/* A 方向：参数表做成带边框圆角 + 表头底 + 斑马纹的精致表 */
.doc-table-wrap {
  overflow-x: auto;
  border: 1px solid var(--border);
  border-radius: 0.5rem;
  box-shadow: 0 1px 3px oklch(0 0 0 / 4%);
}

.doc-table {
  width: 100%;
  table-layout: fixed;
  border-collapse: separate;
  border-spacing: 0;
  font-size: 0.82rem;
}

.doc-table :deep(th) {
  padding: 0.55rem 0.85rem;
  background: color-mix(in oklch, var(--muted) 55%, var(--background));
  border-bottom: 1px solid var(--border);
  text-align: left;
  font-weight: 600;
  color: var(--muted-foreground);
}

.doc-table :deep(td) {
  padding: 0.55rem 0.85rem;
  border-bottom: 1px solid color-mix(in oklch, var(--border) 60%, transparent);
  color: var(--foreground);
}

.doc-table :deep(tbody tr:nth-child(even) td) {
  background: color-mix(in oklch, var(--muted) 28%, transparent);
}

.doc-table :deep(tbody tr:last-child td) {
  border-bottom: 0;
}

.doc-table th,
.doc-table td {
  overflow-wrap: anywhere;
}

.doc-table-code {
  font-family: ui-monospace, "SFMono-Regular", Menlo, Consolas, monospace;
  font-size: 13px;
  color: var(--primary);
}

.doc-table-status-success {
  font-family: ui-monospace, "SFMono-Regular", Menlo, Consolas, monospace;
  color: var(--success);
}

.doc-table-status-error {
  font-family: ui-monospace, "SFMono-Regular", Menlo, Consolas, monospace;
  color: var(--destructive);
}

/* A 方向：endpoint 做成带渐变底的精致卡 */
.doc-meta {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  min-width: 0;
  border: 1px solid var(--border);
  border-radius: 0.5rem;
  padding: 0.6rem 0.85rem;
  background: linear-gradient(180deg, color-mix(in oklch, var(--primary) 4%, var(--background)), var(--background));
  box-shadow: 0 1px 2px oklch(0 0 0 / 4%);
}

.doc-method {
  flex: none;
  border-radius: 0.375rem;
  background: color-mix(in oklch, var(--primary) 12%, transparent);
  padding: 0.2rem 0.6rem;
  font-family: ui-monospace, "SFMono-Regular", Menlo, Consolas, monospace;
  font-size: 0.72rem;
  font-weight: 700;
  letter-spacing: 0.03em;
  color: var(--primary);
}

.doc-url {
  overflow-wrap: anywhere;
  font-family: ui-monospace, "SFMono-Regular", Menlo, Consolas, monospace;
  font-size: 0.82rem;
  color: var(--foreground);
}

/* A 方向：代码块带窗口栏 + 阴影 */
.doc-code {
  border-radius: 0.5rem;
  border: 1px solid oklch(0.28 0.02 264);
  background: oklch(0.18 0.02 264);
  overflow: hidden;
  box-shadow: 0 2px 8px oklch(0 0 0 / 12%);
}

.doc-code-bar {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 0.85rem;
  background: oklch(0.15 0.02 264);
  border-bottom: 1px solid oklch(0.26 0.02 264);
}

.doc-code-dots {
  display: inline-flex;
  gap: 0.375rem;
}

.doc-code-dots i {
  width: 0.625rem;
  height: 0.625rem;
  border-radius: 999px;
}

.doc-code-dots i:nth-child(1) { background: oklch(0.6 0.16 25); }
.doc-code-dots i:nth-child(2) { background: oklch(0.75 0.14 85); }
.doc-code-dots i:nth-child(3) { background: oklch(0.65 0.15 145); }

.doc-code-lang {
  margin-left: auto;
  font-family: ui-monospace, "SFMono-Regular", Menlo, Consolas, monospace;
  font-size: 0.7rem;
  letter-spacing: 0.03em;
  color: oklch(0.68 0.02 264);
  text-transform: uppercase;
}

.doc-code-body {
  margin: 0;
  overflow-x: auto;
  padding: 1rem 1.25rem;
  font-family: ui-monospace, "SFMono-Regular", Menlo, Consolas, monospace;
  font-size: 12.5px;
  line-height: 1.7;
  color: oklch(0.9 0.02 264);
}

.doc-inline {
  border-radius: 0.25rem;
  background: var(--muted);
  padding: 0.1rem 0.4rem;
  font-family: ui-monospace, "SFMono-Regular", Menlo, Consolas, monospace;
  font-size: 0.82em;
  color: var(--primary);
}

/* B 风格：柔和 icon + 图标色外边框 + 淡底卡 */
/* 1:1 复用 Badge 的 Element UI 淡底描边配色（淡底 + 更浅同色描边 + 同色图标） */
.doc-callout {
  --c: #409eff;
  --c-bg: #ecf5ff;
  --c-bd: #d9ecff;
  display: flex;
  gap: 0.65rem;
  border: 1px solid var(--c-bd);
  border-radius: 0.625rem;
  background: var(--c-bg);
  padding: 0.85rem 1rem;
}

.doc-callout--info { --c: #409eff; --c-bg: #ecf5ff; --c-bd: #d9ecff; }
.doc-callout--warning { --c: #e6a23c; --c-bg: #fdf6ec; --c-bd: #faecd8; }
.doc-callout--success { --c: #67c23a; --c-bg: #f0f9eb; --c-bd: #e1f3d8; }
.doc-callout--error { --c: #f56c6c; --c-bg: #fef0f0; --c-bd: #fde2e2; }

/* 图标：饱和纯色，最突出，无底框 */
.doc-callout-badge {
  display: inline-flex;
  flex: none;
  margin-top: 0.15rem;
  color: var(--c);
}

.doc-callout-badge :deep(svg) {
  width: 1.1rem;
  height: 1.1rem;
  stroke-width: 2.2;
}

.doc-callout-body {
  flex: 1;
  min-width: 0;
  overflow-wrap: anywhere;
  word-break: break-word;
}

.doc-callout-body :deep(p:first-child) {
  margin-top: 0;
}

.doc-callout-body :deep(p) {
  color: var(--foreground);
  overflow-wrap: anywhere;
}

.doc-steps {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.75rem;
  margin-top: 1rem;
}

.doc-step {
  display: flex;
  min-width: 0;
  gap: 0.65rem;
  border: 1px solid var(--border);
  border-radius: 0.5rem;
  background: var(--background);
  padding: 0.85rem;
  font-size: 0.875rem;
  line-height: 1.6;
  box-shadow: 0 1px 2px oklch(0 0 0 / 4%);
}

.doc-step-number {
  display: inline-flex;
  width: 1.25rem;
  height: 1.25rem;
  flex: none;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  background: color-mix(in oklch, var(--primary) 12%, transparent);
  font-size: 0.7rem;
  font-weight: 700;
  color: var(--primary);
}

.doc-links {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  margin-top: 1rem;
}

.doc-link {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 0.5rem;
  padding: 0.5rem 1rem;
  font-size: 0.875rem;
  font-weight: 500;
  transition: border-color 150ms ease, color 150ms ease, opacity 150ms ease;
}

.doc-link--primary {
  background: var(--primary);
  color: var(--primary-foreground);
  box-shadow: 0 1px 3px color-mix(in oklch, var(--primary) 30%, transparent);
}

.doc-link--primary:hover {
  opacity: 0.9;
}

.doc-link--secondary {
  border: 1px solid var(--border);
  background: var(--background);
  color: var(--foreground);
}

.doc-link--secondary:hover {
  border-color: color-mix(in oklch, var(--primary) 40%, var(--border));
  color: var(--primary);
}

/* A 方向：FAQ 每条一张精致卡 */
.doc-faq-list {
  display: flex;
  flex-direction: column;
  gap: 0.6rem;
}

.doc-faq-item {
  border: 1px solid var(--border);
  border-radius: 0.5rem;
  background: var(--background);
  padding: 0.85rem 1rem;
  box-shadow: 0 1px 2px oklch(0 0 0 / 4%);
}

.doc-faq-question {
  display: flex;
  gap: 0.5rem;
  font-size: 0.95rem;
  font-weight: 600;
  line-height: 1.6;
  color: var(--foreground);
}

.doc-faq-number {
  flex: none;
  color: var(--primary);
}

.doc-faq-answer {
  margin-top: 0.35rem;
  padding-left: 1.5rem;
}

.doc-page-content--compact .doc-section + .doc-section {
  margin-top: 1.5rem;
}

.doc-page-content--compact .doc-h3 {
  margin-top: 1rem;
}

.doc-page-content--compact .doc-section > .doc-meta,
.doc-page-content--compact .doc-section > .doc-table-wrap,
.doc-page-content--compact .doc-section > .doc-code,
.doc-page-content--compact .doc-section > .doc-callout,
.doc-page-content--compact .doc-section > .doc-steps,
.doc-page-content--compact .doc-section > .doc-links,
.doc-page-content--compact .doc-section > .doc-faq-list,
.doc-page-content--compact .doc-section > .doc-rich {
  margin-top: 0.85rem;
}

.doc-page-content--compact .doc-step,
.doc-page-content--compact .doc-code-body,
.doc-page-content--compact .doc-rich :deep(pre) {
  padding: 0.75rem;
}

@media (max-width: 767px) {
  .doc-steps {
    grid-template-columns: 1fr;
  }

  .doc-table {
    width: max-content;
    min-width: 100%;
    table-layout: auto;
  }
}
</style>
