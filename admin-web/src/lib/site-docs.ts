import DOMPurify from 'dompurify'
import type { DocBlock, DocPage, DocSection } from '@/lib/mock/site-docs'

export const DOC_SLUG_PATTERN = /^[a-z0-9]+(?:-[a-z0-9]+)*$/
export const DOC_ANCHOR_PATTERN = /^[a-z0-9]+(?:-[a-z0-9]+)*$/

/** 富文本只保留文档正文需要的安全标签与属性。 */
export function sanitizeDocHtml(html: string): string {
  return DOMPurify.sanitize(html, {
    ALLOWED_TAGS: [
      'p', 'br', 'strong', 'b', 'em', 'i', 's', 'u', 'h2', 'h3',
      'ul', 'ol', 'li', 'blockquote', 'code', 'pre', 'a', 'img', 'hr',
    ],
    ALLOWED_ATTR: ['href', 'target', 'rel', 'src', 'alt', 'title', 'style'],
    ALLOW_DATA_ATTR: false,
  })
}

/** 仅允许站内路径、锚点和 http(s) 外链。 */
export function safeDocUrl(url: string): string {
  const value = url.trim()
  if (/^(?:\/|#)(?!\/)/.test(value)) return value
  if (/^https?:\/\//i.test(value)) return value
  return '#'
}

export function isValidDocSlug(value: string): boolean {
  return DOC_SLUG_PATTERN.test(value)
}

export function isValidDocAnchor(value: string): boolean {
  return DOC_ANCHOR_PATTERN.test(value)
}

const EMBED_TONES = new Set(['info', 'warning', 'success'])
const COLUMN_KINDS = new Set(['text', 'code', 'status'])

/** 校验并补齐单个内容块；无法识别的块返回 null 由上层丢弃。 */
function normalizeDocBlock(block: DocBlock, sectionId: string, index: number): DocBlock | null {
  if (!block || typeof block !== 'object') return null
  const id = typeof block.id === 'string' && block.id ? block.id : `${sectionId}-block-${index}`
  switch (block.type) {
    case 'richText':
      return { id, type: 'richText', html: typeof block.html === 'string' ? block.html : '' }
    case 'callout':
      return {
        id,
        type: 'callout',
        tone: EMBED_TONES.has(block.tone) ? block.tone : 'info',
        html: typeof block.html === 'string' ? block.html : '',
      }
    case 'endpoint':
      return {
        id,
        type: 'endpoint',
        method: block.method === 'POST' ? 'POST' : 'GET',
        url: typeof block.url === 'string' ? block.url : '',
      }
    case 'table': {
      const columns = Array.isArray(block.columns)
        ? block.columns
            .filter((col) => col && typeof col === 'object' && typeof col.key === 'string')
            .map((col) => ({
              key: col.key,
              label: typeof col.label === 'string' ? col.label : '',
              ...(typeof col.width === 'string' ? { width: col.width } : {}),
              ...(col.kind && COLUMN_KINDS.has(col.kind) ? { kind: col.kind } : {}),
            }))
        : []
      const rows = Array.isArray(block.rows)
        ? block.rows
            .filter((row): row is Record<string, string> => !!row && typeof row === 'object')
            .map((row) => {
              const next: Record<string, string> = {}
              for (const col of columns) next[col.key] = typeof row[col.key] === 'string' ? row[col.key] : ''
              return next
            })
        : []
      return { id, type: 'table', columns, rows }
    }
    case 'code':
      return {
        id,
        type: 'code',
        language: typeof block.language === 'string' ? block.language : 'text',
        code: typeof block.code === 'string' ? block.code : '',
      }
    case 'steps':
      return {
        id,
        type: 'steps',
        items: Array.isArray(block.items) ? block.items.filter((i): i is string => typeof i === 'string') : [],
      }
    case 'links':
      return {
        id,
        type: 'links',
        items: Array.isArray(block.items)
          ? block.items
              .filter((i) => i && typeof i === 'object')
              .map((i) => ({
                label: typeof i.label === 'string' ? i.label : '',
                url: typeof i.url === 'string' ? i.url : '',
                primary: i.primary === true,
              }))
          : [],
      }
    case 'faq':
      return {
        id,
        type: 'faq',
        items: Array.isArray(block.items)
          ? block.items
              .filter((i) => i && typeof i === 'object')
              .map((i) => ({
                q: typeof i.q === 'string' ? i.q : '',
                a: typeof i.a === 'string' ? i.a : '',
              }))
          : [],
      }
    default:
      return null
  }
}

/**
 * V3 保序规范化：保留块原始顺序，正文与结构模块可任意混排。
 * 仅做结构校验、补稳定 id、合并相邻 richText；空章节补一个空 richText。
 */
export function normalizeDocSection(section: DocSection): DocSection {
  const sectionId = typeof section.id === 'string' && section.id ? section.id : 'section'
  const source = Array.isArray(section.blocks) ? section.blocks : []
  const normalized: DocBlock[] = []
  source.forEach((block, index) => {
    const next = normalizeDocBlock(block, sectionId, index)
    if (!next) return
    const prev = normalized[normalized.length - 1]
    if (next.type === 'richText' && prev && prev.type === 'richText') {
      prev.html = `${prev.html}${next.html}`
      return
    }
    normalized.push(next)
  })
  if (normalized.length === 0) {
    normalized.push({ id: `${sectionId}-body`, type: 'richText', html: '' })
  }
  return { ...section, blocks: normalized }
}

export function normalizeDocPages(pages: DocPage[]): DocPage[] {
  return pages.map((page) => ({
    ...page,
    sections: Array.isArray(page.sections) ? page.sections.map(normalizeDocSection) : [],
  }))
}

function stripHtml(html: string): string {
  const cleaned = sanitizeDocHtml(html)
  const container = document.createElement('div')
  container.innerHTML = cleaned
  return container.textContent ?? ''
}

export function blockText(block: DocBlock): string {
  switch (block.type) {
    case 'richText':
    case 'callout':
      return stripHtml(block.html)
    case 'endpoint':
      return `${block.method} ${block.url}`
    case 'table':
      return [
        ...block.columns.map((column) => column.label),
        ...block.rows.flatMap((row) => block.columns.map((column) => row[column.key] ?? '')),
      ].join(' ')
    case 'code':
      return `${block.language} ${block.code}`
    case 'steps':
      return block.items.join(' ')
    case 'links':
      return block.items.map((item) => `${item.label} ${item.url}`).join(' ')
    case 'faq':
      return block.items.map((item) => `${item.q} ${item.a}`).join(' ')
  }
}

export function pageSearchText(page: DocPage): string {
  return [
    page.title,
    page.keywords,
    ...page.sections.flatMap((section) => [
      section.title,
      ...section.blocks.map(blockText),
    ]),
  ].join(' ').toLowerCase()
}

export function blockHasContent(block: DocBlock): boolean {
  switch (block.type) {
    case 'richText':
    case 'callout':
      return stripHtml(block.html).trim().length > 0
    case 'endpoint':
      return block.method.trim().length > 0 && block.url.trim().length > 0
    case 'table':
      return block.columns.length > 0 && block.rows.length > 0
    case 'code':
      return block.code.trim().length > 0
    case 'steps':
      return block.items.some((item) => item.trim())
    case 'links':
      return block.items.some((item) => item.label.trim() && item.url.trim())
    case 'faq':
      return block.items.some((item) => item.q.trim() && item.a.trim())
  }
}

export function pageHasContent(page: DocPage): boolean {
  return page.sections.some((section) => section.blocks.some(blockHasContent))
}
