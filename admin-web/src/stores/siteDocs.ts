import { computed, ref, watch } from 'vue'
import { defineStore } from 'pinia'
import {
  SITE_DOCS_VERSION,
  defaultDocGroups,
  defaultDocPages,
  defaultDocSettings,
  type DocGroup,
  type DocPage,
  type DocSettings,
} from '@/lib/mock/site-docs'
import { normalizeDocPages } from '@/lib/site-docs'
import { fetchSiteConfig, saveSiteConfig } from '@/lib/api/siteConfig'
import { getToken } from '@/lib/api/client'

const STORAGE_KEY = 'site-docs'

interface PersistedDocs {
  version: number
  settings: DocSettings
  groups: DocGroup[]
  pages: DocPage[]
}

function clone<T>(value: T): T {
  return JSON.parse(JSON.stringify(value))
}

function defaults(): PersistedDocs {
  return {
    version: SITE_DOCS_VERSION,
    settings: clone(defaultDocSettings),
    groups: clone(defaultDocGroups),
    pages: normalizeDocPages(clone(defaultDocPages)),
  }
}

function load(): PersistedDocs {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return defaults()
    const parsed = JSON.parse(raw) as Partial<PersistedDocs>
    // 无 version 的早期缓存、v1/v2/v3 结构合法即接受，统一保序迁移到 v3。
    if (
      parsed.settings &&
      Array.isArray(parsed.groups) &&
      Array.isArray(parsed.pages)
    ) {
      return {
        version: SITE_DOCS_VERSION,
        settings: clone(parsed.settings),
        groups: clone(parsed.groups),
        pages: normalizeDocPages(clone(parsed.pages)),
      }
    }
  } catch {
    // 缓存损坏或版本不兼容时回退默认文档。
  }
  return defaults()
}

export const useSiteDocsStore = defineStore('site-docs', () => {
  const initial = load()
  const settings = ref<DocSettings>(initial.settings)
  const groups = ref<DocGroup[]>(initial.groups)
  const pages = ref<DocPage[]>(initial.pages)

  const groupName = computed<Record<string, string>>(() =>
    Object.fromEntries(groups.value.map((group) => [group.id, group.name])),
  )

  const publishedGroups = computed(() =>
    [...groups.value]
      .filter((group) => group.enabled)
      .sort((a, b) => a.sort - b.sort)
      .map((group) => ({
        group,
        pages: pages.value
          .filter((page) => page.groupId === group.id && page.status === 1)
          .sort((a, b) => a.sort - b.sort),
      }))
      .filter((entry) => entry.pages.length > 0),
  )

  const publishedPages = computed(() =>
    publishedGroups.value.flatMap((entry) => entry.pages),
  )

  function getPageBySlug(slug: string): DocPage | undefined {
    return pages.value.find((page) => page.slug === slug)
  }

  function updateSettings(next: DocSettings) {
    settings.value = clone(next)
  }

  function addGroup(group: DocGroup) {
    groups.value.push(clone(group))
  }

  function updateGroup(group: DocGroup) {
    const index = groups.value.findIndex((item) => item.id === group.id)
    if (index >= 0) groups.value[index] = clone(group)
  }

  function removeGroup(id: string): boolean {
    if (pages.value.some((page) => page.groupId === id)) return false
    groups.value = groups.value.filter((group) => group.id !== id)
    return true
  }

  function addPage(page: Omit<DocPage, 'id'>): number {
    const id = Math.max(0, ...pages.value.map((item) => item.id)) + 1
    pages.value.push({ ...clone(page), id })
    return id
  }

  function updatePage(page: DocPage) {
    const index = pages.value.findIndex((item) => item.id === page.id)
    if (index >= 0) pages.value[index] = clone(page)
  }

  function removePage(id: number) {
    pages.value = pages.value.filter((page) => page.id !== id)
  }

  function reset() {
    const next = defaults()
    settings.value = next.settings
    groups.value = next.groups
    pages.value = next.pages
  }

  function snapshot(): PersistedDocs {
    return {
      version: SITE_DOCS_VERSION,
      settings: settings.value,
      groups: groups.value,
      pages: pages.value,
    }
  }

  // hydrate 期间抑制回写，避免拉取赋值又触发一次 save 覆盖后端。
  let hydrated = false
  let suppress = false

  /** 从后端拉取文档并覆盖（后台/官网文档页初始化调用）。失败静默回退本地缓存。 */
  async function hydrate() {
    if (hydrated) return
    hydrated = true
    try {
      const remote = await fetchSiteConfig<PersistedDocs>('docs')
      if (remote && remote.settings && Array.isArray(remote.groups) && Array.isArray(remote.pages)) {
        suppress = true
        settings.value = clone(remote.settings)
        groups.value = clone(remote.groups)
        pages.value = normalizeDocPages(clone(remote.pages))
        suppress = false
      }
    } catch {
      // 后端不可用时用本地缓存
    }
  }

  // 变更即时落 localStorage（即时/离线）+ 防抖推送后端（真持久化）。
  let saveTimer: ReturnType<typeof setTimeout> | null = null
  watch(
    [settings, groups, pages],
    () => {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(snapshot()))
      if (suppress) return
      // 仅后台已登录(有 admin token)时推送后端；官网公开文档页只读，不触发写。
      if (!getToken()) return
      if (saveTimer) clearTimeout(saveTimer)
      saveTimer = setTimeout(() => {
        saveSiteConfig('docs', snapshot()).catch(() => {})
      }, 800)
    },
    { deep: true },
  )

  return {
    settings,
    groups,
    pages,
    groupName,
    publishedGroups,
    publishedPages,
    getPageBySlug,
    hydrate,
    updateSettings,
    addGroup,
    updateGroup,
    removeGroup,
    addPage,
    updatePage,
    removePage,
    reset,
  }
})
