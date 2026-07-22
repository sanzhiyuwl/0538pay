<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { Plus, Pencil, Trash2, Megaphone } from 'lucide-vue-next'
import { Panel, Button, Switch, Drawer } from '@/components/ui'
import {
  fetchAnnounces, createAnnounce, updateAnnounce, setAnnounceStatus, deleteAnnounce,
  type Announce,
} from '@/lib/api/announces'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()
const list = ref<Announce[]>([])
const loading = ref(false)
const busy = ref(false)

async function load() {
  loading.value = true
  try {
    const res = await fetchAnnounces()
    list.value = res.list || []
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载公告失败')
  } finally {
    loading.value = false
  }
}
onMounted(load)

const stats = computed(() => ({
  total: list.value.length,
  shown: list.value.filter((a) => a.status === 1).length,
}))

async function toggleStatus(a: Announce) {
  const next = a.status === 1 ? 0 : 1
  try {
    await setAnnounceStatus(a.id, next)
    a.status = next
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '操作失败')
  }
}
async function remove(id: number) {
  if (busy.value) return
  busy.value = true
  try {
    await deleteAnnounce(id)
    toast.success('已删除')
    await load()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '删除失败')
  } finally {
    busy.value = false
  }
}

// ===== 添加 / 编辑抽屉 =====
const drawerOpen = ref(false)
const editingId = ref<number | null>(null)
const form = reactive({ content: '', color: '', sort: 50 })

function openCreate() {
  editingId.value = null
  Object.assign(form, { content: '', color: '', sort: 50 })
  drawerOpen.value = true
}
function openEdit(a: Announce) {
  editingId.value = a.id
  Object.assign(form, { content: a.content, color: a.color, sort: a.sort })
  drawerOpen.value = true
}
async function save() {
  if (busy.value) return
  if (!form.content.trim()) return toast.error('请填写公告内容')
  busy.value = true
  try {
    if (editingId.value) {
      await updateAnnounce(editingId.value, { content: form.content, color: form.color, sort: form.sort })
      toast.success('已保存')
    } else {
      await createAnnounce({ content: form.content, color: form.color, sort: form.sort })
      toast.success('已发布')
    }
    drawerOpen.value = false
    await load()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '保存失败')
  } finally {
    busy.value = false
  }
}

// 预设颜色
const presetColors = ['', '#e11d48', '#f59e0b', '#2563eb', '#16a34a', '#7c3aed']
</script>

<template>
  <div class="space-y-2.5">
    <!-- 概况 -->
    <Panel title="网站公告" subtitle="发布在商户中心 / 首页的公告，支持排序与文字颜色">
      <template #actions>
        <Button size="sm" @click="openCreate"><Plus />发布公告</Button>
      </template>
      <div class="flex flex-wrap gap-x-10 gap-y-4">
        <div>
          <div class="text-[13px] text-muted-foreground">公告总数</div>
          <div class="mt-1 text-xl font-normal tabular-nums">{{ stats.total }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">显示中</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-success">{{ stats.shown }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">已隐藏</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-muted-foreground">{{ stats.total - stats.shown }}</div>
        </div>
      </div>
    </Panel>

    <!-- 列表 -->
    <Panel title="公告列表" :subtitle="`${list.length} 条`">
      <div class="overflow-x-auto">
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[8%]">排序</th>
              <th class="w-[52%]">公告内容</th>
              <th class="w-[16%]">发布时间</th>
              <th class="col-center w-[12%]">显示</th>
              <th class="col-center w-[12%]">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="a in list" :key="a.id">
              <td class="tabular-nums dim">{{ a.sort }}</td>
              <td>
                <div class="flex items-center gap-2">
                  <Megaphone class="size-4 shrink-0 text-muted-foreground" />
                  <span class="truncate" :style="a.color ? { color: a.color } : {}">{{ a.content }}</span>
                </div>
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
                  <Button variant="ghost" size="sm" class="text-destructive hover:text-destructive" @click="remove(a.id)">
                    <Trash2 class="size-4" />
                  </Button>
                </div>
              </td>
            </tr>
            <tr v-if="!list.length">
              <td colspan="5" class="py-10 text-center dim">暂无公告</td>
            </tr>
          </tbody>
        </table>
      </div>
    </Panel>

    <!-- 添加 / 编辑抽屉 -->
    <Drawer v-model="drawerOpen" :title="editingId ? '编辑公告' : '发布公告'" subtitle="公告将展示在商户中心与首页">
      <div class="space-y-4">
        <div>
          <label class="mb-1.5 block text-sm text-muted-foreground">公告内容</label>
          <textarea
            v-model="form.content"
            rows="5"
            placeholder="输入公告内容"
            class="field-input w-full resize-none py-2"
            style="height: auto"
          />
        </div>
        <div>
          <label class="mb-1.5 block text-sm text-muted-foreground">排序（数字越小越靠前）</label>
          <input v-model.number="form.sort" type="number" class="field-input w-32" />
        </div>
        <div>
          <label class="mb-1.5 block text-sm text-muted-foreground">文字颜色</label>
          <div class="flex items-center gap-2">
            <button
              v-for="c in presetColors"
              :key="c || 'default'"
              class="size-7 rounded-full border-2 transition-transform hover:scale-110"
              :class="form.color === c ? 'border-foreground' : 'border-transparent'"
              :style="{ background: c || 'var(--muted)' }"
              :title="c || '默认色'"
              @click="form.color = c"
            />
            <input v-model="form.color" placeholder="#RRGGBB" maxlength="7" class="field-input ml-2 w-28 font-mono" />
          </div>
        </div>
        <div v-if="form.content" class="bg-muted/40 p-3">
          <div class="mb-1 text-xs text-muted-foreground">预览</div>
          <div class="flex items-center gap-2 text-sm">
            <Megaphone class="size-4 shrink-0 text-muted-foreground" />
            <span :style="form.color ? { color: form.color } : {}">{{ form.content }}</span>
          </div>
        </div>
      </div>
      <template #footer>
        <Button variant="outline" @click="drawerOpen = false">取消</Button>
        <Button @click="save">{{ editingId ? '保存' : '发布' }}</Button>
      </template>
    </Drawer>
  </div>
</template>
