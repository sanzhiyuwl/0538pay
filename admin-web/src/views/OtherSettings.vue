<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { Save, Plug } from 'lucide-vue-next'
import { Panel, Button, Select, Switch } from '@/components/ui'
import { otherSettingTabs, proxyTypeOptions, ipTypeOptions } from '@/lib/mock/sysconfig'
import { fetchConfig, saveConfig } from '@/lib/api/config'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()
const activeTab = ref('proxy')
const saving = ref(false)

// proxy + iptype 合并为 config group=other（键名对齐 epay）
const proxy = reactive({
  proxy: '0', proxy_server: '', proxy_port: '', proxy_user: '', proxy_pwd: '', proxy_type: 'http',
})
const ip = reactive({ ip_type: '2' })

const proxyOn = computed({
  get: () => proxy.proxy === '1',
  set: (v: boolean) => (proxy.proxy = v ? '1' : '0'),
})

onMounted(async () => {
  try {
    const kv = await fetchConfig('other')
    Object.assign(proxy, {
      proxy: kv.proxy ?? '0', proxy_server: kv.proxy_server ?? '', proxy_port: kv.proxy_port ?? '',
      proxy_user: kv.proxy_user ?? '', proxy_pwd: kv.proxy_pwd ?? '', proxy_type: kv.proxy_type || 'http',
    })
    ip.ip_type = kv.ip_type || '2'
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载失败')
  }
})

async function save() {
  saving.value = true
  try {
    await saveConfig('other', { ...proxy, ...ip })
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
        <Button :disabled="saving" @click="save"><Save />保存设置</Button>
        <Button v-if="activeTab === 'proxy' && proxyOn" variant="outline" disabled><Plug />测试连通性</Button>
      </div>
    </Panel>
  </div>
</template>
