<script setup lang="ts">
/**
 * 官网管理 · 首页内容（CMS / DIY 拖拽搭建器）。
 * 三区布局：
 *  - 左·板块管理器：拖拽排序 + 显隐开关（写 draft.sections），点选进入编辑；hero 锁定首项、不可隐藏。
 *  - 中·字段编辑区：各板块文案表单；列表型条目（特性/产品/评价/FAQ/权益行）用拖拽排序。
 *  - 右·实时预览：缩放渲染 ClassicSections(draft, preview)，改文案/拖顺序/切显隐即时反映。
 * 编辑草稿：从 store 深拷一份，点「保存」才提交；「恢复默认」重置为出厂内容（含板块顺序）。
 */
import { reactive, ref, computed, onMounted, onBeforeUnmount } from 'vue'
import draggable from 'vuedraggable'
import {
  Save, Plus, Trash2, RotateCcw, ExternalLink, ChevronDown, GripVertical, Eye, EyeOff,
} from 'lucide-vue-next'
import { Panel, Button, Select, Switch } from '@/components/ui'
import { useSiteContentStore } from '@/stores/siteContent'
import { useToast } from '@/composables/useToast'
import { iconOptions, defaultSiteContent, type SiteContent, type SectionKey } from '@/lib/mock/site-content'
import { sectionMetaMap, type SectionMeta } from '@/lib/site-sections'
import ClassicSections from '@/views/site/templates/classic/ClassicSections.vue'

const store = useSiteContentStore()
const toast = useToast()

// 深拷贝草稿
const draft = reactive<SiteContent>(JSON.parse(JSON.stringify(store.content)))

const activeSection = ref<SectionKey>(draft.sections[0]?.key ?? 'hero')

// ===== 实时预览：按列宽等比缩放（填满可用宽度，尽量清晰）=====
const CANVAS_W = 1280 // 预览画布固定内容宽度（模拟桌面官网）
const previewWrap = ref<HTMLElement | null>(null)
const previewScale = ref(0.4)
let ro: ResizeObserver | null = null
function fitPreview() {
  const el = previewWrap.value
  if (!el) return
  // clientWidth 已扣除滚动条（frame 设 overflow-y:scroll + scrollbar-gutter 稳定）
  const w = el.clientWidth
  if (w > 0) previewScale.value = w / CANVAS_W
}
onMounted(() => {
  // 首帧布局未定时 clientWidth 可能为 0，等一帧再量；ResizeObserver 兜后续变化
  requestAnimationFrame(fitPreview)
  ro = new ResizeObserver(fitPreview)
  if (previewWrap.value) ro.observe(previewWrap.value)
  window.addEventListener('resize', fitPreview)
})
onBeforeUnmount(() => {
  ro?.disconnect()
  window.removeEventListener('resize', fitPreview)
})

const iconSelectOptions = iconOptions.map((v) => ({ value: v, label: v }))

// 板块 key → 元信息（draggable 插槽的 element 类型较宽，用此显式收窄）
function meta(key: SectionKey): SectionMeta {
  return sectionMetaMap[key]
}

// 当前选中板块的元信息（名称/图标）
const activeMeta = computed(() => sectionMetaMap[activeSection.value])

// 板块条目数（左侧徽标）
function sectionCount(key: SectionKey): number | null {
  const m = sectionMetaMap[key]
  return m?.count ? m.count(draft) : null
}

// ===== 折叠状态：key = `${section}:${index}` =====
const expanded = reactive<Set<string>>(new Set())
function toggle(section: string, i: number) {
  const k = `${section}:${i}`
  expanded.has(k) ? expanded.delete(k) : expanded.add(k)
}
function isOpen(section: string, i: number) {
  return expanded.has(`${section}:${i}`)
}

// 列表条目无稳定 id：用对象引用映射稳定 key（reorder 仅移动引用，key 不变；新增项得新 key）
const keyMap = new WeakMap<object, number>()
let keySeq = 0
function itemKey(el: unknown): number {
  const o = el as object
  if (!keyMap.has(o)) keyMap.set(o, keySeq++)
  return keyMap.get(o) as number
}

// ===== 数组增删工具（顺序调整交给 draggable）=====
function addItem<T>(section: string, arr: T[], item: T) {
  arr.push(JSON.parse(JSON.stringify(item)))
  expanded.add(`${section}:${arr.length - 1}`) // 新增项自动展开
}
function removeItem<T>(arr: T[], i: number) {
  arr.splice(i, 1)
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
  activeSection.value = draft.sections[0]?.key ?? 'hero'
  toast.info('已恢复默认内容，点保存后生效')
}

// 新增项模板
const blankFeature = { icon: 'Waypoints', title: '新特性', desc: '特性描述' }
const blankProduct = { icon: 'QrCode', name: '新产品', desc: '产品描述', points: ['特性一'], tags: ['标签'] }
const blankTestimonial = { name: '商户名称', desc: '商户简介', image: '' }
const blankFaq = { q: '问题', a: '答案' }
</script>

<template>
  <div class="space-y-2.5 pb-20">
    <Panel title="首页内容" subtitle="拖拽编排板块顺序与显隐，编辑文案，右侧实时预览，保存后官网生效">
      <!-- 顶部操作条 -->
      <div class="mb-4 flex items-center justify-between gap-3">
        <a href="/" target="_blank" class="inline-flex items-center gap-1.5 text-sm text-primary hover:underline">
          <ExternalLink class="size-3.5" />在新窗口打开官网首页
        </a>
        <button class="inline-flex items-center gap-1.5 text-sm text-muted-foreground transition-colors hover:text-foreground" @click="resetAll">
          <RotateCcw class="size-3.5" />恢复默认
        </button>
      </div>

      <div class="grid gap-5 xl:grid-cols-[216px_minmax(0,1fr)_minmax(360px,42%)]">
        <!-- ===== 左：板块管理器（拖拽排序 + 显隐）===== -->
        <div class="xl:sticky xl:top-4 xl:h-max">
          <div class="mb-2 flex items-center justify-between px-1">
            <span class="text-xs font-medium text-muted-foreground">板块编排</span>
            <span class="text-[11px] text-muted-foreground/70">拖拽排序</span>
          </div>
          <draggable
            v-model="draft.sections"
            item-key="key"
            handle=".sec-handle"
            :animation="180"
            ghost-class="sec-ghost"
            class="space-y-1"
          >
            <template #item="{ element: s }">
              <div
                class="sec-row group"
                :class="[
                  activeSection === s.key ? 'sec-row-active' : '',
                  !s.visible ? 'opacity-55' : '',
                ]"
                @click="activeSection = s.key"
              >
                <!-- 拖拽手柄（hero 锁定不可拖）-->
                <button
                  v-if="!meta(s.key).locked"
                  class="sec-handle"
                  aria-label="拖拽排序"
                  @click.stop
                ><GripVertical class="size-4" /></button>
                <span v-else class="sec-handle-locked" title="固定首项"><GripVertical class="size-4" /></span>

                <component :is="meta(s.key).icon" class="size-4 shrink-0" />
                <span class="flex-1 truncate text-sm">{{ meta(s.key).label }}</span>
                <span v-if="sectionCount(s.key) != null" class="rounded-full bg-muted px-1.5 text-xs tabular-nums text-muted-foreground">{{ sectionCount(s.key) }}</span>

                <!-- 显隐开关（hero 不可隐藏）-->
                <button
                  v-if="!meta(s.key).locked"
                  class="sec-eye"
                  :title="s.visible ? '点击隐藏' : '点击显示'"
                  @click.stop="s.visible = !s.visible"
                >
                  <Eye v-if="s.visible" class="size-4" />
                  <EyeOff v-else class="size-4" />
                </button>
              </div>
            </template>
          </draggable>
        </div>

        <!-- ===== 中：字段编辑区 ===== -->
        <div class="min-w-0">
          <div class="mb-4 flex items-center gap-2 border-b border-border/60 pb-3">
            <component :is="activeMeta.icon" class="size-4 text-primary" />
            <h3 class="text-sm font-semibold">{{ activeMeta.label }}</h3>
          </div>

          <!-- 首屏 Hero -->
          <div v-show="activeSection === 'hero'" class="space-y-3.5">
            <div class="row-field"><label class="lbl">标签徽标</label><input v-model="draft.hero.badge" class="field-input flex-1" /></div>
            <div class="row-field"><label class="lbl">主标题上行</label><input v-model="draft.hero.titleLead" class="field-input flex-1" /></div>
            <div class="row-field"><label class="lbl">主标题高亮</label><input v-model="draft.hero.titleAccent" class="field-input flex-1" /></div>
            <div class="row-field"><label class="lbl">副标题</label><textarea v-model="draft.hero.subtitle" rows="3" class="field-input flex-1 resize-none py-2" style="height:auto" /></div>
            <div class="row-field"><label class="lbl">主按钮文字</label><input v-model="draft.hero.ctaPrimary" class="field-input flex-1" /></div>
            <div class="row-field"><label class="lbl">次按钮文字</label><input v-model="draft.hero.ctaSecondary" class="field-input flex-1" /></div>
            <div class="row-field"><label class="lbl">渠道栏文字</label><input v-model="draft.hero.payMethodsLabel" class="field-input flex-1" /></div>
          </div>

          <!-- 数据背书 -->
          <div v-show="activeSection === 'metrics'" class="space-y-4">
            <p class="text-xs text-muted-foreground">4 个滚动数字。数值填目标数字，前缀如 ¥，后缀如 +、%、ms；小数位数用于成功率类保留一位。</p>
            <div v-for="(m, i) in draft.metrics" :key="i" class="grid grid-cols-[1fr_1fr_70px_70px_60px] items-end gap-3 rounded-md border border-border p-4">
              <div><label class="fld-lbl">标签</label><input v-model="m.label" class="field-input mt-1 w-full" /></div>
              <div><label class="fld-lbl">数值</label><input v-model.number="m.target" type="number" class="field-input mt-1 w-full" /></div>
              <div><label class="fld-lbl">前缀</label><input v-model="m.prefix" class="field-input mt-1 w-full" /></div>
              <div><label class="fld-lbl">后缀</label><input v-model="m.suffix" class="field-input mt-1 w-full" /></div>
              <div><label class="fld-lbl">小数位</label><input v-model.number="m.decimals" type="number" min="0" max="2" class="field-input mt-1 w-full" /></div>
            </div>
          </div>

          <!-- 核心特性 -->
          <div v-show="activeSection === 'features'" class="space-y-4">
            <div class="row-field"><label class="lbl">板块标题</label><input v-model="draft.featuresTitle" class="field-input flex-1" /></div>
            <div class="row-field"><label class="lbl">板块副标题</label><input v-model="draft.featuresSubtitle" class="field-input flex-1" /></div>
            <div class="space-y-2 border-t border-border/60 pt-4">
              <draggable v-model="draft.features" :item-key="itemKey" handle=".item-handle" :animation="180" ghost-class="sec-ghost" class="space-y-2">
                <template #item="{ element: f, index: i }">
                  <div class="overflow-hidden rounded-md border border-border">
                    <div class="collapse-head" :class="isOpen('features', i) ? 'bg-muted/40' : ''" @click="toggle('features', i)">
                      <button class="item-handle" @click.stop><GripVertical class="size-4" /></button>
                      <span class="flex-1 truncate text-sm font-medium">{{ f.title || '未命名特性' }}</span>
                      <span class="head-actions" @click.stop>
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
                </template>
              </draggable>
              <Button variant="outline" size="sm" @click="addItem('features', draft.features, blankFeature)"><Plus class="size-4" />添加特性</Button>
            </div>
          </div>

          <!-- 产品矩阵 -->
          <div v-show="activeSection === 'products'" class="space-y-4">
            <div class="row-field"><label class="lbl">板块标题</label><input v-model="draft.productsTitle" class="field-input flex-1" /></div>
            <div class="row-field"><label class="lbl">板块副标题</label><input v-model="draft.productsSubtitle" class="field-input flex-1" /></div>
            <div class="space-y-2 border-t border-border/60 pt-4">
              <draggable v-model="draft.products" :item-key="itemKey" handle=".item-handle" :animation="180" ghost-class="sec-ghost" class="space-y-2">
                <template #item="{ element: p, index: i }">
                  <div class="overflow-hidden rounded-md border border-border">
                    <div class="collapse-head" :class="isOpen('products', i) ? 'bg-muted/40' : ''" @click="toggle('products', i)">
                      <button class="item-handle" @click.stop><GripVertical class="size-4" /></button>
                      <span class="flex-1 truncate text-sm font-medium">{{ p.name || '未命名产品' }}</span>
                      <span class="head-actions" @click.stop>
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
                </template>
              </draggable>
              <Button variant="outline" size="sm" @click="addItem('products', draft.products, blankProduct)"><Plus class="size-4" />添加产品</Button>
            </div>
          </div>

          <!-- 费率方案 -->
          <div v-show="activeSection === 'pricing'" class="space-y-4">
            <div class="row-field"><label class="lbl">板块标题</label><input v-model="draft.pricingTitle" class="field-input flex-1" /></div>
            <div class="row-field"><label class="lbl">板块副标题</label><input v-model="draft.pricingSubtitle" class="field-input flex-1" /></div>
            <div class="space-y-2 border-t border-border/60 pt-4">
              <draggable v-model="draft.plans" :item-key="itemKey" handle=".item-handle" :animation="180" ghost-class="sec-ghost" class="space-y-2">
                <template #item="{ element: p, index: i }">
                  <div class="overflow-hidden rounded-md border border-border">
                    <div class="collapse-head" :class="isOpen('plans', i) ? 'bg-muted/40' : ''" @click="toggle('plans', i)">
                      <button class="item-handle" @click.stop><GripVertical class="size-4" /></button>
                      <span class="flex-1 truncate text-sm font-medium">{{ p.name || '未命名方案' }}</span>
                      <span v-if="p.highlight" class="rounded bg-primary/10 px-1.5 py-0.5 text-[11px] text-primary">推荐</span>
                      <span v-if="p.hidden" class="rounded bg-muted px-1.5 py-0.5 text-[11px] text-muted-foreground">已隐藏</span>
                      <span class="head-actions" @click.stop>
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
                          <button class="icon-btn text-destructive" @click="removeItem(p.features, Number(fi))"><Trash2 class="size-3.5" /></button>
                        </div>
                        <button class="text-xs text-primary hover:underline" @click="addItem('plans', p.features, { k: '字段', v: '内容' })">+ 添加权益行</button>
                      </div>
                    </div>
                  </div>
                </template>
              </draggable>
            </div>
          </div>

          <!-- 最新动态 -->
          <div v-show="activeSection === 'news'" class="space-y-4">
            <div class="row-field"><label class="lbl">板块标题</label><input v-model="draft.newsTitle" class="field-input flex-1" /></div>
            <div class="row-field"><label class="lbl">板块副标题</label><input v-model="draft.newsSubtitle" class="field-input flex-1" /></div>
            <div class="flex items-start gap-2 rounded-md border border-border bg-muted/30 p-4 text-sm text-muted-foreground">
              <component :is="sectionMetaMap.news.icon" class="mt-0.5 size-4 shrink-0" />
              <div>
                这里只编辑板块标题文案。具体的<span class="font-medium text-foreground">文章内容与分类</span>请到
                <RouterLink to="/admin/articles" class="text-primary hover:underline">「文章管理」</RouterLink>
                维护，保存后在官网「最新动态」板块按分类分列展示。
              </div>
            </div>
          </div>

          <!-- 客户评价 -->
          <div v-show="activeSection === 'testimonials'" class="space-y-4">
            <div class="row-field"><label class="lbl">板块标题</label><input v-model="draft.testimonialsTitle" class="field-input flex-1" /></div>
            <div class="row-field"><label class="lbl">板块副标题</label><input v-model="draft.testimonialsSubtitle" class="field-input flex-1" /></div>
            <div class="space-y-2 border-t border-border/60 pt-4">
              <draggable v-model="draft.testimonials" :item-key="itemKey" handle=".item-handle" :animation="180" ghost-class="sec-ghost" class="space-y-2">
                <template #item="{ element: t, index: i }">
                  <div class="overflow-hidden rounded-md border border-border">
                    <div class="collapse-head" :class="isOpen('testimonials', i) ? 'bg-muted/40' : ''" @click="toggle('testimonials', i)">
                      <button class="item-handle" @click.stop><GripVertical class="size-4" /></button>
                      <span class="flex-1 truncate text-sm font-medium">{{ t.name || '未命名' }}</span>
                      <span class="head-actions" @click.stop>
                        <button class="icon-btn text-destructive" @click="removeItem(draft.testimonials, i)"><Trash2 class="size-3.5" /></button>
                      </span>
                      <ChevronDown class="size-4 shrink-0 text-muted-foreground transition-transform" :class="isOpen('testimonials', i) ? 'rotate-180' : ''" />
                    </div>
                    <div v-show="isOpen('testimonials', i)" class="space-y-3 border-t border-border p-4">
                      <div><label class="fld-lbl">商户名称</label><input v-model="t.name" class="field-input mt-1 w-full" /></div>
                      <div><label class="fld-lbl">门店图片地址</label><input v-model="t.image" placeholder="/assets/xxx.jpg（留空显示占位图块）" class="field-input mt-1 w-full" /></div>
                      <div><label class="fld-lbl">商户简介</label><textarea v-model="t.desc" rows="2" class="field-input mt-1 w-full resize-none py-2" style="height:auto" /></div>
                    </div>
                  </div>
                </template>
              </draggable>
              <Button variant="outline" size="sm" @click="addItem('testimonials', draft.testimonials, blankTestimonial)"><Plus class="size-4" />添加评价</Button>
            </div>
          </div>

          <!-- 常见问题 -->
          <div v-show="activeSection === 'faqs'" class="space-y-4">
            <div class="row-field"><label class="lbl">板块标题</label><input v-model="draft.faqTitle" class="field-input flex-1" /></div>
            <div class="row-field"><label class="lbl">板块副标题</label><input v-model="draft.faqSubtitle" class="field-input flex-1" /></div>
            <div class="space-y-2 border-t border-border/60 pt-4">
              <draggable v-model="draft.faqs" :item-key="itemKey" handle=".item-handle" :animation="180" ghost-class="sec-ghost" class="space-y-2">
                <template #item="{ element: f, index: i }">
                  <div class="overflow-hidden rounded-md border border-border">
                    <div class="collapse-head" :class="isOpen('faqs', i) ? 'bg-muted/40' : ''" @click="toggle('faqs', i)">
                      <button class="item-handle" @click.stop><GripVertical class="size-4" /></button>
                      <span class="flex-1 truncate text-sm font-medium">{{ f.q || '未命名问题' }}</span>
                      <span class="head-actions" @click.stop>
                        <button class="icon-btn text-destructive" @click="removeItem(draft.faqs, i)"><Trash2 class="size-3.5" /></button>
                      </span>
                      <ChevronDown class="size-4 shrink-0 text-muted-foreground transition-transform" :class="isOpen('faqs', i) ? 'rotate-180' : ''" />
                    </div>
                    <div v-show="isOpen('faqs', i)" class="space-y-3 border-t border-border p-4">
                      <div><label class="fld-lbl">问题</label><input v-model="f.q" class="field-input mt-1 w-full" /></div>
                      <div><label class="fld-lbl">答案</label><textarea v-model="f.a" rows="2" class="field-input mt-1 w-full resize-none py-2" style="height:auto" /></div>
                    </div>
                  </div>
                </template>
              </draggable>
              <Button variant="outline" size="sm" @click="addItem('faqs', draft.faqs, blankFaq)"><Plus class="size-4" />添加问题</Button>
            </div>
          </div>

          <!-- 底部 CTA -->
          <div v-show="activeSection === 'cta'" class="space-y-3.5">
            <div class="row-field"><label class="lbl">标题</label><input v-model="draft.cta.title" class="field-input flex-1" /></div>
            <div class="row-field"><label class="lbl">副标题</label><textarea v-model="draft.cta.subtitle" rows="2" class="field-input flex-1 resize-none py-2" style="height:auto" /></div>
            <div class="row-field"><label class="lbl">主按钮文字</label><input v-model="draft.cta.ctaPrimary" class="field-input flex-1" /></div>
            <div class="row-field"><label class="lbl">次按钮文字</label><input v-model="draft.cta.ctaSecondary" class="field-input flex-1" /></div>
          </div>
        </div>

        <!-- ===== 右：实时预览 ===== -->
        <div class="hidden xl:block">
          <div class="sticky top-4">
            <div class="mb-2 flex items-center gap-2 px-1">
              <span class="text-xs font-medium text-muted-foreground">实时预览</span>
              <span class="text-[11px] text-muted-foreground/70">缩略示意，最终以官网为准</span>
            </div>
            <div ref="previewWrap" class="preview-frame">
              <div class="preview-canvas" :style="{ width: CANVAS_W + 'px', zoom: previewScale }">
                <ClassicSections :content="draft" preview />
              </div>
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
/* ===== 板块管理器行 ===== */
.sec-row {
  display: flex;
  cursor: pointer;
  align-items: center;
  gap: 0.5rem;
  border-radius: var(--radius-sm);
  padding: 0.5rem 0.5rem 0.5rem 0.25rem;
  color: var(--muted-foreground);
  transition: background-color 0.15s, color 0.15s;
}
.sec-row:hover {
  background: var(--accent);
  color: var(--foreground);
}
.sec-row-active {
  background: color-mix(in srgb, var(--primary) 8%, transparent);
  color: var(--primary);
  font-weight: 500;
}
.sec-ghost {
  opacity: 0.4;
}
.sec-handle,
.sec-handle-locked,
.item-handle {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--muted-foreground);
  opacity: 0.5;
}
.sec-handle,
.item-handle {
  cursor: grab;
}
.sec-handle:hover,
.item-handle:hover {
  opacity: 1;
}
.sec-handle:active,
.item-handle:active {
  cursor: grabbing;
}
.sec-handle-locked {
  opacity: 0.2;
  cursor: not-allowed;
}
.sec-eye {
  display: inline-flex;
  height: 1.5rem;
  width: 1.5rem;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-sm);
  color: var(--muted-foreground);
  opacity: 0;
  transition: opacity 0.15s, background-color 0.15s, color 0.15s;
}
.sec-row:hover .sec-eye,
.sec-row-active .sec-eye {
  opacity: 0.7;
}
.sec-eye:hover {
  opacity: 1;
  background: var(--accent);
  color: var(--foreground);
}

/* ===== 列表条目折叠头 ===== */
.collapse-head {
  display: flex;
  cursor: pointer;
  align-items: center;
  gap: 0.5rem;
  padding: 0.625rem 0.75rem;
  transition: background-color 0.15s;
}
.collapse-head:hover {
  background: var(--muted);
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

/* ===== 右侧实时预览：定宽画布 + 按列宽等比缩放 ===== */
.preview-frame {
  height: calc(100vh - 8.5rem);
  overflow-x: hidden;
  overflow-y: scroll; /* 常驻纵向滚动条，配 scrollbar-gutter 让 clientWidth 稳定不抖 */
  scrollbar-gutter: stable;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--background);
}
/* 用 zoom 而非 transform：zoom 会等比压缩布局盒，滚动高度自动正确，无需负 margin 兜底 */
.preview-canvas {
  pointer-events: none;
}
</style>
