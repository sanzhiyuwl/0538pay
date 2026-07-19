<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import type { Component } from 'vue'
import {
  ArrowLeft, ArrowLeftRight, ArrowRight, BookOpen, ChevronDown, CreditCard,
  FileText, HelpCircle, List, Rocket, Search, Store, Zap,
} from 'lucide-vue-next'
import DocPageContent from '@/components/site-docs/DocPageContent.vue'
import { pageSearchText } from '@/lib/site-docs'
import { useSiteDocsStore } from '@/stores/siteDocs'

const route = useRoute()
const router = useRouter()
const store = useSiteDocsStore()

const iconMap: Record<string, Component> = {
  FileText,
  Rocket,
  CreditCard,
  Store,
  ArrowLeftRight,
  BookOpen,
  HelpCircle,
}

const groups = computed(() =>
  store.publishedGroups.map((entry) => ({
    ...entry,
    icon: iconMap[entry.group.icon] ?? FileText,
  })),
)
const flatPages = computed(() => store.publishedPages)
const requestedSlug = computed(() => typeof route.query.p === 'string' ? route.query.p : '')
const activePage = computed(() => {
  const requested = flatPages.value.find((page) => page.slug === requestedSlug.value)
  if (requested) return requested
  return flatPages.value.find((page) => page.slug === store.settings.defaultSlug) ?? flatPages.value[0]
})
const activeGroup = computed(() =>
  groups.value.find((entry) => entry.group.id === activePage.value?.groupId),
)

watch(
  [requestedSlug, activePage],
  ([requested, page]) => {
    if (page && requested !== page.slug) {
      void router.replace({ query: { ...route.query, p: page.slug } })
    }
  },
  { immediate: true },
)

const collapsed = ref<Set<string>>(new Set())
function toggleGroup(id: string) {
  const next = new Set(collapsed.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  collapsed.value = next
}

const keyword = ref('')
const searching = computed(() => keyword.value.trim().length > 0)
const searchResults = computed(() => {
  const query = keyword.value.trim().toLowerCase()
  if (!query) return []
  return flatPages.value.filter((page) => pageSearchText(page).includes(query))
})

const activeOutline = computed(() =>
  activePage.value?.sections
    .filter((section) => section.showInOutline)
    .map((section) => ({ id: section.anchor, title: section.title || activePage.value?.title || '' })) ?? [],
)
function scrollToAnchor(id: string) {
  document.getElementById(id)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

function goPage(slug: string) {
  keyword.value = ''
  void router.replace({ query: { ...route.query, p: slug } })
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

const activeIndex = computed(() =>
  activePage.value ? flatPages.value.findIndex((page) => page.id === activePage.value?.id) : -1,
)
const prevPage = computed(() => activeIndex.value > 0 ? flatPages.value[activeIndex.value - 1] : undefined)
const nextPage = computed(() =>
  activeIndex.value >= 0 && activeIndex.value < flatPages.value.length - 1
    ? flatPages.value[activeIndex.value + 1]
    : undefined,
)
</script>

<template>
  <div class="flex min-h-screen flex-col bg-background">
    <header class="sticky top-0 z-40 border-b border-border/50 bg-background">
      <div class="flex h-16 items-center justify-between px-5 lg:px-8">
        <RouterLink to="/" class="flex shrink-0 items-center gap-2">
          <div class="flex size-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
            <Zap class="size-[18px]" />
          </div>
          <span class="text-lg font-bold tracking-tight">0538<span class="text-primary">Pay</span></span>
          <span class="ml-2.5 hidden pl-2.5 text-sm text-muted-foreground/80 sm:inline">{{ store.settings.title }}</span>
        </RouterLink>

        <nav class="flex items-center gap-6 text-sm">
          <RouterLink to="/" class="text-muted-foreground transition-colors hover:text-foreground">返回官网</RouterLink>
          <RouterLink to="/about" class="hidden text-muted-foreground transition-colors hover:text-foreground sm:inline">关于我们</RouterLink>
          <RouterLink to="/m/login" class="text-muted-foreground transition-colors hover:text-foreground">登录</RouterLink>
          <RouterLink to="/m/reg" class="rounded-lg bg-primary px-4 py-1.5 font-medium text-primary-foreground transition-opacity hover:opacity-90">免费注册</RouterLink>
        </nav>
      </div>
    </header>

    <div class="flex w-full flex-1">
      <aside class="hidden w-64 shrink-0 border-r border-border/50 lg:block">
        <div class="docs-nav sticky top-16 overflow-y-auto py-6 [max-height:calc(100vh-4rem)] [scrollbar-gutter:stable]">
          <div class="px-4 pb-4">
            <div class="relative">
              <Search class="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
              <input
                v-model="keyword"
                type="text"
                placeholder="搜索文档"
                class="h-9 w-full border border-border bg-content pl-9 pr-3 text-sm outline-none transition-colors placeholder:text-muted-foreground/70 focus:border-primary/50 focus:bg-background"
              />
            </div>
          </div>

          <nav v-if="searching" class="space-y-0.5">
            <button
              v-for="page in searchResults"
              :key="page.id"
              class="block w-full border-l-2 px-6 py-1.5 text-left text-[13px] transition-colors"
              :class="activePage?.id === page.id
                ? 'border-primary bg-primary/[0.08] font-medium text-primary'
                : 'border-transparent text-muted-foreground hover:bg-accent hover:text-foreground'"
              @click="goPage(page.slug)"
            >
              {{ page.title }}
            </button>
            <p v-if="searchResults.length === 0" class="px-6 py-3 text-[13px] text-muted-foreground">没有匹配的文档</p>
          </nav>

          <nav v-else class="space-y-0.5">
            <div v-for="entry in groups" :key="entry.group.id">
              <button
                class="flex w-full items-center gap-2 px-4 py-2 text-left text-sm font-medium transition-colors hover:text-foreground"
                :class="activeGroup?.group.id === entry.group.id ? 'text-foreground' : 'text-muted-foreground'"
                @click="toggleGroup(entry.group.id)"
              >
                <component :is="entry.icon" class="size-4 shrink-0" :stroke-width="1.75" />
                <span class="flex-1">{{ entry.group.name }}</span>
                <ChevronDown class="size-3.5 transition-transform" :class="collapsed.has(entry.group.id) && '-rotate-90'" />
              </button>
              <div v-show="!collapsed.has(entry.group.id)" class="pb-1">
                <button
                  v-for="page in entry.pages"
                  :key="page.id"
                  class="block w-full border-l-2 py-1.5 pl-10 pr-4 text-left text-[13px] transition-colors"
                  :class="activePage?.id === page.id
                    ? 'border-primary bg-primary/[0.08] font-medium text-primary'
                    : 'border-transparent text-muted-foreground hover:bg-accent hover:text-foreground'"
                  @click="goPage(page.slug)"
                >
                  {{ page.title }}
                </button>
              </div>
            </div>
          </nav>
        </div>
      </aside>

      <main class="min-w-0 flex-1">
        <div v-if="activePage" class="px-8 py-12 lg:px-14">
          <header class="mb-12">
            <div class="flex items-center gap-1.5 text-sm text-muted-foreground">
              <span>{{ activeGroup?.group.name }}</span>
              <ArrowRight class="size-3.5" />
              <span class="text-foreground">{{ activePage.title }}</span>
            </div>
            <p class="mt-2 text-xs text-muted-foreground">{{ store.settings.subtitle }}</p>
          </header>

          <DocPageContent :page="activePage" />

          <nav class="mt-16 flex items-center justify-between border-t border-border pt-6">
            <button
              v-if="prevPage"
              class="group flex items-center gap-2 text-sm text-muted-foreground transition-colors hover:text-primary"
              @click="goPage(prevPage.slug)"
            >
              <ArrowLeft class="size-4 transition-transform group-hover:-translate-x-0.5" />
              <span>{{ prevPage.title }}</span>
            </button>
            <span v-else />
            <button
              v-if="nextPage"
              class="group flex items-center gap-2 text-sm text-muted-foreground transition-colors hover:text-primary"
              @click="goPage(nextPage.slug)"
            >
              <span>{{ nextPage.title }}</span>
              <ArrowRight class="size-4 transition-transform group-hover:translate-x-0.5" />
            </button>
          </nav>
        </div>

        <div v-else class="flex min-h-[60vh] items-center justify-center px-8 text-sm text-muted-foreground">
          暂无已发布的开发文档
        </div>
      </main>

      <aside class="hidden w-52 shrink-0 xl:block">
        <div v-if="activePage" class="sticky top-16 py-12 pr-6">
          <div class="mb-3 flex items-center gap-1.5 text-sm font-semibold text-foreground">
            <List class="size-4 text-muted-foreground" :stroke-width="1.75" />
            大纲
          </div>
          <nav class="space-y-1 border-l border-border">
            <button
              v-for="item in activeOutline"
              :key="item.id"
              class="-ml-px block w-full border-l-2 border-transparent px-3 py-1 text-left text-[13px] text-muted-foreground transition-colors hover:border-primary/40 hover:text-primary"
              @click="scrollToAnchor(item.id)"
            >
              {{ item.title }}
            </button>
          </nav>
        </div>
      </aside>
    </div>
  </div>
</template>

<style scoped>
.docs-nav {
  -webkit-mask-image: linear-gradient(to bottom, transparent 0, #000 16px, #000 calc(100% - 20px), transparent 100%);
  mask-image: linear-gradient(to bottom, transparent 0, #000 16px, #000 calc(100% - 20px), transparent 100%);
}
</style>
