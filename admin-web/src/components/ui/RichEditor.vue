<script setup lang="ts">
/**
 * 富文本编辑器（基于 tiptap）。文章正文编辑用。
 * <RichEditor v-model="html" placeholder="输入正文…" />
 * 输出为 HTML 字符串，官网详情页直接 v-html 渲染。
 */
import { ref, watch, onBeforeUnmount, onMounted } from 'vue'
import { useEditor, EditorContent } from '@tiptap/vue-3'
import StarterKit from '@tiptap/starter-kit'
import Link from '@tiptap/extension-link'
import Image from '@tiptap/extension-image'
import Placeholder from '@tiptap/extension-placeholder'
import TextAlign from '@tiptap/extension-text-align'
import { TextStyle } from '@tiptap/extension-text-style'
import { Color } from '@tiptap/extension-color'
import {
  Bold, Italic, Strikethrough, Heading1, Heading2, Heading3, Heading4,
  List, ListOrdered, Quote, Code, Link as LinkIcon, Image as ImageIcon,
  Undo2, Redo2, AlignLeft, AlignCenter, AlignRight, Minus,
} from 'lucide-vue-next'
import { uploadImage } from '@/lib/api/upload'
import { useToast } from '@/composables/useToast'
import { ApiError } from '@/lib/api/client'

const props = withDefaults(
  defineProps<{ modelValue: string; placeholder?: string }>(),
  { placeholder: '输入正文内容…' },
)
const emit = defineEmits<{ 'update:modelValue': [v: string] }>()
const toast = useToast()

const editor = useEditor({
  content: props.modelValue,
  extensions: [
    StarterKit,
    Link.configure({ openOnClick: false, HTMLAttributes: { rel: 'noopener', target: '_blank' } }),
    Image,
    Placeholder.configure({ placeholder: props.placeholder }),
    TextAlign.configure({ types: ['heading', 'paragraph'] }),
    TextStyle,
    Color, // 文字颜色（依赖 TextStyle）
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

// 点击颜色下拉外部时关闭（下拉 Teleport 到 body，需同时排除下拉自身与颜色按钮）
const colorPanel = ref<HTMLElement | null>(null)
function onDocClick(e: MouseEvent) {
  if (!colorOpen.value) return
  const t = e.target as Node
  if (colorBtn.value?.contains(t) || colorPanel.value?.contains(t)) return
  colorOpen.value = false
}
onMounted(() => document.addEventListener('click', onDocClick))
onBeforeUnmount(() => {
  document.removeEventListener('click', onDocClick)
  editor.value?.destroy()
})

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

// 图片：点击工具栏按钮唤起文件选择，上传后插入
const imgInput = ref<HTMLInputElement | null>(null)
function addImage() {
  imgInput.value?.click()
}
async function onImageFile(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = '' // 清空以便重选同名文件
  if (!file) return
  if (!file.type.startsWith('image/')) {
    toast.error('请选择图片文件')
    return
  }
  if (file.size > 10 * 1024 * 1024) {
    toast.error('图片过大，单张不超过 10MB')
    return
  }
  try {
    const url = await uploadImage(file, 'article')
    editor.value?.chain().focus().setImage({ src: url }).run()
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '图片上传失败')
  }
}

// 文字颜色：标准富文本编辑器色板（多色小色块）
const COLORS = [
  '#000000', '#262626', '#595959', '#8c8c8c', '#bfbfbf', '#d9d9d9', '#f5f5f5', '#ffffff',
  '#e94b4b', '#fa541c', '#fa8c16', '#faad14', '#fadb14', '#a0d911', '#52c41a', '#13c2c2',
  '#1a6dff', '#2f54eb', '#722ed1', '#eb2f96', '#c41d7f', '#873800', '#7cb305', '#08979c',
]
const colorOpen = ref(false)
const colorBtn = ref<HTMLElement | null>(null)
// 下拉用 fixed 定位（Teleport 到 body），脱离抽屉/滚动容器的 overflow 裁剪
const colorPos = ref({ top: 0, left: 0 })
const DROPDOWN_W = 168 // 下拉宽度估算，用于右对齐防溢出
function toggleColor() {
  if (!colorOpen.value && colorBtn.value) {
    const r = colorBtn.value.getBoundingClientRect()
    // 右对齐按钮右边界，且不超出视口左侧
    const left = Math.max(8, r.right - DROPDOWN_W)
    colorPos.value = { top: r.bottom + 4, left }
  }
  colorOpen.value = !colorOpen.value
}
function applyColor(value: string) {
  if (value) editor.value?.chain().focus().setColor(value).run()
  else editor.value?.chain().focus().unsetColor().run()
  colorOpen.value = false
}
function pickCustomColor(e: Event) {
  const v = (e.target as HTMLInputElement).value
  editor.value?.chain().focus().setColor(v).run()
}

// 工具栏按钮配置：按「历史 → 标题 → 文字样式 → 段落块 → 对齐 → 插入」逻辑分组排序。
const groups = [
  // 历史操作
  [
    { icon: Undo2, title: '撤销', active: () => false, run: () => editor.value?.chain().focus().undo().run() },
    { icon: Redo2, title: '重做', active: () => false, run: () => editor.value?.chain().focus().redo().run() },
  ],
  // 标题 H1-H4
  [
    { icon: Heading1, title: '一级标题', active: () => editor.value?.isActive('heading', { level: 1 }), run: () => editor.value?.chain().focus().toggleHeading({ level: 1 }).run() },
    { icon: Heading2, title: '二级标题', active: () => editor.value?.isActive('heading', { level: 2 }), run: () => editor.value?.chain().focus().toggleHeading({ level: 2 }).run() },
    { icon: Heading3, title: '三级标题', active: () => editor.value?.isActive('heading', { level: 3 }), run: () => editor.value?.chain().focus().toggleHeading({ level: 3 }).run() },
    { icon: Heading4, title: '四级标题', active: () => editor.value?.isActive('heading', { level: 4 }), run: () => editor.value?.chain().focus().toggleHeading({ level: 4 }).run() },
  ],
  // 文字样式
  [
    { icon: Bold, title: '加粗', active: () => editor.value?.isActive('bold'), run: () => editor.value?.chain().focus().toggleBold().run() },
    { icon: Italic, title: '斜体', active: () => editor.value?.isActive('italic'), run: () => editor.value?.chain().focus().toggleItalic().run() },
    { icon: Strikethrough, title: '删除线', active: () => editor.value?.isActive('strike'), run: () => editor.value?.chain().focus().toggleStrike().run() },
    { icon: Code, title: '行内代码', active: () => editor.value?.isActive('code'), run: () => editor.value?.chain().focus().toggleCode().run() },
  ],
  // 段落块：列表 / 引用 / 分割线
  [
    { icon: List, title: '无序列表', active: () => editor.value?.isActive('bulletList'), run: () => editor.value?.chain().focus().toggleBulletList().run() },
    { icon: ListOrdered, title: '有序列表', active: () => editor.value?.isActive('orderedList'), run: () => editor.value?.chain().focus().toggleOrderedList().run() },
    { icon: Quote, title: '引用', active: () => editor.value?.isActive('blockquote'), run: () => editor.value?.chain().focus().toggleBlockquote().run() },
    { icon: Minus, title: '分割线', active: () => false, run: () => editor.value?.chain().focus().setHorizontalRule().run() },
  ],
  // 对齐
  [
    { icon: AlignLeft, title: '左对齐', active: () => editor.value?.isActive({ textAlign: 'left' }), run: () => editor.value?.chain().focus().setTextAlign('left').run() },
    { icon: AlignCenter, title: '居中', active: () => editor.value?.isActive({ textAlign: 'center' }), run: () => editor.value?.chain().focus().setTextAlign('center').run() },
    { icon: AlignRight, title: '右对齐', active: () => editor.value?.isActive({ textAlign: 'right' }), run: () => editor.value?.chain().focus().setTextAlign('right').run() },
  ],
  // 插入：链接 / 图片
  [
    { icon: LinkIcon, title: '链接', active: () => editor.value?.isActive('link'), run: setLink },
    { icon: ImageIcon, title: '图片', active: () => false, run: addImage },
  ],
]
</script>

<template>
  <div class="rounded border border-input bg-background">
    <!-- 工具栏 -->
    <div v-if="editor" class="flex flex-wrap items-center gap-0.5 rounded-t border-b border-border bg-muted/30 px-2 py-1.5">
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

      <!-- 文字颜色 -->
      <span class="mx-1 h-5 w-px bg-border" />
      <button
        ref="colorBtn"
        type="button"
        title="文字颜色"
        class="flex h-8 items-center gap-0.5 rounded px-1.5 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
        :class="editor?.getAttributes('textStyle').color ? 'bg-primary/10 text-primary' : ''"
        @click="toggleColor"
      >
        <span class="text-sm font-bold leading-none">A</span>
        <span
          class="block h-1 w-4 rounded-full"
          :style="{ backgroundColor: editor?.getAttributes('textStyle').color || '#111827' }"
        />
      </button>
    </div>

    <!-- 色板下拉：Teleport 到 body + fixed 定位，彻底脱离抽屉/滚动容器的 overflow 裁剪 -->
    <Teleport to="body">
      <div
        v-if="colorOpen"
        ref="colorPanel"
        class="fixed z-[9999] w-max rounded border border-border bg-popover p-2 shadow-lg"
        :style="{ top: colorPos.top + 'px', left: colorPos.left + 'px' }"
      >
        <div class="grid grid-cols-8 gap-1">
          <button
            v-for="c in COLORS"
            :key="c"
            type="button"
            :title="c"
            class="size-4 rounded-[3px] border border-border/50 transition-transform hover:scale-125"
            :style="{ backgroundColor: c }"
            @click="applyColor(c)"
          />
        </div>
        <div class="mt-2 flex items-center justify-between gap-2 border-t border-border pt-2">
          <button
            type="button"
            class="text-xs text-muted-foreground transition-colors hover:text-foreground"
            @click="applyColor('')"
          >清除颜色</button>
          <label class="flex items-center gap-1 text-xs text-muted-foreground">
            自定义
            <input type="color" class="h-5 w-6 cursor-pointer border-0 bg-transparent p-0" @input="pickCustomColor" />
          </label>
        </div>
      </div>
    </Teleport>
    <!-- 编辑区 -->
    <EditorContent :editor="editor" />

    <!-- 隐藏的图片上传 input（工具栏图片按钮触发）-->
    <input ref="imgInput" type="file" accept="image/*" class="hidden" @change="onImageFile" />
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
/* 多级标题：H1-H4 四级各不同样式，与官网详情页 .news-content 保持一致（无阴影）*/
:deep(.tiptap h1) {
  margin: 1.2em 0 0.7em;
  padding: 0.35em 0 0.35em 0.7em;
  border-left: 4px solid var(--primary);
  background: var(--muted);
  font-size: 1.6em;
  font-weight: 700;
  line-height: 1.4;
}
:deep(.tiptap h2) {
  margin: 1.1em 0 0.6em;
  padding: 0.25em 0 0.25em 0.65em;
  border-left: 4px solid var(--primary);
  background: color-mix(in oklch, var(--muted) 55%, transparent);
  font-size: 1.35em;
  font-weight: 700;
  line-height: 1.4;
}
:deep(.tiptap h3) {
  margin: 1em 0 0.5em;
  padding-left: 0.6em;
  border-left: 3px solid var(--primary);
  font-size: 1.18em;
  font-weight: 600;
  line-height: 1.4;
}
:deep(.tiptap h4) {
  position: relative;
  margin: 0.9em 0 0.45em;
  padding-left: 0.85em;
  font-size: 1.05em;
  font-weight: 600;
}
:deep(.tiptap h4)::before {
  content: '';
  position: absolute;
  left: 0;
  top: 0.62em;
  width: 5px;
  height: 5px;
  border-radius: 9999px;
  background: var(--primary);
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
