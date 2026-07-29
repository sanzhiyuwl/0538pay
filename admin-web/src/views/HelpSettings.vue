<script setup lang="ts">
import { reactive, ref, onMounted } from 'vue'
import { Save } from 'lucide-vue-next'
import { Panel, Button, RichEditor } from '@/components/ui'
import { fetchConfig, saveConfig } from '@/lib/api/config'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()

// 使用说明 / 服务协议 / 隐私政策 三块正文，同存 config help 分组。
// 使用说明→商户中心「使用说明」页；服务协议/隐私政策→登录/注册页弹窗与官网。空则前端用内置默认文案。
const form = reactive({ help_content: '', agreement_content: '', privacy_content: '' })
const loading = ref(false)
const saving = ref(false)

const tabs = [
  { key: 'help_content', label: '使用说明', placeholder: '输入使用说明正文，可设置标题/列表/加粗/插图…', hint: '展示在商户中心「使用说明」页。' },
  { key: 'agreement_content', label: '服务协议', placeholder: '输入服务协议正文…', hint: '登录/注册页「服务协议」弹窗与官网服务协议页读取。' },
  { key: 'privacy_content', label: '隐私政策', placeholder: '输入隐私政策正文…', hint: '登录/注册页「隐私政策」弹窗读取。' },
] as const
const active = ref<(typeof tabs)[number]['key']>('help_content')

async function load() {
  loading.value = true
  try {
    const kv = await fetchConfig('help')
    Object.assign(form, kv)
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}
onMounted(load)

async function save() {
  saving.value = true
  try {
    await saveConfig('help', { ...form })
    toast.success('已保存')
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="使用说明与协议" subtitle="编辑商户中心使用说明、服务协议、隐私政策，所见即所得；留空则显示内置默认文案">
      <template #actions>
        <Button size="sm" :disabled="saving || loading" @click="save"><Save />保存</Button>
      </template>
      <div class="space-y-3">
        <!-- Tab 切换 -->
        <div class="flex items-center gap-1 border-b border-border">
          <button
            v-for="t in tabs"
            :key="t.key"
            type="button"
            class="-mb-px border-b-2 px-4 py-2 text-sm font-medium transition-colors"
            :class="active === t.key
              ? 'border-primary text-primary'
              : 'border-transparent text-muted-foreground hover:text-foreground'"
            @click="active = t.key"
          >{{ t.label }}</button>
        </div>

        <!-- 编辑区（v-show 保留各 Tab 编辑器状态） -->
        <template v-for="t in tabs" :key="t.key">
          <div v-show="active === t.key" class="space-y-2">
            <RichEditor v-model="form[t.key]" :placeholder="t.placeholder" />
            <p class="text-xs text-muted-foreground">{{ t.hint }}</p>
          </div>
        </template>

        <p class="text-xs text-muted-foreground">
          三块内容一并保存。epay 原版为硬编码静态页，我方做成后台富文本可编辑（超出 epay）。
        </p>
      </div>
    </Panel>
  </div>
</template>
