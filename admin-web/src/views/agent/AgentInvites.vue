<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { Plus, Copy, Trash2 } from 'lucide-vue-next'
import { Panel, Button, Badge, Switch, Drawer, Pagination } from '@/components/ui'
import { fetchMyInvites, createMyInvite, setMyInviteStatus, deleteMyInvite } from '@/lib/api/agent'
import type { Invite } from '@/lib/api/console'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'
import { useConfirm } from '@/composables/useConfirm'

const toast = useToast()
const confirm = useConfirm()

const invites = ref<Invite[]>([])
const total = ref(0)
const loading = ref(false)
const page = ref(1)
const pageSize = 20
const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

// 客户自助进件公开页地址（公开页在后续批次实现）
const enrollUrl = (code: string) => `${location.origin}/enroll/${code}`

async function load() {
  loading.value = true
  try {
    const { list, total: t } = await fetchMyInvites({ page: page.value, pageSize })
    invites.value = list
    total.value = t
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载邀请链接失败')
  } finally {
    loading.value = false
  }
}

onMounted(load)

function go(p: number) {
  page.value = p
  load()
}

async function copyLink(code: string) {
  try {
    await navigator.clipboard.writeText(enrollUrl(code))
    toast.success('链接已复制')
  } catch {
    toast.info(enrollUrl(code))
  }
}

async function toggle(v: Invite) {
  if (v.status === 2) {
    toast.info('已失效的链接不可再启用，请新建替换')
    return
  }
  const next = v.status === 1 ? 0 : 1
  try {
    await setMyInviteStatus(v.id, next)
    v.status = next
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '操作失败')
  }
}

async function remove(v: Invite) {
  if (!(await confirm(`确定删除邀请链接「${v.name || v.code}」？`, { title: '删除邀请链接', danger: true }))) return
  try {
    await deleteMyInvite(v.id)
    toast.success('已删除')
    await load()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '删除失败')
  }
}

// ===== 新建 =====
const drawer = ref(false)
const saving = ref(false)
const form = reactive({ name: '' })

function openCreate() {
  form.name = ''
  drawer.value = true
}

async function save() {
  saving.value = true
  try {
    await createMyInvite({ name: form.name || undefined })
    drawer.value = false
    toast.success('已生成')
    await load()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '生成失败')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="邀请链接" subtitle="客户扫码/点链接自助进件，进件单自动归属你名下">
      <template #actions>
        <Button size="sm" @click="openCreate"><Plus />生成链接</Button>
      </template>
    </Panel>

    <Panel title="链接列表" :subtitle="`${total} 条`">
      <div class="overflow-x-auto">
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[18%]">Code</th>
              <th class="w-[24%]">备注</th>
              <th class="num w-[12%]">打开数</th>
              <th class="num w-[12%]">提交数</th>
              <th class="col-center w-[12%]">状态</th>
              <th class="col-center w-[22%]">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="v in invites" :key="v.id">
              <td class="truncate font-medium tabular-nums">{{ v.code }}</td>
              <td class="truncate">{{ v.name || '—' }}</td>
              <td class="num tabular-nums">{{ v.open_count }}</td>
              <td class="num tabular-nums">{{ v.submit_count }}</td>
              <td class="col-center">
                <div v-if="v.status === 2" class="flex justify-center">
                  <Badge variant="destructive">已失效</Badge>
                </div>
                <div v-else class="flex justify-center">
                  <Switch :model-value="v.status === 1" size="sm" @update:model-value="toggle(v)" />
                </div>
              </td>
              <td class="col-center">
                <div class="flex items-center justify-center gap-1">
                  <Button variant="ghost" size="sm" @click="copyLink(v.code)"><Copy class="size-4" />复制</Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    class="text-destructive hover:text-destructive"
                    @click="remove(v)"
                  >
                    <Trash2 class="size-4" />
                  </Button>
                </div>
              </td>
            </tr>
            <tr v-if="loading">
              <td colspan="6" class="py-10 text-center dim">加载中…</td>
            </tr>
            <tr v-else-if="!invites.length">
              <td colspan="6" class="py-10 text-center dim">暂无邀请链接</td>
            </tr>
          </tbody>
        </table>
      </div>
      <div class="mt-4 border-t border-border/60 pt-4">
        <Pagination :page="page" :page-count="pageCount" :total="total" :page-size="pageSize" @change="go" />
      </div>
      <p class="mt-3 border-t border-border/60 pt-3 text-xs text-muted-foreground">
        链接有效期锚定终态事件（关单/驳回/退款完成后 24h 起算，审核中不倒计时）。二维码与自助进件公开页在后续批次实现。
      </p>
    </Panel>

    <Drawer v-model="drawer" title="生成邀请链接" subtitle="客户从该链接进来的进件单自动归属你名下">
      <div class="space-y-3.5">
        <div class="row-field">
          <label class="lbl">备注</label>
          <input v-model="form.name" placeholder="选填，便于区分给了哪个客户" class="field-input flex-1" />
        </div>
      </div>
      <template #footer>
        <Button variant="outline" @click="drawer = false">取消</Button>
        <Button :disabled="saving" @click="save">{{ saving ? '生成中…' : '生成' }}</Button>
      </template>
    </Drawer>
  </div>
</template>
