/**
 * 管理员 / 角色 / 操作日志假数据（增强项，epay 原版为单管理员无 RBAC）。
 * RBAC：管理员账号 → 角色 → 角色可访问的菜单模块权限。
 */

/** 可分配的权限模块（对齐主后台一级菜单组） */
export const permModules = [
  { key: 'dashboard', label: '平台概况' },
  { key: 'trade', label: '交易管理' },
  { key: 'merchant', label: '商户管理' },
  { key: 'channel', label: '支付接口' },
  { key: 'finance', label: '财务管理' },
  { key: 'risk', label: '风控管理' },
  { key: 'system', label: '系统设置' },
  { key: 'console', label: 'SaaS 控制台' },
]

/** 角色（pre_role） */
export interface Role {
  id: number
  name: string
  desc: string
  permissions: string[] // permModules 的 key，['*'] 表示全部
  builtin: boolean // 内置角色不可删
}

export const roles: Role[] = [
  { id: 1, name: '超级管理员', desc: '拥有全部权限，不可删除', permissions: ['*'], builtin: true },
  { id: 2, name: '运营管理员', desc: '日常交易、商户、风控运营', permissions: ['dashboard', 'trade', 'merchant', 'risk'], builtin: false },
  { id: 3, name: '财务专员', desc: '结算、付款、账单等财务操作', permissions: ['dashboard', 'finance', 'trade'], builtin: false },
  { id: 4, name: '技术运维', desc: '支付接口、系统设置、数据维护', permissions: ['dashboard', 'channel', 'system'], builtin: false },
  { id: 5, name: '客服专员', desc: '只读查看订单与商户', permissions: ['dashboard', 'trade', 'merchant'], builtin: false },
]

/** 管理员（pre_admin） */
export interface Admin {
  id: number
  username: string
  nickname: string
  roleId: number
  status: 0 | 1 // 0=禁用 1=正常
  lastLoginIp: string
  lastLoginTime: string | null
  createTime: string
}

function pad(n: number, len = 2) {
  return String(n).padStart(len, '0')
}

const adminSeed = [
  { username: 'admin', nickname: '超级管理员', roleId: 1 },
  { username: 'operator01', nickname: '王运营', roleId: 2 },
  { username: 'finance01', nickname: '李财务', roleId: 3 },
  { username: 'devops01', nickname: '张运维', roleId: 4 },
  { username: 'service01', nickname: '刘客服', roleId: 5 },
  { username: 'operator02', nickname: '陈运营', roleId: 2 },
]

export const admins: Admin[] = adminSeed.map((a, i) => ({
  id: i + 1,
  username: a.username,
  nickname: a.nickname,
  roleId: a.roleId,
  status: (i === 4 ? 0 : 1) as 0 | 1,
  lastLoginIp: `${100 + i}.${(i * 17) % 255}.${(i * 7) % 255}.${(i * 3) % 255}`,
  lastLoginTime: i === 5 ? null : `2026-07-${pad(12 - i)} ${pad(9 + i)}:${pad((i * 11) % 60)}:00`,
  createTime: `2026-0${1 + (i % 6)}-${pad(1 + i)} 10:00:00`,
}))

export const adminStatus: Record<number, { text: string; variant: 'success' | 'muted' }> = {
  0: { text: '已禁用', variant: 'muted' },
  1: { text: '正常', variant: 'success' },
}

export function roleName(roleId: number) {
  return roles.find((r) => r.id === roleId)?.name ?? '—'
}

/**
 * 后端管理员 role 为字符串（super/admin/…）。下面是新增/编辑管理员时可选的角色，
 * 与后端 AdminService 的 role 语义对齐（super 受保护，其余为运营角色标签）。
 */
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

/** role 字符串 → 中文标签（未知角色原样返回） */
export function roleLabel(role: string): string {
  return roleLabelMap[role] ?? role
}

/** 角色权限展示文案 */
export function rolePermText(role: Role): string {
  if (role.permissions.includes('*')) return '全部权限'
  return role.permissions
    .map((k) => permModules.find((m) => m.key === k)?.label)
    .filter(Boolean)
    .join('、')
}

/** 操作日志（pre_oplog） */
export interface OpLog {
  id: number
  admin: string // 操作管理员
  action: string // 操作类型
  target: string // 操作对象
  ip: string
  date: string
}

/** 操作类型 → Badge 变体（危险操作标红） */
export const opActionVariant: Record<string, 'default' | 'success' | 'warning' | 'destructive'> = {
  新增: 'success',
  修改: 'default',
  审核通过: 'success',
  停用: 'warning',
  拒绝: 'warning',
  删除: 'destructive',
  清理数据: 'destructive',
  改配置: 'default',
}

const opSeed = [
  { action: '删除', target: '订单 202607121234' },
  { action: '审核通过', target: '商户 1008 实名认证' },
  { action: '改配置', target: '风控设置 - 成功率阈值' },
  { action: '停用', target: '支付通道 #7 微信原生备用' },
  { action: '新增', target: '管理员账号 operator02' },
  { action: '清理数据', target: '30天前订单记录' },
  { action: '修改', target: '用户组 VIP商户 费率' },
  { action: '拒绝', target: '授权域名 pay.test.com' },
  { action: '删除', target: '黑名单 #60012' },
  { action: '改配置', target: '结算设置 - 手续费' },
]
const opAdmins = ['admin', '王运营', '李财务', '张运维']

export const opLogs: OpLog[] = Array.from({ length: 30 }, (_, i) => {
  const s = opSeed[i % opSeed.length]
  const day = 12 - (i % 8)
  return {
    id: 90000 + (30 - i),
    admin: opAdmins[i % opAdmins.length],
    action: s.action,
    target: s.target,
    ip: `${100 + (i % 50)}.${(i * 7) % 255}.${(i * 3) % 255}.${i % 255}`,
    date: `2026-07-${pad(day)} ${pad(8 + (i % 12))}:${pad((i * 7) % 60)}:${pad((i * 13) % 60)}`,
  }
})

export const opActionOptions = [
  { value: '', label: '全部操作' },
  ...['新增', '修改', '删除', '审核通过', '拒绝', '停用', '清理数据', '改配置'].map((a) => ({ value: a, label: a })),
]
