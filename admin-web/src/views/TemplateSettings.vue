<script setup lang="ts">
/**
 * 官网管理 · 首页模板配置（主推 + 副列）。
 * 左·主预览：聚焦当前选中模板。default 为实时缩略（按框宽等比 zoom 渲染真实首页
 *   ClassicSections，始终与官网同步，无需维护截图）；index1~10 暂无组件，用精致占位。
 * 右·副列：全部模板竖列，点选切换左侧预览；行内小缩略 + 名称 + 层级/状态标签。
 * 「应用模板」把选中模板设为当前使用并持久化（localStorage），toast 反馈。
 * 数据源整合自 site/registry.ts（default 取 classic 的名称/描述/层级），epay 皮肤 id 沿用 default+index1~10。
 */
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { Check, LayoutTemplate, ExternalLink, Sparkles } from 'lucide-vue-next'
import { Panel, Button, Badge } from '@/components/ui'
import { useToast } from '@/composables/useToast'
import { useSiteContentStore } from '@/stores/siteContent'
import { siteTemplates, tierMeta, type TemplateTier } from '@/views/site/registry'
import ClassicSections from '@/views/site/templates/classic/ClassicSections.vue'

const toast = useToast()

// ===== 模板清单：default 取 classic 真实元信息，index1~10 为占位 =====
interface TemplateEntry {
  id: string          // epay 皮肤 id（default / index1~10）
  name: string        // 展示名
  desc: string        // 风格一句话
  tier: TemplateTier  // 售卖层级
  ready: boolean      // 是否已实现（有组件）
}
const classic = siteTemplates[0]
const templates: TemplateEntry[] = [
  { id: 'default', name: classic.name, desc: classic.desc, tier: classic.tier, ready: true },
  ...Array.from({ length: 10 }, (_, i) => ({
    id: `index${i + 1}`,
    name: `index${i + 1}`,
    desc: '敬请期待',
    tier: 'free' as TemplateTier,
    ready: false,
  })),
]
const entryMap = Object.fromEntries(templates.map((t) => [t.id, t])) as Record<string, TemplateEntry>

// ===== 当前应用 / 选中态 =====
const STORAGE_KEY = 'site-template'
const applied = ref(localStorage.getItem(STORAGE_KEY) || 'default') // 官网正在用的模板
const active = ref(applied.value) // 左侧预览聚焦的模板（点右列切换）
const activeEntry = computed(() => entryMap[active.value])

function apply() {
  const t = entryMap[active.value]
  if (!t.ready) {
    toast.info(`${t.name} 尚未上线，暂不可应用`)
    return
  }
  applied.value = active.value
  localStorage.setItem(STORAGE_KEY, applied.value)
  toast.success(`已应用「${t.name}」为官网首页模板`)
}

// ===== 左侧主预览：实时缩略（default）=====
const content = useSiteContentStore().content
const CANVAS_W = 1280 // 缩略画布固定内容宽度（模拟桌面官网）
const previewWrap = ref<HTMLElement | null>(null)
const previewZoom = ref(0.4)
let ro: ResizeObserver | null = null
function fitPreview() {
  const w = previewWrap.value?.clientWidth
  if (w && w > 0) previewZoom.value = w / CANVAS_W
}
onMounted(() => {
  requestAnimationFrame(fitPreview) // 首帧布局未定时 clientWidth 可能为 0，等一帧再量
  ro = new ResizeObserver(fitPreview)
  if (previewWrap.value) ro.observe(previewWrap.value)
  window.addEventListener('resize', fitPreview)
})
onBeforeUnmount(() => {
  ro?.disconnect()
  window.removeEventListener('resize', fitPreview)
})
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="首页模板配置" :subtitle="`共 ${templates.length} 套官网首页模板`">
      <template #actions>
        <a
          href="/"
          target="_blank"
          class="mr-1 inline-flex items-center gap-1.5 text-sm text-muted-foreground transition-colors hover:text-foreground"
        >
          <ExternalLink class="size-3.5" />预览官网
        </a>
        <Button size="sm" :disabled="active === applied || !activeEntry.ready" @click="apply">
          <Check class="size-4" />{{ active === applied ? '当前已应用' : !activeEntry.ready ? '暂不可用' : '应用模板' }}
        </Button>
      </template>

      <div class="grid gap-5 lg:grid-cols-[minmax(0,1fr)_300px]">
        <!-- ===== 左：主预览 ===== -->
        <div class="min-w-0">
          <!-- default：真实首页实时缩略 -->
          <div v-if="activeEntry.ready" ref="previewWrap" class="preview-frame">
            <div class="preview-canvas" :style="{ width: CANVAS_W + 'px', zoom: previewZoom }">
              <ClassicSections :content="content" preview />
            </div>
          </div>
          <!-- 占位：精致渐变 + 图标 -->
          <div v-else class="placeholder-frame">
            <div class="flex flex-col items-center gap-3 text-muted-foreground/45">
              <LayoutTemplate class="size-12" :stroke-width="1.25" />
              <span class="text-sm">{{ activeEntry.name }} · 模板即将上线</span>
            </div>
          </div>

          <!-- 主预览信息条 -->
          <div class="mt-3 flex flex-wrap items-center gap-x-3 gap-y-2">
            <h3 class="text-base font-semibold">{{ activeEntry.name }}</h3>
            <Badge :variant="tierMeta[activeEntry.tier].variant">{{ tierMeta[activeEntry.tier].text }}</Badge>
            <Badge v-if="active === applied" variant="success">使用中</Badge>
            <Badge v-else-if="!activeEntry.ready" variant="muted">即将上线</Badge>
            <span class="w-full text-sm leading-relaxed text-muted-foreground sm:w-auto sm:flex-1">{{ activeEntry.desc }}</span>
          </div>
        </div>

        <!-- ===== 右：模板副列 ===== -->
        <div class="lg:border-l lg:border-border/60 lg:pl-5">
          <div class="mb-2.5 flex items-center gap-1.5 px-0.5 text-xs font-medium text-muted-foreground">
            <Sparkles class="size-3.5" />全部模板
          </div>
          <div class="tpl-list">
            <button
              v-for="t in templates"
              :key="t.id"
              type="button"
              class="tpl-row"
              :class="active === t.id ? 'tpl-row-active' : ''"
              @click="active = t.id"
            >
              <!-- 行缩略：default 用品牌色迷你官网草图；占位用图标 -->
              <span class="tpl-thumb">
                <template v-if="t.ready">
                  <span class="tnav"><i /><i class="g" /><i class="g" /></span>
                  <span class="thero">
                    <span class="tcol"><b class="blue" /><b /><b class="sm" /></span>
                    <span class="tcard" />
                  </span>
                </template>
                <LayoutTemplate v-else class="size-5 text-muted-foreground/35" :stroke-width="1.5" />
              </span>

              <span class="min-w-0 flex-1">
                <span class="flex items-center gap-1.5">
                  <span class="truncate text-sm font-medium" :class="t.ready ? '' : 'text-muted-foreground'">{{ t.name }}</span>
                  <Check v-if="applied === t.id" class="size-3.5 shrink-0 text-[#67C23A]" :stroke-width="2.75" />
                </span>
                <span class="mt-0.5 block truncate text-xs text-muted-foreground/70">{{ t.ready ? tierMeta[t.tier].text : '即将上线' }}</span>
              </span>
            </button>
          </div>
        </div>
      </div>

      <p class="mt-4 border-t border-border/60 pt-4 text-xs leading-relaxed text-muted-foreground">
        模板对应 template/ 目录下的皮肤（default 及 index1~10）。default 为当前 SaaS 官网风，左侧预览实时反映官网首页内容；选中后点「应用模板」切换官网首页外观。
      </p>
    </Panel>
  </div>
</template>

<style scoped>
::selection {
  background: color-mix(in srgb, var(--primary) 18%, transparent);
}

/* ===== 左：主预览 ===== */
.preview-frame {
  height: 460px;
  overflow-x: hidden;
  overflow-y: scroll; /* 常驻纵向滚动条，配 scrollbar-gutter 让 clientWidth 稳定不抖 */
  scrollbar-gutter: stable;
  border: 1px solid var(--border);
  background: var(--background);
}
/* 用 zoom 而非 transform：zoom 会等比压缩布局盒，滚动高度自动正确 */
.preview-canvas {
  pointer-events: none;
}
.placeholder-frame {
  display: flex;
  height: 460px;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--border);
  background: linear-gradient(150deg,
    color-mix(in srgb, var(--primary) 6%, var(--background)),
    var(--background) 70%);
}

/* ===== 右：模板副列 ===== */
.tpl-list {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
  max-height: 470px;
  overflow-y: auto;
  padding-right: 0.125rem;
}
.tpl-row {
  display: flex;
  align-items: center;
  gap: 0.625rem;
  border: 1px solid var(--border);
  background: var(--background);
  padding: 0.5rem 0.625rem;
  text-align: left;
  transition: background-color 0.14s ease, border-color 0.14s ease;
}
.tpl-row:focus-visible {
  outline: 2px solid color-mix(in srgb, var(--primary) 55%, transparent);
  outline-offset: 1px;
}
@media (hover: hover) and (pointer: fine) {
  .tpl-row:hover {
    background: var(--accent);
    border-color: color-mix(in srgb, var(--primary) 30%, var(--border));
  }
}
/* 选中态：整块淡蓝底 + 蓝边（无左侧竖线，无阴影，遵守后台扁平规范）*/
.tpl-row-active {
  background: color-mix(in srgb, var(--primary) 8%, transparent);
  border-color: var(--primary);
}

/* 行缩略：品牌色迷你官网草图（default）*/
.tpl-thumb {
  display: flex;
  height: 2.5rem;
  width: 3.5rem;
  flex: none;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  border: 1px solid var(--border);
  background: #fff;
}
.tnav {
  display: flex;
  width: 100%;
  height: 0.4rem;
  align-items: center;
  gap: 2px;
  border-bottom: 1px solid var(--line, #eef0f5);
  padding: 0 3px;
}
.tnav i { height: 2px; width: 8px; border-radius: 1px; background: var(--primary); }
.tnav i.g { width: 5px; background: #d9dde6; }
.thero {
  display: grid;
  width: 100%;
  height: 100%;
  grid-template-columns: 1.2fr 1fr;
  gap: 3px;
  padding: 4px;
}
.tcol { display: flex; flex-direction: column; gap: 2px; }
.tcol b { height: 2.5px; width: 100%; border-radius: 1px; background: #dfe3ec; }
.tcol b.blue { background: var(--primary); width: 70%; }
.tcol b.sm { width: 55%; }
.tcard {
  border-radius: 1px;
  background: linear-gradient(150deg, color-mix(in srgb, var(--primary) 16%, #fff), #f6f8fc);
  border: 1px solid #e5ebf6;
}

@media (prefers-reduced-motion: reduce) {
  .tpl-row { transition: none; }
}
</style>
