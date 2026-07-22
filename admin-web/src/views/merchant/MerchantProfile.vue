<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { Save, ShieldCheck, Link2, Unlink, QrCode } from 'lucide-vue-next'
import { Panel, Button, Select, Switch } from '@/components/ui'
import {
  settleConfig,
  stypeOptions,
  contactConfig,
  msgConfig,
  noticeChannelOptions,
  modeConfig,
  modeOptions,
  bindConfig,
} from '@/lib/mock/merchant/profile'
import { fetchMerchantInfo } from '@/lib/api/merchantAuth'
import { updateProfile, fetchMsgConfig, saveMsgConfig, rebindContact } from '@/lib/api/merchantCenter'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'
import { useMerchantAuthStore } from '@/stores/merchantAuth'

const toast = useToast()
const merchantAuth = useMerchantAuthStore()

const settle = reactive({ ...settleConfig })
const contact = reactive({ ...contactConfig })
const msg = reactive({ ...msgConfig })
const mode = reactive({ ...modeConfig })
const binds = reactive({ ...bindConfig }) // 第三方绑定：绑定跳转需真实 OAuth 凭证

// 拉当前商户资料填充（收款账号/联系方式/扣费模式为真数据）
onMounted(async () => {
  try {
    const info = await fetchMerchantInfo()
    settle.stype = String(info.settle_id || 1)
    settle.account = info.account
    settle.username = info.username
    contact.phone = info.phone
    contact.email = info.email
    contact.qq = info.qq
    contact.url = info.url
    contact.keylogin = info.keylogin === 1
    contact.refund = info.refund === 1
    contact.transfer = info.transfer === 1
    contact.remain_money = info.remain_money || '0.00'
    mode.mode = String(info.mode || 0)
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '资料加载失败')
  }
  // D-3 加载消息提醒配置
  try {
    const { msgconfig } = await fetchMsgConfig()
    const c = JSON.parse(msgconfig || '{}')
    msg.notice_order = !!c.order
    msg.notice_settle = !!c.settle
    msg.notice_login = !!c.login
    msg.notice_balance = !!c.balance
    if (c.balance_threshold) msg.notice_balance_money = String(c.balance_threshold)
  } catch { /* 用默认 */ }
})

// D-3 保存消息提醒
const savingMsg = ref(false)
async function saveMsg() {
  if (savingMsg.value) return
  savingMsg.value = true
  try {
    const cfg = JSON.stringify({
      order: msg.notice_order ? 1 : 0,
      settle: msg.notice_settle ? 1 : 0,
      login: msg.notice_login ? 1 : 0,
      balance: msg.notice_balance ? 1 : 0,
      balance_threshold: msg.notice_balance ? msg.notice_balance_money : '',
    })
    await saveMsgConfig(cfg)
    toast.success('消息提醒设置已保存')
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '保存失败')
  } finally {
    savingMsg.value = false
  }
}

// D-3 换绑手机（登录密码二次确认）
const rebind = reactive({ open: false, value: '', password: '', busy: false })
async function submitRebind() {
  if (rebind.busy) return
  if (!rebind.value.trim()) return toast.error('请输入新手机号')
  rebind.busy = true
  try {
    await rebindContact('phone', rebind.value.trim(), rebind.password)
    contact.phone = rebind.value.trim()
    rebind.open = false
    rebind.value = ''
    rebind.password = ''
    toast.success('手机号换绑成功')
    await merchantAuth.refreshInfo()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '换绑失败')
  } finally {
    rebind.busy = false
  }
}

// 换绑邮箱（登录密码二次确认，复用后端 Rebind field=email 的唯一性/格式校验）
const rebindEmail = reactive({ open: false, value: '', password: '', busy: false })
async function submitRebindEmail() {
  if (rebindEmail.busy) return
  if (!rebindEmail.value.trim()) return toast.error('请输入新邮箱')
  rebindEmail.busy = true
  try {
    await rebindContact('email', rebindEmail.value.trim(), rebindEmail.password)
    contact.email = rebindEmail.value.trim()
    rebindEmail.open = false
    rebindEmail.value = ''
    rebindEmail.password = ''
    toast.success('邮箱换绑成功')
    await merchantAuth.refreshInfo()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '换绑失败')
  } finally {
    rebindEmail.busy = false
  }
}

// 收款账号：微信结算时账号标签变化
const accountLabel = computed(() => {
  switch (settle.stype) {
    case '1': return '支付宝账号'
    case '2': return '微信 OpenId / 微信号'
    case '3': return 'QQ 号码'
    case '4': return '银行卡号'
    default: return '收款账号'
  }
})

const saving = ref(false)
// 保存资料（收款账号 + 联系方式 + 扣费模式，一次提交后端已有字段）
async function save() {
  if (saving.value) return
  saving.value = true
  try {
    await updateProfile({
      settle_id: Number(settle.stype),
      account: settle.account,
      username: settle.username,
      email: contact.email,
      qq: contact.qq,
      url: contact.url,
      mode: Number(mode.mode),
      keylogin: contact.keylogin ? 1 : 0,
      refund: contact.refund ? 1 : 0,
      transfer: contact.transfer ? 1 : 0,
      remain_money: contact.remain_money,
    })
    toast.success('资料已保存')
    await merchantAuth.refreshInfo() // 同步顶栏/工作台
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}
const socials = [
  { key: 'qq', label: 'QQ', color: 'text-[#12b7f5]' },
  { key: 'wx', label: '微信', color: 'text-[#07c160]' },
  { key: 'alipay', label: '支付宝', color: 'text-[#1677ff]' },
] as const
function toggleBind(key: 'qq' | 'wx' | 'alipay') {
  binds[key].bound = !binds[key].bound
}
</script>

<template>
  <div class="space-y-2.5">
    <!-- 收款账号设置 -->
    <Panel title="收款账号设置" subtitle="结算/提现的收款账户，请确保信息准确">
      <div class="max-w-2xl space-y-3.5">
        <div class="row-field">
          <label class="lbl">结算方式</label>
          <Select v-model="settle.stype" :options="stypeOptions" class="flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">{{ accountLabel }}</label>
          <div class="flex flex-1 items-center gap-2">
            <input v-model="settle.account" class="field-input flex-1" />
            <Button v-if="settle.stype === '2'" variant="outline" size="sm"><QrCode class="size-4" />获取 OpenId</Button>
          </div>
        </div>
        <div class="row-field">
          <label class="lbl">真实姓名</label>
          <input v-model="settle.username" class="field-input flex-1" />
        </div>
      </div>
      <div class="mt-5 border-t border-border/60 pt-4"><Button @click="save"><Save />保存收款账号</Button></div>
    </Panel>

    <!-- 联系方式设置 -->
    <Panel title="联系方式与接口设置">
      <div class="max-w-2xl space-y-3.5">
        <div class="row-field">
          <label class="lbl">手机号码</label>
          <div class="flex flex-1 items-center gap-2">
            <input :value="contact.phone || '未绑定'" readonly class="field-input flex-1 bg-muted/40" />
            <Button variant="outline" size="sm" @click="rebind.open = !rebind.open"><ShieldCheck class="size-4" />换绑手机</Button>
          </div>
        </div>
        <div v-if="rebind.open" class="row-field">
          <label class="lbl">新手机号</label>
          <div class="flex flex-1 flex-wrap items-center gap-2">
            <input v-model="rebind.value" placeholder="新手机号" class="field-input w-40" />
            <input v-model="rebind.password" type="password" placeholder="登录密码确认" class="field-input w-40" />
            <Button size="sm" :disabled="rebind.busy" @click="submitRebind">确认换绑</Button>
            <Button variant="outline" size="sm" @click="rebind.open = false">取消</Button>
          </div>
        </div>
        <div class="row-field">
          <label class="lbl">邮箱</label>
          <div class="flex flex-1 items-center gap-2">
            <input :value="contact.email || '未绑定'" readonly class="field-input flex-1 bg-muted/40" />
            <Button variant="outline" size="sm" @click="rebindEmail.open = !rebindEmail.open"><ShieldCheck class="size-4" />换绑邮箱</Button>
          </div>
        </div>
        <div v-if="rebindEmail.open" class="row-field">
          <label class="lbl">新邮箱</label>
          <div class="flex flex-1 flex-wrap items-center gap-2">
            <input v-model="rebindEmail.value" placeholder="新邮箱" class="field-input w-52" />
            <input v-model="rebindEmail.password" type="password" placeholder="登录密码确认" class="field-input w-40" />
            <Button size="sm" :disabled="rebindEmail.busy" @click="submitRebindEmail">确认换绑</Button>
            <Button variant="outline" size="sm" @click="rebindEmail.open = false">取消</Button>
          </div>
        </div>
        <div class="row-field">
          <label class="lbl">QQ</label>
          <input v-model="contact.qq" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">网站域名</label>
          <input v-model="contact.url" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">预留余额</label>
          <div class="flex flex-1 items-center gap-2">
            <input v-model="contact.remain_money" class="field-input w-40" />
            <span class="text-xs text-muted-foreground">自动结算模式下，此金额不参与每日自动结算</span>
          </div>
        </div>
        <div class="row-switch"><span>开启密钥登录</span><Switch v-model="contact.keylogin" /></div>
        <div class="row-switch"><span>订单退款 API 接口</span><Switch v-model="contact.refund" /></div>
        <div class="row-switch"><span>代付 API 接口</span><Switch v-model="contact.transfer" /></div>
      </div>
      <div class="mt-5 border-t border-border/60 pt-4"><Button @click="save"><Save />保存联系方式</Button></div>
    </Panel>

    <!-- 消息提醒接收设置 -->
    <Panel title="消息提醒设置" subtitle="选择接收哪些通知，以及通知渠道">
      <div class="max-w-2xl space-y-3.5">
        <div class="row-field">
          <label class="lbl">通知渠道</label>
          <Select v-model="msg.channel" :options="noticeChannelOptions" class="flex-1" />
        </div>
        <div class="row-switch"><span>新订单通知</span><Switch v-model="msg.notice_order" /></div>
        <div v-if="msg.notice_order" class="row-field">
          <label class="lbl">订单金额大于</label>
          <div class="flex flex-1 items-center gap-2">
            <input v-model="msg.notice_order_money" class="field-input w-40" />
            <span class="text-xs text-muted-foreground">元时才通知（0 为全部通知）</span>
          </div>
        </div>
        <div class="row-switch"><span>结算通知</span><Switch v-model="msg.notice_settle" /></div>
        <div class="row-switch"><span>登录通知</span><Switch v-model="msg.notice_login" /></div>
        <div class="row-switch"><span>余额不足提醒</span><Switch v-model="msg.notice_balance" /></div>
        <div v-if="msg.notice_balance" class="row-field">
          <label class="lbl">余额小于</label>
          <div class="flex flex-1 items-center gap-2">
            <input v-model="msg.notice_balance_money" class="field-input w-40" />
            <span class="text-xs text-muted-foreground">元时提醒</span>
          </div>
        </div>
      </div>
      <div class="mt-5 border-t border-border/60 pt-4"><Button :disabled="savingMsg" @click="saveMsg"><Save />保存消息提醒</Button></div>
    </Panel>

    <!-- 手续费扣除模式 -->
    <Panel title="支付手续费扣除模式">
      <div class="max-w-2xl space-y-3.5">
        <div class="row-field">
          <label class="lbl">扣费模式</label>
          <Select v-model="mode.mode" :options="modeOptions" class="flex-1" />
        </div>
        <p class="text-xs text-muted-foreground">订单加费模式下，买家实际支付金额 = 订单金额 + 手续费。</p>
      </div>
      <div class="mt-5 border-t border-border/60 pt-4"><Button @click="save"><Save />保存扣费模式</Button></div>
    </Panel>

    <!-- 第三方账号绑定 -->
    <Panel title="第三方账号绑定" subtitle="绑定后可使用对应方式快捷登录">
      <div class="max-w-2xl space-y-2.5">
        <div v-for="s in socials" :key="s.key" class="flex items-center gap-3 border border-border/70 px-4 py-3">
          <span :class="['flex size-9 items-center justify-center rounded-full border border-border text-sm font-semibold', s.color]">
            {{ s.label.slice(0, 1) }}
          </span>
          <div class="min-w-0 flex-1">
            <div class="text-sm font-medium">{{ s.label }}快捷登录</div>
            <div class="text-xs text-muted-foreground">
              <span v-if="binds[s.key].bound">已绑定{{ binds[s.key].nick ? '：' + binds[s.key].nick : '' }}</span>
              <span v-else>未绑定</span>
            </div>
          </div>
          <Button
            :variant="binds[s.key].bound ? 'outline' : 'default'"
            size="sm"
            :class="binds[s.key].bound ? 'text-destructive hover:text-destructive' : undefined"
            @click="toggleBind(s.key)"
          >
            <component :is="binds[s.key].bound ? Unlink : Link2" class="size-4" />
            {{ binds[s.key].bound ? '解绑' : '绑定' }}
          </Button>
        </div>
      </div>
    </Panel>
  </div>
</template>
