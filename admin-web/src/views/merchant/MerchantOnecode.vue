<script setup lang="ts">
import { ref } from 'vue'
import { Copy, Check, QrCode, Download, Pencil } from 'lucide-vue-next'
import { Panel, Button } from '@/components/ui'

// 聚合收款码（对齐 epay user/onecode.php）
const payUrl = 'https://0538pay.com/paypage/?merchant=eyJ1aWQiOjEwMDF9'
const codeName = ref('泰安优选商贸')
const editing = ref(false)

// 收款码风格模板
const styles = [
  { key: 'default', label: '默认' },
  { key: 'winter', label: '冬雪' },
  { key: 'pikachu', label: '皮卡丘' },
  { key: 'unionpay', label: '银联' },
  { key: 'cat', label: '猫咪' },
  { key: 'business', label: '商务' },
]
const activeStyle = ref('default')

const copied = ref(false)
function copyUrl() {
  navigator.clipboard?.writeText(payUrl).then(() => {
    copied.value = true
    setTimeout(() => (copied.value = false), 1500)
  }).catch(() => {})
}
function saveName() {
  editing.value = false
}
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="聚合收款码" subtitle="生成固定收款二维码，客户扫码后自选支付宝/微信/QQ 并输入金额付款">
      <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <!-- 左：配置 -->
        <div class="space-y-3.5">
          <div class="row-field">
            <label class="lbl">收款链接</label>
            <div class="flex flex-1 items-center gap-2">
              <input :value="payUrl" readonly class="field-input flex-1 bg-muted/40 font-mono text-[12px]" />
              <Button variant="outline" size="sm" @click="copyUrl">
                <component :is="copied ? Check : Copy" class="size-4" />
              </Button>
            </div>
          </div>
          <div class="row-field">
            <label class="lbl">收款方名称</label>
            <div class="flex flex-1 items-center gap-2">
              <input v-model="codeName" :readonly="!editing" class="field-input flex-1" :class="!editing && 'bg-muted/40'" />
              <Button v-if="!editing" variant="outline" size="sm" @click="editing = true"><Pencil class="size-4" />修改</Button>
              <Button v-else size="sm" @click="saveName"><Check class="size-4" />保存</Button>
            </div>
          </div>

          <!-- 风格模板 -->
          <div>
            <div class="mb-2 text-sm text-muted-foreground">收款码风格</div>
            <div class="grid grid-cols-3 gap-2">
              <button
                v-for="s in styles"
                :key="s.key"
                class="border py-2 text-sm transition-colors"
                :class="activeStyle === s.key ? 'border-primary text-primary ring-1 ring-primary' : 'border-border hover:border-primary/50'"
                @click="activeStyle = s.key"
              >
                {{ s.label }}
              </button>
            </div>
          </div>
        </div>

        <!-- 右：预览 -->
        <div class="flex flex-col items-center justify-center gap-4 border border-border/70 bg-muted/20 py-8">
          <div class="text-sm font-medium">{{ codeName }}</div>
          <div class="flex size-48 items-center justify-center border border-border bg-background text-muted-foreground/40">
            <QrCode class="size-20" />
          </div>
          <div class="flex items-center gap-2 text-xs text-muted-foreground">
            <img v-for="ic in ['alipay', 'wxpay', 'qqpay']" :key="ic" :src="`/assets/icon/${ic}.ico`" class="size-4" onerror="this.style.display='none'" />
            <span>支持支付宝 / 微信 / QQ钱包</span>
          </div>
          <Button variant="outline" size="sm"><Download class="size-4" />下载收款码</Button>
        </div>
      </div>
      <p class="mt-4 border-t border-border/60 pt-4 text-xs text-muted-foreground">
        将收款码打印张贴或分享给客户，客户扫码后进入收银台，选择支付方式并输入金额即可付款，款项实时到账。
      </p>
    </Panel>
  </div>
</template>
