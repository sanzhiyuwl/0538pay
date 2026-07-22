<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Mail, MailOpen, Megaphone } from 'lucide-vue-next'
import { Panel, Badge, Pagination } from '@/components/ui'
import { fetchMessages, readMessage, type MerchantMessage } from '@/lib/api/merchantCenter'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()

const list = ref<MerchantMessage[]>([])
const total = ref(0)
const unread = ref(0)
const loading = ref(false)
const page = ref(1)
const pageSize = 10
const expanded = ref<number | null>(null)

async function load() {
  loading.value = true
  try {
    const res = await fetchMessages({ page: page.value, pageSize })
    list.value = res.list || []
    total.value = res.total || 0
    unread.value = res.unread || 0
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载站内信失败')
  } finally {
    loading.value = false
  }
}
onMounted(load)

async function toggle(m: MerchantMessage) {
  if (expanded.value === m.id) {
    expanded.value = null
    return
  }
  expanded.value = m.id
  if (!m.is_read) {
    try {
      await readMessage(m.id)
      m.is_read = true
      unread.value = Math.max(0, unread.value - 1)
    } catch {
      /* 已读失败不打扰 */
    }
  }
}

function go(p: number) {
  page.value = p
  expanded.value = null
  load()
}
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="站内信" :subtitle="unread > 0 ? `${unread} 封未读` : '暂无未读'">
      <div class="divide-y divide-border/60">
        <div
          v-for="m in list"
          :key="m.id"
          class="cursor-pointer py-3 transition-colors hover:bg-muted/30"
          @click="toggle(m)"
        >
          <div class="flex items-center gap-3 px-1">
            <component
              :is="m.uid === 0 ? Megaphone : m.is_read ? MailOpen : Mail"
              class="size-4 shrink-0"
              :class="m.is_read ? 'text-muted-foreground' : 'text-primary'"
            />
            <span class="flex-1 truncate text-sm" :class="!m.is_read && 'font-medium'">{{ m.title }}</span>
            <Badge v-if="m.uid === 0" variant="muted">公告</Badge>
            <Badge v-if="!m.is_read" variant="success">未读</Badge>
            <span class="shrink-0 text-xs text-muted-foreground">{{ m.date }}</span>
          </div>
          <div v-if="expanded === m.id" class="mt-2 whitespace-pre-wrap px-8 text-sm leading-relaxed text-muted-foreground">
            {{ m.content }}
          </div>
        </div>
        <div v-if="!loading && !list.length" class="py-12 text-center dim">暂无站内信</div>
      </div>
      <div v-if="total > pageSize" class="mt-4 border-t border-border/60 pt-4">
        <Pagination :page="page" :page-count="Math.ceil(total / pageSize)" :total="total" :page-size="pageSize" @change="go" />
      </div>
    </Panel>
  </div>
</template>
