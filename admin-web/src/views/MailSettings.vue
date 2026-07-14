<script setup lang="ts">
import { reactive, computed } from 'vue'
import { Save, Send } from 'lucide-vue-next'
import { Panel, Button, Select } from '@/components/ui'
import {
  mailConfig,
  mailCloudOptions,
  smsConfig,
  smsApiOptions,
} from '@/lib/mock/sysconfig'

const mail = reactive({ ...mailConfig })
const sms = reactive({ ...smsConfig })
// SMTP 与云推送字段互斥显隐
const isSmtp = computed(() => mail.mail_cloud === '0')
// 企信通(0)无 AppId；ThinkAPI(3) 也无独立 AppId
const showSmsAppId = computed(() => sms.sms_api !== '0' && sms.sms_api !== '3')
const showSmsSign = computed(() => sms.sms_api !== '0')
function save() {}
</script>

<template>
  <div class="space-y-2.5">
    <!-- 发信邮箱 -->
    <Panel title="发信邮箱设置" subtitle="用于系统发信的邮箱账号配置">
      <div class="max-w-2xl space-y-3.5">
        <div class="row-field">
          <label class="lbl">发信模式</label>
          <Select v-model="mail.mail_cloud" :options="mailCloudOptions" class="flex-1" />
        </div>
        <template v-if="isSmtp">
          <div class="row-field">
            <label class="lbl">SMTP 服务器</label>
            <input v-model="mail.mail_smtp" class="field-input flex-1" />
          </div>
          <div class="row-field">
            <label class="lbl">SMTP 端口</label>
            <input v-model="mail.mail_port" class="field-input flex-1" />
          </div>
          <div class="row-field">
            <label class="lbl">邮箱账号</label>
            <input v-model="mail.mail_name" class="field-input flex-1" />
          </div>
          <div class="row-field">
            <label class="lbl">邮箱密码</label>
            <input v-model="mail.mail_pwd" type="password" placeholder="QQ 邮箱填 SMTP 授权码" class="field-input flex-1" />
          </div>
        </template>
        <template v-else>
          <div class="row-field">
            <label class="lbl">API_USER</label>
            <input v-model="mail.mail_apiuser" class="field-input flex-1" />
          </div>
          <div class="row-field">
            <label class="lbl">API_KEY</label>
            <input v-model="mail.mail_apikey" class="field-input flex-1" />
          </div>
          <div class="row-field">
            <label class="lbl">发信邮箱</label>
            <input v-model="mail.mail_name2" class="field-input flex-1" />
          </div>
        </template>
        <div class="row-field">
          <label class="lbl">收信邮箱</label>
          <input v-model="mail.mail_recv" placeholder="不填默认为发信邮箱" class="field-input flex-1" />
        </div>
      </div>
      <div class="mt-5 flex items-center gap-2 border-t border-border/60 pt-4">
        <Button @click="save"><Save />保存设置</Button>
        <Button variant="outline" @click="save"><Send />发送测试邮件</Button>
      </div>
    </Panel>

    <!-- 短信接口 -->
    <Panel title="短信接口设置" subtitle="配置短信服务商与各业务短信模板ID">
      <div class="max-w-2xl space-y-3.5">
        <div class="row-field">
          <label class="lbl">接口选择</label>
          <Select v-model="sms.sms_api" :options="smsApiOptions" class="flex-1" />
        </div>
        <div v-if="showSmsAppId" class="row-field">
          <label class="lbl">AppId</label>
          <input v-model="sms.sms_appid" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">AppKey</label>
          <input v-model="sms.sms_appkey" class="field-input flex-1" />
        </div>
        <div v-if="showSmsSign" class="row-field">
          <label class="lbl">短信签名</label>
          <input v-model="sms.sms_sign" placeholder="已审核通过的短信签名" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">注册模板ID</label>
          <input v-model="sms.sms_tpl_reg" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">找回密码模板ID</label>
          <input v-model="sms.sms_tpl_find" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">改结算账号模板ID</label>
          <input v-model="sms.sms_tpl_edit" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">余额不足模板ID</label>
          <input v-model="sms.sms_tpl_balance" placeholder="留空则不开启短信通知" class="field-input flex-1" />
        </div>
      </div>
      <div class="mt-5 border-t border-border/60 pt-4">
        <Button @click="save"><Save />保存设置</Button>
      </div>
    </Panel>
  </div>
</template>
