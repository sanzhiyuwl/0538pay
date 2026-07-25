<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { BookOpen, Zap, CalendarClock, ShieldCheck } from 'lucide-vue-next'
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

// 顶部亮点卡：概括核心规则，快速传达关键信息
const highlights = [
  { icon: Zap, title: '交易即时到账', desc: '付款成功实时入账，随时查看' },
  { icon: CalendarClock, title: 'T+1 提现', desc: '次日到账，费率 0.5% 封顶 25 元' },
  { icon: ShieldCheck, title: '安全合规', desc: '实名收款，密钥加密，资金有保障' },
]
</script>

<template>
  <div class="space-y-2.5">
    <!-- 顶部 Hero：图标 + 标题 + 亮点卡 -->
    <div class="border border-border/60 bg-gradient-to-br from-primary/[0.06] to-transparent p-6">
      <div class="flex items-center gap-3.5">
        <div class="flex size-12 shrink-0 items-center justify-center bg-primary/[0.1] text-primary">
          <BookOpen class="size-6" />
        </div>
        <div>
          <h2 class="text-lg font-semibold">使用说明</h2>
          <p class="mt-0.5 text-sm text-muted-foreground">{{ sitename }} 平台交易规则、到账与提现说明</p>
        </div>
      </div>
      <div class="mt-5 grid grid-cols-1 gap-3 sm:grid-cols-3">
        <div v-for="h in highlights" :key="h.title" class="flex items-start gap-3 border border-border/60 bg-background/60 px-4 py-3">
          <component :is="h.icon" class="mt-0.5 size-5 shrink-0 text-primary" />
          <div class="min-w-0">
            <div class="text-sm font-medium">{{ h.title }}</div>
            <div class="mt-0.5 text-xs leading-relaxed text-muted-foreground">{{ h.desc }}</div>
          </div>
        </div>
      </div>
    </div>

    <Panel title="详细规则" subtitle="完整的交易、提现、结算与安全说明">
      <!-- 加载骨架 -->
      <div v-if="loading" class="space-y-3">
        <div class="h-5 w-40 animate-pulse bg-muted"></div>
        <div class="h-4 w-full animate-pulse bg-muted/70"></div>
        <div class="h-4 w-5/6 animate-pulse bg-muted/70"></div>
        <div class="mt-4 h-5 w-32 animate-pulse bg-muted"></div>
        <div class="h-4 w-full animate-pulse bg-muted/70"></div>
      </div>

      <!-- 后台已编辑：渲染富文本内容 -->
      <div v-else-if="hasContent" class="help-rich" v-html="content"></div>

      <!-- 后台未配置：内置默认文案（对齐 epay help.php 硬编码静态说明） -->
      <div v-else class="space-y-6">
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
/* 展示态与后台 RichEditor 编辑区(.tiptap)层级样式保持一致：H1-H4 左侧竖线/圆点、列表、引用等 */
.help-rich {
  font-size: 14px;
  line-height: 1.75;
  color: var(--foreground);
}
.help-rich :deep(p) {
  margin: 0.5em 0;
  color: var(--muted-foreground);
}
.help-rich :deep(h1) {
  margin: 1.2em 0 0.7em;
  padding: 0.35em 0 0.35em 0.7em;
  border-left: 4px solid var(--primary);
  background: var(--muted);
  font-size: 1.6em;
  font-weight: 700;
  line-height: 1.4;
  color: var(--foreground);
}
.help-rich :deep(h2) {
  margin: 1.1em 0 0.6em;
  padding: 0.25em 0 0.25em 0.65em;
  border-left: 4px solid var(--primary);
  background: color-mix(in oklch, var(--muted) 55%, transparent);
  font-size: 1.35em;
  font-weight: 700;
  line-height: 1.4;
  color: var(--foreground);
}
.help-rich :deep(h3) {
  margin: 1em 0 0.5em;
  padding-left: 0.6em;
  border-left: 3px solid var(--primary);
  font-size: 1.18em;
  font-weight: 600;
  line-height: 1.4;
  color: var(--foreground);
}
.help-rich :deep(h4) {
  position: relative;
  margin: 0.9em 0 0.45em;
  padding-left: 0.85em;
  font-size: 1.05em;
  font-weight: 600;
  color: var(--foreground);
}
.help-rich :deep(h4)::before {
  content: '';
  position: absolute;
  left: 0;
  top: 0.62em;
  width: 5px;
  height: 5px;
  border-radius: 9999px;
  background: var(--primary);
}
.help-rich :deep(ul),
.help-rich :deep(ol) {
  margin: 0.5em 0;
  padding-left: 1.5em;
}
.help-rich :deep(ul) { list-style: disc; }
.help-rich :deep(ol) { list-style: decimal; }
.help-rich :deep(li) { margin: 0.25em 0; }
.help-rich :deep(li p) { margin: 0; }
.help-rich :deep(strong) {
  font-weight: 600;
  color: var(--foreground);
}
.help-rich :deep(blockquote) {
  margin: 0.75em 0;
  border-left: 3px solid var(--primary);
  padding-left: 1em;
  color: var(--muted-foreground);
}
.help-rich :deep(a) {
  color: var(--primary);
  text-decoration: underline;
}
.help-rich :deep(img) {
  max-width: 100%;
}
.help-rich :deep(hr) {
  margin: 1.25em 0;
  border: none;
  border-top: 1px solid var(--border);
}
</style>
