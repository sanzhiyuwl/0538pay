/**
 * 商户中心登录态。持有 merchant token 与当前商户信息，供 /m 路由守卫与顶栏/工作台使用。
 * token 存独立 localStorage key（merchant_token，与 admin 隔离），刷新页面不掉线。
 */
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getMerchantToken, clearMerchantToken } from '@/lib/api/client'
import {
  merchantLogin as apiMerchantLogin,
  fetchMerchantInfo,
  type MerchantInfo,
} from '@/lib/api/merchantAuth'

const INFO_KEY = 'merchant_info'

function loadInfo(): MerchantInfo | null {
  try {
    const raw = localStorage.getItem(INFO_KEY)
    return raw ? (JSON.parse(raw) as MerchantInfo) : null
  } catch {
    return null
  }
}

export const useMerchantAuthStore = defineStore('merchantAuth', () => {
  const token = ref(getMerchantToken())
  const info = ref<MerchantInfo | null>(loadInfo())

  const isLoggedIn = () => !!token.value

  function persistInfo(v: MerchantInfo | null) {
    info.value = v
    if (v) localStorage.setItem(INFO_KEY, JSON.stringify(v))
    else localStorage.removeItem(INFO_KEY)
  }

  async function login(type: 0 | 1, account: string, password: string) {
    const data = await apiMerchantLogin(type, account, password) // 成功后 client 已存 token
    token.value = data.token
    persistInfo(data.info)
  }

  // 刷新当前商户信息（工作台/资料变更后调用）
  async function refreshInfo() {
    const data = await fetchMerchantInfo()
    persistInfo(data)
    return data
  }

  function logout() {
    clearMerchantToken()
    token.value = ''
    persistInfo(null)
  }

  return { token, info, isLoggedIn, login, refreshInfo, logout }
})
