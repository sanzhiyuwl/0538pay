<script setup lang="ts">
import { reactive, ref, onMounted } from 'vue'
import { Save } from 'lucide-vue-next'
import { Panel, Button, RichEditor } from '@/components/ui'
import { fetchConfig, saveConfig } from '@/lib/api/config'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()

// 使用说明正文（商户中心「使用说明」页读取）。空则商户端用内置默认文案。
const form = reactive({ help_content: '' })
const loading = ref(false)
const saving = ref(false)

async function load() {
  loading.value = true
  try {
    const kv = await fetchConfig('help')
    Object.assign(form, kv)
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载使用说明失败')
  } finally {
    loading.value = false
  }
}
onMounted(load)

async function save() {
  saving.value = true
  try {
    await saveConfig('help', { ...form })
    toast.success('使用说明已保存')
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="使用说明" subtitle="编辑商户中心「使用说明」页的内容，所见即所得；留空则显示内置默认文案">
      <template #actions>
        <Button size="sm" :disabled="saving || loading" @click="save"><Save />保存</Button>
      </template>
      <div class="space-y-3">
        <RichEditor v-model="form.help_content" placeholder="输入使用说明正文，可设置标题/列表/加粗/插图…" />
        <p class="text-xs text-muted-foreground">
          该内容展示在商户中心「使用说明」页。epay 原版为硬编码静态页，我方做成后台富文本可编辑（超出 epay）。
        </p>
      </div>
    </Panel>
  </div>
</template>
