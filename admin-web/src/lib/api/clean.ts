/** 数据清理 API（对齐 epay clean.php）。高风险破坏性操作。 */
import { request } from './client'

/** 删除某类型 days 天前的记录，返回删除条数。target: order/settle/record/transfer/psorder */
export function cleanData(target: string, days: number): Promise<{ deleted: number }> {
  return request('/admin/clean', { method: 'POST', body: { target, days } })
}

/** 清理系统设置缓存（对齐 epay clean.php mod=cleancache）。 */
export function cleanCache(): Promise<{ ok: boolean }> {
  return request('/admin/clean/cache', { method: 'POST' })
}
