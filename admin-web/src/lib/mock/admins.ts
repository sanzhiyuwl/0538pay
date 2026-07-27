/**
 * 管理员角色标签常量（Admins.vue 新增/编辑管理员时的角色下拉与状态展示）。
 * 角色体系已切真接口，见 @/lib/api/roles；此文件仅保留 role 字符串 ↔ 中文标签映射。
 * 后端管理员 role 为字符串（super/admin/operator/…），与 AdminService 的 role 语义对齐。
 */

/** 管理员状态 → Badge 变体 + 中文。 */
export const adminStatus: Record<number, { text: string; variant: 'success' | 'muted' }> = {
  0: { text: '已禁用', variant: 'muted' },
  1: { text: '正常', variant: 'success' },
}

/** 新增/编辑管理员时可选的角色（super 受保护，不在可选列表）。 */
export const roleOptions = [
  { value: 'admin', label: '管理员' },
  { value: 'operator', label: '运营' },
  { value: 'finance', label: '财务' },
  { value: 'service', label: '客服' },
]

const roleLabelMap: Record<string, string> = {
  super: '超级管理员',
  admin: '管理员',
  operator: '运营',
  finance: '财务',
  service: '客服',
}

/** role 字符串 → 中文标签（未知角色原样返回）。 */
export function roleLabel(role: string): string {
  return roleLabelMap[role] ?? role
}
