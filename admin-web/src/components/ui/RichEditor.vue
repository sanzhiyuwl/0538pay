<script setup lang="ts">
/**
 * 富文本编辑器（基于 tiptap）。文章正文编辑用。
 * <RichEditor v-model="html" placeholder="输入正文…" />
 * 输出为 HTML 字符串，官网详情页直接 v-html 渲染。
 */
import { watch, onBeforeUnmount } from 'vue'
import { useEditor, EditorContent } from '@tiptap/vue-3'
import StarterKit from '@tiptap/starter-kit'
import Link from '@tiptap/extension-link'
import Image from '@tiptap/extension-image'
import Placeholder from '@tiptap/extension-placeholder'
import TextAlign from '@tiptap/extension-text-align'
import {
  Bold, Italic, Strikethrough, Heading2, Heading3, List, ListOrdered,
  Quote, Code, Link as LinkIcon, Image as ImageIcon, Undo2, Redo2,
  AlignLeft, AlignCenter, AlignRight, Minus,
} from 'lucide-vue-next'

const props = withDefaults(
  defineProps<{ modelValue: string; placeholder?: string }>(),
  { placeholder: '输入正文内容…' },
)
const emit = defineEmits<{ 'update:modelValue': [v: string] }>()

const editor = useEditor({
  content: props.modelValue,
  extensions: [
    StarterKit,
    Link.configure({ openOnClick: false, HTMLAttributes: { rel: 'noopener', target: '_blank' } }),
    Image,
    Placeholder.configure({ placeholder: props.placeholder }),
    TextAlign.configure({ types: ['heading', 'paragraph'] }),
  ],
  editorProps: {
    attributes: {
      class: 'tiptap prose-editor min-h-[320px] px-4 py-3 focus:outline-none',
    },
  },
  onUpdate: ({ editor }) => emit('update:modelValue', editor.getHTML()),
})

// 外部值变化时同步（如切换编辑的文章）
watch(
  () => props.modelValue,
  (v) => {
    if (editor.value && v !== editor.value.getHTML()) {
      editor.value.commands.setContent(v, { emitUpdate: false })
    }
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
  if (url) editor.value?.chain().focus().setImage({ src: url }).run()
}

// 工具栏按钮配置：图标 + tooltip + 激活判断 + 点击
const groups = [
  [
    { icon: Bold, title: '加粗', active: () => editor.value?.isActive('bold'), run: () => editor.value?.chain().focus().toggleBold().run() },
    { icon: Italic, title: '斜体', active: () => editor.value?.isActive('italic'), run: () => editor.value?.chain().focus().toggleItalic().run() },
    { icon: Strikethrough, title: '删除线', active: () => editor.value?.isActive('strike'), run: () => editor.value?.chain().focus().toggleStrike().run() },
    { icon: Code, title: '行内代码', active: () => editor.value?.isActive('code'), run: () => editor.value?.chain().focus().toggleCode().run() },
  ],
  [
    { icon: Heading2, title: '标题 2', active: () => editor.value?.isActive('heading', { level: 2 }), run: () => editor.value?.chain().focus().toggleHeading({ level: 2 }).run() },
    { icon: Heading3, title: '标题 3', active: () => editor.value?.isActive('heading', { level: 3 }), run: () => editor.value?.chain().focus().toggleHeading({ level: 3 }).run() },
    { icon: List, title: '无序列表', active: () => editor.value?.isActive('bulletList'), run: () => editor.value?.chain().focus().toggleBulletList().run() },
    { icon: ListOrdered, title: '有序列表', active: () => editor.value?.isActive('orderedList'), run: () => editor.value?.chain().focus().toggleOrderedList().run() },
    { icon: Quote, title: '引用', active: () => editor.value?.isActive('blockquote'), run: () => editor.value?.chain().focus().toggleBlockquote().run() },
    { icon: Minus, title: '分割线', active: () => false, run: () => editor.value?.chain().focus().setHorizontalRule().run() },
  ],
  [
    { icon: AlignLeft, title: '左对齐', active: () => editor.value?.isActive({ textAlign: 'left' }), run: () => editor.value?.chain().focus().setTextAlign('left').run() },
    { icon: AlignCenter, title: '居中', active: () => editor.value?.isActive({ textAlign: 'center' }), run: () => editor.value?.chain().focus().setTextAlign('center').run() },
    { icon: AlignRight, title: '右对齐', active: () => editor.value?.isActive({ textAlign: 'right' }), run: () => editor.value?.chain().focus().setTextAlign('right').run() },
  ],
  [
    { icon: LinkIcon, title: '链接', active: () => editor.value?.isActive('link'), run: setLink },
    { icon: ImageIcon, title: '图片', active: () => false, run: addImage },
  ],
  [
    { icon: Undo2, title: '撤销', active: () => false, run: () => editor.value?.chain().focus().undo().run() },
    { icon: Redo2, title: '重做', active: () => false, run: () => editor.value?.chain().focus().redo().run() },
  ],
]
</script>

<template>
  <div class="overflow-hidden rounded border border-input bg-background">
    <!-- 工具栏 -->
    <div v-if="editor" class="flex flex-wrap items-center gap-0.5 border-b border-border bg-muted/30 px-2 py-1.5">
      <template v-for="(group, gi) in groups" :key="gi">
        <span v-if="gi > 0" class="mx-1 h-5 w-px bg-border" />
        <button
          v-for="(b, bi) in group"
          :key="bi"
          type="button"
          :title="b.title"
          class="flex size-8 items-center justify-center rounded text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
          :class="b.active() ? 'bg-primary/10 text-primary' : ''"
          @click="b.run()"
        >
          <component :is="b.icon" class="size-4" />
        </button>
      </template>
    </div>
    <!-- 编辑区 -->
    <EditorContent :editor="editor" />
  </div>
</template>

<style scoped>
/* tiptap 编辑区内容样式（与官网详情页 v-html 渲染保持一致）*/
:deep(.tiptap) {
  font-size: 14px;
  line-height: 1.75;
  color: var(--foreground);
}
:deep(.tiptap:focus) {
  outline: none;
}
:deep(.tiptap p) {
  margin: 0.5em 0;
}
:deep(.tiptap h2) {
  margin: 1em 0 0.5em;
  font-size: 1.35em;
  font-weight: 700;
}
:deep(.tiptap h3) {
  margin: 1em 0 0.5em;
  font-size: 1.15em;
  font-weight: 600;
}
:deep(.tiptap ul),
:deep(.tiptap ol) {
  margin: 0.5em 0;
  padding-left: 1.5em;
}
:deep(.tiptap ul) {
  list-style: disc;
}
:deep(.tiptap ol) {
  list-style: decimal;
}
:deep(.tiptap blockquote) {
  margin: 0.75em 0;
  border-left: 3px solid var(--primary);
  padding-left: 1em;
  color: var(--muted-foreground);
}
:deep(.tiptap code) {
  border-radius: 3px;
  background: var(--muted);
  padding: 0.1em 0.35em;
  font-size: 0.9em;
}
:deep(.tiptap pre) {
  margin: 0.75em 0;
  border-radius: 6px;
  background: #1e293b;
  padding: 0.85em 1em;
  color: #e2e8f0;
  overflow-x: auto;
}
:deep(.tiptap pre code) {
  background: transparent;
  padding: 0;
  color: inherit;
}
:deep(.tiptap a) {
  color: var(--primary);
  text-decoration: underline;
}
:deep(.tiptap img) {
  max-width: 100%;
  border-radius: 6px;
}
:deep(.tiptap hr) {
  margin: 1.25em 0;
  border: none;
  border-top: 1px solid var(--border);
}
/* placeholder（空文档提示）*/
:deep(.tiptap p.is-editor-empty:first-child::before) {
  content: attr(data-placeholder);
  float: left;
  height: 0;
  color: var(--muted-foreground);
  pointer-events: none;
}
</style>
