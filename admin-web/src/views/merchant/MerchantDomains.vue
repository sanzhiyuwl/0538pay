<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Plus, Trash2, Globe } from 'lucide-vue-next'
import { Panel, Button, Badge, Modal } from '@/components/ui'
import {
  fetchMerchantDomains,
  addMerchantDomain,
  deleteMerchantDomain,
  type MerchantDomain,
} from '@/lib/api/merchantCenter'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()

// 授权域名（对齐 epay user/domain.php，pre_domain）。status：0待审核 1正常 2拒绝
const domainStatus: Record<number, { text: string; variant: 'success' | 'destructive' | 'muted' }> = {
  0: { text: '审核中', variant: 'muted' },
  1: { text: '正常', variant: 'success' },
  2: { text: '已拒绝', variant: 'destructive' },
}

const list = ref<MerchantDomain[]>([])
const loading = ref(false)
const activeCount = computed(() => list.value.filter((d) => d.status === 1).length)

async function load() {
  loading.value = true
  try {
    const res = await fetchMerchantDomains()
    list.value = res.list || []
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载授权域名失败')
  } finally {
    loading.value = false
  }
}
onMounted(load)

// 添加域名弹窗
const addOpen = ref(false)
const newDomain = ref('')
const busy = ref(false)
async function submitAdd() {
  const d = newDomain.value.trim()
  if (!d || busy.value) return
  busy.value = true
  try {
    await addMerchantDomain(d)
    toast.success('已提交，等待管理员审核')
    addOpen.value = false
    newDomain.value = ''
    await load()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '添加失败')
  } finally {
    busy.value = false
  }
}

// 删除
const delTarget = ref<MerchantDomain | null>(null)
async function confirmDelete() {
  if (!delTarget.value || busy.value) return
  busy.value = true
  try {
    await deleteMerchantDomain(delTarget.value.id)
    toast.success('已删除')
    delTarget.value = null
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
    <Panel title="授权域名" :subtitle="`已绑定 ${list.length} 个域名，${activeCount} 个正常`">
      <template #actions>
        <Button size="sm" @click="addOpen = true"><Plus />添加域名</Button>
      </template>
      <div class="overflow-x-auto">
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[44%]">域名</th>
              <th class="col-center w-[16%]">状态</th>
              <th class="w-[28%]">提交时间</th>
              <th class="col-center w-[12%]">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="d in list" :key="d.id">
              <td>
                <div class="flex items-center gap-2">
                  <Globe class="size-4 text-muted-foreground" />
                  <span class="font-mono text-[13px]">{{ d.domain }}</span>
                </div>
              </td>
              <td class="col-center">
                <Badge :variant="domainStatus[d.status]?.variant || 'muted'">{{ domainStatus[d.status]?.text || '未知' }}</Badge>
              </td>
              <td class="text-xs">{{ d.addtime }}</td>
              <td class="col-center">
                <Button variant="ghost" size="sm" class="text-destructive hover:text-destructive" @click="delTarget = d">
                  <Trash2 class="size-4" />
                </Button>
              </td>
            </tr>
            <tr v-if="!loading && !list.length">
              <td colspan="4" class="py-10 text-center dim">暂无授权域名</td>
            </tr>
          </tbody>
        </table>
      </div>
      <p class="mt-3 border-t border-border/60 pt-3 text-xs text-muted-foreground">
        仅已授权的域名可发起支付请求。支持通配符（如 <span class="font-mono">*.demo.cn</span>）。新增域名需管理员审核后生效。
      </p>
    </Panel>

    <!-- 添加域名弹窗 -->
    <Modal v-model="addOpen" title="添加授权域名" width="max-w-md">
      <div class="space-y-3.5">
        <div class="row-field">
          <label class="lbl">域名</label>
          <input v-model="newDomain" placeholder="如 shop.abc.com 或 *.abc.com" class="field-input flex-1" @keyup.enter="submitAdd" />
        </div>
        <p class="text-xs text-muted-foreground">请填写发起支付的网站域名，无需带 http(s):// 前缀。提交后进入审核。</p>
      </div>
      <template #footer>
        <Button variant="outline" size="sm" @click="addOpen = false">取消</Button>
        <Button size="sm" :disabled="!newDomain.trim() || busy" @click="submitAdd"><Plus />提交审核</Button>
      </template>
    </Modal>

    <!-- 删除确认 -->
    <Modal :model-value="!!delTarget" title="删除授权域名" width="max-w-md" @update:model-value="(v) => !v && (delTarget = null)">
      <p class="text-sm">确定删除域名 <span class="font-mono text-foreground">{{ delTarget?.domain }}</span> 吗？删除后该域名将无法发起支付。</p>
      <template #footer>
        <Button variant="outline" size="sm" @click="delTarget = null">取消</Button>
        <Button variant="destructive" size="sm" :disabled="busy" @click="confirmDelete">删除</Button>
      </template>
    </Modal>
  </div>
</template>
