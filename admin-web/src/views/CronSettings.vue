<script setup lang="ts">
import { reactive, ref, computed, onMounted } from 'vue'
import { Save, Copy, Check } from 'lucide-vue-next'
import { Panel, Button } from '@/components/ui'
import { cronTasks } from '@/lib/mock/sysconfig'
import { fetchConfig, saveConfig } from '@/lib/api/config'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()
const cfg = reactive({ cronkey: '' })
const siteurl = 'https://epvia.com/'
const saving = ref(false)

onMounted(async () => {
  try {
    Object.assign(cfg, await fetchConfig('cron'))
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载失败')
  }
})

const taskUrls = computed(() =>
  cronTasks.map((t) => ({
    ...t,
    url: `${siteurl}${t.path}&key=${cfg.cronkey}`,
  })),
)

// 复制反馈
const copied = ref<string | null>(null)
async function copy(url: string) {
  try {
    await navigator.clipboard.writeText(url)
    copied.value = url
    setTimeout(() => (copied.value = null), 1500)
  } catch {
    // 原型：剪贴板不可用时静默
  }
}
async function save() {
  saving.value = true
  try {
    await saveConfig('cron', { ...cfg })
    toast.success('计划任务密钥已保存')
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="计划任务配置" subtitle="设置计划任务访问密钥，用于服务器定时访问下方任务链接">
      <div class="max-w-2xl space-y-3.5">
        <div class="row-field">
          <label class="lbl">访问密钥</label>
          <input v-model="cfg.cronkey" class="field-input flex-1 font-mono" />
        </div>
      </div>
      <div class="mt-5 border-t border-border/60 pt-4">
        <Button :disabled="saving" @click="save"><Save />保存设置</Button>
      </div>
    </Panel>

    <Panel title="计划任务列表" subtitle="将以下链接配置到服务器的 crontab / 宝塔计划任务中定时访问">
      <div class="space-y-3">
        <div v-for="t in taskUrls" :key="t.path" class="border border-border/70 p-3">
          <div class="flex items-start justify-between gap-3">
            <div class="min-w-0">
              <div class="text-sm font-medium">{{ t.title }}</div>
              <div class="mt-0.5 text-xs text-muted-foreground">{{ t.desc }}</div>
            </div>
            <Button variant="ghost" size="sm" class="shrink-0" @click="copy(t.url)">
              <component :is="copied === t.url ? Check : Copy" class="size-4" />
              {{ copied === t.url ? '已复制' : '复制' }}
            </Button>
          </div>
          <div class="mt-2 overflow-x-auto whitespace-nowrap rounded bg-muted/40 px-3 py-2 font-mono text-xs text-muted-foreground">
            {{ t.url }}
          </div>
        </div>
      </div>
      <p class="mt-4 border-t border-border/60 pt-4 text-xs text-muted-foreground">
        通知重试时间：1 分钟、3 分钟、20 分钟、1 小时、2 小时。建议每分钟执行一次通知重试任务。
      </p>
    </Panel>
  </div>
</template>
