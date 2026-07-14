/**
 * 支付插件数据。1:1 提取自 epay.test/plugins/ 目录下全部 51 个插件的真实元信息
 * （每个 *_plugin.php 头部的 showname/author/link/types/transtypes）。
 * 支付插件页只读展示：插件靠上传源码到 /plugins/ 后"刷新插件列表"识别，后台不增删改。
 */

/** 支付插件（pre_plugin / *_plugin.php 元信息） */
export interface Plugin {
  name: string // 插件标识（目录名）
  showname: string // 插件描述
  author: string // 作者
  link: string // 作者/文档链接（可空）
  types: string // 包含的支付方式（逗号分隔）
  transtypes: string // 包含的转账方式（逗号分隔，可空）
}

/** 全部 51 个支付插件（与 epay.test/plugins/ 完全一致） */
export const plugins: Plugin[] = [
  { name: 'adapay', showname: 'AdaPay聚合支付', author: 'AdaPay', link: 'https://www.adapay.tech/', types: 'alipay,wxpay,bank', transtypes: '' },
  { name: 'alipay', showname: '支付宝官方支付', author: '支付宝', link: 'https://b.alipay.com/signing/productSetV2.htm', types: 'alipay', transtypes: 'alipay,bank' },
  { name: 'alipayd', showname: '支付宝官方支付直付通版', author: '支付宝', link: 'https://b.alipay.com/signing/productSetV2.htm', types: 'alipay', transtypes: '' },
  { name: 'alipayg', showname: '支付宝国际版', author: '支付宝', link: 'https://global.alipay.com/', types: 'alipay', transtypes: '' },
  { name: 'alipayrp', showname: '支付宝现金红包', author: '支付宝', link: 'https://b.alipay.com/signing/productSetV2.htm', types: 'alipay', transtypes: '' },
  { name: 'alipaysl', showname: '支付宝官方支付服务商版', author: '支付宝', link: 'https://b.alipay.com/signing/productSetV2.htm', types: 'alipay', transtypes: '' },
  { name: 'allinpay', showname: '通联支付', author: '通联', link: 'https://www.allinpay.com/', types: 'alipay,wxpay,qqpay,bank', transtypes: '' },
  { name: 'chinaums', showname: '银联商务', author: '银联商务', link: 'https://open.chinaums.com/', types: 'alipay,wxpay,bank', transtypes: '' },
  { name: 'dinpay', showname: '智付', author: '智付', link: 'https://www.dinpay.com/', types: 'alipay,wxpay,bank', transtypes: '' },
  { name: 'duolabao', showname: '哆啦宝支付', author: '哆啦宝', link: 'http://www.duolabao.com/', types: 'alipay,wxpay,qqpay,bank,jdpay', transtypes: '' },
  { name: 'easypay', showname: '易生支付', author: '易生', link: 'https://www.easypay.com.cn/', types: 'alipay,wxpay,bank', transtypes: '' },
  { name: 'epay', showname: '彩虹易支付', author: '彩虹', link: '', types: 'alipay,qqpay,wxpay,bank,jdpay', transtypes: '' },
  { name: 'epayn', showname: '彩虹易支付V2', author: '彩虹', link: '', types: 'alipay,qqpay,wxpay,bank,jdpay', transtypes: 'alipay,wxpay,qqpay,bank' },
  { name: 'fubei', showname: '付呗聚合支付', author: '付呗', link: 'https://www.51fubei.com/', types: 'alipay,wxpay,bank', transtypes: '' },
  { name: 'fuiou2', showname: '富友支付(合作方)', author: '富友', link: 'https://www.fuiou.com/', types: 'alipay,wxpay,bank', transtypes: '' },
  { name: 'heepay', showname: '汇付宝', author: '汇付宝', link: 'https://www.heepay.com/', types: 'alipay,wxpay,bank', transtypes: 'alipay,wxpay,bank' },
  { name: 'hlpay', showname: '汇联支付', author: '汇联', link: '', types: 'alipay,wxpay,bank', transtypes: 'alipay,wxpay' },
  { name: 'hnapay', showname: '新生支付', author: '新生支付', link: 'https://www.hnapay.com/', types: 'alipay,wxpay,bank', transtypes: 'bank' },
  { name: 'huifu', showname: '汇付斗拱平台', author: '汇付天下', link: 'https://paas.huifu.com/', types: 'alipay,wxpay,bank,ecny', transtypes: '' },
  { name: 'huolian', showname: '火脸支付', author: '火脸', link: 'https://www.lianok.com/', types: 'alipay,wxpay,bank', transtypes: '' },
  { name: 'jdpay', showname: '京东支付', author: '京东支付', link: 'https://www.jdpay.com/', types: 'jdpay', transtypes: '' },
  { name: 'jeepay', showname: 'Jeepay聚合支付', author: 'Jeepay', link: 'http://www.xxpay.org/', types: 'alipay,wxpay,bank', transtypes: 'alipay,wxpay,bank' },
  { name: 'kuaiqian', showname: '快钱支付', author: '快钱', link: 'https://www.99bill.com/', types: 'alipay,wxpay,bank', transtypes: 'bank' },
  { name: 'lakala', showname: '拉卡拉', author: '拉卡拉', link: 'https://www.lakala.com/', types: 'alipay,wxpay,bank', transtypes: '' },
  { name: 'ltzf', showname: '蓝兔支付', author: '蓝兔支付', link: 'https://www.ltzf.cn/', types: 'alipay,wxpay', transtypes: '' },
  { name: 'passpay', showname: '精秀支付', author: '精秀', link: 'https://www.jxpays.com/', types: 'alipay,wxpay,qqpay,bank', transtypes: '' },
  { name: 'paypal', showname: 'PayPal', author: 'PayPal', link: 'https://www.paypal.com/', types: 'paypal', transtypes: '' },
  { name: 'qqpay', showname: 'QQ钱包官方支付', author: 'QQ钱包', link: 'https://mp.qpay.tenpay.com/', types: 'qqpay', transtypes: 'qqpay' },
  { name: 'sandpay', showname: '杉德支付', author: '杉德', link: 'https://www.sandpay.com.cn/', types: 'alipay,wxpay,bank', transtypes: 'bank' },
  { name: 'shengpay', showname: '盛付通', author: '盛付通', link: 'https://www.shengpay.com/', types: 'alipay,wxpay,bank', transtypes: '' },
  { name: 'stripe', showname: 'Stripe', author: 'Stripe', link: 'https://stripe.com/', types: 'alipay,wxpay,bank,paypal', transtypes: '' },
  { name: 'suixingpay', showname: '随行付', author: '随行付', link: 'https://www.suixingpay.com/', types: 'alipay,wxpay,bank', transtypes: '' },
  { name: 'swiftpass', showname: '威富通RSA', author: '威富通', link: 'https://www.swiftpass.cn/', types: 'alipay,wxpay,qqpay,bank,jdpay', transtypes: '' },
  { name: 'swiftpass2', showname: '威富通MD5', author: '威富通', link: 'https://www.swiftpass.cn/', types: 'alipay,wxpay,qqpay,bank,jdpay', transtypes: '' },
  { name: 'umfpay', showname: '联动优势', author: '联动优势', link: 'https://xy.umfintech.com/', types: 'alipay,wxpay,bank', transtypes: '' },
  { name: 'unionpay', showname: '银联前置', author: '银联', link: 'http://www.95516.com/', types: 'alipay,wxpay,qqpay,bank,jdpay', transtypes: '' },
  { name: 'vmq', showname: 'V免签', author: 'V免签', link: 'https://github.com/szvone/vmqphp', types: 'alipay,qqpay,wxpay', transtypes: '' },
  { name: 'woaizf', showname: '我爱支付', author: '我爱支付', link: 'https://www.52zhifu.com/', types: 'alipay,wxpay,qqpay,bank', transtypes: 'alipay,wxpay' },
  { name: 'wxpay', showname: '微信官方支付', author: '微信', link: 'https://pay.weixin.qq.com/', types: 'wxpay', transtypes: 'wxpay,bank' },
  { name: 'wxpayn', showname: '微信官方支付V3', author: '微信', link: 'https://pay.weixin.qq.com/', types: 'wxpay', transtypes: 'wxpay' },
  { name: 'wxpayng', showname: '微信支付国际版V3', author: '微信', link: 'https://pay.weixin.qq.com/', types: 'wxpay', transtypes: '' },
  { name: 'wxpaynp', showname: '微信官方支付V3服务商版', author: '微信', link: 'https://pay.weixin.qq.com/partner/public/home', types: 'wxpay', transtypes: '' },
  { name: 'wxpaysl', showname: '微信官方支付服务商版', author: '微信', link: 'https://pay.weixin.qq.com/partner/public/home', types: 'wxpay', transtypes: '' },
  { name: 'xorpay', showname: 'XorPay', author: 'XorPay', link: 'https://xorpay.com/', types: 'alipay,wxpay', transtypes: '' },
  { name: 'xsy', showname: '新生易', author: '新生易', link: 'https://www.hnapay.com/', types: 'wxpay,alipay,bank', transtypes: '' },
  { name: 'xunhupay', showname: '虎皮椒支付', author: '虎皮椒', link: 'https://www.xunhupay.com/', types: 'alipay,wxpay', transtypes: '' },
  { name: 'yeepay', showname: '易宝支付', author: '易宝支付', link: 'https://www.yeepay.com/', types: 'alipay,wxpay,bank', transtypes: '' },
  { name: 'ysepay', showname: '银盛支付', author: '银盛支付', link: 'https://www.ysepay.com/', types: 'alipay,qqpay,wxpay,bank', transtypes: '' },
  { name: 'yseqt', showname: '银盛e企通', author: '银盛', link: 'https://www.ysepay.com/', types: 'alipay,wxpay,bank', transtypes: 'bank' },
  { name: 'zhangyishou', showname: '掌易收聚合支付', author: '掌易收', link: 'http://www.zhangyishou.com/', types: 'alipay,qqpay,wxpay,bank', transtypes: '' },
  { name: 'zyu', showname: '知宇支付', author: '知宇', link: '', types: 'alipay,qqpay,wxpay,bank', transtypes: '' },
]

/** 把逗号分隔的方式字符串拆成数组（用于渲染标签） */
export function splitTypes(s: string): string[] {
  return s ? s.split(',').filter(Boolean) : []
}

/** 汇总统计 */
export function calcPluginStats(list: Plugin[]) {
  const withTransfer = list.filter((p) => p.transtypes).length
  return {
    total: list.length,
    withTransfer,
    onlyPay: list.length - withTransfer,
  }
}
