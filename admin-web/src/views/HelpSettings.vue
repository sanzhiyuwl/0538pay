<script setup lang="ts">
import { reactive, ref, onMounted } from 'vue'
import { Save, Eye } from 'lucide-vue-next'
import { Panel, Button } from '@/components/ui'
import { fetchConfig, saveConfig } from '@/lib/api/config'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()

// 使用说明正文（商户中心「使用说明」页读取）。空则商户端用内置默认文案。
const form = reactive({ help_content: '' })
const loading = ref(false)
const saving = ref(false)
const preview = ref(false)

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
    <Panel title="使用说明" subtitle="编辑商户中心「使用说明」页的内容，支持 HTML；留空则显示内置默认文案">
      <template #actions>
        <Button variant="outline" size="sm" @click="preview = !preview"><Eye />{{ preview ? '编辑' : '预览' }}</Button>
        <Button size="sm" :disabled="saving" @click="save"><Save />保存</Button>
      </template>
      <div class="max-w-3xl space-y-3">
        <textarea
          v-if="!preview"
          v-model="form.help_content"
          rows="18"
          placeholder="支持 HTML，例如：&#10;<h3>一、交易即时到账</h3>&#10;<p>买家付款后款项实时到账……</p>"
          class="field-input w-full font-mono text-[13px] leading-relaxed"
        ></textarea>
        <div class="help-rich min-h-[24rem] border border-border/60 bg-muted/20 p-4 text-sm leading-relaxed" v-else>
          <div v-if="form.help_content.trim()" v-html="form.help_content"></div>
          <p v-else class="text-muted-foreground">（内容为空，商户端将显示内置默认使用说明）</p>
        </div>
        <p class="text-xs text-muted-foreground">
          该内容展示在商户中心「使用说明」页。epay 原版为硬编码静态页，我方做成后台可编辑（超出 epay）。
        </p>
      </div>
    </Panel>
  </div>
</template>

<style scoped>
.help-rich :deep(h1),
.help-rich :deep(h2),
.help-rich :deep(h3),
.help-rich :deep(h4) {
  font-weight: 600;
  margin-top: 1.25rem;
  margin-bottom: 0.5rem;
}
.help-rich :deep(h3) { font-size: 0.95rem; }
.help-rich :deep(p) { margin: 0.5rem 0; color: var(--muted-foreground); }
.help-rich :deep(ul) { list-style: disc; padding-left: 1.25rem; margin: 0.5rem 0; }
.help-rich :deep(ol) { list-style: decimal; padding-left: 1.25rem; margin: 0.5rem 0; }
.help-rich :deep(a) { color: var(--primary); text-decoration: underline; }
</style>
