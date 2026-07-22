<script setup lang="ts">
import { reactive, computed, ref, onMounted } from 'vue'
import { Save } from 'lucide-vue-next'
import { Panel, Button, Switch, Select } from '@/components/ui'
import { regOpenOptions, captchaVersionOptions } from '@/lib/mock/settings'
import { fetchConfig, saveConfig } from '@/lib/api/config'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()

// 键名对齐 epay set.php mod=reg（值一律字符串 "0"/"1"）
const reg = reactive({
  reg_open: '1',
  user_review: '0',
  reg_input_settle: '0',
  reg_pay: '0',
  reg_pay_price: '0',
  captcha_open_login: '0',
  captcha_version: '1',
  captcha_id: '',
  captcha_key: '',
})

// 布尔开关 ↔ 字符串 "0"/"1" 适配（Switch 组件只吃 boolean）
function boolFor(key: keyof typeof reg) {
  return computed({
    get: () => reg[key] === '1',
    set: (v: boolean) => { reg[key] = v ? '1' : '0' },
  })
}
const regAudit = boolFor('user_review')
const regInputSettle = boolFor('reg_input_settle')
const regPay = boolFor('reg_pay')
const loginCaptcha = boolFor('captcha_open_login')

// 关闭注册时，审核/付费等注册相关项无意义
const regClosed = computed(() => reg.reg_open === '0')

const loading = ref(false)
const saving = ref(false)

async function load() {
  loading.value = true
  try {
    Object.assign(reg, await fetchConfig('reg'))
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载注册设置失败')
  } finally {
    loading.value = false
  }
}
onMounted(load)

async function save() {
  saving.value = true
  try {
    await saveConfig('reg', { ...reg })
    toast.success('注册登录设置已保存')
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="注册登录" subtitle="商户注册开关、注册审核与登录验证码设置">
      <div class="max-w-2xl space-y-6">
        <!-- 注册设置 -->
        <div class="space-y-4">
          <h4 class="text-sm font-medium">注册设置</h4>
          <div class="set-field">
            <label class="set-lbl">开放注册</label>
            <div class="min-w-0 flex-1">
              <Select v-model="reg.reg_open" :options="regOpenOptions" class="w-full" />
            </div>
          </div>
          <div class="set-field">
            <label class="set-lbl" :class="regClosed ? 'opacity-50' : ''">开启注册审核</label>
            <div class="min-w-0 flex-1">
              <Switch v-model="regAudit" :disabled="regClosed" />
              <p class="set-hint">开启后新注册商户需管理员审核通过才能登录</p>
            </div>
          </div>
          <div class="set-field">
            <label class="set-lbl" :class="regClosed ? 'opacity-50' : ''">注册后可不填结算账户</label>
            <div class="min-w-0 flex-1">
              <Switch v-model="regInputSettle" :disabled="regClosed" />
              <p class="set-hint">如不做平台代收，可设置为开启</p>
            </div>
          </div>
          <div class="set-field">
            <label class="set-lbl" :class="regClosed ? 'opacity-50' : ''">注册付费</label>
            <div class="min-w-0 flex-1">
              <Switch v-model="regPay" :disabled="regClosed" />
            </div>
          </div>
          <div v-if="regPay && !regClosed" class="set-field">
            <label class="set-lbl">注册付费金额</label>
            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-2">
                <input v-model="reg.reg_pay_price" class="field-input w-32" /><span class="text-sm text-muted-foreground">元</span>
              </div>
            </div>
          </div>
        </div>

        <!-- 登录验证码 -->
        <div class="space-y-4 border-t border-border/60 pt-5">
          <h4 class="text-sm font-medium">登录验证码</h4>
          <div class="set-field">
            <label class="set-lbl">登录开启验证码</label>
            <div class="min-w-0 flex-1">
              <Switch v-model="loginCaptcha" />
              <p class="set-hint">开启后使用极验滑动验证码，需填写下方 ID 与密钥</p>
            </div>
          </div>
          <template v-if="loginCaptcha">
            <div class="set-field">
              <label class="set-lbl">极验版本</label>
              <div class="min-w-0 flex-1">
                <Select v-model="reg.captcha_version" :options="captchaVersionOptions" class="w-full" />
              </div>
            </div>
            <div class="set-field">
              <label class="set-lbl">极验验证码 ID</label>
              <div class="min-w-0 flex-1">
                <input v-model="reg.captcha_id" placeholder="极验后台获取的 ID" class="field-input w-full" />
              </div>
            </div>
            <div class="set-field">
              <label class="set-lbl">极验验证码密钥</label>
              <div class="min-w-0 flex-1">
                <input v-model="reg.captcha_key" placeholder="极验后台获取的 KEY" class="field-input w-full" />
              </div>
            </div>
          </template>
        </div>
      </div>

      <div class="mt-5 border-t border-border/60 pt-4">
        <Button :disabled="saving || loading" @click="save"><Save />保存设置</Button>
      </div>
    </Panel>
  </div>
</template>

<style scoped>
/* 注册登录：统一行结构——标签固定宽右对齐，控件左边缘对齐，说明文字跟随控件下方 */
.set-field {
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
}
.set-lbl {
  width: 9rem;
  flex-shrink: 0;
  padding-top: 0.5rem;
  text-align: right;
  font-size: 0.875rem;
  line-height: 1.25;
  color: var(--muted-foreground);
}
.set-hint {
  margin-top: 0.375rem;
  font-size: 0.75rem;
  color: var(--muted-foreground);
}
</style>
