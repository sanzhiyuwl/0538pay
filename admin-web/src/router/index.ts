import { createRouter, createWebHistory } from 'vue-router'
// 布局外壳 + 各端首屏关键页保持静态：进任一端必加载，懒加载反而多一次 chunk 请求且拖首屏。
// 其余业务页一律动态 import()，按路由拆 chunk 首屏只加载入口，其余按需加载（技术债 #1）。
import AdminLayout from '@/layouts/AdminLayout.vue'
import ConsoleLayout from '@/layouts/ConsoleLayout.vue'
import AgentLayout from '@/layouts/AgentLayout.vue'
import MerchantLayout from '@/layouts/MerchantLayout.vue'
import SiteLayout from '@/layouts/SiteLayout.vue'
import Login from '@/views/Login.vue'
import Dashboard from '@/views/Dashboard.vue'
import Placeholder from '@/views/Placeholder.vue'
import MerchantPlaceholder from '@/views/merchant/MerchantPlaceholder.vue'
import SiteHome from '@/views/site/SiteHome.vue'

// —— 运营后台业务页（懒加载）——
const StyleGuide = () => import('@/views/StyleGuide.vue')
const Orders = () => import('@/views/Orders.vue')
const Merchants = () => import('@/views/Merchants.vue')
const Settle = () => import('@/views/Settle.vue')
const ProfitSharing = () => import('@/views/ProfitSharing.vue')
const Groups = () => import('@/views/Groups.vue')
const Records = () => import('@/views/Records.vue')
const MerchantStats = () => import('@/views/MerchantStats.vue')
const InviteCodes = () => import('@/views/InviteCodes.vue')
const BuyerStats = () => import('@/views/BuyerStats.vue')
const Domains = () => import('@/views/Domains.vue')
const Channels = () => import('@/views/Channels.vue')
const PayTypes = () => import('@/views/PayTypes.vue')
const Plugins = () => import('@/views/Plugins.vue')
const Rolls = () => import('@/views/Rolls.vue')
const Wechat = () => import('@/views/Wechat.vue')
const Transfer = () => import('@/views/Transfer.vue')
const TransferRecords = () => import('@/views/TransferRecords.vue')
const Billing = () => import('@/views/Billing.vue')
const Risk = () => import('@/views/Risk.vue')
const Blacklist = () => import('@/views/Blacklist.vue')
const Settings = () => import('@/views/Settings.vue')
const PaySettings = () => import('@/views/PaySettings.vue')
const RegSettings = () => import('@/views/RegSettings.vue')
const RiskSettings = () => import('@/views/RiskSettings.vue')
const SettleSettings = () => import('@/views/SettleSettings.vue')
const TransferSettings = () => import('@/views/TransferSettings.vue')
const OAuthSettings = () => import('@/views/OAuthSettings.vue')
const NoticeSettings = () => import('@/views/NoticeSettings.vue')
const CertSettings = () => import('@/views/CertSettings.vue')
const OCRSettings = () => import('@/views/OCRSettings.vue')
const TemplateSettings = () => import('@/views/TemplateSettings.vue')
const MailSettings = () => import('@/views/MailSettings.vue')
const CronSettings = () => import('@/views/CronSettings.vue')
const OtherSettings = () => import('@/views/OtherSettings.vue')
const Wework = () => import('@/views/Wework.vue')
const WxkfSettings = () => import('@/views/WxkfSettings.vue')
const GetToken = () => import('@/views/GetToken.vue')
const SiteContent = () => import('@/views/SiteContent.vue')
const Articles = () => import('@/views/Articles.vue')
const DocsContent = () => import('@/views/DocsContent.vue')
const Announcements = () => import('@/views/Announcements.vue')
const Messages = () => import('@/views/Messages.vue')
const HelpSettings = () => import('@/views/HelpSettings.vue')
const Logs = () => import('@/views/Logs.vue')
const Clean = () => import('@/views/Clean.vue')
const Admins = () => import('@/views/Admins.vue')
const Roles = () => import('@/views/Roles.vue')
const OpLogs = () => import('@/views/OpLogs.vue')
const MerchantOpLogs = () => import('@/views/MerchantOpLogs.vue')

// —— 代理控制台（懒加载）——
const ConsoleOverview = () => import('@/views/console/ConsoleOverview.vue')
const ConsoleAgents = () => import('@/views/console/ConsoleAgents.vue')
const ConsoleQuota = () => import('@/views/console/ConsoleQuota.vue')
const ConsoleEnroll = () => import('@/views/console/ConsoleEnroll.vue')
const ConsoleInvites = () => import('@/views/console/ConsoleInvites.vue')
const ConsoleSettlement = () => import('@/views/console/ConsoleSettlement.vue')
const ConsoleSettings = () => import('@/views/console/ConsoleSettings.vue')
const ConsoleWxPartner = () => import('@/views/console/ConsoleWxPartner.vue')
const ConsolePermissions = () => import('@/views/console/ConsolePermissions.vue')
const ConsoleAgentLogs = () => import('@/views/console/ConsoleAgentLogs.vue')
const ConsoleManageLogs = () => import('@/views/console/ConsoleManageLogs.vue')
const ConsoleSystemLogs = () => import('@/views/console/ConsoleSystemLogs.vue')

// —— 独立代理端（懒加载）——
const AgentLogin = () => import('@/views/agent/AgentLogin.vue')
const AgentOverview = () => import('@/views/agent/AgentOverview.vue')
const AgentEnroll = () => import('@/views/agent/AgentEnroll.vue')
const AgentQuota = () => import('@/views/agent/AgentQuota.vue')
const AgentInvites = () => import('@/views/agent/AgentInvites.vue')
const AgentSettlement = () => import('@/views/agent/AgentSettlement.vue')

// —— 商户中心（懒加载）——
const MerchantLogin = () => import('@/views/merchant/MerchantLogin.vue')
const MerchantHome = () => import('@/views/merchant/MerchantHome.vue')
const MerchantOrders = () => import('@/views/merchant/MerchantOrders.vue')
const MerchantRecords = () => import('@/views/merchant/MerchantRecords.vue')
const MerchantSettle = () => import('@/views/merchant/MerchantSettle.vue')
const MerchantApply = () => import('@/views/merchant/MerchantApply.vue')
const MerchantApi = () => import('@/views/merchant/MerchantApi.vue')
const MerchantProfile = () => import('@/views/merchant/MerchantProfile.vue')
const MerchantCertificate = () => import('@/views/merchant/MerchantCertificate.vue')
const MerchantDomains = () => import('@/views/merchant/MerchantDomains.vue')
const MerchantOnecode = () => import('@/views/merchant/MerchantOnecode.vue')
const MerchantTransfer = () => import('@/views/merchant/MerchantTransfer.vue')
const MerchantRecharge = () => import('@/views/merchant/MerchantRecharge.vue')
const MerchantDeposit = () => import('@/views/merchant/MerchantDeposit.vue')
const MerchantGroupbuy = () => import('@/views/merchant/MerchantGroupbuy.vue')
const MerchantInvite = () => import('@/views/merchant/MerchantInvite.vue')
const MerchantHelp = () => import('@/views/merchant/MerchantHelp.vue')
const MerchantTest = () => import('@/views/merchant/MerchantTest.vue')
const MerchantReg = () => import('@/views/merchant/MerchantReg.vue')
const MerchantFindpwd = () => import('@/views/merchant/MerchantFindpwd.vue')
const MerchantComplete = () => import('@/views/merchant/MerchantComplete.vue')
const MerchantOAuthCallback = () => import('@/views/merchant/MerchantOAuthCallback.vue')

// —— 官网 / 支付前台页（懒加载，首页 SiteHome 除外）——
const Paypage = () => import('@/views/site/Paypage.vue')
const EnrollPublic = () => import('@/views/enroll/EnrollPublic.vue')
const ClassicDocs = () => import('@/views/site/templates/classic/ClassicDocs.vue')
const ClassicAbout = () => import('@/views/site/templates/classic/ClassicAbout.vue')
const ClassicAgreement = () => import('@/views/site/templates/classic/ClassicAgreement.vue')
const ClassicPayok = () => import('@/views/site/templates/classic/ClassicPayok.vue')
const ClassicPayerr = () => import('@/views/site/templates/classic/ClassicPayerr.vue')
const CashierMock = () => import('@/views/site/CashierMock.vue')
const PayVerify = () => import('@/views/site/PayVerify.vue')
const ClassicNews = () => import('@/views/site/templates/classic/ClassicNews.vue')
const ClassicNewsList = () => import('@/views/site/templates/classic/ClassicNewsList.vue')
import { allLeaves, consoleLeaves, merchantLeaves, agentLeaves } from '@/config/nav'
import { useAuthStore } from '@/stores/auth'
import { useMerchantAuthStore } from '@/stores/merchantAuth'
import { useAgentAuthStore } from '@/stores/agentAuth'
import { useSiteStore } from '@/stores/site'

// 路径 → 页面名映射：菜单叶子标题 + 少量非菜单页手工补充
const pathTitleMap: Record<string, string> = {
  ...Object.fromEntries(allLeaves.map((l) => [l.to, l.title])),
  ...Object.fromEntries(consoleLeaves.map((l) => [l.to, l.title])),
  ...Object.fromEntries(merchantLeaves.map((l) => [l.to, l.title])),
  ...Object.fromEntries(agentLeaves.map((l) => [l.to, l.title])),
  '/agent/login': '代理登录',
  '/enroll': '商户进件',
  '/admin': '平台概况',
  '/admin/style-guide': '设计规范',
  '/login': '登录',
  '/': '首页',
  '/docs': '开发者文档',
  '/about': '关于我们',
  '/agreement': '服务协议',
  '/payok': '支付成功',
  '/payerr': '支付失败',
  '/pay/verify': '支付安全验证',
}

/** 各端标题后缀 */
function suffixFor(path: string, siteName: string): string {
  if (path.startsWith('/console')) return `${siteName} 控制台`
  if (path.startsWith('/agent')) return `${siteName} 代理端`
  if (path.startsWith('/m')) return `${siteName} 商户中心`
  if (path.startsWith('/admin') || path === '/login') return `${siteName} 管理后台`
  return siteName
}

// 已实现的正式页面（其余菜单项暂用占位页）
const realPages: Record<string, any> = {
  '/admin/orders': Orders,
  '/admin/merchants': Merchants,
  '/admin/settle': Settle,
  '/admin/profit-sharing': ProfitSharing,
  '/admin/groups': Groups,
  '/admin/records': Records,
  '/admin/merchant-stats': MerchantStats,
  '/admin/invite-codes': InviteCodes,
  '/admin/buyer-stats': BuyerStats,
  '/admin/domains': Domains,
  '/admin/messages': Messages,
  '/admin/help-settings': HelpSettings,
  '/admin/channels': Channels,
  '/admin/pay-types': PayTypes,
  '/admin/plugins': Plugins,
  '/admin/rolls': Rolls,
  '/admin/wechat': Wechat,
  '/admin/transfer': Transfer,
  '/admin/transfer-records': TransferRecords,
  '/admin/billing': Billing,
  '/admin/risk': Risk,
  '/admin/blacklist': Blacklist,
  '/admin/settings': Settings,
  '/admin/reg-settings': RegSettings,
  '/admin/announcements': Announcements,
  '/admin/logs': Logs,
  '/admin/clean': Clean,
  '/admin/admins': Admins,
  '/admin/roles': Roles,
  '/admin/oplogs': OpLogs,
  '/admin/merchant-oplogs': MerchantOpLogs,
  '/admin/pay-settings': PaySettings,
  '/admin/risk-settings': RiskSettings,
  '/admin/settle-settings': SettleSettings,
  '/admin/transfer-settings': TransferSettings,
  '/admin/oauth-settings': OAuthSettings,
  '/admin/notice-settings': NoticeSettings,
  '/admin/cert-settings': CertSettings,
  '/admin/ocr-settings': OCRSettings,
  '/admin/template-settings': TemplateSettings,
  '/admin/mail-settings': MailSettings,
  '/admin/cron-settings': CronSettings,
  '/admin/other-settings': OtherSettings,
  '/admin/wework': Wework,
  '/admin/wxkf-settings': WxkfSettings,
  '/admin/gettoken': GetToken,
  '/admin/site-content': SiteContent,
  '/admin/articles': Articles,
  '/admin/docs-content': DocsContent,
}

// 代理控制台正式页面（path → 组件，改 consoleNav + 此表路由自动跟着变）
const consolePages: Record<string, any> = {
  '/console': ConsoleOverview,
  '/console/agents': ConsoleAgents,
  '/console/quota': ConsoleQuota,
  '/console/enroll': ConsoleEnroll,
  '/console/invites': ConsoleInvites,
  '/console/settlement': ConsoleSettlement,
  '/console/settings': ConsoleSettings,
  '/console/wx-partner': ConsoleWxPartner,
  '/console/permissions': ConsolePermissions,
  '/console/agent-logs': ConsoleAgentLogs,
  '/console/manage-logs': ConsoleManageLogs,
  '/console/system-logs': ConsoleSystemLogs,
}

// 主后台子路由（父 path=/admin，children 用相对路径）。排除首页 /admin 与控制台。
const placeholderRoutes = allLeaves
  .filter((i) => i.to !== '/admin' && !i.to.startsWith('/console'))
  .map((i) => ({
    // '/admin/orders' → 'orders'
    path: i.to.replace('/admin/', ''),
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

// 独立代理端正式页面（path → 组件，改 agentNav + 此表路由自动跟着变）
const agentPages: Record<string, any> = {
  '/agent': AgentOverview,
  '/agent/enroll': AgentEnroll,
  '/agent/quota': AgentQuota,
  '/agent/invites': AgentInvites,
  '/agent/settlement': AgentSettlement,
}

// 代理端子路由（父 path=/agent，children 用相对路径）
const agentChildren = agentLeaves.map((i) => ({
  // '/agent' → ''，'/agent/enroll' → 'enroll'
  path: i.to === '/agent' ? '' : i.to.replace('/agent/', ''),
  name: i.to,
  component: agentPages[i.to] ?? Placeholder,
}))

// 商户中心已实现的正式页面（其余子页暂用商户占位页）
const merchantPages: Record<string, any> = {
  '/m': MerchantHome,
  '/m/orders': MerchantOrders,
  '/m/records': MerchantRecords,
  '/m/settle': MerchantSettle,
  '/m/apply': MerchantApply,
  '/m/api': MerchantApi,
  '/m/profile': MerchantProfile,
  '/m/certificate': MerchantCertificate,
  '/m/domains': MerchantDomains,
  '/m/onecode': MerchantOnecode,
  '/m/transfer': MerchantTransfer,
  '/m/recharge': MerchantRecharge,
  '/m/deposit': MerchantDeposit,
  '/m/groupbuy': MerchantGroupbuy,
  '/m/invite': MerchantInvite,
  '/m/help': MerchantHelp,
  '/m/test': MerchantTest,
}

// 商户中心子路由（父 path=/m，children 用相对路径）
const merchantChildren = merchantLeaves.map((i) => ({
  // '/m' → ''，'/m/orders' → 'orders'
  path: i.to === '/m' ? '' : i.to.replace('/m/', ''),
  name: i.to,
  component: merchantPages[i.to] ?? MerchantPlaceholder,
}))

const router = createRouter({
  history: createWebHistory(),
  routes: [
    // 营销官网（根路径，访客第一入口）
    {
      path: '/',
      component: SiteLayout,
      children: [
        { path: '', name: 'site-home', component: SiteHome },
        { path: 'about', name: 'site-about', component: ClassicAbout },
        { path: 'agreement', name: 'site-agreement', component: ClassicAgreement },
        { path: 'news', name: 'site-news-list', component: ClassicNewsList },
        { path: 'news/category/:id', name: 'site-news-category', component: ClassicNewsList },
        { path: 'news/:id', name: 'site-news', component: ClassicNews },
      ],
    },
    // 开发者文档（独立文档站，无官网导航/大页脚，自带顶栏）
    { path: '/docs', name: 'site-docs', component: ClassicDocs },
    // 支付结果页（独立，无官网导航/页脚）
    { path: '/payok', name: 'payok', component: ClassicPayok },
    { path: '/payerr', name: 'payerr', component: ClassicPayerr },
    // 收银台中间页（mock 渠道，下单后 pay_url 跳转至此）
    { path: '/pay/mock/cashier/:trade_no', name: 'cashier-mock', component: CashierMock },
    // B1-04：空 type 下单跳收银台聚合选方式页（复用收银台组件，带 paytypes 时渲染选方式）
    { path: '/pay/cashier/:trade_no', name: 'cashier', component: CashierMock },
    // 支付安全验证页（对齐 epay verify_jump；命中 pay_verify 时下单跳转至此，通过后复发起下单）
    { path: '/pay/verify', name: 'pay-verify', component: PayVerify },
    // 公开聚合收款页（扫码进入，输金额→选方式→走收单链）
    { path: '/paypage', name: 'paypage', component: Paypage },
    // 客户自助进件公开页（免登录，靠邀请 code；自研扩展）
    { path: '/enroll/:code', name: 'enroll-public', component: EnrollPublic },
    // 后台登录页（独立，无侧栏）
    { path: '/login', name: 'login', component: Login },
    // 管理后台（运营端）
    {
      path: '/admin',
      component: AdminLayout,
      meta: { requiresAuth: true },
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
    // 独立代理端登录页（无侧栏）
    { path: '/agent/login', name: 'agent-login', component: AgentLogin },
    // 独立代理端主区（套 AgentLayout，需代理登录态）
    {
      path: '/agent',
      component: AgentLayout,
      meta: { requiresAgent: true },
      children: agentChildren,
    },
    // 商户中心登录态独立页（无侧栏）
    { path: '/m/login', name: 'm-login', component: MerchantLogin },
    { path: '/m/reg', name: 'm-reg', component: MerchantReg },
    { path: '/m/findpwd', name: 'm-findpwd', component: MerchantFindpwd },
    { path: '/m/complete', name: 'm-complete', component: MerchantComplete },
    { path: '/m/oauth/:provider', name: 'm-oauth-callback', component: MerchantOAuthCallback },
    // 商户中心主区（套 MerchantLayout）
    {
      path: '/m',
      component: MerchantLayout,
      meta: { requiresMerchant: true },
      children: merchantChildren,
    },
    // /site 旧路径重定向到官网首页（兼容历史链接）
    { path: '/site', redirect: '/' },
  ],
  // 锚点滚动：带 hash 平滑滚到目标板块（避开吸顶导航高度）；返回上一位置或顶部
  scrollBehavior(to, _from, savedPosition) {
    if (to.hash) {
      return { el: to.hash, top: 72, behavior: 'smooth' }
    }
    if (savedPosition) return savedPosition
    return { top: 0 }
  },
})

// 路由守卫：访问带 requiresAuth 的路由（后台）时校验登录态，未登录跳登录页并记来源
router.beforeEach((to) => {
  // 后台分组
  if (to.matched.some((r) => r.meta.requiresAuth)) {
    const auth = useAuthStore()
    if (!auth.isLoggedIn()) {
      return { name: 'login', query: { redirect: to.fullPath } }
    }
  }
  // 商户中心分组
  if (to.matched.some((r) => r.meta.requiresMerchant)) {
    const m = useMerchantAuthStore()
    if (!m.isLoggedIn()) {
      return { name: 'm-login', query: { redirect: to.fullPath } }
    }
  }
  // 独立代理端分组
  if (to.matched.some((r) => r.meta.requiresAgent)) {
    const ag = useAgentAuthStore()
    if (!ag.isLoggedIn()) {
      return { name: 'agent-login', query: { redirect: to.fullPath } }
    }
  }
  // 已登录时访问登录页，直接进对应端首页
  if (to.name === 'login') {
    const auth = useAuthStore()
    if (auth.isLoggedIn()) return '/admin'
  }
  if (to.name === 'm-login') {
    const m = useMerchantAuthStore()
    if (m.isLoggedIn()) return '/m'
  }
  if (to.name === 'agent-login') {
    const ag = useAgentAuthStore()
    if (ag.isLoggedIn()) return '/agent'
  }
})

// 标题守卫：后台/控制台/商户中心/登录页按「页面名 · 端后缀」动态设标题；
// 官网页面保留 useSiteStore 的 SEO 标题不覆盖。
router.afterEach((to) => {
  const path = to.path
  const isManaged =
    path.startsWith('/admin') ||
    path.startsWith('/console') ||
    path.startsWith('/agent') ||
    path.startsWith('/m') ||
    path === '/login'
  if (!isManaged) return // 官网页面交给 SEO 标题

  const site = useSiteStore()
  const siteName = site.config.sitename || 'Epvia Neo'
  const pageName = pathTitleMap[path]
  const suffix = suffixFor(path, siteName)
  document.title = pageName ? `${pageName} · ${suffix}` : suffix
})

export default router
