/**
 * 独立代理端 /agent 登录态。持有 agent token 与当前代理身份（含权限串），
 * 供 /agent 路由守卫、顶栏、按权限动态渲染菜单使用。
 * token 存独立 localStorage key（agent_token，与 admin/merchant 隔离），刷新页面不掉线。
 */
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { getAgentToken, clearAgentToken } from '@/lib/api/client'
import { agentLogin as apiAgentLogin, fetchAgentProfile } from '@/lib/api/agent'

const NAME_KEY = 'agent_name'
const PERMS_KEY = 'agent_perms'

export const useAgentAuthStore = defineStore('agentAuth', () => {
  const token = ref(getAgentToken())
  const name = ref(localStorage.getItem(NAME_KEY) || '')
  const permissions = ref(localStorage.getItem(PERMS_KEY) || '')

  const isLoggedIn = () => !!token.value

  // 权限点集合（逗号串 → Set），供菜单/按钮门控。
  const permSet = computed(() => new Set(permissions.value.split(',').map((s) => s.trim()).filter(Boolean)))
  const has = (key: string) => permSet.value.has(key)

  function persist(nm: string, perms: string) {
    name.value = nm
    permissions.value = perms
    localStorage.setItem(NAME_KEY, nm)
    localStorage.setItem(PERMS_KEY, perms)
  }

  async function login(account: string, password: string) {
    const data = await apiAgentLogin({ account, password }) // 成功后 client 已存 token
    token.value = data.token
    persist(data.name, data.permissions)
  }

  // 刷新当前代理资料（权限可能被平台调整，进入时拉一次保证菜单准确）。
  async function refresh() {
    const a = await fetchAgentProfile()
    persist(a.name, a.permissions)
    return a
  }

  function logout() {
    clearAgentToken()
    token.value = ''
    name.value = ''
    permissions.value = ''
    localStorage.removeItem(NAME_KEY)
    localStorage.removeItem(PERMS_KEY)
  }

  return { token, name, permissions, isLoggedIn, permSet, has, login, refresh, logout }
})
