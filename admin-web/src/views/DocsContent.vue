<script setup lang="ts">
import { computed, onMounted, reactive, ref, type Component } from 'vue'
import {
  ArrowLeftRight,
  BookOpen,
  ChevronLeft,
  ChevronRight,
  CreditCard,
  ExternalLink,
  FileText,
  FolderTree,
  HelpCircle,
  Pencil,
  Plus,
  Rocket,
  Settings,
  Store as StoreIcon,
  Trash2,
} from 'lucide-vue-next'
import { Badge, Button, Drawer, Modal, Panel, Select, Switch } from '@/components/ui'
import DocContentEditor from '@/components/site-docs/DocContentEditor.vue'
import DocPageContent from '@/components/site-docs/DocPageContent.vue'
import { useToast } from '@/composables/useToast'
import { isValidDocSlug, pageHasContent } from '@/lib/site-docs'
import type {
  DocBlock,
  DocGroup,
  DocPage,
  DocSettings,
} from '@/lib/mock/site-docs'
import { useSiteDocsStore } from '@/stores/siteDocs'

const store = useSiteDocsStore()
// 从后端拉取最新文档（本地缓存先渲染，后端到达后覆盖）
onMounted(() => { store.hydrate() })
const toast = useToast()

function clone<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T
}

let generatedId = 0
function nextId(prefix: string): string {
  generatedId += 1
  return `${prefix}-${Date.now()}-${generatedId}`
}

function pageOrder(a: DocPage, b: DocPage): number {
  const groupSort = new Map(store.groups.map((group) => [group.id, group.sort]))
  return (groupSort.get(a.groupId) ?? Number.MAX_SAFE_INTEGER)
    - (groupSort.get(b.groupId) ?? Number.MAX_SAFE_INTEGER)
    || a.sort - b.sort
    || a.id - b.id
}

const stats = computed(() => ({
  total: store.pages.length,
  published: store.pages.filter((page) => page.status === 1).length,
  draft: store.pages.filter((page) => page.status === 0).length,
  groups: store.groups.length,
}))

const keyword = ref('')
const filterGroup = ref('')
const filterStatus = ref<-1 | 0 | 1>(-1)

const groupOptions = computed(() => [
  { value: '', label: '全部分组' },
  ...[...store.groups]
    .sort((a, b) => a.sort - b.sort)
    .map((group) => ({ value: group.id, label: group.name })),
])
const statusOptions = [
  { value: -1, label: '全部状态' },
  { value: 1, label: '已发布' },
  { value: 0, label: '草稿' },
]
const editStatusOptions = [
  { value: 0, label: '草稿' },
  { value: 1, label: '已发布' },
]
const editGroupOptions = computed(() =>
  [...store.groups]
    .sort((a, b) => a.sort - b.sort)
    .map((group) => ({ value: group.id, label: group.name })),
)

const filteredPages = computed(() => {
  const query = keyword.value.trim().toLowerCase()
  return [...store.pages]
    .filter((page) => !query || [page.title, page.slug, page.keywords].join(' ').toLowerCase().includes(query))
    .filter((page) => !filterGroup.value || page.groupId === filterGroup.value)
    .filter((page) => filterStatus.value === -1 || page.status === filterStatus.value)
    .sort(pageOrder)
})

function onFilterGroup(value: string | number) {
  if (typeof value === 'string') filterGroup.value = value
}

function onFilterStatus(value: string | number) {
  if (value === -1 || value === 0 || value === 1) filterStatus.value = value
}

function groupName(id: string): string {
  return store.groupName[id] ?? '未分组'
}

function sortedPublishedPages(): DocPage[] {
  return [...store.publishedPages].sort(pageOrder)
}

function ensureValidDefault() {
  const published = sortedPublishedPages()
  if (published.some((page) => page.slug === store.settings.defaultSlug)) return
  store.updateSettings({
    ...store.settings,
    defaultSlug: published[0]?.slug ?? '',
  })
}

function previewDocs() {
  ensureValidDefault()
  window.open(
    `/docs?p=${encodeURIComponent(store.settings.defaultSlug)}`,
    '_blank',
    'noopener,noreferrer',
  )
}


const drawerOpen = ref(false)
const editingId = ref<number | null>(null)
const originalPage = ref<DocPage | null>(null)
type DrawerTab = 'info' | 'content' | 'preview'
const activeTab = ref<DrawerTab>('info')
const drawerTabs: { value: DrawerTab; label: string }[] = [
  { value: 'info', label: '页面信息' },
  { value: 'content', label: '文档内容' },
  { value: 'preview', label: '页面预览' },
]
const tabOrder: DrawerTab[] = ['info', 'content', 'preview']
const activeStep = computed(() => tabOrder.indexOf(activeTab.value))
const isFirstStep = computed(() => activeStep.value === 0)
const isLastStep = computed(() => activeStep.value === tabOrder.length - 1)

// 离开「页面信息」前做基础校验，避免带着空标题/分组走到后面。
function canLeaveInfo(): boolean {
  if (!form.title.trim()) {
    toast.error('请填写文档标题')
    return false
  }
  if (!form.groupId || !store.groups.some((group) => group.id === form.groupId)) {
    toast.error('请选择有效分组')
    return false
  }
  return true
}

function goPrevStep() {
  if (isFirstStep.value) return
  activeTab.value = tabOrder[activeStep.value - 1]!
}

function goNextStep() {
  if (isLastStep.value) return
  if (activeTab.value === 'info' && !canLeaveInfo()) return
  activeTab.value = tabOrder[activeStep.value + 1]!
}

// 顶部标签直接跳转：从「页面信息」往后跳同样先校验基础字段。
function selectTab(tab: DrawerTab) {
  if (tab === activeTab.value) return
  if (activeTab.value === 'info' && tab !== 'info' && !canLeaveInfo()) return
  activeTab.value = tab
}

function emptyBlocks(): DocBlock[] {
  return [{ id: nextId('doc-block'), type: 'richText', html: '' }]
}

// 单正文模型：一篇文档 = 一个内容主体。持久化仍用 DocPage.sections（保持官网渲染兼容），
// 编辑时把所有章节的内容块拉平成一个主体；保存时再包回单个章节。
function flattenBlocks(page: DocPage): DocBlock[] {
  const blocks = page.sections.flatMap((section) => section.blocks)
  return blocks.length > 0 ? blocks : emptyBlocks()
}

function blankPage(): DocPage {
  return {
    id: 0,
    groupId: [...store.groups].sort((a, b) => a.sort - b.sort)[0]?.id ?? '',
    slug: '',
    title: '',
    keywords: '',
    sort: 50,
    status: 0,
    sections: [],
  }
}

const form = reactive<DocPage>(blankPage())
const contentBlocks = ref<DocBlock[]>(emptyBlocks())

// 预览用：把当前正文包成单章节的 DocPage。
const previewPage = computed<DocPage>(() => ({
  ...clone(form),
  sections: [
    {
      id: 'preview-section',
      anchor: form.slug || 'overview',
      title: '',
      showInOutline: true,
      blocks: contentBlocks.value,
    },
  ],
}))

function openCreate() {
  editingId.value = null
  originalPage.value = null
  Object.assign(form, blankPage())
  contentBlocks.value = emptyBlocks()
  activeTab.value = 'info'
  drawerOpen.value = true
}

function openEdit(page: DocPage) {
  const copy = clone(page)
  editingId.value = page.id
  originalPage.value = clone(page)
  Object.assign(form, copy)
  contentBlocks.value = flattenBlocks(copy)
  activeTab.value = 'info'
  drawerOpen.value = true
}

function onFormGroup(value: string | number) {
  if (typeof value === 'string') form.groupId = value
}

function onFormStatus(value: string | number) {
  if (value === 0 || value === 1) form.status = value
}

// 新增文档时自动生成唯一 slug（doc-N），编辑时沿用原值——用户无需手填访问路径。
function generateSlug(): string {
  const used = new Set(store.pages.map((page) => page.slug))
  let n = store.pages.length + 1
  let slug = `doc-${n}`
  while (used.has(slug)) slug = `doc-${++n}`
  return slug
}

// 排序按发布先后自动递增：新增文档取当前最大 sort + 1；无文档时从 1 开始。
function nextSort(): number {
  return Math.max(0, ...store.pages.map((page) => page.sort)) + 1
}

function normalizedPage(): DocPage {
  const page = clone(form)
  page.title = page.title.trim()
  page.groupId = page.groupId.trim()
  page.keywords = page.keywords.trim()
  if (editingId.value === null) {
    page.slug = generateSlug()
    page.sort = nextSort()
  } else {
    // 编辑时不改动访问路径与排序，保持链接与顺序稳定。
    page.slug = (originalPage.value?.slug ?? page.slug).trim()
    page.sort = originalPage.value?.sort ?? page.sort
  }
  // 单正文包成单个章节，锚点用 slug，供官网渲染与大纲使用。
  page.sections = [
    {
      id: originalPage.value?.sections[0]?.id ?? nextId('doc-section'),
      anchor: page.slug || 'overview',
      title: '',
      showInOutline: true,
      blocks: clone(contentBlocks.value),
    },
  ]
  return page
}

function validatePage(page: DocPage): boolean {
  if (!page.title) {
    toast.error('请填写文档标题')
    return false
  }
  if (!page.groupId || !store.groups.some((group) => group.id === page.groupId)) {
    toast.error('请选择有效分组')
    return false
  }
  if (page.status === 1 && !pageHasContent(page)) {
    toast.error('发布文档前请先填写文档内容')
    return false
  }
  return true
}

function savePage() {
  const page = normalizedPage()
  if (!validatePage(page)) return

  if (editingId.value === null) {
    store.addPage({
      groupId: page.groupId,
      slug: page.slug,
      title: page.title,
      keywords: page.keywords,
      sort: page.sort,
      status: page.status,
      sections: page.sections,
    })
    toast.success('文档已新增')
  } else {
    const wasDefault = originalPage.value?.slug === store.settings.defaultSlug
    store.updatePage({ ...page, id: editingId.value })
    if (wasDefault && page.status === 1) {
      store.updateSettings({ ...store.settings, defaultSlug: page.slug })
    }
    toast.success('文档已保存')
  }
  ensureValidDefault()
  drawerOpen.value = false
}

function togglePageStatus(page: DocPage, enabled: boolean) {
  if (enabled && !pageHasContent(page)) {
    toast.error('发布文档前请至少添加一个有内容的内容块')
    return
  }
  store.updatePage({ ...clone(page), status: enabled ? 1 : 0 })
  ensureValidDefault()
  toast.success(enabled ? '文档已发布' : '文档已转为草稿')
}

function removePage(page: DocPage) {
  if (!window.confirm(`确定删除文档“${page.title}”吗？此操作不可撤销。`)) return
  store.removePage(page.id)
  ensureValidDefault()
  toast.info('文档已删除')
}

const groupModalOpen = ref(false)
const groupEditingId = ref<string | null>(null)
const groupForm = reactive<DocGroup>({
  id: '',
  name: '',
  icon: 'FileText',
  sort: 50,
  enabled: true,
})

const iconOptions = [
  { value: 'FileText', label: '文件' },
  { value: 'Rocket', label: '入门' },
  { value: 'CreditCard', label: '支付' },
  { value: 'Store', label: '商户' },
  { value: 'ArrowLeftRight', label: '转账' },
  { value: 'BookOpen', label: '手册' },
  { value: 'HelpCircle', label: '帮助' },
]
const iconComponents: Record<string, Component> = {
  FileText,
  Rocket,
  CreditCard,
  Store: StoreIcon,
  ArrowLeftRight,
  BookOpen,
  HelpCircle,
}

function groupIcon(icon: string): Component {
  return iconComponents[icon] ?? FileText
}

function resetGroupForm() {
  groupEditingId.value = null
  Object.assign(groupForm, {
    id: '',
    name: '',
    icon: 'FileText',
    sort: 50,
    enabled: true,
  })
}

function openGroupModal() {
  resetGroupForm()
  groupModalOpen.value = true
}

function editGroup(group: DocGroup) {
  groupEditingId.value = group.id
  Object.assign(groupForm, clone(group))
}

function onGroupIcon(value: string | number) {
  if (typeof value === 'string' && iconComponents[value]) groupForm.icon = value
}

function saveGroup() {
  const next: DocGroup = {
    id: groupForm.id.trim(),
    name: groupForm.name.trim(),
    icon: groupForm.icon,
    sort: Number.isFinite(groupForm.sort) ? groupForm.sort : 0,
    enabled: groupForm.enabled,
  }
  if (!isValidDocSlug(next.id)) {
    toast.error('分组 ID 仅支持小写字母、数字和连字符')
    return
  }
  if (!next.name) {
    toast.error('请填写分组名称')
    return
  }
  if (store.groups.some((group) => group.id === next.id && group.id !== groupEditingId.value)) {
    toast.error('分组 ID 已存在')
    return
  }

  if (groupEditingId.value === null) {
    store.addGroup(next)
    toast.success('分组已新增')
  } else if (groupEditingId.value === next.id) {
    store.updateGroup(next)
    toast.success('分组已保存')
  } else {
    const oldId = groupEditingId.value
    store.addGroup(next)
    store.pages
      .filter((page) => page.groupId === oldId)
      .forEach((page) => store.updatePage({ ...clone(page), groupId: next.id }))
    store.removeGroup(oldId)
    toast.success('分组及所属文档已更新')
  }
  ensureValidDefault()
  resetGroupForm()
}

function toggleGroupEnabled(group: DocGroup, enabled: boolean) {
  store.updateGroup({ ...clone(group), enabled })
  ensureValidDefault()
}

function removeGroup(group: DocGroup) {
  if (!window.confirm(`确定删除分组“${group.name}”吗？`)) return
  if (!store.removeGroup(group.id)) {
    toast.error('该分组仍包含文档，无法删除，请先移动或删除所属文档')
    return
  }
  if (groupEditingId.value === group.id) resetGroupForm()
  ensureValidDefault()
  toast.info('分组已删除')
}

const settingsModalOpen = ref(false)
const settingsForm = reactive<DocSettings>({ title: '', subtitle: '', defaultSlug: '' })
const publishedPageOptions = computed(() => {
  const options = sortedPublishedPages().map((page) => ({
    value: page.slug,
    label: `${page.title}（${page.slug}）`,
  }))
  return options.length > 0 ? options : [{ value: '', label: '暂无已发布页面' }]
})

function openSettingsModal() {
  ensureValidDefault()
  Object.assign(settingsForm, clone(store.settings))
  settingsModalOpen.value = true
}

function onDefaultSlug(value: string | number) {
  if (typeof value === 'string') settingsForm.defaultSlug = value
}

function saveSettings() {
  const next: DocSettings = {
    title: settingsForm.title.trim(),
    subtitle: settingsForm.subtitle.trim(),
    defaultSlug: settingsForm.defaultSlug,
  }
  const published = sortedPublishedPages()
  if (!next.title) {
    toast.error('请填写文档站标题')
    return
  }
  if (published.length > 0 && !published.some((page) => page.slug === next.defaultSlug)) {
    toast.error('请选择有效的已发布默认页面')
    return
  }
  if (published.length === 0 && next.defaultSlug !== '') {
    toast.error('暂无已发布页面，默认页面必须留空')
    return
  }
  store.updateSettings(next)
  settingsModalOpen.value = false
  toast.success('文档设置已保存')
}
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="文档管理" subtitle="管理开发者文档的页面、分组与文档站设置">
      <template #actions>
        <div class="flex flex-wrap items-center justify-end gap-2">
          <Button variant="outline" size="sm" @click="previewDocs">
            <ExternalLink class="size-4" />预览文档站
          </Button>
          <Button variant="outline" size="sm" @click="openSettingsModal">
            <Settings class="size-4" />文档设置
          </Button>
          <Button variant="outline" size="sm" @click="openGroupModal">
            <FolderTree class="size-4" />分组管理
          </Button>
          <Button size="sm" @click="openCreate">
            <Plus class="size-4" />新增文档
          </Button>
        </div>
      </template>

      <div class="flex flex-wrap gap-x-10 gap-y-4">
        <div>
          <div class="text-[13px] text-muted-foreground">文档总数</div>
          <div class="mt-1 text-xl font-normal tabular-nums">{{ stats.total }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">已发布</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-success">{{ stats.published }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">草稿</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-muted-foreground">{{ stats.draft }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">分组数</div>
          <div class="mt-1 text-xl font-normal tabular-nums">{{ stats.groups }}</div>
        </div>
      </div>
    </Panel>

    <Panel title="文档列表" :subtitle="`${filteredPages.length} 篇`">
      <div class="filter-bar mb-4">
        <div class="filter-item">
          <label class="filter-label">关键词</label>
          <input
            v-model="keyword"
            class="field-input w-56"
            placeholder="标题、slug 或关键词"
          />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">分组</label>
          <Select
            :model-value="filterGroup"
            :options="groupOptions"
            class="w-36"
            @update:model-value="onFilterGroup"
          />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">状态</label>
          <Select
            :model-value="filterStatus"
            :options="statusOptions"
            class="w-28"
            @update:model-value="onFilterStatus"
          />
        </div>
      </div>

      <div class="overflow-x-auto">
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[53%]">标题</th>
              <th class="w-[20%]">分组</th>
              <th class="col-center w-[12%]">发布</th>
              <th class="col-center w-[15%]">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="page in filteredPages" :key="page.id">
              <td>
                <div class="flex min-w-0 items-center gap-2">
                  <FileText class="size-4 shrink-0 text-muted-foreground" />
                  <div class="truncate text-sm font-medium">{{ page.title }}</div>
                  <Badge v-if="store.settings.defaultSlug === page.slug" variant="outline" class="shrink-0">
                    默认
                  </Badge>
                </div>
              </td>
              <td><Badge variant="muted">{{ groupName(page.groupId) }}</Badge></td>
              <td class="col-center">
                <div class="flex justify-center">
                  <Switch
                    :model-value="page.status === 1"
                    size="sm"
                    @update:model-value="togglePageStatus(page, $event)"
                  />
                </div>
              </td>
              <td class="col-center">
                <div class="flex items-center justify-center gap-1">
                  <Button variant="ghost" size="sm" title="编辑" @click="openEdit(page)">
                    <Pencil class="size-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    class="text-destructive hover:text-destructive"
                    title="删除"
                    @click="removePage(page)"
                  >
                    <Trash2 class="size-4" />
                  </Button>
                </div>
              </td>
            </tr>
            <tr v-if="filteredPages.length === 0">
              <td colspan="4" class="py-10 text-center dim">暂无符合条件的文档</td>
            </tr>
          </tbody>
        </table>
      </div>
    </Panel>

    <Drawer
      v-model="drawerOpen"
      :title="editingId === null ? '新增文档' : '编辑文档'"
      subtitle="配置页面信息与文档内容，发布后将同步到文档站"
      width="max-w-6xl"
    >
      <!-- 抽屉内标签页导航 -->
      <div class="-mt-1 mb-5 flex items-center gap-6 border-b border-border">
        <button
          v-for="tab in drawerTabs"
          :key="tab.value"
          type="button"
          class="relative -mb-px border-b-2 px-0.5 pb-2.5 text-sm transition-colors"
          :class="activeTab === tab.value
            ? 'border-primary font-medium text-primary'
            : 'border-transparent text-muted-foreground hover:text-foreground'"
          @click="selectTab(tab.value)"
        >
          {{ tab.label }}
        </button>
      </div>

      <!-- 页面信息 -->
      <div v-show="activeTab === 'info'" class="max-w-3xl space-y-5">
        <div class="grid gap-x-8 gap-y-5 sm:grid-cols-2">
          <div>
            <label class="fld-lbl"><span class="text-destructive">*</span> 标题</label>
            <input v-model="form.title" class="field-input mt-1.5 w-full" placeholder="请输入文档标题" />
          </div>
          <div>
            <label class="fld-lbl"><span class="text-destructive">*</span> 分组</label>
            <Select
              :model-value="form.groupId"
              :options="editGroupOptions"
              class="mt-1.5 w-full"
              placeholder="请选择分组"
              @update:model-value="onFormGroup"
            />
          </div>
          <div>
            <label class="fld-lbl">关键词</label>
            <input
              v-model="form.keywords"
              class="field-input mt-1.5 w-full"
              placeholder="用于文档搜索，空格分隔"
            />
          </div>
          <div>
            <label class="fld-lbl">状态</label>
            <Select
              :model-value="form.status"
              :options="editStatusOptions"
              class="mt-1.5 w-full"
              @update:model-value="onFormStatus"
            />
            <p class="mt-1 text-xs text-muted-foreground">发布后同步到文档站，需至少一个有内容的内容块</p>
          </div>
        </div>
      </div>

      <!-- 文档内容 -->
      <div v-show="activeTab === 'content'">
        <DocContentEditor v-model="contentBlocks" />
      </div>

      <!-- 页面预览 -->
      <div v-show="activeTab === 'preview'" class="min-h-64 bg-muted/40 p-5">
        <DocPageContent :page="previewPage" compact />
      </div>

      <template #footer>
        <span class="mr-auto text-xs text-muted-foreground">
          第 {{ activeStep + 1 }} / {{ tabOrder.length }} 步 · {{ drawerTabs[activeStep].label }}
        </span>
        <Button variant="outline" @click="drawerOpen = false">取消</Button>
        <Button v-if="!isFirstStep" variant="outline" @click="goPrevStep">
          <ChevronLeft class="size-4" />上一步
        </Button>
        <Button v-if="!isLastStep" @click="goNextStep">
          下一步<ChevronRight class="size-4" />
        </Button>
        <Button v-else @click="savePage">{{ editingId === null ? '新增文档' : '保存文档' }}</Button>
      </template>
    </Drawer>

    <Modal v-model="groupModalOpen" title="分组管理" width="max-w-3xl">
      <div class="space-y-4">
        <div class="max-h-72 space-y-2 overflow-y-auto">
          <div
            v-for="group in [...store.groups].sort((a, b) => a.sort - b.sort)"
            :key="group.id"
            class="flex items-center gap-3 border border-border px-3 py-2"
          >
            <component :is="groupIcon(group.icon)" class="size-4 shrink-0 text-muted-foreground" />
            <div class="min-w-0 flex-1">
              <div class="truncate text-sm font-medium">{{ group.name }}</div>
              <div class="truncate font-mono text-xs text-muted-foreground">{{ group.id }}</div>
            </div>
            <span class="text-xs tabular-nums text-muted-foreground">
              {{ store.pages.filter((page) => page.groupId === group.id).length }} 篇
            </span>
            <span class="text-xs tabular-nums text-muted-foreground">排序 {{ group.sort }}</span>
            <Switch
              :model-value="group.enabled"
              size="sm"
              @update:model-value="toggleGroupEnabled(group, $event)"
            />
            <Button variant="ghost" size="sm" title="编辑分组" @click="editGroup(group)">
              <Pencil class="size-4" />
            </Button>
            <Button
              variant="ghost"
              size="sm"
              class="text-destructive hover:text-destructive"
              title="删除分组"
              @click="removeGroup(group)"
            >
              <Trash2 class="size-4" />
            </Button>
          </div>
          <div v-if="store.groups.length === 0" class="py-8 text-center text-sm text-muted-foreground">
            暂无分组
          </div>
        </div>

        <div class="space-y-3 border-t border-border pt-4">
          <div class="text-sm font-medium">{{ groupEditingId === null ? '新增分组' : '编辑分组' }}</div>
          <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
            <div>
              <label class="fld-lbl">分组 ID</label>
              <input v-model="groupForm.id" class="field-input mt-1 w-full font-mono" placeholder="getting-started" />
            </div>
            <div>
              <label class="fld-lbl">分组名称</label>
              <input v-model="groupForm.name" class="field-input mt-1 w-full" placeholder="入门" />
            </div>
            <div>
              <label class="fld-lbl">图标</label>
              <Select
                :model-value="groupForm.icon"
                :options="iconOptions"
                class="mt-1 w-full"
                @update:model-value="onGroupIcon"
              />
            </div>
            <div>
              <label class="fld-lbl">排序</label>
              <input v-model.number="groupForm.sort" type="number" class="field-input mt-1 w-full" />
            </div>
          </div>
          <label class="flex items-center gap-2 text-sm">
            <Switch v-model="groupForm.enabled" size="sm" />
            启用分组
          </label>
        </div>
      </div>
      <template #footer>
        <Button v-if="groupEditingId !== null" variant="outline" @click="resetGroupForm">取消编辑</Button>
        <Button @click="saveGroup">{{ groupEditingId === null ? '新增分组' : '保存分组' }}</Button>
      </template>
    </Modal>

    <Modal v-model="settingsModalOpen" title="文档设置" width="max-w-lg">
      <div class="space-y-4">
        <div>
          <label class="fld-lbl">文档站标题</label>
          <input v-model="settingsForm.title" class="field-input mt-1 w-full" placeholder="开发者文档" />
        </div>
        <div>
          <label class="fld-lbl">副标题</label>
          <textarea
            v-model="settingsForm.subtitle"
            class="field-input mt-1 min-h-20 w-full resize-y py-2"
            placeholder="文档站简介"
          />
        </div>
        <div>
          <label class="fld-lbl">默认页面</label>
          <Select
            :model-value="settingsForm.defaultSlug"
            :options="publishedPageOptions"
            class="mt-1 w-full"
            @update:model-value="onDefaultSlug"
          />
          <p class="mt-1.5 text-xs text-muted-foreground">仅可选择已发布且所属分组已启用的页面。</p>
        </div>
      </div>
      <template #footer>
        <Button variant="outline" @click="settingsModalOpen = false">取消</Button>
        <Button @click="saveSettings">保存设置</Button>
      </template>
    </Modal>
  </div>
</template>
