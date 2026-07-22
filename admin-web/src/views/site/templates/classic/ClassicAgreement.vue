<script setup lang="ts">
import { onMounted, computed } from 'vue'
import { useSiteStore } from '@/stores/site'

// 服务协议：站点名从网站设置 CMS 读取（后台可改，官网实时联动）。
const site = useSiteStore()
onMounted(() => site.hydrate())
const brand = computed(() => site.config.sitename || '0538Pay')

const sections = computed(() => [
  {
    title: '一、服务说明',
    body: [
      `${brand.value}（以下简称"本平台"）为商户提供聚合支付技术服务，包括但不限于多渠道收款、订单管理、结算提现、开放 API 等功能。`,
      '商户在使用本平台服务前，应完整阅读并同意本协议全部条款。一旦注册或使用，即视为已充分理解并接受本协议。',
    ],
  },
  {
    title: '二、账户与实名',
    body: [
      '商户须提供真实、准确、完整的注册信息，并按平台要求完成个人或企业实名认证。',
      '商户应妥善保管账户、密钥与登录凭证，因保管不善造成的损失由商户自行承担。',
    ],
  },
  {
    title: '三、费率与结算',
    body: [
      '本平台按约定费率收取交易手续费，费率以商户所属用户组/会员等级为准，可在费率方案页查看。',
      '结算方式支持 T+1 自动结算与手动提现，具体到账时间以实际结算通道为准。提现手续费按平台规则收取。',
    ],
  },
  {
    title: '四、商户义务',
    body: [
      '商户不得利用本平台从事任何违法违规交易，包括但不限于赌博、诈骗、洗钱、销售违禁品等。',
      '商户应确保交易真实合法，配合平台的风控核查。平台有权对异常账户采取限制收款、冻结结算等措施。',
    ],
  },
  {
    title: '五、风控与安全',
    body: [
      '本平台通过实时风控引擎、黑名单、限额策略等手段保障资金与账户安全。',
      '对于触发风控规则的交易或账户，平台有权暂停服务并要求商户配合核实。',
    ],
  },
  {
    title: '六、责任限制',
    body: [
      '因不可抗力、第三方支付渠道故障、政策调整等非平台原因造成的服务中断或损失，平台不承担赔偿责任。',
      '本平台仅提供支付技术服务，不介入商户与其客户之间的交易纠纷。',
    ],
  },
  {
    title: '七、协议变更',
    body: [
      '本平台有权根据业务需要修订本协议，修订后的协议将在平台公示。商户继续使用即视为接受变更。',
    ],
  },
])
</script>

<template>
  <div class="site-surface min-h-[70vh] border-b border-border">
    <div class="mx-auto max-w-3xl px-4 py-16 lg:px-6">
      <h1 class="text-3xl font-bold tracking-tight">服务协议</h1>
      <p class="mt-2 text-sm text-muted-foreground">最后更新：2026-07-14</p>

      <div class="mt-10 space-y-8">
        <section v-for="s in sections" :key="s.title">
          <h2 class="text-lg font-semibold">{{ s.title }}</h2>
          <p v-for="(p, i) in s.body" :key="i" class="mt-2 text-sm leading-relaxed text-muted-foreground">{{ p }}</p>
        </section>
      </div>

      <p class="mt-12 border-t border-border pt-6 text-xs text-muted-foreground">
        本协议为原型示例文本，不构成正式法律文件。正式上线前请由法务审定。
      </p>
    </div>
  </div>
</template>
