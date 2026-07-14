/**
 * 邀请码假数据。对齐 epay admin/invitecode.php（pre_invitecode）。
 * 仅邀请注册模式（reg_open==2）下启用：批量生成邀请码，注册时填写核销。
 * pre_invitecode 字段：code / status（0未用 1已用）/ addtime / usetime / uid（使用者）。
 */

/** 邀请码（pre_invitecode） */
export interface InviteCode {
  id: number
  code: string // 邀请码
  status: 0 | 1 // 0=未使用 1=已使用
  addtime: string // 生成时间
  usetime: string | null // 使用时间（未用为 null）
  uid: number | null // 使用者商户号（未用为 null）
}

/** 状态字典（对齐 invitecode.php formatter：已使用红 / 未使用绿） */
export const inviteStatus: Record<number, { text: string; variant: 'success' | 'default' }> = {
  0: { text: '未使用', variant: 'success' },
  1: { text: '已使用', variant: 'default' },
}

export const statusOptions = [
  { value: -1, label: '全部状态' },
  { value: 0, label: '未使用' },
  { value: 1, label: '已使用' },
]

function pad(n: number, len = 2) {
  return String(n).padStart(len, '0')
}

// 8 位随机码字符集（对齐 epay getkm / random(8)）
const CHARS = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'
// 用索引确定性地生成码，避免 Math.random（构建期/SSR 友好，且刷新稳定）
function genCode(seed: number): string {
  let s = ''
  for (let i = 0; i < 8; i++) {
    s += CHARS[(seed * 31 + i * 17 + 7) % CHARS.length]
  }
  return s
}

// 生成 36 条邀请码：约 1/3 已使用
export const inviteCodes: InviteCode[] = Array.from({ length: 36 }, (_, i) => {
  const used = i % 3 === 0
  const day = 12 - (i % 8)
  const useDay = day + (i % 3)
  return {
    id: 90000 + (36 - i),
    code: genCode(90000 + (36 - i)),
    status: (used ? 1 : 0) as 0 | 1,
    addtime: `2026-07-${pad(day)} ${pad(9 + (i % 10))}:${pad((i * 11) % 60)}:${pad((i * 17) % 60)}`,
    usetime: used ? `2026-07-${pad(useDay)} ${pad(10 + (i % 8))}:${pad((i * 13) % 60)}:${pad((i * 7) % 60)}` : null,
    uid: used ? 1000 + (i % 20) + 1 : null,
  }
})

/** 汇总统计 */
export function calcInviteStats(list: InviteCode[]) {
  return {
    total: list.length,
    used: list.filter((c) => c.status === 1).length,
    unused: list.filter((c) => c.status === 0).length,
  }
}
