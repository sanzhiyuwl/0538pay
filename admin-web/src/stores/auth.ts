/**
 * 后台登录态。持有 token 与管理员身份，供路由守卫与顶栏使用。
 * token 存 localStorage（经 api/client 读写），刷新页面不掉线。
 */
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getToken, clearToken } from '@/lib/api/client'
import { login as apiLogin } from '@/lib/api/auth'

const NICK_KEY = 'admin_nickname'
const ROLE_KEY = 'admin_role'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(getToken())
  const nickname = ref(localStorage.getItem(NICK_KEY) || '')
  const role = ref(localStorage.getItem(ROLE_KEY) || '')

  const isLoggedIn = () => !!token.value

  async function login(username: string, password: string) {
    const data = await apiLogin(username, password) // 成功后 client 已存 token
    token.value = data.token
    nickname.value = data.nickname
    role.value = data.role
    localStorage.setItem(NICK_KEY, data.nickname)
    localStorage.setItem(ROLE_KEY, data.role)
  }

  function logout() {
    clearToken()
    token.value = ''
    nickname.value = ''
    role.value = ''
    localStorage.removeItem(NICK_KEY)
    localStorage.removeItem(ROLE_KEY)
  }

  // 更新显示昵称并持久化（账号设置保存后同步顶栏）。
  function setNickname(nick: string) {
    nickname.value = nick
    localStorage.setItem(NICK_KEY, nick)
  }

  return { token, nickname, role, isLoggedIn, login, logout, setNickname }
})
