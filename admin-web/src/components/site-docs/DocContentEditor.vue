<script setup lang="ts">
/**
 * 文档章节统一编辑器。
 * 正文（富文本）与接口/表格/代码/提示/步骤/链接/FAQ 模块在同一个 Tiptap 画布中按顺序混排；
 * 模块通过工具栏「插入」菜单落在当前光标位置，并以 DocEmbedNodeView 原位编辑。
 * 对外仍以 DocBlock[] 双向绑定，官网渲染与持久化格式不变。
 */
import { onBeforeUnmount, ref, watch } from 'vue'
import { useEditor, EditorContent, VueNodeViewRenderer } from '@tiptap/vue-3'
import {
  AlignCenter, AlignLeft, AlignRight, Bold, ChevronDown, Code, Heading2, Heading3,
  Image as ImageIcon, Italic, Link as LinkIcon, List, ListOrdered, Minus, Plus,
  Quote, Redo2, Strikethrough, Undo2,
} from 'lucide-vue-next'
import { onClickOutside } from '@vueuse/core'
import DocEmbedNodeView from '@/components/site-docs/editor/DocEmbedNodeView.vue'
import {
  blocksToEditorJson,
  createDocBlockId,
  createDocEditorExtensions,
  createEmptyEmbedBlock,
  editorJsonToBlocks,
  type EmbedBlockType,
} from '@/lib/site-docs-editor'
import type { DocBlock } from '@/lib/mock/site-docs'

const props = defineProps<{ modelValue: DocBlock[] }>()
const emit = defineEmits<{ 'update:modelValue': [value: DocBlock[]] }>()

// 防止 onUpdate → emit → 父更新 modelValue → watch 回写编辑器 的回环。
let applyingExternal = false
let lastEmitted = ''
// 记录最近一次光标落点，供工具栏「插入」在编辑器失焦后仍能插到正确位置。
let lastCursorPos: number | null = null

const editor = useEditor({
  content: blocksToEditorJson(props.modelValue),
  extensions: createDocEditorExtensions(VueNodeViewRenderer(DocEmbedNodeView)),
  editorProps: {
    attributes: { class: 'tiptap doc-content-editor min-h-[320px] px-4 py-3 focus:outline-none' },
  },
  onUpdate: ({ editor }) => {
    const blocks = editorJsonToBlocks(editor.getJSON() as never, props.modelValue)
    lastEmitted = JSON.stringify(blocks)
    emit('update:modelValue', blocks)
  },
  onSelectionUpdate: ({ editor }) => {
    lastCursorPos = editor.state.selection.to
  },
})

watch(
  () => props.modelValue,
  (value) => {
    if (!editor.value) return
    // 来自本组件 onUpdate 的更新无需回写编辑器。
    if (JSON.stringify(value) === lastEmitted) return
    applyingExternal = true
    editor.value.commands.setContent(blocksToEditorJson(value), { emitUpdate: false })
    applyingExternal = false
  },
)

onBeforeUnmount(() => editor.value?.destroy())

function setLink() {
  const prev = editor.value?.getAttributes('link').href
  const url = window.prompt('链接地址', prev ?? 'https://')
  if (url === null) return
  if (url === '') {
    editor.value?.chain().focus().extendMarkRange('link').unsetLink().run()
    return
  }
  editor.value?.chain().focus().extendMarkRange('link').setLink({ href: url }).run()
}

function addImage() {
  const url = window.prompt('图片地址', 'https://')
  if (!url) return
  editor.value?.chain().focus().setImage({ src: url }).run()
}

const groups = [
  [
    { icon: Bold, title: '加粗', active: () => editor.value?.isActive('bold'), run: () => editor.value?.chain().focus().toggleBold().run() },
    { icon: Italic, title: '斜体', active: () => editor.value?.isActive('italic'), run: () => editor.value?.chain().focus().toggleItalic().run() },
    { icon: Strikethrough, title: '删除线', active: () => editor.value?.isActive('strike'), run: () => editor.value?.chain().focus().toggleStrike().run() },
    { icon: Code, title: '行内代码', active: () => editor.value?.isActive('code'), run: () => editor.value?.chain().focus().toggleCode().run() },
  ],
  [
    { icon: Heading2, title: '二级标题', active: () => editor.value?.isActive('heading', { level: 2 }), run: () => editor.value?.chain().focus().toggleHeading({ level: 2 }).run() },
    { icon: Heading3, title: '三级标题', active: () => editor.value?.isActive('heading', { level: 3 }), run: () => editor.value?.chain().focus().toggleHeading({ level: 3 }).run() },
    { icon: List, title: '无序列表', active: () => editor.value?.isActive('bulletList'), run: () => editor.value?.chain().focus().toggleBulletList().run() },
    { icon: ListOrdered, title: '有序列表', active: () => editor.value?.isActive('orderedList'), run: () => editor.value?.chain().focus().toggleOrderedList().run() },
    { icon: Quote, title: '引用', active: () => editor.value?.isActive('blockquote'), run: () => editor.value?.chain().focus().toggleBlockquote().run() },
    { icon: Minus, title: '分割线', active: () => false, run: () => editor.value?.chain().focus().setHorizontalRule().run() },
  ],
  [
    { icon: AlignLeft, title: '左对齐', active: () => editor.value?.isActive({ textAlign: 'left' }), run: () => editor.value?.chain().focus().setTextAlign('left').run() },
    { icon: AlignCenter, title: '居中', active: () => editor.value?.isActive({ textAlign: 'center' }), run: () => editor.value?.chain().focus().setTextAlign('center').run() },
    { icon: AlignRight, title: '右对齐', active: () => editor.value?.isActive({ textAlign: 'right' }), run: () => editor.value?.chain().focus().setTextAlign('right').run() },
    { icon: LinkIcon, title: '链接', active: () => editor.value?.isActive('link'), run: setLink },
    { icon: ImageIcon, title: '图片', active: () => false, run: addImage },
  ],
  [
    { icon: Undo2, title: '撤销', active: () => false, run: () => editor.value?.chain().focus().undo().run() },
    { icon: Redo2, title: '重做', active: () => false, run: () => editor.value?.chain().focus().redo().run() },
  ],
]

const insertOptions: { type: EmbedBlockType; label: string }[] = [
  { type: 'callout', label: '提示信息' },
  { type: 'endpoint', label: '接口地址' },
  { type: 'table', label: '参数表格' },
  { type: 'code', label: '代码示例' },
  { type: 'steps', label: '步骤列表' },
  { type: 'links', label: '相关链接' },
  { type: 'faq', label: '常见问题' },
]

const insertMenuOpen = ref(false)
const insertMenuRef = ref<HTMLElement | null>(null)
onClickOutside(insertMenuRef, () => { insertMenuOpen.value = false })

function insertModule(type: EmbedBlockType) {
  insertMenuOpen.value = false
  const instance = editor.value
  if (!instance) return
  const block = createEmptyEmbedBlock(type)
  const { id, type: blockType, ...payload } = block
  const content = [
    { type: 'docEmbed', attrs: { id, blockType, payload } },
    // 模块后补一个空段落，方便继续输入正文。
    { type: 'paragraph' },
  ]
  // 用记录的光标位置插入；若编辑器已失焦或无有效选区，则插到文档末尾，
  // 避免 focus() 把选区落到 atom 节点上导致 insertContent 替换/删除现有模块。
  const size = instance.state.doc.content.size
  const at = lastCursorPos !== null && lastCursorPos <= size ? lastCursorPos : size
  instance.chain().focus().insertContentAt(at, content).run()
}

// applyingExternal 仅用于抑制回写期间的副作用；createDocBlockId 供外部潜在复用保持稳定 id 语义。
void applyingExternal
void createDocBlockId
</script>

<template>
  <div class="overflow-hidden rounded border border-input bg-background">
    <div v-if="editor" class="flex flex-wrap items-center gap-0.5 border-b border-border bg-muted/30 px-2 py-1.5">
      <template v-for="(group, gi) in groups" :key="gi">
        <span v-if="gi > 0" class="mx-1 h-5 w-px bg-border" />
        <button
          v-for="(b, bi) in group"
          :key="bi"
          type="button"
          class="flex size-8 items-center justify-center rounded text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
          :class="b.active() ? 'bg-primary/10 text-primary' : ''"
          :title="b.title"
          @click="b.run()"
        >
          <component :is="b.icon" class="size-4" />
        </button>
      </template>

      <span class="mx-1 h-5 w-px bg-border" />
      <div ref="insertMenuRef" class="relative">
        <button
          type="button"
          class="flex h-8 items-center gap-1 rounded px-2 text-sm text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
          :class="insertMenuOpen ? 'bg-primary/10 text-primary' : ''"
          title="插入模块"
          @click="insertMenuOpen = !insertMenuOpen"
        >
          <Plus class="size-4" />
          <span>插入</span>
          <ChevronDown class="size-3.5" />
        </button>
        <div
          v-if="insertMenuOpen"
          class="absolute left-0 top-full z-20 mt-1 w-36 overflow-hidden rounded-md border border-border bg-popover py-1 shadow-lg"
        >
          <button
            v-for="opt in insertOptions"
            :key="opt.type"
            type="button"
            class="block w-full px-3 py-1.5 text-left text-sm text-foreground transition-colors hover:bg-accent"
            @click="insertModule(opt.type)"
          >
            {{ opt.label }}
          </button>
        </div>
      </div>
    </div>

    <EditorContent :editor="editor" />
  </div>
</template>

<style scoped>
:deep(.doc-content-editor) {
  font-size: 14px;
  line-height: 1.75;
  color: var(--foreground);
}
:deep(.doc-content-editor:focus) {
  outline: none;
}
:deep(.doc-content-editor p) {
  margin: 0.5em 0;
}
:deep(.doc-content-editor h2) {
  margin: 1em 0 0.5em;
  font-size: 1.3em;
  font-weight: 700;
}
:deep(.doc-content-editor h3) {
  margin: 1em 0 0.5em;
  font-size: 1.1em;
  font-weight: 600;
}
:deep(.doc-content-editor ul),
:deep(.doc-content-editor ol) {
  margin: 0.5em 0;
  padding-left: 1.4em;
}
:deep(.doc-content-editor ul) {
  list-style: disc;
}
:deep(.doc-content-editor ol) {
  list-style: decimal;
}
:deep(.doc-content-editor blockquote) {
  margin: 0.75em 0;
  border-left: 3px solid var(--primary);
  padding-left: 0.9em;
  color: var(--muted-foreground);
}
:deep(.doc-content-editor code) {
  border-radius: 0.25rem;
  background: var(--muted);
  padding: 0.1rem 0.35rem;
  font-family: ui-monospace, "SFMono-Regular", Menlo, Consolas, monospace;
  font-size: 0.85em;
}
:deep(.doc-content-editor a) {
  color: var(--primary);
  text-decoration: underline;
}
:deep(.doc-content-editor hr) {
  margin: 1em 0;
  border: none;
  border-top: 1px solid var(--border);
}
:deep(.doc-content-editor p.is-editor-empty:first-child::before) {
  content: attr(data-placeholder);
  float: left;
  height: 0;
  color: var(--muted-foreground);
  pointer-events: none;
}
</style>
