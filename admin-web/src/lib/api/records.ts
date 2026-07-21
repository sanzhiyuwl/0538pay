/** 后台资金流水 API（列表 + 统计）。对齐 epay admin/record.php + ajax_user.php recordList/record_stats。 */
import { request, type PageResult } from './client'

/** 资金明细行（对齐后端 dto.RecordView / mock FundRecord） */
export interface FundRecord {
  id: number
  uid: number
  action: 1 | 2 // 1=增加 2=减少
  money: number
  oldmoney: number
  newmoney: number
  type: string
  trade_no: string
  date: string
}

/** 资金明细统计（对齐后端 dto.RecordStats） */
export interface RecordStats {
  incMoney: number
  decMoney: number
  totalMoney: number
  incCount: number
  decCount: number
  totalCount: number
}

export interface RecordListParams {
  page?: number
  pageSize?: number
  uid?: number
  type?: string
  column?: string
  value?: string
  starttime?: string
  endtime?: string
}

/** 拉取后台资金流水（分页 + 筛选） */
export function fetchRecords(params: RecordListParams = {}): Promise<PageResult<FundRecord>> {
  return request<PageResult<FundRecord>>('/admin/records', { query: { ...params } })
}

/** 当前筛选条件下的资金明细统计 */
export function fetchRecordStats(params: RecordListParams = {}): Promise<RecordStats> {
  return request<RecordStats>('/admin/records/stats', { query: { ...params } })
}
