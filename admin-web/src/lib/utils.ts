import { type ClassValue, clsx } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

/** 金额格式化：分位符 + 两位小数 */
export function formatMoney(v: number | string): string {
  const n = typeof v === 'string' ? parseFloat(v) : v
  if (isNaN(n)) return '0.00'
  return n.toLocaleString('zh-CN', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  })
}

/** 大数字缩写：12345 -> 1.23万 */
export function formatCompact(v: number): string {
  if (v >= 1e8) return (v / 1e8).toFixed(2) + '亿'
  if (v >= 1e4) return (v / 1e4).toFixed(2) + '万'
  return v.toLocaleString('zh-CN')
}
