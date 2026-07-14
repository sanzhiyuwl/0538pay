/**
 * 公众号 / 小程序假数据。对齐 epay admin/pay_weixin.php（pre_weixin）。
 * 配置微信服务号 / 小程序的 APPID/APPSECRET，用于 JSAPI 支付、网页授权等场景。
 */

/** 公众号 / 小程序（pre_weixin） */
export interface WeixinApp {
  id: number
  type: 0 | 1 // 0=微信服务号 1=微信小程序
  name: string // 名称（仅显示）
  appid: string // APPID
  appsecret: string // APPSECRET（脱敏展示）
}

/** 类别字典（对齐 display_type） */
export const weixinType: Record<number, string> = {
  0: '微信服务号',
  1: '微信小程序',
}

export const weixinApps: WeixinApp[] = [
  { id: 1, type: 0, name: '主商城服务号', appid: 'wx8a1b2c3d4e5f6a7b', appsecret: 'a1b2****************c3d4' },
  { id: 2, type: 1, name: '收银台小程序', appid: 'wx9f8e7d6c5b4a3f2e', appsecret: 'e5f6****************a7b8' },
  { id: 3, type: 0, name: '会员中心服务号', appid: 'wx1122334455667788', appsecret: 'c3d4****************e5f6' },
  { id: 4, type: 1, name: '门店点单小程序', appid: 'wxaabbccddeeff0011', appsecret: 'f6a7****************b8c9' },
]

/** 汇总统计 */
export function calcWeixinStats(list: WeixinApp[]) {
  return {
    total: list.length,
    official: list.filter((w) => w.type === 0).length,
    mini: list.filter((w) => w.type === 1).length,
  }
}
