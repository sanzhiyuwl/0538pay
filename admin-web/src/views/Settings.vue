<script setup lang="ts">
import { ref, reactive } from 'vue'
import { Save, Upload, UploadCloud, X } from 'lucide-vue-next'
import { Panel, Button } from '@/components/ui'
import { logoItems, siteConfig } from '@/lib/mock/settings'

const tabs = [
  { key: 'site', label: '网站信息' },
  { key: 'beian', label: '版权备案' },
  { key: 'logo', label: 'LOGO 设置' },
]
const activeTab = ref('site')

const site = reactive({ ...siteConfig })

// ===== LOGO 上传（原型：FileReader 本地预览，不落库）=====
const logos = reactive<Record<string, string>>({})
const logoInputs = ref<Record<string, HTMLInputElement | null>>({})
function setLogoInput(key: string, el: any) {
  logoInputs.value[key] = el as HTMLInputElement | null
}
function pickLogo(key: string) {
  logoInputs.value[key]?.click()
}
function onLogoChange(key: string, e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file || !file.type.startsWith('image/')) return
  const reader = new FileReader()
  reader.onload = () => {
    logos[key] = reader.result as string
  }
  reader.readAsDataURL(file)
}
function removeLogo(key: string) {
  delete logos[key]
  const el = logoInputs.value[key]
  if (el) el.value = ''
}

function save() {
  // 原型阶段：仅提示（不落库）
}
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="网站设置" subtitle="网站基础信息与 LOGO 配置">
      <!-- 分组 Tab -->
      <div class="mb-4 flex flex-wrap gap-1 border-b border-border">
        <button
          v-for="t in tabs"
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

      <!-- 网站信息 -->
      <div v-if="activeTab === 'site'" class="max-w-2xl space-y-3.5">
        <div class="row-field">
          <label class="lbl">网站名称</label>
          <input v-model="site.sitename" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">首页标题</label>
          <input v-model="site.title" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">关键字</label>
          <input v-model="site.keywords" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">网站描述</label>
          <input v-model="site.description" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">公司名称</label>
          <input v-model="site.company" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">联系邮箱</label>
          <input v-model="site.email" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">客服QQ</label>
          <input v-model="site.qq" class="field-input flex-1" />
        </div>
      </div>

      <!-- 版权备案 -->
      <div v-else-if="activeTab === 'beian'" class="max-w-2xl space-y-6">
        <!-- 版权信息 -->
        <div class="space-y-3.5">
          <h4 class="text-sm font-medium">版权信息</h4>
          <div class="beian-field">
            <label class="beian-lbl">版权链接</label>
            <div class="min-w-0 flex-1">
              <input v-model="site.copyrightLink" placeholder="https://beian.miit.gov.cn/" class="field-input w-full" />
              <p class="beian-hint">点击 PC 底部版权文字跳转的链接</p>
            </div>
          </div>
          <div class="beian-field">
            <label class="beian-lbl mt-1.5">版权信息</label>
            <div class="min-w-0 flex-1">
              <textarea
                v-model="site.copyright"
                rows="3"
                placeholder="Copyright © 2026 0538Pay 版权所有"
                class="field-input w-full resize-none py-2"
                style="height: auto"
              />
              <p class="beian-hint">显示在 PC 端底部的版权文字</p>
            </div>
          </div>
        </div>

        <!-- 备案设置 -->
        <div class="space-y-3.5 border-t border-border/60 pt-5">
          <h4 class="text-sm font-medium">备案设置</h4>
          <div class="beian-field">
            <label class="beian-lbl">网站 ICP 备案号</label>
            <div class="min-w-0 flex-1">
              <input v-model="site.icp" placeholder="如 鲁ICP备2026000538号-1" class="field-input w-full" />
              <p class="beian-hint">工信部核发的 ICP 备案号，显示在 PC 端底部</p>
            </div>
          </div>
          <div class="beian-field">
            <label class="beian-lbl">网站公安备案</label>
            <div class="min-w-0 flex-1">
              <input v-model="site.police" placeholder="如 鲁公网安备 37098202000538号" class="field-input w-full" />
              <p class="beian-hint">公安部门登记的备案信息，显示在 PC 端底部</p>
            </div>
          </div>
          <div class="beian-field">
            <label class="beian-lbl">网站公安链接</label>
            <div class="min-w-0 flex-1">
              <input v-model="site.policeLink" placeholder="https://beian.mps.gov.cn/" class="field-input w-full" />
              <p class="beian-hint">PC 底部公安备案号点击跳转的链接</p>
            </div>
          </div>
          <div class="beian-field">
            <label class="beian-lbl">市场监督管理局链接</label>
            <div class="min-w-0 flex-1">
              <input v-model="site.marketLink" placeholder="https://www.gsxt.gov.cn/" class="field-input w-full" />
              <p class="beian-hint">PC 底部市场监督管理局点击跳转的链接</p>
            </div>
          </div>
        </div>
      </div>

      <!-- LOGO 设置 -->
      <div v-else-if="activeTab === 'logo'" class="max-w-3xl space-y-5">
        <div v-for="item in logoItems" :key="item.key" class="flex items-start gap-4">
          <label class="w-32 shrink-0 whitespace-nowrap pt-1 text-right text-sm text-muted-foreground">{{ item.label }}</label>
          <button
            type="button"
            class="group flex size-20 shrink-0 flex-col items-center justify-center gap-1 overflow-hidden rounded-lg border border-dashed border-border bg-muted/30 transition-colors hover:border-primary hover:bg-primary/[0.04]"
            @click="pickLogo(item.key)"
          >
            <img v-if="logos[item.key]" :src="logos[item.key]" :alt="item.label" class="size-full object-contain p-1" />
            <template v-else>
              <UploadCloud class="size-6 text-muted-foreground/50 group-hover:text-primary" />
              <span class="text-xs text-muted-foreground/70">上传</span>
            </template>
          </button>
          <div class="flex flex-col gap-1.5 pt-0.5">
            <div class="flex items-center gap-2">
              <button
                type="button"
                class="inline-flex items-center gap-1 text-sm font-medium text-primary hover:underline"
                @click="pickLogo(item.key)"
              >
                <Upload class="size-3.5" />{{ logos[item.key] ? '重新上传' : '上传' }}
              </button>
              <button
                v-if="logos[item.key]"
                type="button"
                class="inline-flex items-center gap-1 text-sm text-destructive hover:underline"
                @click="removeLogo(item.key)"
              >
                <X class="size-3.5" />移除
              </button>
            </div>
            <span class="text-xs text-muted-foreground">{{ item.desc }}</span>
          </div>
          <input
            :ref="(el) => setLogoInput(item.key, el)"
            type="file"
            accept="image/*"
            class="hidden"
            @change="(e) => onLogoChange(item.key, e)"
          />
        </div>
      </div>

      <!-- 保存 -->
      <div class="mt-5 border-t border-border/60 pt-4">
        <Button @click="save"><Save />保存设置</Button>
      </div>
    </Panel>
  </div>
</template>

<style scoped>
/* 版权备案：标签+控件横排，标签固定宽（比通用 .lbl 宽以容纳"市场监督管理局链接"不折行），
   控件下方跟随说明文字，与输入框左对齐 */
.beian-field {
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
}
.beian-lbl {
  width: 9rem;
  flex-shrink: 0;
  padding-top: 0.5rem;
  text-align: right;
  font-size: 0.875rem;
  line-height: 1.25;
  color: var(--muted-foreground);
}
.beian-hint {
  margin-top: 0.375rem;
  font-size: 0.75rem;
  color: var(--muted-foreground);
}
</style>
