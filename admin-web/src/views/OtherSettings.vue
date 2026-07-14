<script setup lang="ts">
import { ref, reactive, computed } from 'vue'
import { Save, Plug } from 'lucide-vue-next'
import { Panel, Button, Select, Switch } from '@/components/ui'
import {
  otherSettingTabs,
  proxyConfig,
  proxyTypeOptions,
  ipConfig,
  ipTypeOptions,
} from '@/lib/mock/sysconfig'

const activeTab = ref('proxy')

// 中转代理
const proxy = reactive({ ...proxyConfig })
const proxyOn = computed({
  get: () => proxy.proxy === '1',
  set: (v: boolean) => (proxy.proxy = v ? '1' : '0'),
})

// IP 获取方式
const ip = reactive({ ...ipConfig })

function save() {}
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="其余设置" subtitle="中转代理 / IP 获取方式">
      <!-- 分组 Tab -->
      <div class="mb-4 flex flex-wrap gap-1 border-b border-border">
        <button
          v-for="t in otherSettingTabs"
          :key="t.key"
          class="-mb-px border-b-2 px-4 py-2 text-sm transition-colors"
          :class="
            activeTab === t.key
              ? 'border-primary font-medium text-primary'
              : 'border-transparent text-muted-foreground hover:text-foreground'
          "
          @click="activeTab = t.key"
        >
          {{ t.label }}
        </button>
      </div>

      <!-- 中转代理 -->
      <div v-if="activeTab === 'proxy'" class="max-w-2xl space-y-3.5">
        <div class="row-switch"><span>中转代理开关</span><Switch v-model="proxyOn" /></div>
        <template v-if="proxyOn">
          <div class="row-field">
            <label class="lbl">代理 IP</label>
            <input v-model="proxy.proxy_server" class="field-input flex-1" />
          </div>
          <div class="row-field">
            <label class="lbl">代理端口</label>
            <input v-model="proxy.proxy_port" class="field-input flex-1" />
          </div>
          <div class="row-field">
            <label class="lbl">代理账号</label>
            <input v-model="proxy.proxy_user" class="field-input flex-1" />
          </div>
          <div class="row-field">
            <label class="lbl">代理密码</label>
            <input v-model="proxy.proxy_pwd" type="password" class="field-input flex-1" />
          </div>
          <div class="row-field">
            <label class="lbl">代理协议</label>
            <Select v-model="proxy.proxy_type" :options="proxyTypeOptions" class="flex-1" />
          </div>
        </template>
        <p class="text-xs text-muted-foreground">
          开启后异步回调经中转代理访问商户网站，可解决仅国内可访问的回调问题，并隐藏本站服务器 IP。
        </p>
      </div>

      <!-- IP 获取方式 -->
      <div v-else-if="activeTab === 'iptype'" class="max-w-2xl space-y-3.5">
        <div class="row-field">
          <label class="lbl">IP 获取方式</label>
          <Select v-model="ip.ip_type" :options="ipTypeOptions" class="flex-1" />
        </div>
        <p class="text-xs text-muted-foreground">
          用于防止伪造 IP 请求。使用 CDN 时选 X_REAL_IP；无 CDN 建议选 REMOTE_ADDR。请选择能显示真实地址的选项。
        </p>
      </div>

      <!-- 保存 -->
      <div class="mt-5 flex items-center gap-2 border-t border-border/60 pt-4">
        <Button @click="save"><Save />保存设置</Button>
        <Button v-if="activeTab === 'proxy' && proxyOn" variant="outline" @click="save"><Plug />测试连通性</Button>
      </div>
    </Panel>
  </div>
</template>
