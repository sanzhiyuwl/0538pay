<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { Panel } from '@/components/ui'
import { fetchHelp } from '@/lib/api/merchantCenter'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()
const content = ref('')
const sitename = ref('0538Pay')
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    const res = await fetchHelp()
    content.value = res.content || ''
    sitename.value = res.sitename || '0538Pay'
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载使用说明失败')
  } finally {
    loading.value = false
  }
}
onMounted(load)

const hasContent = computed(() => content.value.trim().length > 0)
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="使用说明" subtitle="平台交易规则与到账说明">
      <!-- 后台已编辑：渲染富文本内容 -->
      <div v-if="hasContent" class="help-rich max-w-3xl text-sm leading-relaxed" v-html="content"></div>

      <!-- 后台未配置：内置默认文案（对齐 epay help.php 硬编码静态说明） -->
      <div v-else class="max-w-3xl space-y-6">
        <section>
          <h4 class="text-sm font-semibold">一、交易即时到账</h4>
          <p class="mt-2 text-sm leading-relaxed text-muted-foreground">
            {{ sitename }} 支持支付宝、微信、财付通、QQ钱包等多种支付方式，买家付款后款项实时到账至您的商户余额，无需等待。
          </p>
        </section>
        <section class="border-t border-border/60 pt-5">
          <h4 class="text-sm font-semibold">二、T+1 提现方案</h4>
          <p class="mt-2 text-sm leading-relaxed text-muted-foreground">
            商户余额支持手动申请提现，默认 T+1 到账（当日交易次日可提）。提现申请提交后由系统或人工审核后下发到您的收款账户。
          </p>
        </section>
        <section class="border-t border-border/60 pt-5">
          <h4 class="text-sm font-semibold">三、提现费率</h4>
          <div class="mt-3 overflow-x-auto">
            <table class="tbl w-full">
              <thead>
                <tr><th>项目</th><th>规则</th></tr>
              </thead>
              <tbody>
                <tr><td>起提金额</td><td>单笔 ≥ 10 元</td></tr>
                <tr><td>提现费率</td><td>0.5%，最高 25 元</td></tr>
                <tr><td>最低手续费</td><td>不足 1 元按 1 元收取</td></tr>
              </tbody>
            </table>
          </div>
        </section>
        <section class="border-t border-border/60 pt-5">
          <h4 class="text-sm font-semibold">四、结算方式</h4>
          <ul class="mt-2 space-y-1.5 text-sm leading-relaxed text-muted-foreground">
            <li>· 小额提现：官方企业支付宝 → 您的个人支付宝，实时到账。</li>
            <li>· 大额提现：官方对公账户 → 您的个人银行卡，1~2 个工作日到账。</li>
            <li>· 具体结算方式与限额以平台风控策略为准，如有疑问请联系客服。</li>
          </ul>
        </section>
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
