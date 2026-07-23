<script setup lang="ts">
/**
 * 官网管理 · 文章管理。
 * 管理官网「最新动态」板块的文章与分类：
 * - 文章列表（按分类/状态筛选）+ 新增/编辑抽屉（含 tiptap 富文本正文）
 * - 分类管理（Modal，增删改）
 * 数据经 useArticlesStore 持久化 → 官网首页板块 + 文章详情页实时联动。
 */
import { ref, reactive, computed, onMounted, watch } from 'vue'
import {
  Plus, Pencil, Trash2, FileText, Tags, Star, Eye, X,
} from 'lucide-vue-next'
import { Panel, Button, Select, Switch, Drawer, Modal, Badge, RichEditor, ImageUpload } from '@/components/ui'
import { useArticlesStore } from '@/stores/articles'
import { useToast } from '@/composables/useToast'
import { ApiError } from '@/lib/api/client'
import type { Article, ArticleCategory } from '@/lib/api/articles'

const store = useArticlesStore()
const toast = useToast()

onMounted(() => store.hydrate())

// ===== 概况 =====
const stats = computed(() => ({
  total: store.articles.length,
  published: store.articles.filter((a) => a.status === 1).length,
  categories: store.categories.length,
}))

// ===== 筛选 =====
const filterCat = ref<number | 0>(0) // 0=全部
const filterStatus = ref<number | -1>(-1) // -1=全部
const catFilterOptions = computed(() => [
  { value: 0, label: '全部分类' },
  ...store.categories.map((c) => ({ value: c.id, label: c.name })),
])
const statusFilterOptions = [
  { value: -1, label: '全部状态' },
  { value: 1, label: '已发布' },
  { value: 0, label: '草稿' },
]

const filteredList = computed(() =>
  [...store.articles]
    .filter((a) => filterCat.value === 0 || a.categoryId === filterCat.value)
    .filter((a) => filterStatus.value === -1 || a.status === filterStatus.value)
    .sort((a, b) => a.sort - b.sort),
)

// ===== 文章增删改（抽屉）=====
const drawerOpen = ref(false)
const editingId = ref<number | null>(null)
// 抽屉内分步：0=基础信息(基本信息+展示设置)，1=正文，2=发布设置。步进式引导。
const STEPS = ['基础信息', '正文', '发布设置'] as const
const activeStep = ref(0)
const blankForm = (): Omit<Article, 'id'> => ({
  categoryId: store.categories[0]?.id ?? 1,
  title: '', summary: '', content: '', cover: '', tags: [],
  isNew: false, isTop: false, status: 1, sort: 50, views: 0,
  addtime: new Date().toISOString().slice(0, 10),
})
const form = reactive<Omit<Article, 'id'>>(blankForm())

// 标签片输入：输入框回车/逗号生成标签片，Backspace 删末项，chip 点 × 删除
const tagInput = ref('')
const tagInputEl = ref<HTMLInputElement | null>(null)
/** 提交当前输入框内容为一个标签（去空、去重）。 */
function commitTag() {
  const t = tagInput.value.trim()
  tagInput.value = ''
  if (!t) return
  if (!form.tags) form.tags = []
  if (!form.tags.includes(t)) form.tags.push(t)
}
function removeTag(i: number) {
  form.tags?.splice(i, 1)
}
/** 逗号也触发生成；输入框为空时按 Backspace 删除末尾标签。 */
function onTagKeydown(e: KeyboardEvent) {
  if (e.key === ',' || e.key === '，') {
    e.preventDefault()
    commitTag()
  } else if (e.key === 'Backspace' && tagInput.value === '' && form.tags?.length) {
    removeTag(form.tags.length - 1)
  }
}

const catEditOptions = computed(() => store.categories.map((c) => ({ value: c.id, label: c.name })))

// ===== 新增文章草稿自动留存（防误关丢稿）=====
// 仅新增模式（editingId === null）留存到 localStorage；发布/保存成功后清空。
const DRAFT_KEY = 'article-draft-new'
const draftRestored = ref(false) // 本次打开是否由草稿恢复而来（用于显示提示与「放弃草稿」）

/** 判断表单是否有实质内容（避免把空白也当草稿存）。 */
function formHasContent(): boolean {
  return !!(form.title.trim() || form.summary.trim() || form.content.trim() || (form.tags ?? []).length || form.cover.trim())
}
function saveDraft() {
  try {
    localStorage.setItem(DRAFT_KEY, JSON.stringify({ ...form }))
  } catch {
    // localStorage 满/禁用时静默
  }
}
function clearDraft() {
  localStorage.removeItem(DRAFT_KEY)
  draftRestored.value = false
}
function loadDraft(): Omit<Article, 'id'> | null {
  try {
    const raw = localStorage.getItem(DRAFT_KEY)
    if (raw) return JSON.parse(raw)
  } catch {
    // 损坏草稿忽略
  }
  return null
}

// 抽屉打开且处于新增模式时，表单任何变化自动存草稿（深度监听）。
watch(
  form,
  () => {
    if (drawerOpen.value && editingId.value === null && formHasContent()) {
      saveDraft()
    }
  },
  { deep: true },
)

function openCreate() {
  editingId.value = null
  activeStep.value = 0
  const draft = loadDraft()
  if (draft) {
    // 有未提交草稿：恢复并提示
    Object.assign(form, blankForm(), draft)
    draftRestored.value = true
    toast.info('已恢复上次未发布的草稿')
  } else {
    Object.assign(form, blankForm())
    draftRestored.value = false
  }
  drawerOpen.value = true
}

/** 放弃草稿：清空草稿并重置为空白表单（仍停留在新增抽屉）。 */
function discardDraft() {
  clearDraft()
  Object.assign(form, blankForm())
  toast.info('已放弃草稿')
}
function openEdit(a: Article) {
  editingId.value = a.id
  activeStep.value = 0
  draftRestored.value = false // 编辑既有文章不涉及新增草稿
  Object.assign(form, JSON.parse(JSON.stringify(a)))
  drawerOpen.value = true
}

// ===== 分步导航（上一步 / 下一步）=====
/** 校验当前步是否可进入下一步。 */
function validateStep(step: number): boolean {
  if (step === 0 && !form.title.trim()) {
    toast.error('请先填写文章标题')
    return false
  }
  if (step === 1 && !form.content.trim()) {
    toast.error('请先填写文章正文')
    return false
  }
  return true
}
function nextStep() {
  if (!validateStep(activeStep.value)) return
  if (activeStep.value < STEPS.length - 1) activeStep.value += 1
}
function prevStep() {
  if (activeStep.value > 0) activeStep.value -= 1
}
/** 点击步骤条跳转：只允许回退或校验通过后前进。 */
function gotoStep(target: number) {
  if (target === activeStep.value) return
  if (target < activeStep.value) {
    activeStep.value = target
    return
  }
  // 前进：逐步校验中间每一步
  for (let s = activeStep.value; s < target; s++) {
    if (!validateStep(s)) return
  }
  activeStep.value = target
}

const saving = ref(false)
async function saveArticle() {
  if (!form.title.trim()) {
    toast.error('请填写文章标题')
    return
  }
  if (!form.content.trim()) {
    toast.error('请填写文章正文')
    return
  }
  if (saving.value) return
  saving.value = true
  try {
    if (editingId.value === null) {
      await store.addArticle({ ...form })
      clearDraft() // 新增成功，清掉草稿
      toast.success('文章已新增')
    } else {
      await store.updateArticle({ ...form, id: editingId.value })
      toast.success('文章已保存')
    }
    drawerOpen.value = false
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}
async function removeArticle(id: number) {
  try {
    await store.removeArticle(id)
    toast.info('文章已删除')
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '删除失败')
  }
}
async function toggleStatus(a: Article) {
  try {
    await store.setArticleStatus(a.id, a.status === 1 ? 0 : 1)
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '操作失败')
  }
}

// ===== 分类管理（Modal）=====
const catModalOpen = ref(false)
const catEditingId = ref<number | null>(null)
const catForm = reactive<Omit<ArticleCategory, 'id'>>({ name: '', enName: '', cover: '', sort: 50 })

function openCatCreate() {
  catEditingId.value = null
  Object.assign(catForm, { name: '', enName: '', cover: '', sort: 50 })
}
function openCatEdit(c: ArticleCategory) {
  catEditingId.value = c.id
  Object.assign(catForm, { name: c.name, enName: c.enName, cover: c.cover, sort: c.sort })
}
const catSaving = ref(false)
async function saveCat() {
  if (!catForm.name.trim()) {
    toast.error('请填写分类名称')
    return
  }
  if (catSaving.value) return
  catSaving.value = true
  try {
    if (catEditingId.value === null) {
      await store.addCategory({ ...catForm })
      toast.success('分类已新增')
    } else {
      await store.updateCategory({ ...catForm, id: catEditingId.value })
      toast.success('分类已保存')
    }
    openCatCreate() // 重置表单，便于连续添加
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '保存失败')
  } finally {
    catSaving.value = false
  }
}
async function removeCat(id: number) {
  const count = store.articles.filter((a) => a.categoryId === id).length
  if (count > 0 && !window.confirm(`该分类下有 ${count} 篇文章，删除分类会一并删除这些文章，确定继续？`)) return
  try {
    await store.removeCategory(id)
    toast.info('分类已删除')
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '删除失败')
  }
}

const catName = (id: number) => store.categoryName[id] ?? '—'
</script>

<template>
  <div class="space-y-2.5">
    <!-- 概况 -->
    <Panel title="文章管理" subtitle="管理官网「最新动态」板块的文章与分类，保存后官网实时生效">
      <template #actions>
        <Button variant="outline" size="sm" @click="catModalOpen = true"><Tags class="size-4" />分类管理</Button>
        <Button size="sm" @click="openCreate"><Plus class="size-4" />新增文章</Button>
      </template>
      <div class="flex flex-wrap gap-x-10 gap-y-4">
        <div>
          <div class="text-[13px] text-muted-foreground">文章总数</div>
          <div class="mt-1 text-xl font-normal tabular-nums">{{ stats.total }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">已发布</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-success">{{ stats.published }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">草稿</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-muted-foreground">{{ stats.total - stats.published }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">分类数</div>
          <div class="mt-1 text-xl font-normal tabular-nums">{{ stats.categories }}</div>
        </div>
      </div>
    </Panel>

    <!-- 列表 -->
    <Panel title="文章列表" :subtitle="`${filteredList.length} 篇`">
      <template #actions>
        <div class="flex items-center gap-2">
          <Select v-model="filterCat" :options="catFilterOptions" class="w-32" />
          <Select v-model="filterStatus" :options="statusFilterOptions" class="w-28" />
        </div>
      </template>
      <div class="overflow-x-auto">
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[7%]">排序</th>
              <th class="w-[40%]">标题</th>
              <th class="w-[12%]">分类</th>
              <th class="col-center w-[9%]">浏览</th>
              <th class="w-[12%]">发布时间</th>
              <th class="col-center w-[10%]">发布</th>
              <th class="col-center w-[10%]">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="a in filteredList" :key="a.id">
              <td class="tabular-nums dim">{{ a.sort }}</td>
              <td>
                <div class="flex items-center gap-2">
                  <FileText class="size-4 shrink-0 text-muted-foreground" />
                  <span class="truncate">{{ a.title }}</span>
                  <Star v-if="a.isTop" class="size-3.5 shrink-0 fill-amber-400 text-amber-400" title="置顶" />
                  <Badge v-if="a.isNew" variant="destructive" class="shrink-0">new</Badge>
                </div>
              </td>
              <td><Badge variant="outline">{{ catName(a.categoryId) }}</Badge></td>
              <td class="col-center">
                <span class="inline-flex items-center gap-1 tabular-nums dim"><Eye class="size-3.5" />{{ a.views }}</span>
              </td>
              <td class="text-xs">{{ a.addtime }}</td>
              <td class="col-center">
                <div class="flex justify-center">
                  <Switch :model-value="a.status === 1" size="sm" @update:model-value="toggleStatus(a)" />
                </div>
              </td>
              <td class="col-center">
                <div class="flex items-center justify-center gap-1">
                  <Button variant="ghost" size="sm" @click="openEdit(a)"><Pencil class="size-4" /></Button>
                  <Button variant="ghost" size="sm" class="text-destructive hover:text-destructive" @click="removeArticle(a.id)">
                    <Trash2 class="size-4" />
                  </Button>
                </div>
              </td>
            </tr>
            <tr v-if="!filteredList.length">
              <td colspan="7" class="py-10 text-center dim">暂无文章</td>
            </tr>
          </tbody>
        </table>
      </div>
    </Panel>

    <!-- 新增 / 编辑文章抽屉 -->
    <Drawer
      v-model="drawerOpen"
      :title="editingId ? '编辑文章' : '新增文章'"
      subtitle="正文支持富文本，保存后在官网「最新动态」板块展示"
      width="max-w-3xl"
    >
      <div class="space-y-5">
        <!-- 草稿恢复提示（仅新增模式且由草稿恢复时显示）-->
        <div
          v-if="draftRestored && !editingId"
          class="flex items-center gap-2 rounded bg-warning/10 px-3 py-2 text-xs text-warning"
        >
          <FileText class="size-3.5 shrink-0" />
          <span class="flex-1">已恢复上次未发布的草稿，可继续编辑。发布或新增成功后草稿自动清除。</span>
          <button class="shrink-0 font-medium underline-offset-2 hover:underline" @click="discardDraft">放弃草稿</button>
        </div>

        <!-- 步骤指示器：小巧文字 tab，选中项下划线高亮 -->
        <div class="flex gap-5 border-b border-border">
          <button
            v-for="(s, i) in STEPS"
            :key="i"
            type="button"
            class="relative -mb-px border-b-2 pb-2 text-sm transition-colors"
            :class="i === activeStep
              ? 'border-primary font-medium text-primary'
              : 'border-transparent text-muted-foreground hover:text-foreground'"
            @click="gotoStep(i)"
          >{{ s }}</button>
        </div>

        <!-- 第 1 步：基础信息（基本信息 + 展示设置）-->
        <div v-show="activeStep === 0" class="space-y-6">
          <!-- 基本信息 -->
          <section class="space-y-4">
            <div class="text-xs font-medium uppercase tracking-wide text-muted-foreground/70">基本信息</div>
            <div>
              <label class="mb-1.5 block text-sm text-muted-foreground">标题</label>
              <input v-model="form.title" placeholder="输入文章标题" class="field-input w-full" />
            </div>
            <div class="grid gap-4 sm:grid-cols-2">
              <div>
                <label class="mb-1.5 block text-sm text-muted-foreground">分类</label>
                <Select v-model="form.categoryId" :options="catEditOptions" class="w-full" />
              </div>
              <div>
                <label class="mb-1.5 block text-sm text-muted-foreground">发布时间</label>
                <input v-model="form.addtime" type="date" class="field-input w-full" />
              </div>
            </div>
            <div>
              <label class="mb-1.5 block text-sm text-muted-foreground">摘要 <span class="text-muted-foreground/60">（首页头条展示，2-3 行）</span></label>
              <textarea
                v-model="form.summary"
                rows="3"
                placeholder="输入文章摘要"
                class="field-input w-full resize-none py-2"
                style="height: auto"
              />
            </div>
          </section>
        </div>

        <!-- 第 2 步：正文 -->
        <div v-show="activeStep === 1" class="space-y-6">
          <section class="space-y-4">
            <div class="text-xs font-medium uppercase tracking-wide text-muted-foreground/70">正文</div>
            <RichEditor v-model="form.content" placeholder="输入文章正文…" />
          </section>
        </div>

        <!-- 第 3 步：发布设置（各字段竖排，无分组小标题）-->
        <div v-show="activeStep === 2" class="space-y-5">
          <!-- 封面图 -->
          <div>
            <label class="mb-1.5 block text-sm text-muted-foreground">封面图 <span class="text-muted-foreground/60">（可选，留空不显示封面）</span></label>
            <ImageUpload v-model="form.cover" dir="cover" />
          </div>

          <!-- 标签 -->
          <div>
            <label class="mb-1.5 block text-sm text-muted-foreground">标签 <span class="text-muted-foreground/60">（输入后回车生成，用于资讯列表标签与侧栏标签云）</span></label>
            <!-- 标签片输入：已生成的标签为可删除的 chip，末尾是输入框，回车/逗号生成 -->
            <div
              class="field-input flex min-h-9 w-full flex-wrap items-center gap-1.5 py-1.5"
              @click="tagInputEl?.focus()"
            >
              <span
                v-for="(t, i) in form.tags"
                :key="t"
                class="inline-flex items-center gap-1 rounded-sm bg-muted px-2 py-0.5 text-xs text-foreground"
              >
                {{ t }}
                <button type="button" class="text-muted-foreground/60 transition-colors hover:text-destructive" @click.stop="removeTag(i)">
                  <X class="size-3" />
                </button>
              </span>
              <input
                ref="tagInputEl"
                v-model="tagInput"
                :placeholder="form.tags?.length ? '' : '输入标签后回车，如：标准版系统'"
                class="min-w-[8rem] flex-1 border-0 bg-transparent p-0 text-sm outline-none placeholder:text-muted-foreground/60"
                @keydown.enter.prevent="commitTag"
                @keydown="onTagKeydown"
                @blur="commitTag"
              />
            </div>
          </div>

          <!-- 排序 -->
          <div>
            <label class="mb-1.5 block text-sm text-muted-foreground">排序 <span class="text-muted-foreground/60">（越小越靠前）</span></label>
            <input v-model.number="form.sort" type="number" class="field-input w-40" />
          </div>

          <!-- 虚拟阅读量 -->
          <div>
            <label class="mb-1.5 block text-sm text-muted-foreground">虚拟阅读量 <span class="text-muted-foreground/60">（初始基数，叠加在真实浏览量之上）</span></label>
            <input v-model.number="form.views" type="number" min="0" class="field-input w-40" />
          </div>

          <!-- 状态开关：浅灰底行块，左标题+说明堆叠，开关靠右居中 -->
          <div class="max-w-md space-y-2 pt-1">
            <label class="flex cursor-pointer items-center justify-between gap-4 bg-muted/40 px-3.5 py-2.5">
              <span class="flex flex-col">
                <span class="text-sm text-foreground">置顶</span>
                <span class="mt-0.5 text-xs text-muted-foreground/70">首页头条优先展示</span>
              </span>
              <Switch v-model="form.isTop" size="sm" />
            </label>
            <label class="flex cursor-pointer items-center justify-between gap-4 bg-muted/40 px-3.5 py-2.5">
              <span class="flex flex-col">
                <span class="text-sm text-foreground">new 标记</span>
                <span class="mt-0.5 text-xs text-muted-foreground/70">资讯列表角标显示 new</span>
              </span>
              <Switch v-model="form.isNew" size="sm" />
            </label>
            <label class="flex cursor-pointer items-center justify-between gap-4 bg-muted/40 px-3.5 py-2.5">
              <span class="flex flex-col">
                <span class="text-sm text-foreground">发布</span>
                <span class="mt-0.5 text-xs text-muted-foreground/70">关闭则存为草稿，官网不展示</span>
              </span>
              <Switch :model-value="form.status === 1" size="sm" @update:model-value="form.status = form.status === 1 ? 0 : 1" />
            </label>
          </div>
        </div>
      </div>
      <template #footer>
        <span v-if="!editingId" class="mr-auto text-xs text-muted-foreground/70">编辑内容自动留存草稿，误关不丢失</span>
        <!-- 非首步显示「上一步」 -->
        <Button v-if="activeStep > 0" variant="outline" @click="prevStep">上一步</Button>
        <!-- 非末步显示「下一步」，末步显示「保存/发布」 -->
        <Button v-if="activeStep < STEPS.length - 1" @click="nextStep">下一步</Button>
        <Button v-else @click="saveArticle">{{ editingId ? '保存' : '发布' }}</Button>
      </template>
    </Drawer>

    <!-- 分类管理 Modal -->
    <Modal v-model="catModalOpen" title="分类管理" width="max-w-lg">
      <div class="space-y-4">
        <!-- 现有分类列表 -->
        <div class="space-y-2">
          <div
            v-for="c in store.categories"
            :key="c.id"
            class="flex items-center gap-3 rounded border border-border px-3 py-2"
          >
            <span class="flex-1">
              <span class="text-sm font-medium">{{ c.name }}</span>
              <span class="ml-2 text-xs text-muted-foreground">{{ c.enName }}</span>
            </span>
            <span class="text-xs tabular-nums dim">{{ store.articles.filter((a) => a.categoryId === c.id).length }} 篇</span>
            <Button variant="ghost" size="sm" @click="openCatEdit(c)"><Pencil class="size-4" /></Button>
            <Button variant="ghost" size="sm" class="text-destructive hover:text-destructive" @click="removeCat(c.id)">
              <Trash2 class="size-4" />
            </Button>
          </div>
          <div v-if="!store.categories.length" class="py-6 text-center dim">暂无分类</div>
        </div>

        <!-- 新增 / 编辑分类表单 -->
        <div class="space-y-3 border-t border-border pt-4">
          <div class="text-sm font-medium">{{ catEditingId ? '编辑分类' : '新增分类' }}</div>
          <div class="grid gap-3 sm:grid-cols-2">
            <div>
              <label class="fld-lbl">分类名称</label>
              <input v-model="catForm.name" placeholder="如：产品动态" class="field-input mt-1 w-full" />
            </div>
            <div>
              <label class="fld-lbl">英文小标题</label>
              <input v-model="catForm.enName" placeholder="如：Function Update" class="field-input mt-1 w-full" />
            </div>
          </div>
          <div class="grid gap-3 sm:grid-cols-[1fr_100px]">
            <div class="sm:col-span-2">
              <label class="fld-lbl">头图（可选，留空用占位）</label>
              <div class="mt-1"><ImageUpload v-model="catForm.cover" dir="category" /></div>
            </div>
            <div>
              <label class="fld-lbl">排序</label>
              <input v-model.number="catForm.sort" type="number" class="field-input mt-1 w-full" />
            </div>
          </div>
        </div>
      </div>
      <template #footer>
        <Button v-if="catEditingId" variant="outline" @click="openCatCreate">取消编辑</Button>
        <Button @click="saveCat">{{ catEditingId ? '保存分类' : '添加分类' }}</Button>
      </template>
    </Modal>
  </div>
</template>
