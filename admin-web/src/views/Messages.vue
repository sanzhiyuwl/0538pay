<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { Plus, Trash2, Megaphone, User } from 'lucide-vue-next'
import { Panel, Button, Badge, Pagination, Drawer, Modal } from '@/components/ui'
import { fetchAdminMessages, sendMessage, deleteMessage, type AdminMessage } from '@/lib/api/messages'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()

const page = ref(1)
const pageSize = 15
const total = ref(0)
const rows = ref<AdminMessage[]>([])
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    const res = await fetchAdminMessages({ page: page.value, pageSize })
    rows.value = res.list
    total.value = res.total
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载站内信失败')
    rows.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}
function go(p: number) {
  page.value = p
  load()
}
onMounted(load)
const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

// ===== 下发 =====
const busy = ref(false)
const sendOpen = ref(false)
const form = reactive({ target: 'all', uid: '', title: '', content: '' })
function openSend() {
  form.target = 'all'
  form.uid = ''
  form.title = ''
  form.content = ''
  sendOpen.value = true
}
async function submitSend() {
  if (busy.value) return
  if (!form.title.trim()) return toast.error('请填写标题')
  if (!form.content.trim()) return toast.error('请填写内容')
  const uid = form.target === 'direct' ? Number(form.uid) : 0
  if (form.target === 'direct' && !(uid > 0)) return toast.error('请填写有效的商户号')
  busy.value = true
  try {
    await sendMessage({ uid, title: form.title.trim(), content: form.content.trim() })
    toast.success(form.target === 'all' ? '已全体广播' : `已发送给商户 ${uid}`)
    sendOpen.value = false
    page.value = 1
    await load()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '下发失败')
  } finally {
    busy.value = false
  }
}

// ===== 删除 =====
const delRow = ref<AdminMessage | null>(null)
const delOpen = ref(false)
function askDelete(m: AdminMessage) {
  delRow.value = m
  delOpen.value = true
}
async function doDelete() {
  if (!delRow.value || busy.value) return
  busy.value = true
  try {
    await deleteMessage(delRow.value.id)
    toast.success('已删除')
    delOpen.value = false
    await load()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '删除失败')
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="站内信下发" :subtitle="`共 ${total} 条`">
      <template #actions>
        <Button size="sm" @click="openSend"><Plus />下发站内信</Button>
      </template>
      <div class="overflow-x-auto">
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[14%]">接收方</th>
              <th class="w-[26%]">标题</th>
              <th class="w-[36%]">内容</th>
              <th class="w-[16%]">下发时间</th>
              <th class="col-center w-[8%]">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="m in rows" :key="m.id">
              <td>
                <Badge v-if="m.uid === 0" variant="muted"><Megaphone class="mr-1 size-3" />全体</Badge>
                <span v-else class="flex items-center gap-1 tabular-nums text-primary"><User class="size-3" />{{ m.uid }}</span>
              </td>
              <td class="truncate">{{ m.title }}</td>
              <td class="truncate text-xs text-muted-foreground">{{ m.content }}</td>
              <td class="text-xs">{{ m.date }}</td>
              <td class="col-center">
                <Button variant="ghost" size="sm" class="text-destructive hover:text-destructive" @click="askDelete(m)">
                  <Trash2 class="size-4" />
                </Button>
              </td>
            </tr>
            <tr v-if="loading">
              <td colspan="5" class="py-10 text-center dim">加载中…</td>
            </tr>
            <tr v-else-if="!rows.length">
              <td colspan="5" class="py-10 text-center dim">还没有下发任何站内信</td>
            </tr>
          </tbody>
        </table>
      </div>
      <div class="mt-4 border-t border-border/60 pt-4">
        <Pagination :page="page" :page-count="pageCount" :total="total" :page-size="pageSize" @change="go" />
      </div>
      <p class="mt-3 border-t border-border/60 pt-3 text-xs text-muted-foreground">
        站内信为我方扩展功能（epay 无此实体）。「全体广播」对所有商户可见，商户在商户中心「站内信」查看并可标记已读。
      </p>
    </Panel>

    <!-- 下发抽屉 -->
    <Drawer v-model="sendOpen" title="下发站内信" subtitle="向全体商户广播或定向发送给单个商户" width="max-w-lg">
      <div class="space-y-3.5">
        <div class="row-field">
          <label class="lbl">接收方</label>
          <div class="flex flex-1 gap-2">
            <label class="flex items-center gap-1.5 text-sm"><input v-model="form.target" type="radio" value="all" />全体广播</label>
            <label class="flex items-center gap-1.5 text-sm"><input v-model="form.target" type="radio" value="direct" />指定商户</label>
          </div>
        </div>
        <div v-if="form.target === 'direct'" class="row-field">
          <label class="lbl">商户号</label>
          <input v-model="form.uid" type="number" placeholder="接收的商户 UID" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">标题</label>
          <input v-model="form.title" placeholder="站内信标题" class="field-input flex-1" />
        </div>
        <div class="row-field items-start">
          <label class="lbl pt-2">内容</label>
          <textarea v-model="form.content" rows="5" placeholder="站内信正文" class="field-input flex-1"></textarea>
        </div>
      </div>
      <template #footer>
        <Button variant="outline" size="sm" @click="sendOpen = false">取消</Button>
        <Button size="sm" :disabled="busy" @click="submitSend"><Plus />确认下发</Button>
      </template>
    </Drawer>

    <!-- 删除确认 -->
    <Modal v-model="delOpen" title="删除确认" width="max-w-md">
      <p class="text-sm text-muted-foreground">确认删除站内信「{{ delRow?.title }}」？删除后商户端将不再显示。</p>
      <template #footer>
        <Button variant="outline" size="sm" @click="delOpen = false">取消</Button>
        <Button size="sm" :disabled="busy" @click="doDelete">确认删除</Button>
      </template>
    </Modal>
  </div>
</template>
