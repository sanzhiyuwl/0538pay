/**
 * 文档章节编辑器的数据转换层。
 * DocBlock[] 始终是持久化与官网渲染的权威格式；编辑态临时转成 Tiptap JSON：
 * - richText 用标准 Tiptap 节点表达；
 * - 其余结构化块用统一的 docEmbed atom 节点承载（attrs: id / blockType / payload）。
 * 正文与结构模块在同一画布按真实顺序混排。
 */
import { Node, generateHTML, generateJSON, type AnyExtension } from '@tiptap/core'
import type { NodeViewRenderer } from '@tiptap/core'
import StarterKit from '@tiptap/starter-kit'
import Link from '@tiptap/extension-link'
import Image from '@tiptap/extension-image'
import Placeholder from '@tiptap/extension-placeholder'
import TextAlign from '@tiptap/extension-text-align'
import { sanitizeDocHtml } from '@/lib/site-docs'
import type {
  DocBlock,
  DocCalloutBlock,
  DocCodeBlock,
  DocEndpointBlock,
  DocFaqBlock,
  DocLinksBlock,
  DocStepsBlock,
  DocTableBlock,
  DocTableColumn,
} from '@/lib/mock/site-docs'

/** docEmbed 承载的结构化块（DocBlock 去掉 richText）。 */
export type EmbedBlock = Exclude<DocBlock, { type: 'richText' }>
export type EmbedBlockType = EmbedBlock['type']
/** payload = 结构块去掉 id 与 type 后的纯数据。 */
export type EmbedPayload = Omit<EmbedBlock, 'id' | 'type'>

interface EditorJsonNode {
  type: string
  attrs?: Record<string, unknown>
  content?: EditorJsonNode[]
  [key: string]: unknown
}

interface EditorJsonDoc {
  type: 'doc'
  content: EditorJsonNode[]
}

const EMBED_TYPES: EmbedBlockType[] = ['callout', 'endpoint', 'table', 'code', 'steps', 'links', 'faq']

let embedIdCounter = 0

/** 生成稳定唯一的块 id。 */
export function createDocBlockId(prefix = 'doc-block'): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return `${prefix}-${crypto.randomUUID()}`
  }
  embedIdCounter += 1
  return `${prefix}-${Date.now()}-${embedIdCounter}`
}

/** 统一的 docEmbed 节点定义。编辑器与转换共用同一个 schema。 */
export const DocEmbed = Node.create({
  name: 'docEmbed',
  group: 'block',
  atom: true,
  selectable: true,
  draggable: true,

  addAttributes() {
    return {
      id: {
        default: null,
        parseHTML: (element) => element.getAttribute('data-id'),
        renderHTML: (attributes) => (attributes.id ? { 'data-id': attributes.id as string } : {}),
      },
      blockType: {
        default: 'callout',
        parseHTML: (element) => element.getAttribute('data-block-type'),
        renderHTML: (attributes) => ({ 'data-block-type': attributes.blockType as string }),
      },
      payload: {
        default: {},
        parseHTML: (element) => {
          const raw = element.getAttribute('data-payload')
          if (!raw) return {}
          try {
            return JSON.parse(decodeURIComponent(raw))
          } catch {
            return {}
          }
        },
        renderHTML: (attributes) => ({
          'data-payload': encodeURIComponent(JSON.stringify(attributes.payload ?? {})),
        }),
      },
    }
  },

  parseHTML() {
    return [{ tag: 'div[data-doc-embed]' }]
  },

  renderHTML({ HTMLAttributes }) {
    return ['div', { 'data-doc-embed': '', ...HTMLAttributes }]
  },
})

/**
 * 文档章节编辑器扩展集合。文档专用，不影响文章共享的 RichEditor。
 * @param nodeView 可选的 docEmbed Vue NodeView 渲染器（编辑器实例使用；纯转换时可省略）。
 */
export function createDocEditorExtensions(nodeView?: NodeViewRenderer): AnyExtension[] {
  const embed = nodeView
    ? DocEmbed.extend({ addNodeView: () => nodeView })
    : DocEmbed
  return [
    StarterKit.configure({ heading: { levels: [2, 3] } }),
    Link.configure({ openOnClick: false, autolink: false }),
    Image,
    Placeholder.configure({ placeholder: '输入正文，或用工具栏插入接口、表格、代码等模块…' }),
    TextAlign.configure({ types: ['heading', 'paragraph'] }),
    embed,
  ]
}

const CONVERSION_EXTENSIONS = createDocEditorExtensions()

/** 校验并规整 payload，非法字段回退到安全默认。 */
function sanitizeEmbedPayload(blockType: EmbedBlockType, raw: unknown): EmbedPayload | null {
  const payload = (raw && typeof raw === 'object') ? (raw as Record<string, unknown>) : {}
  switch (blockType) {
    case 'callout': {
      const tone = payload.tone
      return {
        tone: tone === 'warning' || tone === 'success' ? tone : 'info',
        html: sanitizeDocHtml(typeof payload.html === 'string' ? payload.html : ''),
      } satisfies Omit<DocCalloutBlock, 'id' | 'type'>
    }
    case 'endpoint':
      return {
        method: payload.method === 'POST' ? 'POST' : 'GET',
        url: typeof payload.url === 'string' ? payload.url : '',
      } satisfies Omit<DocEndpointBlock, 'id' | 'type'>
    case 'table': {
      const columnsRaw = Array.isArray(payload.columns) ? payload.columns : []
      const columns = columnsRaw
        .filter((col): col is Record<string, unknown> => !!col && typeof col === 'object')
        .map((col): DocTableColumn => {
          const kind = col.kind
          return {
            key: typeof col.key === 'string' ? col.key : '',
            label: typeof col.label === 'string' ? col.label : '',
            ...(typeof col.width === 'string' ? { width: col.width } : {}),
            ...(kind === 'code' || kind === 'status' || kind === 'text' ? { kind } : {}),
          }
        })
        .filter((col) => col.key)
      const rowsRaw = Array.isArray(payload.rows) ? payload.rows : []
      const rows = rowsRaw
        .filter((row): row is Record<string, unknown> => !!row && typeof row === 'object')
        .map((row) => {
          const next: Record<string, string> = {}
          for (const col of columns) {
            const value = row[col.key]
            next[col.key] = typeof value === 'string' ? value : ''
          }
          return next
        })
      return { columns, rows } satisfies Omit<DocTableBlock, 'id' | 'type'>
    }
    case 'code':
      return {
        language: typeof payload.language === 'string' ? payload.language : 'text',
        code: typeof payload.code === 'string' ? payload.code : '',
      } satisfies Omit<DocCodeBlock, 'id' | 'type'>
    case 'steps': {
      const items = Array.isArray(payload.items) ? payload.items : []
      return {
        items: items.filter((item): item is string => typeof item === 'string'),
      } satisfies Omit<DocStepsBlock, 'id' | 'type'>
    }
    case 'links': {
      const items = Array.isArray(payload.items) ? payload.items : []
      return {
        items: items
          .filter((item): item is Record<string, unknown> => !!item && typeof item === 'object')
          .map((item) => ({
            label: typeof item.label === 'string' ? item.label : '',
            url: typeof item.url === 'string' ? item.url : '',
            primary: item.primary === true,
          })),
      } satisfies Omit<DocLinksBlock, 'id' | 'type'>
    }
    case 'faq': {
      const items = Array.isArray(payload.items) ? payload.items : []
      return {
        items: items
          .filter((item): item is Record<string, unknown> => !!item && typeof item === 'object')
          .map((item) => ({
            q: typeof item.q === 'string' ? item.q : '',
            a: typeof item.a === 'string' ? item.a : '',
          })),
      } satisfies Omit<DocFaqBlock, 'id' | 'type'>
    }
  }
}

/** 结构块 → docEmbed payload（去掉 id / type）。 */
function blockToPayload(block: EmbedBlock): EmbedPayload {
  const { id: _id, type: _type, ...payload } = block
  void _id
  void _type
  return payload as EmbedPayload
}

/** 新建一个带默认内容的结构块。 */
export function createEmptyEmbedBlock(blockType: EmbedBlockType): EmbedBlock {
  const id = createDocBlockId('doc-embed')
  switch (blockType) {
    case 'callout':
      return { id, type: 'callout', tone: 'info', html: '' }
    case 'endpoint':
      return { id, type: 'endpoint', method: 'GET', url: '' }
    case 'table':
      return {
        id,
        type: 'table',
        columns: [
          { key: 'field_1', label: '参数', kind: 'code' },
          { key: 'field_2', label: '说明', kind: 'text' },
        ],
        rows: [{ field_1: '', field_2: '' }],
      }
    case 'code':
      return { id, type: 'code', language: 'json', code: '' }
    case 'steps':
      return { id, type: 'steps', items: [''] }
    case 'links':
      return { id, type: 'links', items: [{ label: '', url: '', primary: false }] }
    case 'faq':
      return { id, type: 'faq', items: [{ q: '', a: '' }] }
  }
}

/** DocBlock[] → Tiptap JSON 文档。 */
export function blocksToEditorJson(blocks: DocBlock[]): EditorJsonDoc {
  const content: EditorJsonNode[] = []
  for (const block of blocks) {
    if (block.type === 'richText') {
      const html = sanitizeDocHtml(block.html)
      if (!html.trim()) continue
      const doc = generateJSON(html, CONVERSION_EXTENSIONS) as EditorJsonDoc
      if (Array.isArray(doc.content)) content.push(...doc.content)
      continue
    }
    content.push({
      type: 'docEmbed',
      attrs: {
        id: block.id || createDocBlockId('doc-embed'),
        blockType: block.type,
        payload: blockToPayload(block),
      },
    })
  }
  if (content.length === 0) content.push({ type: 'paragraph' })
  return { type: 'doc', content }
}

function isEmptyParagraph(node: EditorJsonNode): boolean {
  return node.type === 'paragraph' && (!node.content || node.content.length === 0)
}

/**
 * Tiptap JSON → DocBlock[]。
 * 顶层节点按顺序扫描：连续的富文本节点合并成一个 richText，遇到 docEmbed 时先 flush。
 * previousBlocks 用于按顺序复用已有 richText id，避免每次编辑都改 id。
 */
export function editorJsonToBlocks(doc: EditorJsonDoc, previousBlocks: DocBlock[] = []): DocBlock[] {
  const richTextIds = previousBlocks.filter((b) => b.type === 'richText').map((b) => b.id)
  let richTextCursor = 0
  const nextRichTextId = () => richTextIds[richTextCursor++] ?? createDocBlockId('doc-rich')

  const result: DocBlock[] = []
  let buffer: EditorJsonNode[] = []

  const flush = () => {
    const meaningful = buffer.filter((node) => !isEmptyParagraph(node))
    buffer = []
    if (meaningful.length === 0) return
    const html = sanitizeDocHtml(
      generateHTML({ type: 'doc', content: meaningful }, CONVERSION_EXTENSIONS),
    )
    if (!html.trim()) return
    result.push({ id: nextRichTextId(), type: 'richText', html })
  }

  for (const node of doc.content ?? []) {
    if (node.type === 'docEmbed') {
      flush()
      const attrs = node.attrs ?? {}
      const blockType = attrs.blockType
      if (typeof blockType !== 'string' || !EMBED_TYPES.includes(blockType as EmbedBlockType)) continue
      const payload = sanitizeEmbedPayload(blockType as EmbedBlockType, attrs.payload)
      if (!payload) continue
      const id = typeof attrs.id === 'string' && attrs.id ? attrs.id : createDocBlockId('doc-embed')
      result.push({ id, type: blockType, ...payload } as DocBlock)
      continue
    }
    buffer.push(node)
  }
  flush()

  if (result.length === 0) {
    result.push({ id: nextRichTextId(), type: 'richText', html: '' })
  }
  return result
}
