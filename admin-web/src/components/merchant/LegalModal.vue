<script setup lang="ts">
import { ref, computed } from 'vue'
import { Modal } from '@/components/ui'
import { fetchLegal } from '@/lib/api/merchantAuth'

/**
 * 服务协议 / 隐私政策弹窗。登录页 / 注册页共用。
 * 内容后台「使用说明与协议」可编辑，留空则展示内置默认文案。
 * 用法：<LegalModal ref="legal" /> + legal.value?.open('agreement' | 'privacy')
 */
const open = ref(false)
const kind = ref<'agreement' | 'privacy'>('agreement')
const title = computed(() => (kind.value === 'agreement' ? '服务协议' : '隐私政策'))
const cache = ref<{ agreement: string; privacy: string } | null>(null)
const loading = ref(false)

const body = computed(() => {
  const c = cache.value?.[kind.value]?.trim()
  return c || defaultLegal[kind.value]
})

async function show(k: 'agreement' | 'privacy') {
  kind.value = k
  open.value = true
  if (cache.value) return
  loading.value = true
  try {
    const r = await fetchLegal()
    cache.value = { agreement: r.agreement, privacy: r.privacy }
  } catch {
    cache.value = { agreement: '', privacy: '' } // 拉取失败退回内置默认文案
  } finally {
    loading.value = false
  }
}

defineExpose({ open: show })

// 内置默认文案（后台未配置时展示）
const defaultLegal = {
  agreement:
    '<p>欢迎使用本平台聚合支付技术服务。在使用前，请完整阅读并同意本服务协议全部条款。一旦注册或使用，即视为您已充分理解并接受本协议。</p>' +
    '<p>商户须提供真实、准确、完整的注册信息，妥善保管账户与密钥；不得利用本平台从事任何违法违规交易。平台按约定费率收取交易手续费，并有权对异常账户采取风控措施。</p>' +
    '<p class="tip">本文本为默认示例，正式条款请在后台「使用说明与协议」中编辑，并由法务审定。</p>',
  privacy:
    '<p>本平台重视并保护您的个人信息。我们仅在为您提供支付技术服务所必需的范围内收集、使用您的信息，包括注册信息、实名信息、交易记录等。</p>' +
    '<p>未经您同意，我们不会向第三方披露您的个人信息，法律法规另有规定或为完成交易结算所必需的除外。我们采取加密存储、访问控制等安全措施保障信息安全。</p>' +
    '<p class="tip">本文本为默认示例，正式条款请在后台「使用说明与协议」中编辑，并由法务审定。</p>',
}
</script>

<template>
  <Modal v-model="open" :title="title" width="max-w-2xl">
    <div v-if="loading" class="py-8 text-center text-sm text-muted-foreground">加载中…</div>
    <div v-else class="legal-body max-h-[60vh] overflow-y-auto" v-html="body" />
  </Modal>
</template>

<style scoped>
.legal-body {
  font-size: 14px;
  line-height: 1.75;
  color: var(--foreground);
}
.legal-body :deep(p) {
  margin: 0 0 0.75rem;
}
.legal-body :deep(h1),
.legal-body :deep(h2),
.legal-body :deep(h3) {
  margin: 1rem 0 0.5rem;
  font-weight: 600;
}
.legal-body :deep(ul),
.legal-body :deep(ol) {
  margin: 0 0 0.75rem;
  padding-left: 1.5rem;
}
.legal-body :deep(.tip) {
  margin-top: 1rem;
  padding-top: 0.75rem;
  border-top: 1px solid var(--border);
  font-size: 12px;
  color: var(--muted-foreground);
}
</style>
