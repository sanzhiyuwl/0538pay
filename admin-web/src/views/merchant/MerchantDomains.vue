<script setup lang="ts">
import { ref, computed } from 'vue'
import { Plus, Trash2, Globe } from 'lucide-vue-next'
import { Panel, Button, Badge, Modal } from '@/components/ui'

// 授权域名（对齐 epay user/domain.php，pre_domain）
interface Domain {
  id: number
  domain: string
  status: 1 | 2 | 3 // 1正常 2拒绝 3审核中
  addtime: string
}
const domainStatus: Record<number, { text: string; variant: 'success' | 'destructive' | 'muted' }> = {
  1: { text: '正常', variant: 'success' },
  2: { text: '已拒绝', variant: 'destructive' },
  3: { text: '审核中', variant: 'muted' },
}

const list = ref<Domain[]>([
  { id: 3, domain: 'shop.abc.com', status: 1, addtime: '2026-06-20 10:12:33' },
  { id: 2, domain: '*.demo.cn', status: 1, addtime: '2026-06-25 14:08:20' },
  { id: 1, domain: 'pay.newsite.net', status: 3, addtime: '2026-07-11 09:30:00' },
])

const activeCount = computed(() => list.value.filter((d) => d.status === 1).length)

// 添加域名弹窗
const addOpen = ref(false)
const newDomain = ref('')
function submitAdd() {
  addOpen.value = false
  newDomain.value = ''
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
              <td class="col-center"><Badge :variant="domainStatus[d.status].variant">{{ domainStatus[d.status].text }}</Badge></td>
              <td class="text-xs">{{ d.addtime }}</td>
              <td class="col-center">
                <Button variant="ghost" size="sm" class="text-destructive hover:text-destructive"><Trash2 class="size-4" /></Button>
              </td>
            </tr>
            <tr v-if="!list.length">
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
          <input v-model="newDomain" placeholder="如 shop.abc.com 或 *.abc.com" class="field-input flex-1" />
        </div>
        <p class="text-xs text-muted-foreground">请填写发起支付的网站域名，无需带 http(s):// 前缀。提交后进入审核。</p>
      </div>
      <template #footer>
        <Button variant="outline" size="sm" @click="addOpen = false">取消</Button>
        <Button size="sm" :disabled="!newDomain.trim()" @click="submitAdd"><Plus />提交审核</Button>
      </template>
    </Modal>
  </div>
</template>
