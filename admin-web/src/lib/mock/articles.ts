/**
 * 官网文章资讯数据源（CMS 默认值）。
 * 运营后台「官网管理 / 文章管理」编辑 → useArticlesStore 持久化 → 官网首页「最新动态」板块 + 文章详情页读取。
 *
 * 数据分两层：分类(ArticleCategory) + 文章(Article)。
 * 首页「最新动态」板块按分类分列展示；每列取该分类下的文章（按 sort 升序、置顶优先）。
 */

/** 文章分类：对应首页「最新动态」的一列（产品动态 / 公司新闻 / 行业新闻…）*/
export interface ArticleCategory {
  id: number
  name: string // 中文名，如「产品动态」
  enName: string // 英文小标题，如「Function Update」
  cover: string // 该列顶部深色头图（留空用占位）
  sort: number // 越小越靠前
}

/** 文章 */
export interface Article {
  id: number
  categoryId: number // 所属分类
  title: string
  summary: string // 摘要（首页头条展示）
  content: string // 正文 HTML（tiptap 输出）
  cover: string // 文章封面（可选）
  isNew: boolean // 是否标记 new
  isTop: boolean // 是否置顶（首页头条优先）
  status: number // 1=发布 0=草稿(不在官网展示)
  sort: number // 越小越靠前
  views: number // 浏览量
  addtime: string // 发布时间 YYYY-MM-DD
}

export const defaultArticleCategories: ArticleCategory[] = [
  { id: 1, name: '产品动态', enName: 'Function Update', cover: '/assets/news-product.jpg', sort: 1 },
  { id: 2, name: '公司新闻', enName: 'Company Dynamics', cover: '/assets/news-company.jpg', sort: 2 },
  { id: 3, name: '行业新闻', enName: 'Industry News', cover: '/assets/news-industry.jpg', sort: 3 },
]

export const defaultArticles: Article[] = [
  // ===== 产品动态 =====
  {
    id: 1, categoryId: 1,
    title: '0538Pay 标准版系统 v3.0 正式发布上线',
    summary: '0538Pay 标准版系统 v3.0 正式发布上线！全面重构支付内核，支持多渠道聚合、实时到账、开放 API，让收款更简单高效。',
    content: '<h2>v3.0 版本重磅升级</h2><p>本次 v3.0 版本对支付内核进行了全面重构，带来更稳定、更高效的收款体验：</p><ul><li>多渠道聚合：一次对接支持支付宝、微信、QQ钱包、云闪付</li><li>实时到账：买家付款秒级入账，T+1 自动结算</li><li>开放 API：RESTful 接口 + MD5/RSA 双签名</li></ul><p>欢迎各位商户升级体验！</p>',
    cover: '', isNew: true, isTop: true, status: 1, sort: 1, views: 1280, addtime: '2026-07-10',
  },
  {
    id: 2, categoryId: 1,
    title: '陀螺匠系统 v2.4 正式发布，快来升级新版本',
    summary: '陀螺匠系统 v2.4 正式发布，新增多项实用功能，优化系统性能，欢迎升级体验。',
    content: '<p>陀螺匠系统 v2.4 正式发布，本次更新包含多项功能优化与体验提升。</p>',
    cover: '', isNew: false, isTop: false, status: 1, sort: 2, views: 860, addtime: '2026-07-10',
  },
  {
    id: 3, categoryId: 1,
    title: '0538Pay Pro 私域会员电商系统 v4.1 正式发布',
    summary: '私域会员电商系统 v4.1 正式发布，助力商户打造专属会员运营体系。',
    content: '<p>私域会员电商系统 v4.1 正式发布，新增会员分层运营、积分商城等功能。</p>',
    cover: '', isNew: false, isTop: false, status: 1, sort: 3, views: 720, addtime: '2026-06-30',
  },
  {
    id: 4, categoryId: 1,
    title: '0538Pay 多商户系统（Java）v2.3 正式发布',
    summary: '多商户系统 Java 版 v2.3 正式发布，支持多店铺入驻、分账结算。',
    content: '<p>多商户系统 Java 版 v2.3 正式发布。</p>',
    cover: '', isNew: false, isTop: false, status: 1, sort: 4, views: 540, addtime: '2026-06-23',
  },
  {
    id: 5, categoryId: 1,
    title: '0538Pay 多商户系统（PHP）v4.0 正式发布',
    summary: '多商户系统 PHP 版 v4.0 正式发布，全新架构，性能大幅提升。',
    content: '<p>多商户系统 PHP 版 v4.0 正式发布。</p>',
    cover: '', isNew: false, isTop: false, status: 1, sort: 5, views: 480, addtime: '2026-06-15',
  },
  // ===== 公司新闻 =====
  {
    id: 6, categoryId: 2,
    title: '0538Pay 主题广场｜以创意赋能，让装修更简单',
    summary: '为了帮大家解决装修痛点，让每一位 0538Pay 用户都能轻松拥有专业级商城装修，主题广场正式上线。',
    content: '<h2>主题广场正式上线</h2><p>为了帮大家解决装修痛点，让每一位用户都能轻松拥有专业级商城装修，我们推出了主题广场。</p>',
    cover: '', isNew: true, isTop: true, status: 1, sort: 1, views: 960, addtime: '2026-03-05',
  },
  {
    id: 7, categoryId: 2,
    title: '免费福利！0538Pay 原创后台图标已上线',
    summary: '0538Pay 原创设计的后台图标库正式上线，免费提供给所有用户使用。',
    content: '<p>原创后台图标库正式上线，免费开放使用。</p>',
    cover: '', isNew: false, isTop: false, status: 1, sort: 2, views: 640, addtime: '2026-03-05',
  },
  {
    id: 8, categoryId: 2,
    title: '0538Pay 2026 年春节放假通知',
    summary: '2026 年春节放假安排通知，请各位商户提前做好业务安排。',
    content: '<p>2026 年春节放假安排通知，具体时间见正文。</p>',
    cover: '', isNew: false, isTop: false, status: 1, sort: 3, views: 420, addtime: '2026-02-13',
  },
  {
    id: 9, categoryId: 2,
    title: '创新赋能，聚力共赢｜2026 战略年会圆满举行',
    summary: '2026 战略年会圆满举行，共话未来发展蓝图。',
    content: '<p>2026 战略年会圆满举行。</p>',
    cover: '', isNew: false, isTop: false, status: 1, sort: 4, views: 380, addtime: '2026-02-12',
  },
  {
    id: 10, categoryId: 2,
    title: '营销中心 2025 团建之旅圆满结束',
    summary: '营销中心 2025 团建活动圆满结束，凝聚团队力量。',
    content: '<p>营销中心 2025 团建活动圆满结束。</p>',
    cover: '', isNew: false, isTop: false, status: 1, sort: 5, views: 260, addtime: '2025-10-20',
  },
  // ===== 行业新闻 =====
  {
    id: 11, categoryId: 3,
    title: '2026 商业相关性报告：对话式商业成新趋势',
    summary: '未来商业竞争正在从流量获取迈向决策效率竞争。从研究结果可以看到，消费者真正需要的是帮助理解与决策的服务。',
    content: '<h2>对话式商业成新趋势</h2><p>未来商业竞争正在从流量获取迈向决策效率竞争，对话式商业成为新的增长点。</p>',
    cover: '', isNew: true, isTop: true, status: 1, sort: 1, views: 1120, addtime: '2026-07-14',
  },
  {
    id: 12, categoryId: 3,
    title: '新电商法即将发布：对平台加重处罚',
    summary: '新电商法即将发布，对平台责任提出更高要求，加重违规处罚力度。',
    content: '<p>新电商法即将发布，平台需提前做好合规准备。</p>',
    cover: '', isNew: false, isTop: false, status: 1, sort: 2, views: 780, addtime: '2026-07-14',
  },
  {
    id: 13, categoryId: 3,
    title: '重磅！微信小店推出最高等级店铺',
    summary: '微信小店推出最高等级店铺权益，为优质商家提供更多流量扶持。',
    content: '<p>微信小店推出最高等级店铺权益。</p>',
    cover: '', isNew: false, isTop: false, status: 1, sort: 3, views: 690, addtime: '2026-07-14',
  },
  {
    id: 14, categoryId: 3,
    title: '中小企业应该如何推进数字化？',
    summary: '中小企业数字化转型路径探讨，从工具选型到组织变革。',
    content: '<p>中小企业数字化转型路径探讨。</p>',
    cover: '', isNew: false, isTop: false, status: 1, sort: 4, views: 450, addtime: '2026-07-14',
  },
  {
    id: 15, categoryId: 3,
    title: '长周期与高回报：2026 B2B 营销数据深度洞察',
    summary: '2026 B2B 营销数据深度洞察，长周期投入带来高回报。',
    content: '<p>2026 B2B 营销数据深度洞察。</p>',
    cover: '', isNew: false, isTop: false, status: 1, sort: 5, views: 320, addtime: '2026-07-14',
  },
]
