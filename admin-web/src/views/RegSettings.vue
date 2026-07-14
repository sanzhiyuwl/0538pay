<script setup lang="ts">
import { reactive, computed } from 'vue'
import { Save } from 'lucide-vue-next'
import { Panel, Button, Switch, Select } from '@/components/ui'
import { regConfig, regOpenOptions, captchaVersionOptions } from '@/lib/mock/settings'

const reg = reactive({ ...regConfig })

// 关闭注册时，审核/付费等注册相关项无意义
const regClosed = computed(() => reg.regOpen === '0')

function save() {
  // 原型阶段：仅提示（不落库）
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
              <Select v-model="reg.regOpen" :options="regOpenOptions" class="w-full" />
            </div>
          </div>
          <div class="set-field">
            <label class="set-lbl" :class="regClosed ? 'opacity-50' : ''">开启注册审核</label>
            <div class="min-w-0 flex-1">
              <Switch v-model="reg.regAudit" :disabled="regClosed" />
              <p class="set-hint">开启后新注册商户需管理员审核通过才能登录</p>
            </div>
          </div>
          <div class="set-field">
            <label class="set-lbl" :class="regClosed ? 'opacity-50' : ''">注册后可不填结算账户</label>
            <div class="min-w-0 flex-1">
              <Switch v-model="reg.regInputSettle" :disabled="regClosed" />
              <p class="set-hint">如不做平台代收，可设置为开启</p>
            </div>
          </div>
          <div class="set-field">
            <label class="set-lbl" :class="regClosed ? 'opacity-50' : ''">注册付费</label>
            <div class="min-w-0 flex-1">
              <Switch v-model="reg.regPay" :disabled="regClosed" />
            </div>
          </div>
          <div v-if="reg.regPay && !regClosed" class="set-field">
            <label class="set-lbl">注册付费金额</label>
            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-2">
                <input v-model="reg.regPayPrice" class="field-input w-32" /><span class="text-sm text-muted-foreground">元</span>
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
              <Switch v-model="reg.loginCaptcha" />
              <p class="set-hint">开启后使用极验滑动验证码，需填写下方 ID 与密钥</p>
            </div>
          </div>
          <template v-if="reg.loginCaptcha">
            <div class="set-field">
              <label class="set-lbl">极验版本</label>
              <div class="min-w-0 flex-1">
                <Select v-model="reg.captchaVersion" :options="captchaVersionOptions" class="w-full" />
              </div>
            </div>
            <div class="set-field">
              <label class="set-lbl">极验验证码 ID</label>
              <div class="min-w-0 flex-1">
                <input v-model="reg.captchaId" placeholder="极验后台获取的 ID" class="field-input w-full" />
              </div>
            </div>
            <div class="set-field">
              <label class="set-lbl">极验验证码密钥</label>
              <div class="min-w-0 flex-1">
                <input v-model="reg.captchaKey" placeholder="极验后台获取的 KEY" class="field-input w-full" />
              </div>
            </div>
          </template>
        </div>
      </div>

      <div class="mt-5 border-t border-border/60 pt-4">
        <Button @click="save"><Save />保存设置</Button>
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
