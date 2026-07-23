import { createRouter, createWebHistory } from 'vue-router'
import AdminLayout from '@/layouts/AdminLayout.vue'
import Login from '@/views/Login.vue'
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
import InviteCodes from '@/views/InviteCodes.vue'
import BuyerStats from '@/views/BuyerStats.vue'
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
import TransferSettings from '@/views/TransferSettings.vue'
import OAuthSettings from '@/views/OAuthSettings.vue'
import NoticeSettings from '@/views/NoticeSettings.vue'
import CertSettings from '@/views/CertSettings.vue'
import TemplateSettings from '@/views/TemplateSettings.vue'
import MailSettings from '@/views/MailSettings.vue'
import CronSettings from '@/views/CronSettings.vue'
import OtherSettings from '@/views/OtherSettings.vue'
import Wework from '@/views/Wework.vue'
import WxkfSettings from '@/views/WxkfSettings.vue'
import GetToken from '@/views/GetToken.vue'
import SiteContent from '@/views/SiteContent.vue'
// 内容管理含 tiptap 富文本编辑器，体积较大，懒加载拆出主包
const Articles = () => import('@/views/Articles.vue')
const DocsContent = () => import('@/views/DocsContent.vue')
import Announcements from '@/views/Announcements.vue'
import Messages from '@/views/Messages.vue'
import HelpSettings from '@/views/HelpSettings.vue'
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
import MerchantLayout from '@/layouts/MerchantLayout.vue'
import MerchantLogin from '@/views/merchant/MerchantLogin.vue'
import MerchantPlaceholder from '@/views/merchant/MerchantPlaceholder.vue'
import MerchantHome from '@/views/merchant/MerchantHome.vue'
import MerchantOrders from '@/views/merchant/MerchantOrders.vue'
import MerchantRecords from '@/views/merchant/MerchantRecords.vue'
import MerchantSettle from '@/views/merchant/MerchantSettle.vue'
import MerchantApply from '@/views/merchant/MerchantApply.vue'
import MerchantApi from '@/views/merchant/MerchantApi.vue'
import MerchantProfile from '@/views/merchant/MerchantProfile.vue'
import MerchantCertificate from '@/views/merchant/MerchantCertificate.vue'
import MerchantDomains from '@/views/merchant/MerchantDomains.vue'
import MerchantOnecode from '@/views/merchant/MerchantOnecode.vue'
import MerchantTransfer from '@/views/merchant/MerchantTransfer.vue'
import MerchantRecharge from '@/views/merchant/MerchantRecharge.vue'
import MerchantDeposit from '@/views/merchant/MerchantDeposit.vue'
import MerchantGroupbuy from '@/views/merchant/MerchantGroupbuy.vue'
import MerchantInvite from '@/views/merchant/MerchantInvite.vue'
import MerchantHelp from '@/views/merchant/MerchantHelp.vue'
import MerchantTest from '@/views/merchant/MerchantTest.vue'
import MerchantMessages from '@/views/merchant/MerchantMessages.vue'
import Paypage from '@/views/site/Paypage.vue'
import MerchantReg from '@/views/merchant/MerchantReg.vue'
import MerchantFindpwd from '@/views/merchant/MerchantFindpwd.vue'
import MerchantComplete from '@/views/merchant/MerchantComplete.vue'
import MerchantOAuthCallback from '@/views/merchant/MerchantOAuthCallback.vue'
import SiteLayout from '@/layouts/SiteLayout.vue'
import SiteHome from '@/views/site/SiteHome.vue'
import ClassicDocs from '@/views/site/templates/classic/ClassicDocs.vue'
import ClassicAbout from '@/views/site/templates/classic/ClassicAbout.vue'
import ClassicAgreement from '@/views/site/templates/classic/ClassicAgreement.vue'
import ClassicPayok from '@/views/site/templates/classic/ClassicPayok.vue'
import ClassicPayerr from '@/views/site/templates/classic/ClassicPayerr.vue'
import CashierMock from '@/views/site/CashierMock.vue'
import ClassicNews from '@/views/site/templates/classic/ClassicNews.vue'
import ClassicNewsList from '@/views/site/templates/classic/ClassicNewsList.vue'
import { allLeaves, consoleLeaves, merchantLeaves } from '@/config/nav'
import { useAuthStore } from '@/stores/auth'
import { useMerchantAuthStore } from '@/stores/merchantAuth'
import { useSiteStore } from '@/stores/site'

// 路径 → 页面名映射：菜单叶子标题 + 少量非菜单页手工补充
const pathTitleMap: Record<string, string> = {
  ...Object.fromEntries(allLeaves.map((l) => [l.to, l.title])),
  ...Object.fromEntries(consoleLeaves.map((l) => [l.to, l.title])),
  ...Object.fromEntries(merchantLeaves.map((l) => [l.to, l.title])),
  '/admin': '平台概况',
  '/admin/style-guide': '设计规范',
  '/login': '登录',
  '/': '首页',
  '/docs': '开发者文档',
  '/about': '关于我们',
  '/agreement': '服务协议',
  '/payok': '支付成功',
  '/payerr': '支付失败',
}

/** 各端标题后缀 */
function suffixFor(path: string, siteName: string): string {
  if (path.startsWith('/console')) return `${siteName} 控制台`
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
  '/admin/pay-settings': PaySettings,
  '/admin/risk-settings': RiskSettings,
  '/admin/settle-settings': SettleSettings,
  '/admin/transfer-settings': TransferSettings,
  '/admin/oauth-settings': OAuthSettings,
  '/admin/notice-settings': NoticeSettings,
  '/admin/cert-settings': CertSettings,
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

// 控制台独立后台的正式页面（其余子页暂用占位）
const consolePages: Record<string, any> = {
  '/console': Console,
  '/console/plans': ConsolePlans,
  '/console/billing': ConsoleBilling,
  '/console/overview': ConsoleOverview,
  '/console/logs': ConsoleLogs,
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
  '/m/messages': MerchantMessages,
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
    // 公开聚合收款页（扫码进入，输金额→选方式→走收单链）
    { path: '/paypage', name: 'paypage', component: Paypage },
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
  // 已登录时访问登录页，直接进对应端首页
  if (to.name === 'login') {
    const auth = useAuthStore()
    if (auth.isLoggedIn()) return '/admin'
  }
  if (to.name === 'm-login') {
    const m = useMerchantAuthStore()
    if (m.isLoggedIn()) return '/m'
  }
})

// 标题守卫：后台/控制台/商户中心/登录页按「页面名 · 端后缀」动态设标题；
// 官网页面保留 useSiteStore 的 SEO 标题不覆盖。
router.afterEach((to) => {
  const path = to.path
  const isManaged =
    path.startsWith('/admin') || path.startsWith('/console') || path.startsWith('/m') || path === '/login'
  if (!isManaged) return // 官网页面交给 SEO 标题

  const site = useSiteStore()
  const siteName = site.config.sitename || '0538Pay'
  const pageName = pathTitleMap[path]
  const suffix = suffixFor(path, siteName)
  document.title = pageName ? `${pageName} · ${suffix}` : suffix
})

export default router
