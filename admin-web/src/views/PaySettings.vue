<script setup lang="ts">
import { reactive, ref, computed, onMounted } from 'vue'
import { Save } from 'lucide-vue-next'
import { Panel, Button, Select, Switch } from '@/components/ui'
import { fetchConfig, saveConfig } from '@/lib/api/config'
import { fetchPluginMeta } from '@/lib/api/channels'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()

const refundFeeOptions = [
  { value: '0', label: '平台承担（退款退回商户分成）' },
  { value: '1', label: '商户承担（全额退时扣实付）' },
]

const notifyOrdernameOptions = [
  { value: '0', label: '原样回传商品名称' },
  { value: '1', label: '统一回传为 product（隐藏真实商品名）' },
]
const switchOptions = [
  { value: '0', label: '关闭' },
  { value: '1', label: '开启' },
]

// 支付安全验证模式（对齐 epay checkPayVerifyOpen pay_verify）
const payVerifyOptions = [
  { value: '0', label: '关闭' },
  { value: '1', label: '智能验证（按成功率 / 同 IP 失败率触发）' },
  { value: '2', label: '指定商户开启' },
  { value: '3', label: '全部订单开启' },
]
// 验证方式（对齐 epay pay_verify_type）
const payVerifyTypeOptions = [
  { value: '0', label: '跳转确认页（无需第三方，可闭环）' },
  { value: '1', label: '极验隐形验证（需极验凭证）' },
  { value: '2', label: '极验滑块验证（需极验凭证）' },
]

// 键名对齐 epay set.php mod=pay
const form = reactive({
  pay_maxmoney: '50000',
  pay_minmoney: '0.01',
  // 内部收款通道：平台自身充值/进件/测试/聚合等内部收款统一走此通道（默认七相聚合），
  // 收银台按此通道支持的支付方式（微信/支付宝）分派。对齐后端 internal_pay_plugin。
  internal_pay_plugin: 'qixiang',
  blockname: '博彩|赌博|违禁|毒品|枪支',
  blockalert: '温馨提醒该商品禁止出售',
  refund_fee_type: '0',
  // 最低手续费兜底（手续费低于阈值时按 mincost 收取）
  payfee_lessthan: '0',
  payfee_mincost: '0',
  // 随机增减金额（防同额并单）：realmoney≥start 时 +random(min,max)
  pay_payaddstart: '0',
  pay_payaddmin: '0',
  pay_payaddmax: '0',
  // 回调商品名策略 + 强制填QQ + 同IP/同买家当日限单
  notifyordername: '0',
  forceqq: '0',
  pay_iplimit: '0',
  pay_userlimit: '0',
  // 支付安全验证（对齐 epay pay_verify 系列）
  pay_verify: '0',
  pay_verify_type: '0',
  pay_verify_check_uid: '',
  pay_verify_check_second: '0',
  pay_verify_check_count: '0',
  pay_verify_check_rate: '0',
  pay_verify_check_ip: '0',
})

// 聚合收款码全局开关（config group=onecode，对齐 epay set.php「开启聚合收款码」）。
// 开启后全站商户可用聚合收款码；关闭时仅后台单独授权(商户 open_code=1)的商户可用。
const onecode = reactive({ onecode: '0' })
const onecodeOn = computed({
  get: () => onecode.onecode === '1',
  set: (v: boolean) => { onecode.onecode = v ? '1' : '0' },
})

// 测试支付全局开关（config group=test，对齐 epay set.php「测试支付」）。
// 开启后商户可在「测试支付」页下真实测试单；test_pay_uid 指定收款商户(0=下到当前商户)。
const test = reactive({ test_open: '0', test_pay_uid: '0' })
const testOn = computed({
  get: () => test.test_open === '1',
  set: (v: boolean) => { test.test_open = v ? '1' : '0' },
})

// 商户保证金门槛（config group=deposit，对齐 epay set.php mod=pay「商户支付功能设置」）。
// 开启后商户余额低于最低保证金时拦截下单；冻结天数在提取保证金时校验近 N 天无订单。
const deposit = reactive({ user_deposit: '0', user_deposit_min: '0', user_deposit_day: '0' })
const depositOn = computed({
  get: () => deposit.user_deposit === '1',
  set: (v: boolean) => { deposit.user_deposit = v ? '1' : '0' },
})

const loading = ref(false)
const saving = ref(false)

// 内部收款通道候选：从已注册插件里取支持微信/支付宝的聚合/直连通道（默认七相置顶）。
const internalPluginOptions = ref<{ value: string; label: string }[]>([
  { value: 'qixiang', label: '七相聚合（自用通道）' },
])

async function loadInternalPluginOptions() {
  try {
    const metas = await fetchPluginMeta()
    const opts = metas
      .filter((m) => !m.delegate) // 排除已退役的形态包（仅作聚合门面委托目标）
      .filter((m) => (m.methods ?? []).some((x) => x === 'alipay' || x === 'wxpay'))
      .map((m) => ({
        value: m.key,
        label: m.key === 'qixiang' ? `${m.showname}（自用通道）` : m.showname,
      }))
    // 七相置顶，其余按原序
    opts.sort((a, b) => (a.value === 'qixiang' ? -1 : b.value === 'qixiang' ? 1 : 0))
    if (opts.length) internalPluginOptions.value = opts
  } catch {
    // 拉取失败保留默认七相选项，不阻断设置页加载
  }
}

async function load() {
  loading.value = true
  try {
    const [payKv, onecodeKv, testKv, depositKv] = await Promise.all([
      fetchConfig('pay'), fetchConfig('onecode'), fetchConfig('test'), fetchConfig('deposit'),
    ])
    Object.assign(form, payKv)
    Object.assign(onecode, onecodeKv)
    Object.assign(test, testKv)
    Object.assign(deposit, depositKv)
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载支付设置失败')
  } finally {
    loading.value = false
  }
}
onMounted(() => {
  load()
  loadInternalPluginOptions()
})

async function save() {
  saving.value = true
  try {
    await Promise.all([
      saveConfig('pay', { ...form }),
      saveConfig('onecode', { ...onecode }),
      saveConfig('test', { ...test }),
      saveConfig('deposit', {
        user_deposit: deposit.user_deposit,
        user_deposit_min: deposit.user_deposit_min,
        user_deposit_day: deposit.user_deposit_day,
      }),
    ])
    toast.success('支付设置已保存')
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="支付设置" subtitle="全站支付金额限制、商品屏蔽词与退款手续费策略">
      <div class="max-w-2xl space-y-3.5">
        <div class="row-field">
          <label class="lbl">最大支付金额</label>
          <div class="flex flex-1 items-center gap-2">
            <input v-model="form.pay_maxmoney" class="field-input w-40" /><span class="text-sm text-muted-foreground">元</span>
          </div>
        </div>
        <div class="row-field">
          <label class="lbl">最小支付金额</label>
          <div class="flex flex-1 items-center gap-2">
            <input v-model="form.pay_minmoney" class="field-input w-40" /><span class="text-sm text-muted-foreground">元</span>
          </div>
        </div>
        <div class="row-field">
          <label class="lbl">屏蔽关键词</label>
          <input v-model="form.blockname" placeholder="多个用竖线 | 分隔" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">屏蔽提示语</label>
          <input v-model="form.blockalert" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">退款手续费</label>
          <Select v-model="form.refund_fee_type" :options="refundFeeOptions" class="flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">内部收款通道</label>
          <Select v-model="form.internal_pay_plugin" :options="internalPluginOptions" class="flex-1" />
        </div>
        <p class="rounded bg-muted/40 px-3 py-2 text-xs text-muted-foreground">
          屏蔽关键词命中商品名时拦截下单并记风控。退款手续费策略决定退款时从商户扣分成还是扣实付全额。
        </p>
        <p class="rounded bg-warning/[0.08] px-3 py-2 text-xs text-warning">
          内部收款通道：平台自身的余额充值、进件工本费、测试支付、聚合收款等所有内部扫码收款，统一走此通道（默认七相聚合），
          收银台按该通道支持的支付方式（微信/支付宝）分派。需先在「支付通道」里配置并开启对应通道。
        </p>
      </div>
    </Panel>

    <!-- 手续费兜底 + 金额随机微调 -->
    <Panel title="手续费与金额策略" subtitle="最低手续费兜底与随机增减金额（防同额并单）">
      <div class="max-w-2xl space-y-3.5">
        <div class="row-field">
          <label class="lbl">最低手续费阈值</label>
          <div class="flex flex-1 items-center gap-2">
            <input v-model="form.payfee_lessthan" class="field-input w-40" /><span class="text-sm text-muted-foreground">元（0=关闭兜底）</span>
          </div>
        </div>
        <div class="row-field">
          <label class="lbl">最低手续费金额</label>
          <div class="flex flex-1 items-center gap-2">
            <input v-model="form.payfee_mincost" class="field-input w-40" /><span class="text-sm text-muted-foreground">元</span>
          </div>
        </div>
        <div class="row-field">
          <label class="lbl">随机微调起始额</label>
          <div class="flex flex-1 items-center gap-2">
            <input v-model="form.pay_payaddstart" class="field-input w-40" /><span class="text-sm text-muted-foreground">元（0=关闭；实付≥此值才微调）</span>
          </div>
        </div>
        <div class="row-field">
          <label class="lbl">微调最小/最大</label>
          <div class="flex flex-1 items-center gap-2">
            <input v-model="form.pay_payaddmin" class="field-input w-28" /><span class="text-sm text-muted-foreground">~</span>
            <input v-model="form.pay_payaddmax" class="field-input w-28" /><span class="text-sm text-muted-foreground">元</span>
          </div>
        </div>
        <p class="rounded bg-muted/40 px-3 py-2 text-xs text-muted-foreground">
          手续费低于阈值时按「最低手续费金额」收取。随机微调在实付金额上叠加 min~max 的随机小数，避免相同金额订单并单串单。
        </p>
      </div>
    </Panel>

    <!-- 回调策略 + 下单风控限制 -->
    <Panel title="回调与下单限制" subtitle="回调商品名策略、强制填 QQ、同 IP/同买家当日限单">
      <div class="max-w-2xl space-y-3.5">
        <div class="row-field">
          <label class="lbl">回调商品名</label>
          <Select v-model="form.notifyordername" :options="notifyOrdernameOptions" class="flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">强制填写 QQ</label>
          <Select v-model="form.forceqq" :options="switchOptions" class="flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">同 IP 当日限单</label>
          <div class="flex flex-1 items-center gap-2">
            <input v-model="form.pay_iplimit" class="field-input w-40" /><span class="text-sm text-muted-foreground">笔（0=不限）</span>
          </div>
        </div>
        <div class="row-field">
          <label class="lbl">同买家当日限单</label>
          <div class="flex flex-1 items-center gap-2">
            <input v-model="form.pay_userlimit" class="field-input w-40" /><span class="text-sm text-muted-foreground">笔（0=不限；按 openid/buyer）</span>
          </div>
        </div>
        <p class="rounded bg-muted/40 px-3 py-2 text-xs text-muted-foreground">
          回调商品名设为 product 可对下游隐藏真实商品名。同买家限单在回调阶段按支付账号(openid/buyer)统计当日成功单数。
        </p>
      </div>
    </Panel>

    <!-- 支付安全验证 -->
    <Panel title="支付安全验证" subtitle="下单时插入人机验证，拦截刷单/攻击（对齐 epay 支付安全验证）">
      <div class="max-w-2xl space-y-3.5">
        <div class="row-field">
          <label class="lbl">验证模式</label>
          <Select v-model="form.pay_verify" :options="payVerifyOptions" class="flex-1" />
        </div>
        <div v-if="form.pay_verify !== '0'" class="row-field">
          <label class="lbl">验证方式</label>
          <Select v-model="form.pay_verify_type" :options="payVerifyTypeOptions" class="flex-1" />
        </div>
        <div v-if="form.pay_verify === '2'" class="row-field">
          <label class="lbl">指定商户 UID</label>
          <input v-model="form.pay_verify_check_uid" placeholder="多个用竖线 | 分隔" class="field-input flex-1" />
        </div>
        <template v-if="form.pay_verify === '1'">
          <div class="row-field">
            <label class="lbl">成功率统计窗口</label>
            <div class="flex flex-1 items-center gap-2">
              <input v-model="form.pay_verify_check_second" class="field-input w-40" /><span class="text-sm text-muted-foreground">秒</span>
            </div>
          </div>
          <div class="row-field">
            <label class="lbl">窗口最少订单数</label>
            <div class="flex flex-1 items-center gap-2">
              <input v-model="form.pay_verify_check_count" class="field-input w-40" /><span class="text-sm text-muted-foreground">笔（达到才判定，避免小样本误伤）</span>
            </div>
          </div>
          <div class="row-field">
            <label class="lbl">成功率低于</label>
            <div class="flex flex-1 items-center gap-2">
              <input v-model="form.pay_verify_check_rate" class="field-input w-40" /><span class="text-sm text-muted-foreground">% 触发验证</span>
            </div>
          </div>
          <div class="row-field">
            <label class="lbl">同 IP 失败单数</label>
            <div class="flex flex-1 items-center gap-2">
              <input v-model="form.pay_verify_check_ip" class="field-input w-40" /><span class="text-sm text-muted-foreground">笔（近 1 小时最近 N 单全未支付则触发；0=不启用）</span>
            </div>
          </div>
        </template>
        <p class="rounded bg-muted/40 px-3 py-2 text-xs text-muted-foreground">
          命中验证策略时，跳转支付会先要求通过安全验证再继续下单；API（create）下单会提示改用跳转支付接口。
          验证方式选「跳转确认页」无需第三方即可闭环；极验方式需在「注册登录鉴权」页配置极验凭证。
        </p>
      </div>
    </Panel>

    <!-- 聚合收款码全局开关 -->
    <Panel title="聚合收款码" subtitle="全站聚合收款码开关，商户可生成固定收款二维码，客户扫码自选支付方式并输入金额付款">
      <div class="max-w-2xl space-y-3.5">
        <div class="row-field">
          <label class="lbl">开启聚合收款码</label>
          <Switch v-model="onecodeOn" />
        </div>
        <p class="rounded bg-muted/40 px-3 py-2 text-xs text-muted-foreground">
          开启后全站商户均可在「聚合收款码」页生成收款码。关闭时仅在商户管理中单独授权（开启聚合收款）的商户可用。
        </p>
      </div>
    </Panel>

    <!-- 测试支付全局开关 -->
    <Panel title="测试支付" subtitle="全站测试支付开关，商户可在「测试支付」页下真实测试订单验证收款链路">
      <div class="max-w-2xl space-y-3.5">
        <div class="row-field">
          <label class="lbl">开启测试支付</label>
          <Switch v-model="testOn" />
        </div>
        <div v-if="testOn" class="row-field">
          <label class="lbl">测试收款商户</label>
          <div class="flex flex-1 items-center gap-2">
            <input v-model="test.test_pay_uid" placeholder="商户 UID" class="field-input w-40" /><span class="text-sm text-muted-foreground">填 0 则下到发起测试的商户本人</span>
          </div>
        </div>
        <p class="rounded bg-muted/40 px-3 py-2 text-xs text-muted-foreground">
          开启后商户可在「测试支付」页下真实订单验证收单链路。测试收款商户 UID 指定款项落到哪个商户，填 0 则下到发起测试的商户本人。
        </p>
      </div>
    </Panel>

    <!-- 商户保证金门槛（对齐 epay mod=pay 商户支付功能设置） -->
    <Panel title="商户保证金" subtitle="商户余额低于保证金门槛时拦截下单，提取保证金时校验冻结天数">
      <div class="max-w-2xl space-y-3.5">
        <div class="row-field">
          <label class="lbl">开启保证金门槛</label>
          <Switch v-model="depositOn" />
        </div>
        <template v-if="depositOn">
          <div class="row-field">
            <label class="lbl">最低保证金</label>
            <div class="flex flex-1 items-center gap-2">
              <input v-model="deposit.user_deposit_min" class="field-input w-40" /><span class="text-sm text-muted-foreground">元（留空/0=不限制最低充值）</span>
            </div>
          </div>
          <div class="row-field">
            <label class="lbl">提取冻结天数</label>
            <div class="flex flex-1 items-center gap-2">
              <input v-model="deposit.user_deposit_day" class="field-input w-40" /><span class="text-sm text-muted-foreground">天（0=不校验；提取时近 N 天需无订单、无投诉）</span>
            </div>
          </div>
        </template>
        <p class="rounded bg-muted/40 px-3 py-2 text-xs text-muted-foreground">
          开启后商户需充足额保证金才能发起支付。冻结天数用于提取保证金时校验最近 N 天内无成功订单、无投诉，防止套现跑路。
        </p>
      </div>
      <div class="mt-5 border-t border-border/60 pt-4">
        <Button :disabled="saving || loading" @click="save"><Save />保存设置</Button>
      </div>
    </Panel>
  </div>
</template>
