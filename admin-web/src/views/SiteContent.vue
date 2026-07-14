<script setup lang="ts">
/**
 * 官网管理 · 首页内容（CMS）。
 * 编辑官网首页各板块营销文案 → useSiteContentStore 持久化 → 官网首页实时联动。
 * 编辑草稿：从 store 深拷一份，点「保存」才提交；「恢复默认」重置为出厂内容。
 *
 * 布局：左侧板块导航 + 右侧编辑区。列表型板块（特性/产品/场景/方案/评价/FAQ）用折叠卡片，
 * 卡头显示序号+图标+主标题预览，便于快速定位；点开才展开编辑。保存栏吸底常驻。
 */
import { reactive, ref, computed } from 'vue'
import {
  Save, Plus, Trash2, RotateCcw, ArrowUp, ArrowDown, ExternalLink, ChevronDown,
  LayoutTemplate, BarChart3, Sparkles, Grid3x3, Tag, MessageSquareQuote, HelpCircle,
} from 'lucide-vue-next'
import { Panel, Button, Select, Switch } from '@/components/ui'
import { useSiteContentStore } from '@/stores/siteContent'
import { useToast } from '@/composables/useToast'
import { iconOptions, defaultSiteContent, type SiteContent } from '@/lib/mock/site-content'

const store = useSiteContentStore()
const toast = useToast()

// 深拷贝草稿
const draft = reactive<SiteContent>(JSON.parse(JSON.stringify(store.content)))

// 板块导航（带图标 + 数量徽标，右侧滚动定位）
const sections = [
  { key: 'hero', label: '首屏 / CTA', icon: LayoutTemplate },
  { key: 'metrics', label: '数据背书', icon: BarChart3 },
  { key: 'features', label: '核心特性', icon: Sparkles },
  { key: 'products', label: '产品矩阵', icon: Grid3x3 },
  { key: 'plans', label: '费率方案', icon: Tag },
  { key: 'testimonials', label: '客户评价', icon: MessageSquareQuote },
  { key: 'faqs', label: '常见问题', icon: HelpCircle },
]
const activeSection = ref('hero')

// 各列表板块的条目数（导航徽标）
const counts = computed<Record<string, number>>(() => ({
  features: draft.features.length,
  products: draft.products.length,
  plans: draft.plans.length,
  testimonials: draft.testimonials.length,
  faqs: draft.faqs.length,
}))

const iconSelectOptions = iconOptions.map((v) => ({ value: v, label: v }))

// ===== 折叠状态：key = `${section}:${index}` =====
const expanded = reactive<Set<string>>(new Set())
function toggle(section: string, i: number) {
  const k = `${section}:${i}`
  expanded.has(k) ? expanded.delete(k) : expanded.add(k)
}
function isOpen(section: string, i: number) {
  return expanded.has(`${section}:${i}`)
}

// ===== 数组增删移工具 =====
function addItem<T>(section: string, arr: T[], item: T) {
  arr.push(JSON.parse(JSON.stringify(item)))
  expanded.add(`${section}:${arr.length - 1}`) // 新增项自动展开
}
function removeItem<T>(arr: T[], i: number) {
  arr.splice(i, 1)
}
function moveItem<T>(arr: T[], i: number, dir: -1 | 1) {
  const j = i + dir
  if (j < 0 || j >= arr.length) return
  ;[arr[i], arr[j]] = [arr[j], arr[i]]
}
// 顿号 / 逗号分隔字符串 ↔ 数组（用于 points / tags 的行内编辑）
function joinList(list: string[]) {
  return list.join('、')
}
function splitList(v: string): string[] {
  return v.split(/[、,，]/).map((s) => s.trim()).filter(Boolean)
}

function save() {
  store.update(draft)
  toast.success('保存成功')
}

function resetAll() {
  Object.assign(draft, JSON.parse(JSON.stringify(defaultSiteContent)))
  expanded.clear()
  toast.info('已恢复默认内容，点保存后生效')
}

// 新增项模板
const blankFeature = { icon: 'Waypoints', title: '新特性', desc: '特性描述' }
const blankProduct = { icon: 'QrCode', name: '新产品', desc: '产品描述', points: ['特性一'], tags: ['标签'] }
const blankTestimonial = { name: '客户', role: '职位', avatar: '客', text: '评价内容' }
const blankFaq = { q: '问题', a: '答案' }
</script>

<template>
  <div class="space-y-2.5 pb-20">
    <Panel title="首页内容" subtitle="编辑官网首页各板块文案，保存后官网实时生效">
      <!-- 顶部操作条 -->
      <div class="mb-4 flex items-center justify-between gap-3">
        <a href="/" target="_blank" class="inline-flex items-center gap-1.5 text-sm text-primary hover:underline">
          <ExternalLink class="size-3.5" />预览官网首页
        </a>
        <button class="inline-flex items-center gap-1.5 text-sm text-muted-foreground transition-colors hover:text-foreground" @click="resetAll">
          <RotateCcw class="size-3.5" />恢复默认
        </button>
      </div>

      <div class="grid gap-6 lg:grid-cols-[188px_1fr]">
        <!-- 左：板块导航 -->
        <nav class="flex flex-row flex-wrap gap-1 lg:sticky lg:top-4 lg:h-max lg:flex-col">
          <button
            v-for="s in sections"
            :key="s.key"
            class="flex items-center gap-2.5 rounded-md px-3 py-2 text-left text-sm transition-colors"
            :class="activeSection === s.key ? 'bg-primary/[0.08] font-medium text-primary' : 'text-muted-foreground hover:bg-accent hover:text-foreground'"
            @click="activeSection = s.key"
          >
            <component :is="s.icon" class="size-4 shrink-0" />
            <span class="flex-1">{{ s.label }}</span>
            <span v-if="counts[s.key] != null" class="rounded-full bg-muted px-1.5 text-xs tabular-nums text-muted-foreground">{{ counts[s.key] }}</span>
          </button>
        </nav>

        <!-- 右：编辑区 -->
        <div class="min-w-0">
          <!-- ===== 首屏 / CTA ===== -->
          <div v-show="activeSection === 'hero'" class="max-w-2xl space-y-6">
            <div class="space-y-3.5">
              <h4 class="text-sm font-medium">首屏 Hero</h4>
              <div class="row-field"><label class="lbl">标签徽标</label><input v-model="draft.hero.badge" class="field-input flex-1" /></div>
              <div class="row-field"><label class="lbl">主标题上行</label><input v-model="draft.hero.titleLead" class="field-input flex-1" /></div>
              <div class="row-field"><label class="lbl">主标题高亮</label><input v-model="draft.hero.titleAccent" class="field-input flex-1" /></div>
              <div class="row-field"><label class="lbl">副标题</label><textarea v-model="draft.hero.subtitle" rows="3" class="field-input flex-1 resize-none py-2" style="height:auto" /></div>
              <div class="row-field"><label class="lbl">主按钮文字</label><input v-model="draft.hero.ctaPrimary" class="field-input flex-1" /></div>
              <div class="row-field"><label class="lbl">次按钮文字</label><input v-model="draft.hero.ctaSecondary" class="field-input flex-1" /></div>
              <div class="row-field"><label class="lbl">渠道栏文字</label><input v-model="draft.hero.payMethodsLabel" class="field-input flex-1" /></div>
            </div>
            <div class="space-y-3.5 border-t border-border/60 pt-5">
              <h4 class="text-sm font-medium">底部 CTA</h4>
              <div class="row-field"><label class="lbl">标题</label><input v-model="draft.cta.title" class="field-input flex-1" /></div>
              <div class="row-field"><label class="lbl">副标题</label><textarea v-model="draft.cta.subtitle" rows="2" class="field-input flex-1 resize-none py-2" style="height:auto" /></div>
              <div class="row-field"><label class="lbl">主按钮文字</label><input v-model="draft.cta.ctaPrimary" class="field-input flex-1" /></div>
              <div class="row-field"><label class="lbl">次按钮文字</label><input v-model="draft.cta.ctaSecondary" class="field-input flex-1" /></div>
            </div>
          </div>

          <!-- ===== 数据背书 ===== -->
          <div v-show="activeSection === 'metrics'" class="max-w-3xl space-y-4">
            <p class="text-xs text-muted-foreground">4 个滚动数字。数值填目标数字，前缀如 ¥，后缀如 +、%、ms；小数位数用于成功率类保留一位。</p>
            <div v-for="(m, i) in draft.metrics" :key="i" class="grid grid-cols-[1fr_1fr_80px_80px_70px] items-end gap-3 rounded-md border border-border p-4">
              <div><label class="fld-lbl">标签</label><input v-model="m.label" class="field-input mt-1 w-full" /></div>
              <div><label class="fld-lbl">数值</label><input v-model.number="m.target" type="number" class="field-input mt-1 w-full" /></div>
              <div><label class="fld-lbl">前缀</label><input v-model="m.prefix" class="field-input mt-1 w-full" /></div>
              <div><label class="fld-lbl">后缀</label><input v-model="m.suffix" class="field-input mt-1 w-full" /></div>
              <div><label class="fld-lbl">小数位</label><input v-model.number="m.decimals" type="number" min="0" max="2" class="field-input mt-1 w-full" /></div>
            </div>
          </div>

          <!-- ===== 核心特性 ===== -->
          <div v-show="activeSection === 'features'" class="max-w-3xl space-y-4">
            <div class="row-field"><label class="lbl">板块标题</label><input v-model="draft.featuresTitle" class="field-input flex-1" /></div>
            <div class="row-field"><label class="lbl">板块副标题</label><input v-model="draft.featuresSubtitle" class="field-input flex-1" /></div>
            <div class="space-y-2 border-t border-border/60 pt-4">
              <div v-for="(f, i) in draft.features" :key="i" class="overflow-hidden rounded-md border border-border">
                <div class="collapse-head" :class="isOpen('features', i) ? 'bg-muted/40' : ''" @click="toggle('features', i)">
                  <span class="idx">{{ i + 1 }}</span>
                  <span class="flex-1 truncate text-sm font-medium">{{ f.title || '未命名特性' }}</span>
                  <span class="head-actions" @click.stop>
                    <button class="icon-btn" :disabled="i === 0" @click="moveItem(draft.features, i, -1)"><ArrowUp class="size-3.5" /></button>
                    <button class="icon-btn" :disabled="i === draft.features.length - 1" @click="moveItem(draft.features, i, 1)"><ArrowDown class="size-3.5" /></button>
                    <button class="icon-btn text-destructive" @click="removeItem(draft.features, i)"><Trash2 class="size-3.5" /></button>
                  </span>
                  <ChevronDown class="size-4 shrink-0 text-muted-foreground transition-transform" :class="isOpen('features', i) ? 'rotate-180' : ''" />
                </div>
                <div v-show="isOpen('features', i)" class="space-y-3 border-t border-border p-4">
                  <div class="grid gap-3 sm:grid-cols-[160px_1fr]">
                    <div><label class="fld-lbl">图标</label><Select v-model="f.icon" :options="iconSelectOptions" class="mt-1" /></div>
                    <div><label class="fld-lbl">标题</label><input v-model="f.title" class="field-input mt-1 w-full" /></div>
                  </div>
                  <div><label class="fld-lbl">描述</label><input v-model="f.desc" class="field-input mt-1 w-full" /></div>
                </div>
              </div>
              <Button variant="outline" size="sm" @click="addItem('features', draft.features, blankFeature)"><Plus class="size-4" />添加特性</Button>
            </div>
          </div>

          <!-- ===== 产品矩阵 ===== -->
          <div v-show="activeSection === 'products'" class="max-w-3xl space-y-4">
            <div class="row-field"><label class="lbl">板块标题</label><input v-model="draft.productsTitle" class="field-input flex-1" /></div>
            <div class="row-field"><label class="lbl">板块副标题</label><input v-model="draft.productsSubtitle" class="field-input flex-1" /></div>
            <div class="space-y-2 border-t border-border/60 pt-4">
              <div v-for="(p, i) in draft.products" :key="i" class="overflow-hidden rounded-md border border-border">
                <div class="collapse-head" :class="isOpen('products', i) ? 'bg-muted/40' : ''" @click="toggle('products', i)">
                  <span class="idx">{{ i + 1 }}</span>
                  <span class="flex-1 truncate text-sm font-medium">{{ p.name || '未命名产品' }}</span>
                  <span class="head-actions" @click.stop>
                    <button class="icon-btn" :disabled="i === 0" @click="moveItem(draft.products, i, -1)"><ArrowUp class="size-3.5" /></button>
                    <button class="icon-btn" :disabled="i === draft.products.length - 1" @click="moveItem(draft.products, i, 1)"><ArrowDown class="size-3.5" /></button>
                    <button class="icon-btn text-destructive" @click="removeItem(draft.products, i)"><Trash2 class="size-3.5" /></button>
                  </span>
                  <ChevronDown class="size-4 shrink-0 text-muted-foreground transition-transform" :class="isOpen('products', i) ? 'rotate-180' : ''" />
                </div>
                <div v-show="isOpen('products', i)" class="space-y-3 border-t border-border p-4">
                  <div class="grid gap-3 sm:grid-cols-[160px_1fr]">
                    <div><label class="fld-lbl">图标</label><Select v-model="p.icon" :options="iconSelectOptions" class="mt-1" /></div>
                    <div><label class="fld-lbl">名称</label><input v-model="p.name" class="field-input mt-1 w-full" /></div>
                  </div>
                  <div><label class="fld-lbl">描述</label><input v-model="p.desc" class="field-input mt-1 w-full" /></div>
                  <div><label class="fld-lbl">特性点（顿号 / 逗号分隔）</label>
                    <input :value="joinList(p.points)" class="field-input mt-1 w-full" @input="p.points = splitList(($event.target as HTMLInputElement).value)" /></div>
                  <div><label class="fld-lbl">场景标签（顿号 / 逗号分隔）</label>
                    <input :value="joinList(p.tags)" class="field-input mt-1 w-full" @input="p.tags = splitList(($event.target as HTMLInputElement).value)" /></div>
                </div>
              </div>
              <Button variant="outline" size="sm" @click="addItem('products', draft.products, blankProduct)"><Plus class="size-4" />添加产品</Button>
            </div>
          </div>

          <!-- ===== 费率方案 ===== -->
          <div v-show="activeSection === 'plans'" class="max-w-3xl space-y-4">
            <div class="row-field"><label class="lbl">板块标题</label><input v-model="draft.pricingTitle" class="field-input flex-1" /></div>
            <div class="row-field"><label class="lbl">板块副标题</label><input v-model="draft.pricingSubtitle" class="field-input flex-1" /></div>
            <div class="space-y-2 border-t border-border/60 pt-4">
              <div v-for="(p, i) in draft.plans" :key="i" class="overflow-hidden rounded-md border border-border">
                <div class="collapse-head" :class="isOpen('plans', i) ? 'bg-muted/40' : ''" @click="toggle('plans', i)">
                  <span class="idx">{{ i + 1 }}</span>
                  <span class="flex-1 truncate text-sm font-medium">{{ p.name || '未命名方案' }}</span>
                  <span v-if="p.highlight" class="rounded bg-primary/10 px-1.5 py-0.5 text-[11px] text-primary">推荐</span>
                  <span v-if="p.hidden" class="rounded bg-muted px-1.5 py-0.5 text-[11px] text-muted-foreground">已隐藏</span>
                  <span class="head-actions" @click.stop>
                    <button class="icon-btn" :disabled="i === 0" @click="moveItem(draft.plans, i, -1)"><ArrowUp class="size-3.5" /></button>
                    <button class="icon-btn" :disabled="i === draft.plans.length - 1" @click="moveItem(draft.plans, i, 1)"><ArrowDown class="size-3.5" /></button>
                    <button class="icon-btn text-destructive" @click="removeItem(draft.plans, i)"><Trash2 class="size-3.5" /></button>
                  </span>
                  <ChevronDown class="size-4 shrink-0 text-muted-foreground transition-transform" :class="isOpen('plans', i) ? 'rotate-180' : ''" />
                </div>
                <div v-show="isOpen('plans', i)" class="space-y-3 border-t border-border p-4">
                  <div class="grid gap-3 sm:grid-cols-2">
                    <div><label class="fld-lbl">名称</label><input v-model="p.name" class="field-input mt-1 w-full" /></div>
                    <div><label class="fld-lbl">描述</label><input v-model="p.desc" class="field-input mt-1 w-full" /></div>
                    <div><label class="fld-lbl">费率数字</label><input v-model="p.price" class="field-input mt-1 w-full" /></div>
                    <div><label class="fld-lbl">单位</label><input v-model="p.unit" class="field-input mt-1 w-full" /></div>
                    <div><label class="fld-lbl">按钮文字</label><input v-model="p.cta" class="field-input mt-1 w-full" /></div>
                    <div><label class="fld-lbl">配色主题</label>
                      <Select v-model="p.theme" :options="[{ value: 'gray', label: '灰（普通）' }, { value: 'wechat', label: '微信绿' }, { value: 'alipay', label: '支付宝蓝' }]" class="mt-1" /></div>
                  </div>
                  <div class="flex flex-wrap gap-x-6 gap-y-2">
                    <label class="row-switch text-sm"><Switch v-model="p.highlight" /><span class="text-muted-foreground">标记「推荐」</span></label>
                    <label class="row-switch text-sm"><Switch v-model="p.hidden" /><span class="text-muted-foreground">隐藏此方案</span></label>
                  </div>
                  <div>
                    <div class="fld-lbl mb-1.5">权益明细</div>
                    <div v-for="(ft, fi) in p.features" :key="fi" class="mb-2 flex items-center gap-2">
                      <input v-model="ft.k" placeholder="字段名" class="field-input w-40" />
                      <input v-model="ft.v" placeholder="内容" class="field-input flex-1" />
                      <button class="icon-btn text-destructive" @click="removeItem(p.features, fi)"><Trash2 class="size-3.5" /></button>
                    </div>
                    <button class="text-xs text-primary hover:underline" @click="addItem('plans', p.features, { k: '字段', v: '内容' })">+ 添加权益行</button>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- ===== 客户评价 ===== -->
          <div v-show="activeSection === 'testimonials'" class="max-w-3xl space-y-4">
            <div class="row-field"><label class="lbl">板块标题</label><input v-model="draft.testimonialsTitle" class="field-input flex-1" /></div>
            <div class="row-field"><label class="lbl">板块副标题</label><input v-model="draft.testimonialsSubtitle" class="field-input flex-1" /></div>
            <div class="space-y-2 border-t border-border/60 pt-4">
              <div v-for="(t, i) in draft.testimonials" :key="i" class="overflow-hidden rounded-md border border-border">
                <div class="collapse-head" :class="isOpen('testimonials', i) ? 'bg-muted/40' : ''" @click="toggle('testimonials', i)">
                  <span class="idx">{{ i + 1 }}</span>
                  <span class="flex-1 truncate text-sm font-medium">{{ t.name || '未命名' }} <span class="font-normal text-muted-foreground">· {{ t.role }}</span></span>
                  <span class="head-actions" @click.stop>
                    <button class="icon-btn" :disabled="i === 0" @click="moveItem(draft.testimonials, i, -1)"><ArrowUp class="size-3.5" /></button>
                    <button class="icon-btn" :disabled="i === draft.testimonials.length - 1" @click="moveItem(draft.testimonials, i, 1)"><ArrowDown class="size-3.5" /></button>
                    <button class="icon-btn text-destructive" @click="removeItem(draft.testimonials, i)"><Trash2 class="size-3.5" /></button>
                  </span>
                  <ChevronDown class="size-4 shrink-0 text-muted-foreground transition-transform" :class="isOpen('testimonials', i) ? 'rotate-180' : ''" />
                </div>
                <div v-show="isOpen('testimonials', i)" class="space-y-3 border-t border-border p-4">
                  <div class="grid gap-3 sm:grid-cols-[1fr_1fr_80px]">
                    <div><label class="fld-lbl">姓名</label><input v-model="t.name" class="field-input mt-1 w-full" /></div>
                    <div><label class="fld-lbl">职位</label><input v-model="t.role" class="field-input mt-1 w-full" /></div>
                    <div><label class="fld-lbl">头像字</label><input v-model="t.avatar" maxlength="1" class="field-input mt-1 w-full" /></div>
                  </div>
                  <div><label class="fld-lbl">评价内容</label><textarea v-model="t.text" rows="2" class="field-input mt-1 w-full resize-none py-2" style="height:auto" /></div>
                </div>
              </div>
              <Button variant="outline" size="sm" @click="addItem('testimonials', draft.testimonials, blankTestimonial)"><Plus class="size-4" />添加评价</Button>
            </div>
          </div>

          <!-- ===== 常见问题 ===== -->
          <div v-show="activeSection === 'faqs'" class="max-w-3xl space-y-4">
            <div class="row-field"><label class="lbl">板块标题</label><input v-model="draft.faqTitle" class="field-input flex-1" /></div>
            <div class="row-field"><label class="lbl">板块副标题</label><input v-model="draft.faqSubtitle" class="field-input flex-1" /></div>
            <div class="space-y-2 border-t border-border/60 pt-4">
              <div v-for="(f, i) in draft.faqs" :key="i" class="overflow-hidden rounded-md border border-border">
                <div class="collapse-head" :class="isOpen('faqs', i) ? 'bg-muted/40' : ''" @click="toggle('faqs', i)">
                  <span class="idx">{{ i + 1 }}</span>
                  <span class="flex-1 truncate text-sm font-medium">{{ f.q || '未命名问题' }}</span>
                  <span class="head-actions" @click.stop>
                    <button class="icon-btn" :disabled="i === 0" @click="moveItem(draft.faqs, i, -1)"><ArrowUp class="size-3.5" /></button>
                    <button class="icon-btn" :disabled="i === draft.faqs.length - 1" @click="moveItem(draft.faqs, i, 1)"><ArrowDown class="size-3.5" /></button>
                    <button class="icon-btn text-destructive" @click="removeItem(draft.faqs, i)"><Trash2 class="size-3.5" /></button>
                  </span>
                  <ChevronDown class="size-4 shrink-0 text-muted-foreground transition-transform" :class="isOpen('faqs', i) ? 'rotate-180' : ''" />
                </div>
                <div v-show="isOpen('faqs', i)" class="space-y-3 border-t border-border p-4">
                  <div><label class="fld-lbl">问题</label><input v-model="f.q" class="field-input mt-1 w-full" /></div>
                  <div><label class="fld-lbl">答案</label><textarea v-model="f.a" rows="2" class="field-input mt-1 w-full resize-none py-2" style="height:auto" /></div>
                </div>
              </div>
              <Button variant="outline" size="sm" @click="addItem('faqs', draft.faqs, blankFaq)"><Plus class="size-4" />添加问题</Button>
            </div>
          </div>
        </div>
      </div>
    </Panel>

    <!-- 吸底保存栏 -->
    <div class="fixed inset-x-0 bottom-0 z-30 border-t border-border bg-background/90 backdrop-blur lg:pl-[11.25rem]">
      <div class="flex items-center justify-end gap-3 px-6 py-3">
        <span class="mr-auto text-xs text-muted-foreground">改动即时预览需保存后生效</span>
        <Button variant="outline" @click="resetAll"><RotateCcw class="size-4" />恢复默认</Button>
        <Button @click="save"><Save class="size-4" />保存设置</Button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.collapse-head {
  display: flex;
  cursor: pointer;
  align-items: center;
  gap: 0.625rem;
  padding: 0.625rem 0.875rem;
  transition: background-color 0.15s;
}
.collapse-head:hover {
  background: var(--muted);
}
.idx {
  display: inline-flex;
  height: 1.375rem;
  min-width: 1.375rem;
  align-items: center;
  justify-content: center;
  border-radius: 9999px;
  background: color-mix(in srgb, var(--primary) 10%, transparent);
  padding: 0 0.375rem;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--primary);
}
.head-actions {
  display: inline-flex;
  align-items: center;
  gap: 0.125rem;
}
.fld-lbl {
  font-size: 0.75rem;
  color: var(--muted-foreground);
}
.icon-btn {
  display: inline-flex;
  height: 1.75rem;
  width: 1.75rem;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-sm);
  color: var(--muted-foreground);
  transition: background-color 0.15s, color 0.15s;
}
.icon-btn:hover:not(:disabled) {
  background: var(--accent);
  color: var(--foreground);
}
.icon-btn:disabled {
  opacity: 0.3;
  cursor: not-allowed;
}
</style>
