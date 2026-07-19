<script setup lang="ts">
/**
 * 官网管理 · 文章管理。
 * 管理官网「最新动态」板块的文章与分类：
 * - 文章列表（按分类/状态筛选）+ 新增/编辑抽屉（含 tiptap 富文本正文）
 * - 分类管理（Modal，增删改）
 * 数据经 useArticlesStore 持久化 → 官网首页板块 + 文章详情页实时联动。
 */
import { ref, reactive, computed } from 'vue'
import {
  Plus, Pencil, Trash2, FileText, Tags, Star, Eye,
} from 'lucide-vue-next'
import { Panel, Button, Select, Switch, Drawer, Modal, Badge, RichEditor } from '@/components/ui'
import { useArticlesStore } from '@/stores/articles'
import { useToast } from '@/composables/useToast'
import type { Article, ArticleCategory } from '@/lib/mock/articles'

const store = useArticlesStore()
const toast = useToast()

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
const blankForm = (): Omit<Article, 'id'> => ({
  categoryId: store.categories[0]?.id ?? 1,
  title: '', summary: '', content: '', cover: '',
  isNew: false, isTop: false, status: 1, sort: 50, views: 0,
  addtime: new Date().toISOString().slice(0, 10),
})
const form = reactive<Omit<Article, 'id'>>(blankForm())

const catEditOptions = computed(() => store.categories.map((c) => ({ value: c.id, label: c.name })))

function openCreate() {
  editingId.value = null
  Object.assign(form, blankForm())
  drawerOpen.value = true
}
function openEdit(a: Article) {
  editingId.value = a.id
  Object.assign(form, JSON.parse(JSON.stringify(a)))
  drawerOpen.value = true
}
function saveArticle() {
  if (!form.title.trim()) {
    toast.error('请填写文章标题')
    return
  }
  if (editingId.value === null) {
    store.addArticle({ ...form })
    toast.success('文章已新增')
  } else {
    store.updateArticle({ ...form, id: editingId.value })
    toast.success('文章已保存')
  }
  drawerOpen.value = false
}
function removeArticle(id: number) {
  store.removeArticle(id)
  toast.info('文章已删除')
}
function toggleStatus(a: Article) {
  store.updateArticle({ ...a, status: a.status === 1 ? 0 : 1 })
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
function saveCat() {
  if (!catForm.name.trim()) {
    toast.error('请填写分类名称')
    return
  }
  if (catEditingId.value === null) {
    store.addCategory({ ...catForm })
    toast.success('分类已新增')
  } else {
    store.updateCategory({ ...catForm, id: catEditingId.value })
    toast.success('分类已保存')
  }
  openCatCreate() // 重置表单，便于连续添加
}
function removeCat(id: number) {
  const count = store.articles.filter((a) => a.categoryId === id).length
  if (count > 0 && !window.confirm(`该分类下有 ${count} 篇文章，删除分类会一并删除这些文章，确定继续？`)) return
  store.removeCategory(id)
  toast.info('分类已删除')
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
      <div class="space-y-4">
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
          <label class="mb-1.5 block text-sm text-muted-foreground">摘要（首页头条展示，2-3 行）</label>
          <textarea
            v-model="form.summary"
            rows="3"
            placeholder="输入文章摘要"
            class="field-input w-full resize-none py-2"
            style="height: auto"
          />
        </div>
        <div>
          <label class="mb-1.5 block text-sm text-muted-foreground">封面图地址（可选）</label>
          <input v-model="form.cover" placeholder="/assets/xxx.jpg（留空不显示封面）" class="field-input w-full" />
        </div>
        <div>
          <label class="mb-1.5 block text-sm text-muted-foreground">正文</label>
          <RichEditor v-model="form.content" placeholder="输入文章正文…" />
        </div>
        <div class="grid gap-4 sm:grid-cols-2">
          <div>
            <label class="mb-1.5 block text-sm text-muted-foreground">排序（越小越靠前）</label>
            <input v-model.number="form.sort" type="number" class="field-input w-full" />
          </div>
          <div class="flex items-end gap-6 pb-1">
            <label class="flex items-center gap-2 text-sm">
              <Switch v-model="form.isTop" size="sm" /> 置顶
            </label>
            <label class="flex items-center gap-2 text-sm">
              <Switch v-model="form.isNew" size="sm" /> new 标记
            </label>
            <label class="flex items-center gap-2 text-sm">
              <Switch :model-value="form.status === 1" size="sm" @update:model-value="form.status = form.status === 1 ? 0 : 1" /> 发布
            </label>
          </div>
        </div>
      </div>
      <template #footer>
        <Button variant="outline" @click="drawerOpen = false">取消</Button>
        <Button @click="saveArticle">{{ editingId ? '保存' : '新增' }}</Button>
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
            <div>
              <label class="fld-lbl">头图地址（可选）</label>
              <input v-model="catForm.cover" placeholder="/assets/xxx.jpg（留空用占位）" class="field-input mt-1 w-full" />
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
