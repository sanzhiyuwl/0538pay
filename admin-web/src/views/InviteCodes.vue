<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { Search, RotateCcw, Plus, Trash2, Eraser } from 'lucide-vue-next'
import { Panel, Button, Badge, Select, Pagination, Drawer, Modal } from '@/components/ui'
import {
  fetchInviteCodes, generateInviteCodes, deleteInviteCode, clearInviteCodes,
  type InviteCode,
} from '@/lib/api/stats'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()

const inviteStatus: Record<number, { text: string; variant: 'success' | 'default' }> = {
  0: { text: '未使用', variant: 'success' },
  1: { text: '已使用', variant: 'default' },
}
const statusOptions = [
  { value: -1, label: '全部状态' },
  { value: 0, label: '未使用' },
  { value: 1, label: '已使用' },
]

// ===== 筛选（kw 精确等值）=====
const filters = reactive({ kw: '', status: -1 })
const page = ref(1)
const pageSize = 15
const total = ref(0)
const rows = ref<InviteCode[]>([])
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    const res = await fetchInviteCodes({
      page: page.value, pageSize,
      kw: filters.kw.trim() || undefined,
      status: filters.status > -1 ? filters.status : undefined,
    })
    rows.value = res.list
    total.value = res.total
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载邀请码失败')
    rows.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}
function applySearch() {
  page.value = 1
  load()
}
function resetFilters() {
  filters.kw = ''
  filters.status = -1
  applySearch()
}
function go(p: number) {
  page.value = p
  load()
}
onMounted(load)
const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

// ===== 生成 =====
const busy = ref(false)
const genOpen = ref(false)
const genNum = ref('10')
async function submitGen() {
  const n = Number(genNum.value)
  if (!(n > 0)) return toast.error('请输入生成个数')
  if (busy.value) return
  busy.value = true
  try {
    const res = await generateInviteCodes(n)
    toast.success(`已生成 ${res.generated} 个邀请码`)
    genOpen.value = false
    genNum.value = '10'
    page.value = 1
    await load()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '生成失败')
  } finally {
    busy.value = false
  }
}

// ===== 删除 / 清空 =====
const delRow = ref<InviteCode | null>(null)
const delOpen = ref(false)
const clearOpen = ref(false)
function askDelete(c: InviteCode) {
  delRow.value = c
  delOpen.value = true
}
async function doDelete() {
  if (!delRow.value || busy.value) return
  busy.value = true
  try {
    await deleteInviteCode(delRow.value.id)
    toast.success('已删除')
    delOpen.value = false
    await load()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '删除失败')
  } finally {
    busy.value = false
  }
}
async function doClearUsed() {
  if (busy.value) return
  busy.value = true
  try {
    const res = await clearInviteCodes('used')
    toast.success(`已清空 ${res.deleted} 个已使用邀请码`)
    clearOpen.value = false
    page.value = 1
    await load()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '清空失败')
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="space-y-2.5">
    <!-- 筛选 -->
    <Panel title="邀请码管理" :subtitle="`共 ${total} 个`">
      <template #actions>
        <Button size="sm" @click="genOpen = true"><Plus />生成邀请码</Button>
      </template>
      <div class="filter-bar">
        <div class="filter-item">
          <label class="filter-label">邀请码</label>
          <input v-model="filters.kw" placeholder="精确匹配邀请码" class="field-input w-52" @keyup.enter="applySearch" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">状态</label>
          <Select v-model="filters.status" :options="statusOptions" class="w-32" />
        </div>
        <div class="ml-auto flex items-center gap-2">
          <Button size="sm" @click="applySearch"><Search />搜索</Button>
          <Button variant="outline" size="sm" @click="resetFilters"><RotateCcw />重置</Button>
        </div>
      </div>
    </Panel>

    <!-- 列表 -->
    <Panel title="邀请码列表" :subtitle="`${total} 个`">
      <div class="overflow-x-auto">
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[20%]">邀请码</th>
              <th class="w-[10%]">状态</th>
              <th class="w-[18%]">生成时间</th>
              <th class="w-[18%]">使用时间</th>
              <th class="w-[14%]">使用者</th>
              <th class="col-center w-[10%]">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="c in rows" :key="c.id">
              <td class="font-mono text-[13px] font-medium">{{ c.code }}</td>
              <td>
                <Badge :variant="inviteStatus[c.status].variant">{{ inviteStatus[c.status].text }}</Badge>
              </td>
              <td class="text-xs">{{ c.addtime }}</td>
              <td class="text-xs">
                <span v-if="c.usetime">{{ c.usetime }}</span>
                <span v-else class="dim">—</span>
              </td>
              <td class="tabular-nums">
                <span v-if="c.uid" class="text-primary">{{ c.uid }}</span>
                <span v-else class="dim">—</span>
              </td>
              <td class="col-center">
                <Button variant="ghost" size="sm" class="text-destructive hover:text-destructive" @click="askDelete(c)">
                  <Trash2 class="size-4" />
                </Button>
              </td>
            </tr>
            <tr v-if="loading">
              <td colspan="6" class="py-10 text-center dim">加载中…</td>
            </tr>
            <tr v-else-if="!rows.length">
              <td colspan="6" class="py-10 text-center dim">没有符合条件的邀请码</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="mt-4 border-t border-border/60 pt-4">
        <Pagination :page="page" :page-count="pageCount" :total="total" :page-size="pageSize" @change="go" />
      </div>
      <p class="mt-3 flex items-center gap-3 border-t border-border/60 pt-3 text-xs text-muted-foreground">
        <span>仅邀请注册模式（注册登录设置开启「仅邀请注册」）下生效，商户注册时需填写有效邀请码。</span>
        <Button variant="ghost" size="sm" class="ml-auto shrink-0 text-destructive hover:text-destructive" @click="clearOpen = true">
          <Eraser class="size-4" />清空已使用
        </Button>
      </p>
    </Panel>

    <!-- 生成抽屉 -->
    <Drawer v-model="genOpen" title="生成邀请码" subtitle="批量生成新的未使用邀请码" width="max-w-md">
      <div class="space-y-3.5">
        <div class="row-field">
          <label class="lbl">生成个数</label>
          <input v-model="genNum" type="number" min="1" max="200" placeholder="生成的个数" class="field-input flex-1" />
        </div>
        <p class="text-xs text-muted-foreground">单次最多生成 200 个。生成后邀请码状态为「未使用」，可在列表中查看与分发。</p>
      </div>
      <template #footer>
        <Button variant="outline" size="sm" @click="genOpen = false">取消</Button>
        <Button size="sm" :disabled="busy" @click="submitGen"><Plus />确认生成</Button>
      </template>
    </Drawer>

    <!-- 删除确认 -->
    <Modal v-model="delOpen" title="删除确认" width="max-w-md">
      <p class="text-sm text-muted-foreground">确认删除邀请码「{{ delRow?.code }}」？</p>
      <template #footer>
        <Button variant="outline" size="sm" @click="delOpen = false">取消</Button>
        <Button size="sm" :disabled="busy" @click="doDelete">确认删除</Button>
      </template>
    </Modal>

    <!-- 清空已使用确认 -->
    <Modal v-model="clearOpen" title="清空已使用邀请码" width="max-w-md">
      <p class="text-sm text-muted-foreground">确认清空所有已使用的邀请码？此操作不可撤销。</p>
      <template #footer>
        <Button variant="outline" size="sm" @click="clearOpen = false">取消</Button>
        <Button size="sm" :disabled="busy" @click="doClearUsed">确认清空</Button>
      </template>
    </Modal>
  </div>
</template>
