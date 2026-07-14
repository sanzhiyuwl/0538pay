/**
 * 授权支付域名假数据。对齐 epay admin/domain.php + ajax_user.php?act=domainList（pre_domain）。
 * 商户绑定的支付回调/发起域名，需审核；支持通配符 *.
 */
import { merchants } from './merchants'

/** 授权域名（pre_domain） */
export interface DomainItem {
  id: number
  uid: number // 商户号
  domain: string // 域名（支持 *.通配符）
  status: 0 | 1 | 2 // 0=待审核 1=正常 2=拒绝
  addtime: string // 添加时间
  endtime: string | null // 审核时间
}

/** 域名状态字典（对齐 domain.php formatter） */
export const domainStatus: Record<
  number,
  { text: string; variant: 'default' | 'success' | 'destructive' }
> = {
  0: { text: '审核中', variant: 'default' },
  1: { text: '正常', variant: 'success' },
  2: { text: '拒绝', variant: 'destructive' },
}

const domainSuffixes = ['com', 'cn', 'net', 'shop', 'vip']
const subPrefixes = ['pay', 'shop', 'www', 'api', 'm', '*.']

function pad(n: number, len = 2) {
  return String(n).padStart(len, '0')
}

const uids = Array.from(new Map(merchants.map((m) => [m.uid, m])).values()).map((m) => m.uid)

// 生成 40 条授权域名
export const domains: DomainItem[] = Array.from({ length: 40 }, (_, i) => {
  const uid = uids[i % uids.length]
  const sub = subPrefixes[i % subPrefixes.length]
  const suffix = domainSuffixes[i % domainSuffixes.length]
  const domain = `${sub === '*.' ? '*.' : sub + '.'}merchant${uid}.${suffix}`
  // 状态分布：多数正常，部分待审核、少量拒绝
  const statusPool: DomainItem['status'][] = [1, 1, 0, 1, 2, 1, 0, 1]
  const status = statusPool[i % statusPool.length]
  const day = 12 - (i % 8)
  const reviewed = status !== 0
  return {
    id: 40000 + (40 - i),
    uid,
    domain,
    status,
    addtime: `2026-07-${pad(day)} ${pad(9 + (i % 10))}:${pad((i * 7) % 60)}:${pad((i * 13) % 60)}`,
    endtime: reviewed ? `2026-07-${pad(day)} ${pad(10 + (i % 9))}:${pad((i * 11) % 60)}:${pad((i * 17) % 60)}` : null,
  }
})

/** 状态可执行的操作（对齐 domain.php 操作列） */
export function domainActions(status: number): string[] {
  if (status === 1) return ['改为拒绝', '删除']
  if (status === 2) return ['改为通过', '删除']
  return ['通过', '拒绝', '删除'] // 待审核
}

/** 汇总统计 */
export function calcDomainStats(list: DomainItem[]) {
  return {
    total: list.length,
    pending: list.filter((d) => d.status === 0).length,
    normal: list.filter((d) => d.status === 1).length,
    rejected: list.filter((d) => d.status === 2).length,
  }
}
