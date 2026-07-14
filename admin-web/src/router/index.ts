import { createRouter, createWebHistory } from 'vue-router'
import AdminLayout from '@/layouts/AdminLayout.vue'
import Dashboard from '@/views/Dashboard.vue'
import Placeholder from '@/views/Placeholder.vue'
import StyleGuide from '@/views/StyleGuide.vue'
import Orders from '@/views/Orders.vue'
import Merchants from '@/views/Merchants.vue'
import Settle from '@/views/Settle.vue'
import ProfitSharing from '@/views/ProfitSharing.vue'
import Groups from '@/views/Groups.vue'
import Records from '@/views/Records.vue'
import MerchantStats from '@/views/MerchantStats.vue'
import Domains from '@/views/Domains.vue'
import Channels from '@/views/Channels.vue'
import PayTypes from '@/views/PayTypes.vue'
import Plugins from '@/views/Plugins.vue'
import Rolls from '@/views/Rolls.vue'
import Wechat from '@/views/Wechat.vue'
import Transfer from '@/views/Transfer.vue'
import TransferRecords from '@/views/TransferRecords.vue'
import Billing from '@/views/Billing.vue'
import Risk from '@/views/Risk.vue'
import Blacklist from '@/views/Blacklist.vue'
import Settings from '@/views/Settings.vue'
import PaySettings from '@/views/PaySettings.vue'
import RegSettings from '@/views/RegSettings.vue'
import RiskSettings from '@/views/RiskSettings.vue'
import SettleSettings from '@/views/SettleSettings.vue'
import Announcements from '@/views/Announcements.vue'
import Logs from '@/views/Logs.vue'
import Clean from '@/views/Clean.vue'
import Admins from '@/views/Admins.vue'
import Roles from '@/views/Roles.vue'
import OpLogs from '@/views/OpLogs.vue'
import ConsoleLayout from '@/layouts/ConsoleLayout.vue'
import Console from '@/views/Console.vue'
import ConsolePlans from '@/views/ConsolePlans.vue'
import ConsoleBilling from '@/views/ConsoleBilling.vue'
import ConsoleOverview from '@/views/ConsoleOverview.vue'
import ConsoleLogs from '@/views/ConsoleLogs.vue'
import { allLeaves, consoleLeaves } from '@/config/nav'

// 已实现的正式页面（其余菜单项暂用占位页）
const realPages: Record<string, any> = {
  '/orders': Orders,
  '/merchants': Merchants,
  '/settle': Settle,
  '/profit-sharing': ProfitSharing,
  '/groups': Groups,
  '/records': Records,
  '/merchant-stats': MerchantStats,
  '/domains': Domains,
  '/channels': Channels,
  '/pay-types': PayTypes,
  '/plugins': Plugins,
  '/rolls': Rolls,
  '/wechat': Wechat,
  '/transfer': Transfer,
  '/transfer-records': TransferRecords,
  '/billing': Billing,
  '/risk': Risk,
  '/blacklist': Blacklist,
  '/settings': Settings,
  '/reg-settings': RegSettings,
  '/announcements': Announcements,
  '/logs': Logs,
  '/clean': Clean,
  '/admins': Admins,
  '/roles': Roles,
  '/oplogs': OpLogs,
  '/pay-settings': PaySettings,
  '/risk-settings': RiskSettings,
  '/settle-settings': SettleSettings,
}

// 控制台独立后台的正式页面（其余子页暂用占位）
const consolePages: Record<string, any> = {
  '/console': Console,
  '/console/plans': ConsolePlans,
  '/console/billing': ConsoleBilling,
  '/console/overview': ConsoleOverview,
  '/console/logs': ConsoleLogs,
}

// 主后台占位路由（排除首页与控制台——控制台走独立 ConsoleLayout）
const placeholderRoutes = allLeaves
  .filter((i) => i.to !== '/' && !i.to.startsWith('/console'))
  .map((i) => ({
    path: i.to,
    name: i.to,
    component: realPages[i.to] ?? Placeholder,
  }))

// 控制台子路由（父 path=/console，children 用相对路径）
const consoleChildren = consoleLeaves.map((i) => ({
  // '/console' → ''，'/console/plans' → 'plans'
  path: i.to === '/console' ? '' : i.to.replace('/console/', ''),
  name: i.to,
  component: consolePages[i.to] ?? Placeholder,
}))

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      component: AdminLayout,
      children: [
        { path: '', name: 'dashboard', component: Dashboard },
        { path: 'style-guide', name: 'style-guide', component: StyleGuide },
        ...placeholderRoutes,
      ],
    },
    {
      path: '/console',
      component: ConsoleLayout,
      children: consoleChildren,
    },
  ],
})

export default router
