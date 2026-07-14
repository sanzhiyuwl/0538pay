/**
 * 官网首页营销内容（CMS 可编辑数据源的默认值）。
 * 运营后台「官网管理 / 首页内容」编辑 → useSiteContentStore 持久化 → 官网首页读取。
 *
 * 说明：图标是 Vue 组件，不能存进 localStorage，故这里只存「图标名」字符串，
 * 由 ClassicHome 的 iconMap 映射成组件。支付渠道的 SVG logo/品牌色属于底层资产，
 * 不进 CMS，仍在模板内以常量维护。
 */

/** 可选图标名（供后台下拉选择；与 ClassicHome iconMap 对应）*/
export const iconOptions = [
  'Waypoints', 'BadgePercent', 'Gauge', 'Radar', 'Webhook', 'MonitorSmartphone',
  'QrCode', 'MessageCircle', 'Smartphone', 'AppWindow', 'Network', 'Building2', 'Globe',
  'Store', 'ShieldCheck', 'LayoutDashboard',
] as const

export interface Metric {
  target: number
  suffix: string
  prefix?: string
  label: string
  decimals: number // 显示保留小数位（成功率=1，其余=0）
}
export interface Feature { icon: string; title: string; desc: string }
export interface Product { icon: string; name: string; desc: string; points: string[]; tags: string[]; art?: string }
export interface PlanRate { chan: string; name: string; rate: string }
export interface PlanFeature { k: string; v: string }
export interface Plan {
  name: string; price: string; unit: string; desc: string; cta: string
  highlight: boolean; theme: string; hidden: boolean
  rates: PlanRate[]; features: PlanFeature[]
}
export interface Testimonial { name: string; role: string; avatar: string; text: string }
export interface Faq { q: string; a: string }

export interface SiteContent {
  hero: {
    badge: string
    titleLead: string // 主标题第一行
    titleAccent: string // 主标题第二行（品牌色高亮）
    subtitle: string
    ctaPrimary: string
    ctaSecondary: string
    payMethodsLabel: string
  }
  metrics: Metric[]
  featuresTitle: string
  featuresSubtitle: string
  features: Feature[]
  productsTitle: string
  productsSubtitle: string
  products: Product[]
  pricingTitle: string
  pricingSubtitle: string
  plans: Plan[]
  testimonialsTitle: string
  testimonialsSubtitle: string
  testimonials: Testimonial[]
  faqTitle: string
  faqSubtitle: string
  faqs: Faq[]
  cta: {
    title: string
    subtitle: string
    ctaPrimary: string
    ctaSecondary: string
  }
}

export const defaultSiteContent: SiteContent = {
  hero: {
    badge: '聚合支付 · 服务商模式',
    titleLead: '让每一笔收款',
    titleAccent: '简单而可靠',
    subtitle: '一次对接，聚合支付宝、微信、QQ钱包、云闪付等多渠道收款。费率低至 0.28%，秒级到账，开放 API，助你快速开启线上收款。',
    ctaPrimary: '免费注册',
    ctaSecondary: '查看文档',
    payMethodsLabel: '已接入主流支付渠道',
  },
  metrics: [
    { target: 50000, suffix: '+', label: '接入商户', decimals: 0 },
    { target: 12, suffix: '亿+', prefix: '¥', label: '日均交易额', decimals: 0 },
    { target: 99.9, suffix: '%', label: '支付成功率', decimals: 1 },
    { target: 8, suffix: 'ms', label: '平均响应', decimals: 0 },
  ],
  featuresTitle: '为什么选择 0538Pay',
  featuresSubtitle: '一站式聚合支付解决方案，让收款更简单',
  features: [
    { icon: 'Waypoints', title: '多渠道聚合', desc: '一次对接，支持支付宝、微信、QQ钱包、云闪付等主流支付方式。' },
    { icon: 'BadgePercent', title: '超低费率', desc: '费率低至 0.28%，会员等级越高费率越低，成本清晰可控。' },
    { icon: 'Gauge', title: '实时到账', desc: '买家付款秒级入账，T+1 自动结算，资金周转更快。' },
    { icon: 'Radar', title: '安全风控', desc: '多维度风控引擎，实时拦截异常交易，保障资金安全。' },
    { icon: 'Webhook', title: '开放 API', desc: 'RESTful 接口 + MD5/RSA 双签名，文档完善，最快 1 天对接。' },
    { icon: 'MonitorSmartphone', title: '多端管理', desc: '商户中心、运营后台、SaaS 控制台，PC 与移动端随时管理。' },
  ],
  productsTitle: '全场景产品矩阵',
  productsSubtitle: '从线上到线下，覆盖你的每个收款场景',
  products: [
    { icon: 'QrCode', name: '扫码支付', desc: '线下贴码收款首选，顾客用支付宝 / 微信主扫或被扫，秒级到账。', points: ['固定码 / 动态码 / 一码多付', '语音播报到账金额', '支持店员多子账号对账'], tags: ['门店', '地摊', '餐饮'], art: '/assets/Rectangle1.png' },
    { icon: 'MessageCircle', name: '公众号支付', desc: '微信内 JSAPI 收款，用户关注公众号即可下单支付，转化更高。', points: ['微信内免跳转直接支付', '自动获取 openid 会员沉淀', '支持模板消息提醒'], tags: ['公众号', '会员'], art: '/assets/shouyedt3.png' },
    { icon: 'AppWindow', name: '小程序支付', desc: '小程序一键下单，拉起微信支付，适合线上商城与预约服务。', points: ['小程序原生收银体验', '支持商品券 / 代金券核销', '订单与会员打通'], tags: ['电商', '预约', '零售'], art: '/assets/shouyeqzq1.png' },
    { icon: 'Smartphone', name: 'H5 支付', desc: '手机浏览器唤起收银台，一个链接即可分享收款，无需下载 App。', points: ['扫码 / 链接皆可发起', '自动适配支付宝、微信环境', '支持自定义收款页'], tags: ['社交', '推广', '活动'], art: '/assets/Rectangle2.png' },
    { icon: 'MonitorSmartphone', name: 'APP 支付', desc: '原生 SDK 收款，App 内深度集成支付宝、微信支付。', points: ['iOS / Android 原生 SDK', '客户端一键唤起支付', '异步通知自动回调'], tags: ['App', '游戏', '出行'], art: '/assets/shouyeqzq2.png' },
    { icon: 'Network', name: '网关支付', desc: 'PC 网页收银台，跳转标准收银页，适合传统电商与网站收款。', points: ['标准收银台页面', '支持多渠道聚合选择', 'PC / 移动自适应'], tags: ['网站', 'PC 商城'], art: '/assets/Rectangle3.png' },
    { icon: 'Building2', name: '企业收付款', desc: '对公代付批量结算，支持提现、代发佣金与供应链结算。', points: ['批量代付 / 单笔付款', '提现自动审核出款', '资金流水可导出对账'], tags: ['佣金', '结算', '供应链'], art: '/assets/shouyedt2.png' },
    { icon: 'Globe', name: '跨境支付', desc: '多币种境外收款，支持主流外卡与本地钱包，助力出海业务。', points: ['多币种收款结汇', '支持主流外卡组织', '合规风控保障'], tags: ['出海', '外贸', '留学'], art: '/assets/shouyeqzq3.png' },
  ],
  pricingTitle: '透明的费率方案',
  pricingSubtitle: '按需选择，费率清晰，无隐藏费用',
  plans: [
    {
      name: '普通商户', price: '0.38', unit: '%', desc: '注册即用，适合起步', cta: '免费注册', highlight: false, theme: 'gray', hidden: false,
      rates: [
        { chan: 'wxpay', name: '微信支付', rate: '0.38%' },
        { chan: 'alipay', name: '支付宝', rate: '0.38%' },
      ],
      features: [
        { k: '开通资费', v: '¥100' },
        { k: '资质要求', v: '有无营业执照均可' },
        { k: '支持方式', v: '电脑扫码 + 线下扫码 + H5/APP 唤起支付' },
        { k: '结算方式', v: '官方 D+1 结算至银行卡' },
      ],
    },
    {
      name: '微信特约商户', price: '0.20', unit: '%', desc: '微信支付专属低费率', cta: '立即开通', highlight: true, theme: 'wechat', hidden: false,
      rates: [
        { chan: 'wxpay', name: '微信支付', rate: '0.20%' },
      ],
      features: [
        { k: '开通资费', v: '¥200' },
        { k: '资质要求', v: '个体户及企业营业执照' },
        { k: '支持方式', v: '电脑端扫码 + 公众号 JSAPI 支付 + 手机端 H5/APP 唤起支付 + 小程序支付' },
        { k: '结算方式', v: '官方 D+1 结算至银行卡' },
      ],
    },
    {
      name: '支付宝特约商户', price: '0.38', unit: '%', desc: '支付宝专属收款方案', cta: '立即开通', highlight: false, theme: 'alipay', hidden: true,
      rates: [
        { chan: 'alipay', name: '支付宝', rate: '0.38%' },
      ],
      features: [
        { k: '开通资费', v: '¥100' },
        { k: '资质要求', v: '个体户及企业营业执照' },
        { k: '支持方式', v: '电脑端扫码 + 手机端 H5/APP 唤起支付' },
        { k: '结算方式', v: '官方 D+1 结算至银行卡' },
      ],
    },
  ],
  testimonialsTitle: '他们都在用 0538Pay',
  testimonialsSubtitle: '来自各行业商户的真实反馈',
  testimonials: [
    { name: '陈先生', role: '连锁餐饮 · 财务负责人', avatar: '陈', text: '接入两周就跑通了全部门店的扫码收款，到账很快，对账也清楚，客服响应也及时。' },
    { name: '林女士', role: '跨境电商 · 创始人', avatar: '林', text: '跨境多币种收款帮我们省了不少事，费率透明，结算稳定，出海业务终于不用为收款发愁。' },
    { name: '王工', role: 'SaaS 服务商 · 技术负责人', avatar: '王', text: 'API 文档写得很清楚，MD5 / RSA 双签名，一天就对接完上线，异步通知也很稳。' },
  ],
  faqTitle: '常见问题',
  faqSubtitle: '关于接入、费率、资金安全的高频疑问',
  faqs: [
    { q: '接入需要什么资质？', a: '普通商户注册即用，有无营业执照均可；微信 / 支付宝特约商户需要个体户或企业营业执照。具体资质要求见费率方案说明。' },
    { q: '费率是多少？有没有隐藏费用？', a: '费率低至 0.28%，按商户等级与渠道透明计费，无隐藏费用。开通资费与结算费率在费率方案中逐项列明。' },
    { q: '资金安全吗？会不会有二清风险？', a: '本平台是收单外包服务机构，不涉及资金清算、不触碰用户资金。资金由持牌支付机构与你直接清算，不存在二清。' },
    { q: '多久能到账？如何结算？', a: '买家付款秒级入账到你的商户余额，官方 D+1 自动结算至绑定银行卡，也支持手动申请结算。' },
    { q: '技术对接复杂吗？', a: '提供 RESTful API + 完善文档 + 示例代码，支持 MD5 / RSA 双签名，最快 1 天即可完成对接上线。' },
    { q: '支持哪些支付方式？', a: '支持支付宝、微信支付、QQ 钱包、云闪付等主流渠道，覆盖扫码 / 公众号 / 小程序 / H5 / App / 网关 / 企业收付款 / 跨境八大产品。' },
  ],
  cta: {
    title: '立即开启你的收款业务',
    subtitle: '注册即可免费接入，多渠道聚合、超低费率、实时到账，助你的生意收款无忧。',
    ctaPrimary: '免费注册',
    ctaSecondary: '查看接入文档',
  },
}
