<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { Save, Copy, ArrowRight } from 'lucide-vue-next'
import { useRouter } from 'vue-router'
import { Panel, Button, Select, Switch } from '@/components/ui'
import { fetchConfig, saveConfig, fetchWxkfAccounts } from '@/lib/api/config'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const router = useRouter()
const toast = useToast()
const saving = ref(false)

// 回调 URL 对齐 epay set_wxkf.php：$siteurl.'wework.php'
const callbackUrl = window.location.origin + '/wework.php'

// 消息模式选项（对齐 epay set_wxkf.php wework_paymsgmode）
const paymsgModeOptions = [
  { value: '0', label: '发送确认消息，用户回复后发支付链接（默认）' },
  { value: '1', label: '直接发送支付链接（用户将无法支付第二单）' },
]

// 指定客服账号下拉（0=多客服账号轮询 + 启用企微下的客服账号）
const kfAccountOptions = ref<{ value: string; label: string }[]>([{ value: '0', label: '多客服账号轮询' }])

// config group=wxkf 全部字段
const cfg = reactive({
  wework_token: '',
  wework_aeskey: '',
  wework_payopen: '0',
  wework_paymsgmode: '0',
  wework_paykfid: '0',
  wework_contact: '',
  wework_remark: '',
})

const payOn = computed({
  get: () => cfg.wework_payopen === '1',
  set: (v: boolean) => (cfg.wework_payopen = v ? '1' : '0'),
})

function copyCallback() {
  navigator.clipboard
    ?.writeText(callbackUrl)
    .then(() => toast.success('回调 URL 已复制'))
    .catch(() => toast.error('复制失败，请手动复制'))
}

onMounted(async () => {
  try {
    const [kv, accounts] = await Promise.all([fetchConfig('wxkf'), fetchWxkfAccounts()])
    Object.assign(cfg, {
      wework_token: kv.wework_token ?? '',
      wework_aeskey: kv.wework_aeskey ?? '',
      wework_payopen: kv.wework_payopen ?? '0',
      wework_paymsgmode: kv.wework_paymsgmode ?? '0',
      wework_paykfid: kv.wework_paykfid ?? '0',
      wework_contact: kv.wework_contact ?? '',
      wework_remark: kv.wework_remark ?? '',
    })
    kfAccountOptions.value = [
      { value: '0', label: '多客服账号轮询' },
      ...accounts.list.map((a) => ({ value: String(a.id), label: `${a.openkfid} - ${a.name}` })),
    ]
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载失败')
  }
})

async function save() {
  if (saving.value) return
  saving.value = true
  try {
    await saveConfig('wxkf', { ...cfg })
    toast.success('设置已保存')
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="H5 跳转微信客服支付" subtitle="从手机网站跳转到微信客服完成支付，需先配置企业微信账号">
      <template #actions>
        <Button variant="outline" size="sm" @click="router.push('/admin/wework')">
          企业微信账号列表<ArrowRight />
        </Button>
      </template>
      <div class="max-w-2xl space-y-3.5">
        <!-- 回调配置 -->
        <div class="row-field">
          <label class="lbl">回调 URL</label>
          <div class="flex flex-1 items-center gap-2">
            <input :value="callbackUrl" readonly class="field-input flex-1 bg-muted/40 font-mono text-xs" />
            <Button variant="outline" size="sm" @click="copyCallback"><Copy class="size-4" /></Button>
          </div>
        </div>
        <div class="row-field">
          <label class="lbl">Token</label>
          <input v-model="cfg.wework_token" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">EncodingAESKey</label>
          <input v-model="cfg.wework_aeskey" class="field-input flex-1" />
        </div>

        <!-- 支付配置 -->
        <div class="border-t border-border/60 pt-3.5">
          <div class="row-switch"><span>开启 H5 跳转微信客服支付</span><Switch v-model="payOn" /></div>
        </div>
        <div class="row-field">
          <label class="lbl">支付消息模式</label>
          <Select v-model="cfg.wework_paymsgmode" :options="paymsgModeOptions" class="flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">客服账号</label>
          <Select v-model="cfg.wework_paykfid" :options="kfAccountOptions" class="flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">人工客服链接</label>
          <input v-model="cfg.wework_contact" placeholder="选填，追加在支付消息后面" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">消息尾部内容</label>
          <input v-model="cfg.wework_remark" placeholder="选填，支持变量 [qq] 当前商户联系QQ" class="field-input flex-1" />
        </div>
      </div>
      <div class="mt-5 border-t border-border/60 pt-4">
        <Button :disabled="saving" @click="save"><Save />保存设置</Button>
      </div>
      <p class="mt-4 border-t border-border/60 pt-4 text-xs text-muted-foreground">
        开启前请确保配置正确，否则会导致手机浏览器无法微信支付。仅能使用独立版微信客服获取 token 并配置回调，不能开启企业微信内的微信客服应用。
      </p>
    </Panel>
  </div>
</template>
