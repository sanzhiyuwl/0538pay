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
import MerchantReg from '@/views/merchant/MerchantReg.vue'
import MerchantFindpwd from '@/views/merchant/MerchantFindpwd.vue'
import MerchantComplete from '@/views/merchant/MerchantComplete.vue'
import SiteLayout from '@/layouts/SiteLayout.vue'
import SiteHome from '@/views/site/SiteHome.vue'
import ClassicDocs from '@/views/site/templates/classic/ClassicDocs.vue'
import ClassicAbout from '@/views/site/templates/classic/ClassicAbout.vue'
import ClassicAgreement from '@/views/site/templates/classic/ClassicAgreement.vue'
import ClassicPayok from '@/views/site/templates/classic/ClassicPayok.vue'
import ClassicPayerr from '@/views/site/templates/classic/ClassicPayerr.vue'
import { allLeaves, consoleLeaves, merchantLeaves } from '@/config/nav'

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
        { path: 'docs', name: 'site-docs', component: ClassicDocs },
        { path: 'about', name: 'site-about', component: ClassicAbout },
        { path: 'agreement', name: 'site-agreement', component: ClassicAgreement },
      ],
    },
    // 支付结果页（独立，无官网导航/页脚）
    { path: '/payok', name: 'payok', component: ClassicPayok },
    { path: '/payerr', name: 'payerr', component: ClassicPayerr },
    // 管理后台（运营端）
    {
      path: '/admin',
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
    // 商户中心登录态独立页（无侧栏）
    { path: '/m/login', name: 'm-login', component: MerchantLogin },
    { path: '/m/reg', name: 'm-reg', component: MerchantReg },
    { path: '/m/findpwd', name: 'm-findpwd', component: MerchantFindpwd },
    { path: '/m/complete', name: 'm-complete', component: MerchantComplete },
    // 商户中心主区（套 MerchantLayout）
    {
      path: '/m',
      component: MerchantLayout,
      children: merchantChildren,
    },
    // /site 旧路径重定向到官网首页（兼容历史链接）
    { path: '/site', redirect: '/' },
  ],
})

export default router
