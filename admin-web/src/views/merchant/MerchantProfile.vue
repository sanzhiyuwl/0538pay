<script setup lang="ts">
import { reactive, computed } from 'vue'
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

const settle = reactive({ ...settleConfig })
const contact = reactive({ ...contactConfig })
const msg = reactive({ ...msgConfig })
const mode = reactive({ ...modeConfig })
const binds = reactive({ ...bindConfig })

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

function save() {}
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
            <input :value="contact.phone" readonly class="field-input flex-1 bg-muted/40" />
            <Button variant="outline" size="sm"><ShieldCheck class="size-4" />短信验证改绑</Button>
          </div>
        </div>
        <div class="row-field">
          <label class="lbl">邮箱</label>
          <input v-model="contact.email" class="field-input flex-1" />
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
      <div class="mt-5 border-t border-border/60 pt-4"><Button @click="save"><Save />保存消息提醒</Button></div>
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
